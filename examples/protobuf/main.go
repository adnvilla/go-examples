//nolint:staticcheck // golang/protobuf v1.5 compat wrapper — upgrade requires protoc toolchain
package main

import (
	"fmt"
	"log"
	"os"

	proto "github.com/golang/protobuf/proto"
)

func main() {
	fname := "test.dat"
	book := &AddressBook{
		People: []*Person{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
	}

	out, err := proto.Marshal(book)
	if err != nil {
		log.Fatalln("Failed to encode address book:", err)
	}
	if err := os.WriteFile(fname, out, 0600); err != nil {
		log.Fatalln("Failed to write address book:", err)
	}

	in, err := os.ReadFile(fname)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}
	book2 := &AddressBook{}
	if err := proto.Unmarshal(in, book2); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}

	fmt.Println(book2)
}
