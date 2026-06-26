# go-examples

Colección de ejemplos en Go, organizados por tema. Requiere Go 1.24+.

## Ejemplos

| Carpeta | Descripción | Temas |
|---------|-------------|-------|
| [benchmark](examples/benchmark/) | Benchmarks con `testing.B` | testing, performance |
| [channels](examples/channels/) | Comunicación entre goroutines con channels | concurrencia |
| [config](examples/config/) | Lectura de configuración desde JSON | stdlib |
| [datetime-parse](examples/datetime-parse/) | Parsing de fechas en múltiples formatos | librerías |
| [dynamodb](examples/dynamodb/) | CRUD con DynamoDB usando AWS SDK v1 | AWS, bases de datos |
| [flags](examples/flags/) | Flags de línea de comandos con `flag` | stdlib, CLI |
| [gin](examples/gin/) | API HTTP mínima con Gin | HTTP, frameworks |
| [inject](examples/inject/) | Inyección de dependencias por reflection | DI, patrones |
| [lambda](examples/lambda/) | Función AWS Lambda en Go | AWS, serverless |
| [metric](examples/metric/) | Envío de métricas a Datadog vía StatsD | observabilidad |
| [mysql](examples/mysql/) | Conexión y queries a MySQL | bases de datos |
| [oop](examples/oop/) | Composición y métodos en structs | patrones OOP en Go |
| [pool](examples/pool/) | Worker pool con manejo de errores por tarea | concurrencia, patrones |
| [profiling](examples/profiling/) | CPU/memory profiling con `pkg/profile` | performance |
| [protobuf](examples/protobuf/) | Serialización binaria con Protocol Buffers | serialización |
| [recover](examples/recover/) | Manejo de panics con `recover` | manejo de errores |
| [redis](examples/redis/) | Cola de tareas con Redis y Gin | Redis, HTTP |
| [reflection-bench](examples/reflection-bench/) | Benchmarks de reflection vs tipos directos | performance, reflection |
| [scatter-gather](examples/scatter-gather/) | Patrón scatter/gather con goroutines | concurrencia, patrones |
| [serialization](examples/serialization/) | JSON flexible con arrays/objetos intercambiables | serialización |
| [share-memory-by-communicating](examples/share-memory-by-communicating/) | Poller HTTP con channels — ejemplo del blog de Go | concurrencia |
| [testing](examples/testing/) | Tests unitarios básicos | testing |
| [typecast](examples/typecast/) | Benchmarks de type switch vs type assertion | performance |
| [wire](examples/wire/) | Inyección de dependencias generada con Wire | DI, code generation |
| [worker-pool](examples/worker-pool/) | Worker pool simple con jobs/results channels | concurrencia |
| [workers](examples/workers/) | Fan-out con timeout por worker | concurrencia, patrones |

## Correr un ejemplo

```bash
go run ./examples/channels/
go run ./examples/gin/
```

## Tests

```bash
# Todos los tests
go test ./...

# Tests de integración de DynamoDB (requiere DynamoDB local en :8000)
DYNAMODB_LOCAL=1 go test ./examples/dynamodb/...

# Con race detector
go test -race ./...
```

## Requisitos

- Go 1.24+
- Para `dynamodb/`: DynamoDB Local o credenciales AWS
- Para `redis/`: Redis en `192.168.99.101:6379`
- Para `mysql/`: MySQL accesible
- Para `protobuf/`: `protoc` + `protoc-gen-go` para regenerar el `.pb.go`
