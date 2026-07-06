# DynamoDB

**Category:** database
**Difficulty:** Advanced

## Objective

Show a full DynamoDB CRUD cycle with AWS SDK v1: create a table, put items (individually and from a JSON fixture), read one item and scan/filter many, update, delete an item, and drop the table — plus a small expression-builder helper for constructing filter/projection expressions.

## Concepts Covered

- `session.NewSession` pointed at a custom `Endpoint` (`http://localhost:8000`) — the standard way to redirect the AWS SDK at DynamoDB Local instead of real AWS
- `dynamodbattribute.MarshalMap`/`UnmarshalMap` — converting between Go structs and DynamoDB's attribute-value map representation
- `expression.NewBuilder().WithFilter(...).WithProjection(...).Build()` — constructing a `Scan` filter and projection without hand-building DynamoDB's expression syntax
- Sequential, stateful integration tests: each `Test*` function depends on table/item state left behind by the previous one (Go runs tests in file-declaration order, not parallel or randomized by default)

## Prerequisites

- Go 1.24+
- A running DynamoDB Local instance (this repo's `docker-compose.yml` provides one on `localhost:8000`):
  ```bash
  docker compose up -d dynamodb
  # or, from this directory:
  make infra-up
  ```
- **Dummy AWS credentials.** DynamoDB Local doesn't validate credentials, but the AWS SDK's credential chain still errors out (`NoCredentialProviders`) if it can't find *any* — set placeholder values:
  ```bash
  export AWS_ACCESS_KEY_ID=dummy
  export AWS_SECRET_ACCESS_KEY=dummy
  ```
- Tests only run with `DYNAMODB_LOCAL=1` set; without it, they're skipped (`t.Skip`)

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
make test       # sets DYNAMODB_LOCAL=1 and dummy AWS creds automatically
```

## Expected Output

```
=== RUN   TestListAlltables
Tables:

--- PASS: TestListAlltables (0.13s)
=== RUN   TestCreateTable
Created the table Movies in us-west-1
--- PASS: TestCreateTable (0.06s)
=== RUN   TestCreateItem
Successfully added 'The Big New Movie' (2015) to Movies table
--- PASS: TestCreateItem (0.03s)
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

Found 1 movie(s) with a rating above 1 in 2011
--- PASS: TestReadItems (0.04s)
=== RUN   TestUpdateItem
Successfully updated 'The Big New Movie' (2015) rating to 0.5
--- PASS: TestUpdateItem (0.01s)
=== RUN   TestDeleteItem
Deleted 'The Big New Movie' (2015)
--- PASS: TestDeleteItem (0.01s)
=== RUN   TestDeleteTable
Delete the table Movies in us-west-1
--- PASS: TestDeleteTable (0.01s)
PASS
```

## Code Walkthrough

- `GetSession`/`GetDynamoDB` build an SDK session pointed at `http://localhost:8000` — this single line (the custom `Endpoint`) is what redirects every subsequent call away from real AWS to the local Docker container.
- The tests run in a deliberate sequence (Go executes `Test*` functions in the order they appear in the file): `TestCreateTable` creates the `Movies` table, `TestCreateItem`/`TestCreateItems` populate it (one hardcoded item, then everything in `movie_data.json`), `TestReadItem` fetches a specific item by its composite key (`year` + `title`), `TestReadItems` scans with a filter and projection, `TestUpdateItem` and `TestDeleteItem` mutate and remove an item, and `TestDeleteTable` cleans up — leaving the database in the same empty state it started in, so the whole suite can run again.
- `FilterGreaterThan`/`ProjectionNames` (in `main.go`) are small helpers wrapping the `expression` package's builder API: the former builds a `ConditionBuilder` for "field greater than value," the latter builds a `ProjectionBuilder` listing which fields a `Scan` should return.
- `TestReadItems` combines both: filters for movies with `info.rating > 1.0`, projects only `title`/`year`/`info.rating`, and scans the table — demonstrating that DynamoDB's `Scan` filters happen *after* reading all items (unlike a SQL `WHERE` clause pushed into an index), so filters reduce what's returned, not what's read.

## Common Pitfalls

- **Running the tests without DynamoDB Local up, or without `DYNAMODB_LOCAL=1`.** Each test calls `requireLocalDynamo(t)` first, which skips (not fails) the test if the env var isn't set — a clean way to keep these integration tests out of a normal `go test ./...` run.
- **Missing AWS credentials, even for a local-only endpoint.** The SDK's credential chain check happens before any request is sent, regardless of whether the endpoint is real AWS or a local emulator — dummy credentials are required either way.
- **Assuming test order is guaranteed across files or packages.** It's guaranteed *within* a single file (declaration order), which is what this suite relies on — this pattern doesn't generalize to tests split across multiple files without additional coordination.
- **Interrupting the suite partway through (e.g., `Ctrl+C` mid-run).** Since later tests clean up what earlier ones created, stopping early can leave the `Movies` table in place — rerunning from `TestCreateTable` would then fail on "table already exists." `docker compose restart dynamodb` resets the in-memory store entirely.
- **This uses AWS SDK v1** (`github.com/aws/aws-sdk-go`), which is deprecated in favor of `aws-sdk-go-v2` — kept here since it's what the original example used; new code should target v2.

## References

- [AWS SDK for Go v1 — DynamoDB API docs](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/)
- [DynamoDB Local (Docker)](https://hub.docker.com/r/amazon/dynamodb-local)
- [AWS SDK for Go — expression package](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/expression/)

## Next Steps

- [mysql](../mysql/) — a relational-database counterpart, for comparison
- [serialization](../serialization/) — more on marshaling data into and out of Go structs
