# Go Testing

    go build
    go test --coverprofile=cover.out
    go tool cover -html=cover.out -o coverage.html


    go test
    go test -v
    go test -cover



For fail unit tests if coverage is below certain percentage:



```go

func TestMain(m *testing.M) {
    // call flag.Parse() here if TestMain uses flags
    rc := m.Run()

    // rc 0 means we've passed, 
    // and CoverMode will be non empty if run with -cover
    if rc == 0 && testing.CoverMode() != "" {
        c := testing.Coverage()
        if c < 0.8 {
            fmt.Println("Tests passed but coverage failed at", c)
            rc = -1
        }
    }
    os.Exit(rc)
}
```

# Resources

https://golang.org/pkg/testing/

https://blog.golang.org/cover

https://blog.alexellis.io/golang-writing-unit-tests/

https://golang.org/pkg/testing/#hdr-Main

