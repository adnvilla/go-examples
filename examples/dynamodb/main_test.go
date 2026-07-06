package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Unlike SDK v1's dynamodbattribute, v2's attributevalue does not fall back
// to `json` tags — attribute names come from `dynamodbav` tags only.
type ItemInfo struct {
	Plot   string  `json:"plot" dynamodbav:"plot"`
	Rating float64 `json:"rating" dynamodbav:"rating"`
}

type Item struct {
	Year  int      `json:"year" dynamodbav:"year"`
	Title string   `json:"title" dynamodbav:"title"`
	Info  ItemInfo `json:"info" dynamodbav:"info"`
}

// requireLocalDynamo skips the test unless DYNAMODB_LOCAL=1, then returns a
// client pointed at DynamoDB Local. DynamoDB Local ignores credentials, but
// the SDK's credential chain still needs *some*, so static dummies are wired
// in here rather than via environment variables.
func requireLocalDynamo(t *testing.T) *dynamodb.Client {
	t.Helper()
	if os.Getenv("DYNAMODB_LOCAL") == "" {
		t.Skip("set DYNAMODB_LOCAL=1 to run DynamoDB integration tests (requires local DynamoDB on :8000)")
	}

	cfg, err := config.LoadDefaultConfig(t.Context(),
		config.WithRegion("us-west-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")),
	)
	if err != nil {
		t.Fatalf("loading AWS config: %v", err)
	}

	return dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:8000")
	})
}

func TestListAllTables(t *testing.T) {
	svc := requireLocalDynamo(t)

	result, err := svc.ListTables(t.Context(), &dynamodb.ListTablesInput{})
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}

	fmt.Println("Tables:")
	fmt.Println("")

	for _, n := range result.TableNames {
		fmt.Println(n)
	}
}

func TestCreateTable(t *testing.T) {
	svc := requireLocalDynamo(t)

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("year"),
				AttributeType: types.ScalarAttributeTypeN,
			},
			{
				AttributeName: aws.String("title"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("year"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("title"),
				KeyType:       types.KeyTypeRange,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		TableName: aws.String("Movies"),
	}

	if _, err := svc.CreateTable(t.Context(), input); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	fmt.Println("Created the table Movies in us-west-1")
}

func TestCreateItem(t *testing.T) {
	svc := requireLocalDynamo(t)

	item := Item{
		Year:  2015,
		Title: "The Big New Movie",
		Info: ItemInfo{
			Plot:   "Nothing happens at all.",
			Rating: 0.0,
		},
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		t.Fatalf("marshalling item: %v", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("Movies"),
	}

	if _, err := svc.PutItem(t.Context(), input); err != nil {
		t.Fatalf("PutItem: %v", err)
	}

	fmt.Println("Successfully added 'The Big New Movie' (2015) to Movies table")
}

func getItems(t *testing.T) []Item {
	t.Helper()

	raw, err := os.ReadFile("./movie_data.json")
	if err != nil {
		t.Fatalf("reading movie data: %v", err)
	}

	var items []Item
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("parsing movie data: %v", err)
	}
	return items
}

func TestCreateItems(t *testing.T) {
	svc := requireLocalDynamo(t)

	items := getItems(t)

	// Add each item to Movies table:
	for _, item := range items {
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			t.Fatalf("marshalling item: %v", err)
		}

		// Create item in table Movies
		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String("Movies"),
		}

		if _, err := svc.PutItem(t.Context(), input); err != nil {
			t.Fatalf("PutItem: %v", err)
		}

		fmt.Println("Successfully added '", item.Title, "' (", item.Year, ") to Movies table")
	}
}

func TestReadItem(t *testing.T) {
	svc := requireLocalDynamo(t)

	result, err := svc.GetItem(t.Context(), &dynamodb.GetItemInput{
		TableName: aws.String("Movies"),
		Key: map[string]types.AttributeValue{
			"year":  &types.AttributeValueMemberN{Value: "2015"},
			"title": &types.AttributeValueMemberS{Value: "The Big New Movie"},
		},
	})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	item := Item{}
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		t.Fatalf("unmarshalling record: %v", err)
	}

	if item.Title == "" {
		t.Fatal("could not find 'The Big New Movie' (2015)")
	}

	fmt.Println("Found item:")
	fmt.Println("Year:  ", item.Year)
	fmt.Println("Title: ", item.Title)
	fmt.Println("Plot:  ", item.Info.Plot)
	fmt.Println("Rating:", item.Info.Rating)
}

func TestReadItems(t *testing.T) {
	svc := requireLocalDynamo(t)

	minRating := 1.0

	// Filter for movies rated above minRating; project back only the
	// attributes the demonstration prints.
	filter := FilterGreaterThan("info.rating", minRating)
	proj := ProjectionNames("title", "year", "info.rating")

	expr, err := expression.NewBuilder().
		WithFilter(filter).
		WithProjection(proj).Build()
	if err != nil {
		t.Fatalf("building expression: %v", err)
	}

	// Build the scan input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String("Movies"),
	}

	result, err := svc.Scan(t.Context(), params)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	numItems := 0

	for _, i := range result.Items {
		item := Item{}
		if err := attributevalue.UnmarshalMap(i, &item); err != nil {
			t.Fatalf("unmarshalling record: %v", err)
		}

		numItems++

		fmt.Println("Title: ", item.Title)
		fmt.Println("Year: ", item.Year)
		fmt.Println("Rating:", item.Info.Rating)
		fmt.Println()
	}

	fmt.Println("Found", numItems, "movie(s) with a rating above", minRating)
}

func TestUpdateItem(t *testing.T) {
	svc := requireLocalDynamo(t)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":r": &types.AttributeValueMemberN{Value: "0.5"},
		},
		TableName: aws.String("Movies"),
		Key: map[string]types.AttributeValue{
			"year":  &types.AttributeValueMemberN{Value: "2015"},
			"title": &types.AttributeValueMemberS{Value: "The Big New Movie"},
		},
		ReturnValues:     types.ReturnValueUpdatedNew,
		UpdateExpression: aws.String("set info.rating = :r"),
	}

	if _, err := svc.UpdateItem(t.Context(), input); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}

	fmt.Println("Successfully updated 'The Big New Movie' (2015) rating to 0.5")
}

func TestDeleteItem(t *testing.T) {
	svc := requireLocalDynamo(t)

	input := &dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			"year":  &types.AttributeValueMemberN{Value: "2015"},
			"title": &types.AttributeValueMemberS{Value: "The Big New Movie"},
		},
		TableName: aws.String("Movies"),
	}

	if _, err := svc.DeleteItem(t.Context(), input); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	fmt.Println("Deleted 'The Big New Movie' (2015)")
}

func TestDeleteTable(t *testing.T) {
	svc := requireLocalDynamo(t)

	input := &dynamodb.DeleteTableInput{
		TableName: aws.String("Movies"),
	}

	if _, err := svc.DeleteTable(t.Context(), input); err != nil {
		t.Fatalf("DeleteTable: %v", err)
	}

	fmt.Println("Deleted the table Movies in us-west-1")
}
