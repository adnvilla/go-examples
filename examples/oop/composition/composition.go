// Package composition demonstrates struct embedding as Go's alternative to inheritance.
// Embedding promotes fields and methods, but the embedded type is still accessible directly.
package composition

import "fmt"

type Address struct {
	Number string
	Street string
	City   string
	State  string
	Zip    string
}

type Person struct {
	Name    string
	Address Address
}

func (p *Person) Talk() {
	fmt.Println("Hi, my name is", p.Name)
}

func (p *Person) Location() {
	fmt.Printf("I'm at %s %s, %s %s %s\n",
		p.Address.Number, p.Address.Street, p.Address.City, p.Address.State, p.Address.Zip)
}

// Citizen embeds Person, promoting all of Person's fields and methods.
// It can override Talk to specialise behaviour.
type Citizen struct {
	Country string
	Person
}

func (c *Citizen) Talk() {
	fmt.Printf("Hello, my name is %s and I'm from %s\n", c.Name, c.Country)
}

func (c *Citizen) Nationality() {
	fmt.Printf("%s is a citizen of %s\n", c.Name, c.Country)
}
