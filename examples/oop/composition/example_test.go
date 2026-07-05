package composition_test

import "github.com/adnvilla/go-examples/examples/oop/composition"

// Example shows struct embedding: Citizen embeds Person, promoting its fields
// and methods, while overriding Talk to specialize its behavior.
func Example() {
	c := composition.Citizen{
		Country: "Wakanda",
		Person: composition.Person{
			Name: "T'Challa",
			Address: composition.Address{
				Number: "1",
				Street: "Palace Way",
				City:   "Birnin Zana",
				State:  "N/A",
				Zip:    "00000",
			},
		},
	}

	c.Talk()        // Citizen's own override
	c.Location()    // promoted from the embedded Person
	c.Nationality() // Citizen's own method

	// Output:
	// Hello, my name is T'Challa and I'm from Wakanda
	// I'm at 1 Palace Way, Birnin Zana N/A 00000
	// T'Challa is a citizen of Wakanda
}
