# Go Profiling


Step #1 .- Get Package


    go get github.com/pkg/profile


Step #2 Use Package in your code

```go
package main

import (
    //...
	"github.com/pkg/profile"
)

func main() {
	// CPU profiling by default
	defer profile.Start().Stop()

    //...
}
```

Step #3 Build and Run your program

    go build

    go run yourmain.go


After run program, obtain path "cpu.pprof" file


Step #4 Install Graphviz if you don't have it installed yet

    https://www.graphviz.org/download/

    Update %PATH% and re-open terminal or VSCode or whatever


Step #5 Go Tool Prof and generate pdf file

    go tool pprof --pdf /path/yourbinary /path/to/your/cpu.pprof > file.pdf

Step #6 Happy Profiling :D

![alt text](./SamplePdf.PNG "Sample Profiling")



# Resources

https://blog.golang.org/profiling-go-programs

https://github.com/pkg/profile

https://www.graphviz.org/