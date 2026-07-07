# Lambda

**Category:** cloud
**Difficulty:** Beginner

## Objective

Show the minimal shape of an AWS Lambda function written in Go: a handler function matching one of `aws-lambda-go`'s supported signatures, registered with `lambda.Start`, plus reading configuration from a Lambda environment variable.

## Concepts Covered

- `lambda.Start(handlerFunc)` — hands control to the Lambda runtime, which invokes `handlerFunc` once per incoming event
- A handler with signature `func() (string, error)` — one of several signatures `aws-lambda-go` supports (others take an event/context parameter)
- Reading a Lambda-configured environment variable (`os.Getenv("V1")`) — how Lambda functions typically receive configuration, since there's no CLI or config file at invocation time
- Why this program fails immediately when run outside an actual Lambda environment (see Expected Output)

## Prerequisites

- Go 1.25+
- No external services needed to **build** the example
- Actually **invoking** it requires either deploying to AWS Lambda, or a local Lambda emulator (e.g. the [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)); an AWS account/credentials are needed to deploy

## Project Structure

```
lambda/
├── go.mod
├── main.go
└── README.md
```

## How to Run

Running it directly (`go run .` / `make run`) is expected to fail — see Expected Output. To actually deploy it to Lambda:

```bash
make build-lambda   # cross-compiles for linux/amd64 and produces main.zip
```

Then upload `main.zip` to a Lambda function using the `provided.al2` or `provided.al2023` custom runtime (Go's dedicated `go1.x` managed runtime has been deprecated by AWS in favor of this), with the handler set to `bootstrap`.

## Expected Output

Running the binary directly, outside a real Lambda environment:
```
2026/07/05 17:33:12 expected AWS Lambda environment variables [_LAMBDA_SERVER_PORT AWS_LAMBDA_RUNTIME_API] are not defined
```

This is `lambda.Start` correctly detecting it isn't running inside the Lambda runtime, and failing fast rather than hanging. Once actually deployed and invoked (with `V1=some-value` set as a Lambda environment variable), the handler returns:
```
Hello ƛ! some-value
```

## Code Walkthrough

- `hello() (string, error)` is the handler — it takes no event parameter (a valid `aws-lambda-go` handler signature for functions that don't need input), reads the `V1` environment variable, and returns a greeting string plus a `nil` error.
- `lambda.Start(hello)` is the entire `main` function's job: register `hello` and hand control to the Lambda runtime's event loop. `lambda.Start` never returns under normal operation inside Lambda — it blocks, waiting for and dispatching invocations for the lifetime of the execution environment.
- Outside Lambda, `lambda.Start` checks for the environment variables the real runtime always sets (`AWS_LAMBDA_RUNTIME_API`, `_LAMBDA_SERVER_PORT`) and exits immediately with an error if they're missing — which is what running this locally demonstrates.
- Configuration via environment variable (`V1`) is the standard way Lambda functions receive settings, since there's no command line or config file involved in an invocation.

## Common Pitfalls

- **Expecting `go run .` to actually invoke the handler locally.** It can't — there's no Lambda runtime API to talk to outside AWS (or a local emulator like SAM CLI). The clear failure message is a feature, not a bug.
- **Naming the compiled binary anything other than `bootstrap` for the `provided.al2`/`provided.al2023` runtimes.** These custom runtimes specifically look for an executable named `bootstrap` in the deployment package; the older `go1.x` managed runtime (deprecated) used different conventions (e.g. a binary named `main`).
- **Forgetting to cross-compile.** Lambda runs on Linux/amd64 (or arm64) regardless of what OS you develop on — `GOOS=linux GOARCH=amd64 go build` (as `make build-lambda` does) is required when building from macOS or Windows.
- **Assuming the handler must take an event parameter.** `aws-lambda-go` supports handlers with zero, one, or two parameters (event and/or `context.Context`) — a zero-parameter handler like `hello` is valid when the function doesn't need any input.

## References

- [aws-lambda-go GitHub repository](https://github.com/aws/aws-lambda-go)
- [AWS Lambda — Building Lambda functions with Go](https://docs.aws.amazon.com/lambda/latest/dg/lambda-golang.html)
- [AWS Lambda — Handler function signatures](https://docs.aws.amazon.com/lambda/latest/dg/golang-handler.html)
- [AWS Lambda — Environment variables](https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html)

## Next Steps

- [http-server](../http-server/) — a "regular" long-running server, for contrast with Lambda's per-invocation execution model
- [config](../config/) — an alternative configuration source (a file) instead of environment variables
