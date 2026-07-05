# Protobuf

**Category:** serialization
**Difficulty:** Intermediate

## Objective

Show binary serialization with Protocol Buffers: define a message schema in a `.proto` file, generate Go types from it, then marshal/unmarshal those types to/from bytes — round-tripping through a file on disk.

## Concepts Covered

- A `.proto` schema (`addressbook.proto`) defining `Person` (with a nested `PhoneNumber` message and `PhoneType` enum) and `AddressBook`
- Generated Go types (`addressbook.pb.go`) from that schema, via `protoc` + the Go protobuf plugin
- `proto.Marshal`/`proto.Unmarshal` to convert between Go structs and their binary wire format
- Why this example still imports the older `github.com/golang/protobuf` compatibility wrapper instead of `google.golang.org/protobuf` directly (see Common Pitfalls)

## Prerequisites

- Go 1.24+
- No external services or environment variables required to **run** the example
- Regenerating `addressbook.pb.go` from `addressbook.proto` requires the [protoc compiler](https://github.com/protocolbuffers/protobuf/releases) and the Go plugin (`protoc-gen-go`) on `PATH`

## Project Structure

```
protobuf/
├── addressbook.proto   (schema)
├── addressbook.pb.go   (generated — do not hand-edit)
├── go.mod
├── main.go
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

To regenerate `addressbook.pb.go` after changing `addressbook.proto` (requires `protoc` + `protoc-gen-go`):
```bash
make generate
# or
protoc -I=./ --go_out=./ ./addressbook.proto
```

## Expected Output

```
people:<name:"Alice" email:"alice@example.com" > people:<name:"Bob" email:"bob@example.com" >
```

(Protobuf's text format omits zero-value fields — `Id` isn't shown for either person since it defaults to `0` and was never set.)

## Code Walkthrough

- `addressbook.proto` declares the schema: a `Person` message with `name`, `id`, `email`, and a `repeated PhoneNumber phones` field (itself a nested message with a `PhoneType` enum) — plus an `AddressBook` message wrapping `repeated Person people`.
- `addressbook.pb.go` is what `protoc` generated from that schema: Go structs (`Person`, `AddressBook`, `Person_PhoneNumber`) with the right field types, plus all the boilerplate (`Reset`, `String`, `ProtoReflect`, etc.) protobuf's runtime needs.
- `main` builds an `*AddressBook` with two `*Person` values (only `Name`/`Email` set — `Id` stays at its zero value), marshals it to bytes with `proto.Marshal`, writes those bytes to `test.dat`, then reads the file back and unmarshals it into a fresh `*AddressBook` to prove the round-trip works.
- `fmt.Println(book2)` uses `AddressBook`'s generated `String()` method, which renders protobuf's text format — a debugging-oriented representation, not the wire format itself (which is binary and not human-readable).

## Common Pitfalls

- **This file imports `github.com/golang/protobuf/proto`, not `google.golang.org/protobuf`.** The former is a thin compatibility wrapper Google maintains around the latter (the actual, current protobuf-go implementation) — it's kept here because the generated `addressbook.pb.go` predates the newer API and regenerating it requires the `protoc` toolchain, which isn't assumed to be installed. New protobuf code should generate against and import `google.golang.org/protobuf` directly.
- **Hand-editing `addressbook.pb.go`.** Like `wire_gen.go` in [wire](../wire/), it's generated — any manual change is lost (and can silently diverge from the schema) the next time `protoc` regenerates it.
- **Assuming the marshaled bytes are human-readable.** Protobuf's wire format is a dense binary encoding; only `String()` (used for debugging, as `main` does) or explicit JSON marshaling (via `protojson`) produce readable text.
- **`test.dat` is a runtime artifact, not source.** Every run of this program overwrites it — it doesn't belong in version control (previously committed by mistake; removed during this migration).

## References

- [Protocol Buffers — Go Tutorial](https://protobuf.dev/getting-started/gotutorial/)
- [google.golang.org/protobuf package docs](https://pkg.go.dev/google.golang.org/protobuf)
- [Protocol Buffers releases (protoc compiler)](https://github.com/protocolbuffers/protobuf/releases)

## Next Steps

- [serialization](../serialization/) — JSON-based serialization, contrasted with protobuf's binary, schema-driven approach
- [wire](../wire/) — another example with a generated-and-committed Go file (`wire_gen.go`) alongside its hand-written source
