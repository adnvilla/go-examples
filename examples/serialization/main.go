// Demonstrates flexible JSON deserialization for APIs that return either an object
// or an array for the same field — a common pattern in XML-derived JSON APIs.
// The Hotels.Hotel field can be either a single object {} or an array [].
package main

import (
	"fmt"
	"log"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Root struct {
	Field  string `json:"Field"`
	Hotels Hotels `json:"Hotels"`
}

type Hotels struct {
	Field string  `json:"Field"`
	Hotel []Hotel `json:"Hotel"`
}

type Hotel struct {
	Field string `json:"Field"`
}

// UnmarshalJSON handles the case where Hotel is either an array or a single object.
func (h *Hotels) UnmarshalJSON(data []byte) error {
	var hotels []Hotel
	if err := json.Unmarshal([]byte(jsoniter.Get(data, "Hotel").ToString()), &hotels); err == nil {
		h.Hotel = hotels
		return nil
	}

	var single Hotel
	if err := json.Unmarshal([]byte(jsoniter.Get(data, "Hotel").ToString()), &single); err != nil {
		return err
	}
	h.Hotel = []Hotel{single}
	return nil
}

func main() {
	arrayInput := `{
		"Field": "example",
		"Hotels": {
			"Field": "",
			"Hotel": [{"Field": "1"}, {"Field": "2"}]
		}
	}`

	singleInput := `{
		"Field": "example",
		"Hotels": {
			"Field": "",
			"Hotel": {"Field": "1"}
		}
	}`

	for _, input := range []string{arrayInput, singleInput} {
		var root Root
		if err := json.Unmarshal([]byte(input), &root); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("hotels: %+v\n", root.Hotels.Hotel)
	}
}
