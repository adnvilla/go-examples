package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Configuration schema
type Configuration struct {
	Users             []string
	Groups            []string
	Name              string
	ConnectionStrings map[string]string
}

func main() {

	file, _ := os.Open("config.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(configuration.Users)
	fmt.Println(configuration.Name)

	fmt.Println(configuration.ConnectionStrings)
}
