# Serialization: Object-or-Array JSON Field

**Category:** basics
**Difficulty:** Intermediate

## Objective

Show a custom `UnmarshalJSON` handling a field that's sometimes a JSON object and sometimes a JSON array of the same shape — a common quirk in APIs derived from XML (where a single child element and a repeated one look identical except for cardinality).

## Concepts Covered

- Implementing `json.Unmarshaler` (`UnmarshalJSON([]byte) error`) to customize decoding for one type
- `jsoniter.Get(data, "Hotel").ToString()` to extract a raw sub-value from JSON without decoding the rest of the document first
- "Try array, fall back to single object" as a general pattern for handling this kind of ambiguous field
- `jsoniter.ConfigCompatibleWithStandardLibrary` — a drop-in, faster alternative to `encoding/json` with the same semantics

## Prerequisites

- Go 1.24+
- No external services or environment variables required

## Project Structure

```
serialization/
├── go.mod
├── main.go
├── main_test.go
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

## Expected Output

```
hotels: [{Field:1} {Field:2}]
hotels: [{Field:1}]
```

## Code Walkthrough

- `Hotels.Hotel` is always represented in Go as `[]Hotel` — a slice — regardless of whether the source JSON had an array or a single object for that field. This is the key design choice: callers of `Hotels` never need to check which shape the original JSON used.
- `Hotels.UnmarshalJSON` first tries to decode the raw `"Hotel"` sub-value as `[]Hotel`. If that succeeds, it's stored directly.
- If the array decode fails (because the JSON value is actually a single object, not an array), it falls back to decoding as a single `Hotel` and wraps it in a one-element slice.
- `jsoniter.Get(data, "Hotel").ToString()` is what extracts just the `"Hotel"` sub-document as raw JSON text, without needing a full two-pass parse of the outer `Hotels` object — `jsoniter`'s `Get` navigates the JSON structure directly.
- `main` demonstrates both shapes: `arrayInput` has `"Hotel": [...]`, `singleInput` has `"Hotel": {...}` — both unmarshal into the same `Root` struct shape (`root.Hotels.Hotel` is always `[]Hotel`).

## Common Pitfalls

- **Trying the single-object decode first.** Attempting the array decode first, and falling back to a single object on failure, is the right order here — decoding a JSON array into a single struct fails immediately and cleanly, whereas some decoders are more permissive about decoding a single object into a slice-typed field, which could mask real errors. Match the fallback order to which failure mode is actually distinguishable in your JSON library.
- **Forgetting this only applies to the specific type with the custom `UnmarshalJSON`.** Only `Hotels` gets this flexible handling — `Root` and `Hotel` use ordinary struct-tag-based unmarshaling; the custom logic doesn't propagate automatically to nested or sibling types.
- **Assuming `jsoniter.ConfigCompatibleWithStandardLibrary` behaves identically to `encoding/json` in every edge case.** It's designed to be a compatible, faster drop-in replacement, but subtle differences in edge cases (e.g. number precision, invalid UTF-8 handling) are worth verifying if migrating existing code that has encoding/json-specific assumptions baked in.

## References

- [json-iterator/go GitHub repository](https://github.com/json-iterator/go)
- [encoding/json package docs — the Unmarshaler interface](https://pkg.go.dev/encoding/json#Unmarshaler)

## Next Steps

- [config](../config/) — a more conventional `encoding/json` decode, without the object-or-array ambiguity
- [protobuf](../protobuf/) — a schema-driven alternative where field shapes are unambiguous by construction
