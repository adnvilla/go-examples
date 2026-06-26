//nolint:staticcheck // legacy AWS SDK v1 example; SA4006 false-positives on err reassignment chains
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type ItemInfo struct {
	Plot   string  `json:"plot"`
	Rating float64 `json:"rating"`
}

type Item struct {
	Year  int      `json:"year"`
	Title string   `json:"title"`
	Info  ItemInfo `json:"info"`
}

func GetSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String("us-west-1"),
		Endpoint: aws.String("http://localhost:8000"),
	})
	return sess, err
}

func GetDynamoDB() *dynamodb.DynamoDB {
	sess, err := GetSession()

	if err != nil {
		fmt.Println("Error creating session:")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// Create DynamoDB client
	svc := dynamodb.New(sess)
	return svc
}

func requireLocalDynamo(t *testing.T) {
	t.Helper()
	if os.Getenv("DYNAMODB_LOCAL") == "" {
		t.Skip("set DYNAMODB_LOCAL=1 to run DynamoDB integration tests (requires local DynamoDB on :8000)")
	}
}

func TestListAlltables(t *testing.T) {
	requireLocalDynamo(t)

	sess, err := GetSession()

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	result, err := svc.ListTables(&dynamodb.ListTablesInput{})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Tables:")
	fmt.Println("")

	for _, n := range result.TableNames {
		fmt.Println(*n)
	}

}

func TestCreateTable(t *testing.T) {
	requireLocalDynamo(t)
	sess, err := GetSession()

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("year"),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String("title"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("year"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("title"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		TableName: aws.String("Movies"),
	}

	_, err = svc.CreateTable(input)

	if err != nil {
		fmt.Println("Got error calling CreateTable:")
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Created the table Movies in us-west-1")
}

func TestCreateItem(t *testing.T) {
	requireLocalDynamo(t)
	sess, err := GetSession()
	// Create DynamoDB client
	svc := dynamodb.New(sess)

	info := ItemInfo{
		Plot:   "Nothing happens at all.",
		Rating: 0.0,
	}

	item := Item{
		Year:  2015,
		Title: "The Big New Movie",
		Info:  info,
	}

	av, err := dynamodbattribute.MarshalMap(item)
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("Movies"),
	}

	_, err = svc.PutItem(input)

	if err != nil {
		fmt.Println("Got error calling PutItem:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Successfully added 'The Big New Movie' (2015) to Movies table")

}

func getItems() []Item {
	raw, err := os.ReadFile("./movie_data.json")

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var items []Item
	if err := json.Unmarshal(raw, &items); err != nil {
		fmt.Println("error parsing movie data:", err)
		os.Exit(1)
	}
	return items
}
func TestCreateItems(t *testing.T) {
	requireLocalDynamo(t)

	svc := GetDynamoDB()

	items := getItems()

	// Add each item to Movies table:
	for _, item := range items {
		av, err := dynamodbattribute.MarshalMap(item)

		if err != nil {
			fmt.Println("Got error marshalling map:")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// Create item in table Movies
		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String("Movies"),
		}

		_, err = svc.PutItem(input)

		if err != nil {
			fmt.Println("Got error calling PutItem:")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Println("Successfully added '", item.Title, "' (", item.Year, ") to Movies table")
	}
}

func TestReadItem(t *testing.T) {
	requireLocalDynamo(t)

	svc := GetDynamoDB()
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("Movies"),
		Key: map[string]*dynamodb.AttributeValue{
			"year": {
				N: aws.String("2015"),
			},
			"title": {
				S: aws.String("The Big New Movie"),
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	item := Item{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)

	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	if item.Title == "" {
		fmt.Println("Could not find 'The Big New Movie' (2015)")
		return
	}

	fmt.Println("Found item:")
	fmt.Println("Year:  ", item.Year)
	fmt.Println("Title: ", item.Title)
	fmt.Println("Plot:  ", item.Info.Plot)
	fmt.Println("Rating:", item.Info.Rating)

}

func TestReadItems(t *testing.T) {
	requireLocalDynamo(t)
	min_rating := 1.0
	year := 2011

	svc := GetDynamoDB()

	// Create the Expression to fill the input struct with.
	// Get all movies in that year; we'll pull out those with a higher rating later
	// filt := expression.Name("year").Equal(expression.Value(year))

	// Or we could get by ratings and pull out those with the right year later
	// filt := expression.Name("info.rating").GreaterThan(expression.Value(min_rating))
	filter := FilterGreaterThan("info.rating", min_rating)

	// Get back the title, year, and rating
	// proj := expression.NamesList(expression.Name("title"), expression.Name("year"), expression.Name("info.rating"))
	proj := ProjectionNames("title", "year", "info.rating")

	// expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
	expr, err := expression.NewBuilder().
		WithFilter(filter).
		WithProjection(proj).Build()

	if err != nil {
		fmt.Println("Got error building expression:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String("Movies"),
	}

	// Make the DynamoDB Query API call
	result, err := svc.Scan(params)

	if err != nil {
		fmt.Println("Query API call failed:")
		fmt.Println((err.Error()))
		os.Exit(1)
	}

	num_items := 0

	for _, i := range result.Items {
		item := Item{}

		err = dynamodbattribute.UnmarshalMap(i, &item)

		if err != nil {
			fmt.Println("Got error unmarshalling:")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// Which ones had a higher rating?
		// if item.Info.Rating > min_rating {
		// Or it we had filtered by rating previously:
		//   if item.Year == year {
		num_items += 1

		fmt.Println("Title: ", item.Title)
		fmt.Println("Year: ", item.Year)
		fmt.Println("Rating:", item.Info.Rating)
		fmt.Println()
		// }
	}

	fmt.Println("Found", num_items, "movie(s) with a rating above", min_rating, "in", year)

}
func TestUpdateItem(t *testing.T) {
	requireLocalDynamo(t)
	svc := GetDynamoDB()

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":r": {
				N: aws.String("0.5"),
			},
		},
		TableName: aws.String("Movies"),
		Key: map[string]*dynamodb.AttributeValue{
			"year": {
				N: aws.String("2015"),
			},
			"title": {
				S: aws.String("The Big New Movie"),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set info.rating = :r"),
	}

	_, err := svc.UpdateItem(input)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Successfully updated 'The Big New Movie' (2015) rating to 0.5")

}

func TestDeleteItem(t *testing.T) {
	requireLocalDynamo(t)
	svc := GetDynamoDB()

	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"year": {
				N: aws.String("2015"),
			},
			"title": {
				S: aws.String("The Big New Movie"),
			},
		},
		TableName: aws.String("Movies"),
	}

	_, err := svc.DeleteItem(input)

	if err != nil {
		fmt.Println("Got error calling DeleteItem")
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Deleted 'The Big New Movie' (2015)")

}

func TestDeleteTable(t *testing.T) {
	requireLocalDynamo(t)
	svc := GetDynamoDB()

	input := &dynamodb.DeleteTableInput{
		TableName: aws.String("Movies"),
	}

	_, err := svc.DeleteTable(input)

	if err != nil {
		fmt.Println("Got error calling DeleteTable:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Delete the table Movies in us-west-1")
}
