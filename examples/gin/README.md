# Gin

**Category:** http
**Difficulty:** Beginner

## Objective

Show the smallest possible HTTP API using the [Gin](https://github.com/gin-gonic/gin) web framework: one route, one JSON response.

## Concepts Covered

- `gin.Default()` — a router pre-configured with logging and panic-recovery middleware
- `r.GET(path, handler)` — registering a route with a handler taking `*gin.Context`
- `c.JSON(status, gin.H{...})` — writing a JSON response with a status code, where `gin.H` is a shorthand for `map[string]any`
- `r.Run()` — starts the HTTP server, defaulting to `:8080` (or the `PORT` environment variable, if set)

## Prerequisites

- Go 1.24+
- No external services; listens on `:8080`

## Project Structure

```
gin/
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

In another terminal:
```bash
curl http://localhost:8080/ping
```

## Expected Output

Server log on startup and on the request (Gin logs every request by default via its logging middleware):
```
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)
[GIN-debug] GET    /ping                     --> main.main.func1 (3 handlers)
[GIN-debug] Listening and serving HTTP on :8080
[GIN] 2026/07/05 - 18:04:40 | 200 |      68.791µs |             ::1 | GET      "/ping"
```

Client response:
```
$ curl http://localhost:8080/ping
{"message":"pong"}
```

## Code Walkthrough

- `gin.Default()` returns a `*gin.Engine` with `Logger()` and `Recovery()` middleware already attached — every request is logged, and a handler panic is caught and turned into a 500 instead of crashing the process (see [http-server](../http-server/) for the same idea implemented by hand with the standard library).
- `r.GET("/ping", handler)` registers a route matching GET requests to `/ping`; the handler receives a `*gin.Context`, Gin's per-request object bundling the request, response writer, and helper methods.
- `c.JSON(200, gin.H{"message": "pong"})` sets the `Content-Type` header to `application/json`, writes status `200`, and serializes the `gin.H` map (`map[string]any`) as the response body.
- `r.Run()` blocks, listening on `:8080` by default — this call never returns during normal operation; the process exits only via a signal (there's no graceful shutdown handling in this minimal example, unlike [http-server](../http-server/)).

## Common Pitfalls

- **Running in debug mode in production.** The startup warnings tell you exactly this — set `GIN_MODE=release` (or call `gin.SetMode(gin.ReleaseMode)`) for production deployments; debug mode adds overhead and verbose logging not meant for production traffic.
- **Using `gin.Default()` when you don't want the built-in middleware.** `gin.New()` returns a bare engine with no middleware attached, if you want to compose your own logging/recovery stack instead.
- **No graceful shutdown.** `r.Run()` is a thin wrapper around `http.ListenAndServe` with no signal handling — killing the process drops in-flight requests immediately. See [http-server](../http-server/) for a graceful-shutdown pattern (built without a framework), which can be adapted to Gin by using `r.Handler()` as the `http.Server`'s handler instead of `r.Run()`.
- **Forgetting `PORT` is read from the environment.** `r.Run()` with no argument checks `PORT` before defaulting to `:8080` — a deployment platform setting `PORT` will change where the server actually listens.

## References

- [Gin GitHub repository](https://github.com/gin-gonic/gin)
- [Gin documentation](https://gin-gonic.com/docs/)

## Next Steps

- [http-server](../http-server/) — the same kind of API built on the standard library, including graceful shutdown
- [redis](../redis/) — a Gin-based task queue backed by Redis
