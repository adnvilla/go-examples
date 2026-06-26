# Go Lambda


```go
// main.go
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
)

func hello() (string, error) {
	return "Hello ƛ!", nil
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(hello)
}
```

    go.exe get -u github.com/aws/aws-lambda-go/cmd/build-lambda-zip
    set GOOS=linux
    set GOARCH=amd64
    go build -o main main.go
    %USERPROFILE%\Go\bin\build-lambda-zip.exe -o main.zip main




# Resources

https://docs.aws.amazon.com/es_es/lambda/latest/dg/go-programming-model-context.html

https://docs.aws.amazon.com/es_es/lambda/latest/dg/best-practices.html

https://docs.aws.amazon.com/es_es/lambda/latest/dg/go-programming-model-env-variables.html

https://github.com/aws/aws-lambda-go