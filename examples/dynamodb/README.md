# DynamoDB

**Category:** database
**Difficulty:** Advanced

## Objective

Show a full DynamoDB CRUD cycle with AWS SDK v2: create a table, put items (individually and from a JSON fixture), read one item and scan/filter many, update, delete an item, and drop the table — plus a small expression-builder helper for constructing filter/projection expressions.

## Concepts Covered

- `config.LoadDefaultConfig` + `dynamodb.NewFromConfig` with a `BaseEndpoint` override (`http://localhost:8000`) — the standard way to redirect SDK v2 at DynamoDB Local instead of real AWS
- `credentials.NewStaticCredentialsProvider` — wiring dummy credentials in code so the local-only example needs no `AWS_*` environment variables
- `attributevalue.MarshalMap`/`UnmarshalMap` — converting between Go structs and DynamoDB's attribute-value map representation, driven by `dynamodbav` struct tags
- Typed attribute values (`types.AttributeValueMemberN`, `types.AttributeValueMemberS`) — v2's compile-time-safe replacement for v1's pointer-heavy `*dynamodb.AttributeValue`
- `expression.NewBuilder().WithFilter(...).WithProjection(...).Build()` — constructing a `Scan` filter and projection without hand-building DynamoDB's expression syntax
- `t.Context()` (Go 1.24) — a per-test `context.Context` that is canceled when the test ends
- Sequential, stateful integration tests: each `Test*` function depends on table/item state left behind by the previous one (Go runs tests in file-declaration order, not parallel or randomized by default)

## Prerequisites

- Go 1.25+
- A running DynamoDB Local instance (this repo's `docker-compose.yml` provides one on `localhost:8000`):
  ```bash
  docker compose up -d dynamodb
  # or, from this directory:
  make infra-up
  ```
- Tests only run with `DYNAMODB_LOCAL=1` set; without it, they're skipped (`t.Skip`). No AWS credentials needed — dummy static credentials are wired into the client in code.

## Project Structure

```
dynamodb/
├── go.mod
├── main.go
├── main_test.go
├── movie_data.json
└── README.md
```

## How to Run

This example's logic lives entirely in its integration tests (`main.go`'s `main` is empty) — there's nothing to `go run`. Its tests are the demonstration:

```bash
make infra-up   # start DynamoDB Local
make test       # sets DYNAMODB_LOCAL=1 automatically
```

## Expected Output

```
=== RUN   TestListAllTables
Tables:

--- PASS: TestListAllTables (0.01s)
=== RUN   TestCreateTable
Created the table Movies in us-west-1
--- PASS: TestCreateTable (0.01s)
=== RUN   TestCreateItem
Successfully added 'The Big New Movie' (2015) to Movies table
--- PASS: TestCreateItem (0.01s)
=== RUN   TestCreateItems
Successfully added ' Turn It Down, Or Else! ' ( 2013 ) to Movies table
Successfully added ' The Big New Movie ' ( 2015 ) to Movies table
--- PASS: TestCreateItems (0.01s)
=== RUN   TestReadItem
Found item:
Year:   2015
Title:  The Big New Movie
Plot:   Nothing happens at all.
Rating: 0
--- PASS: TestReadItem (0.01s)
=== RUN   TestReadItems
Title:  Turn It Down, Or Else!
Year:  2013
Rating: 6.2

Found 1 movie(s) with a rating above 1
--- PASS: TestReadItems (0.01s)
=== RUN   TestUpdateItem
Successfully updated 'The Big New Movie' (2015) rating to 0.5
--- PASS: TestUpdateItem (0.01s)
=== RUN   TestDeleteItem
Deleted 'The Big New Movie' (2015)
--- PASS: TestDeleteItem (0.01s)
=== RUN   TestDeleteTable
Deleted the table Movies in us-west-1
--- PASS: TestDeleteTable (0.00s)
PASS
```

## Code Walkthrough

- `requireLocalDynamo` does double duty: it skips the test unless `DYNAMODB_LOCAL=1`, then builds the client with `config.LoadDefaultConfig` and points it at the Docker container via the `BaseEndpoint` option — that single option is what redirects every call away from real AWS. Static dummy credentials satisfy the SDK's credential chain (DynamoDB Local never validates them, but the chain errors out if it finds none at all).
- The tests run in a deliberate sequence (Go executes `Test*` functions in the order they appear in the file): `TestCreateTable` creates the `Movies` table, `TestCreateItem`/`TestCreateItems` populate it (one hardcoded item, then everything in `movie_data.json`), `TestReadItem` fetches a specific item by its composite key (`year` + `title`), `TestReadItems` scans with a filter and projection, `TestUpdateItem` and `TestDeleteItem` mutate and remove an item, and `TestDeleteTable` cleans up — leaving the database in the same empty state it started in, so the whole suite can run again.
- Keys and expression values are built with v2's typed attribute values (`&types.AttributeValueMemberN{Value: "2015"}`) instead of v1's `*dynamodb.AttributeValue` with a dozen optional pointer fields — invalid combinations no longer compile.
- `FilterGreaterThan`/`ProjectionNames` (in `main.go`) are small helpers wrapping the `expression` package's builder API: the former builds a `ConditionBuilder` for "field greater than value," the latter builds a `ProjectionBuilder` listing which fields a `Scan` should return.
- `TestReadItems` combines both: filters for movies with `info.rating > 1.0`, projects only `title`/`year`/`info.rating`, and scans the table — demonstrating that DynamoDB's `Scan` filters happen *after* reading all items (unlike a SQL `WHERE` clause pushed into an index), so filters reduce what's returned, not what's read.

## Common Pitfalls

- **`dynamodbav` tags are not optional in v2.** v1's `dynamodbattribute` fell back to `json` tags; v2's `attributevalue` only honors `dynamodbav` tags. With `json` tags alone, items marshal under Go field names (`Year`, `Title`) and `PutItem` fails with `ValidationException: One of the required keys was not given a value`.
- **Running the tests without DynamoDB Local up, or without `DYNAMODB_LOCAL=1`.** Each test calls `requireLocalDynamo(t)` first, which skips (not fails) the test if the env var isn't set — a clean way to keep these integration tests out of a normal `go test ./...` run.
- **Missing AWS credentials, even for a local-only endpoint.** The SDK's credential chain check happens before any request is sent, regardless of whether the endpoint is real AWS or a local emulator — this example sidesteps it with `credentials.NewStaticCredentialsProvider("dummy", "dummy", "")`.
- **Assuming test order is guaranteed across files or packages.** It's guaranteed *within* a single file (declaration order), which is what this suite relies on — this pattern doesn't generalize to tests split across multiple files without additional coordination.
- **Interrupting the suite partway through (e.g., `Ctrl+C` mid-run).** Since later tests clean up what earlier ones created, stopping early can leave the `Movies` table in place — rerunning from `TestCreateTable` would then fail on "table already exists." `docker compose restart dynamodb` resets the in-memory store entirely.

## References

- [AWS SDK for Go v2 — DynamoDB client](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/dynamodb)
- [AWS SDK for Go v2 — attributevalue package](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue)
- [AWS SDK for Go v2 — expression package](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression)
- [DynamoDB Local (Docker)](https://hub.docker.com/r/amazon/dynamodb-local)
- [Migrating from v1 to v2 of the AWS SDK for Go](https://aws.github.io/aws-sdk-go-v2/docs/migrating/)

## Next Steps

- [mysql](../mysql/) — a relational-database counterpart, for comparison
- [serialization](../serialization/) — more on marshaling data into and out of Go structs
