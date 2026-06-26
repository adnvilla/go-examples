// Dependency injection via constructor functions — the idiomatic Go approach.
// No reflection, no struct tags, no framework: dependencies are explicit parameters.
//
// For compile-time DI code generation see the wire/ example.
package main

import (
	"fmt"
	"net/http"
)

// Namer resolves a user ID to a display name.
type Namer interface {
	Name(id uint64) string
}

// Planner resolves a user ID to their home planet.
type Planter interface {
	Planet(id uint64) string
}

// NameAPI is an HTTP-backed implementation of Namer.
type NameAPI struct {
	transport http.RoundTripper
}

func NewNameAPI(transport http.RoundTripper) *NameAPI {
	return &NameAPI{transport: transport}
}

func (n *NameAPI) Name(_ uint64) string {
	// real impl would use n.transport to call a remote service
	return "Spock"
}

// PlanetAPI is an HTTP-backed implementation of Planter.
type PlanetAPI struct {
	transport http.RoundTripper
}

func NewPlanetAPI(transport http.RoundTripper) *PlanetAPI {
	return &PlanetAPI{transport: transport}
}

func (p *PlanetAPI) Planet(_ uint64) string {
	return "Vulcan"
}

// App composes the two APIs. Dependencies are injected via the constructor,
// not through struct tags or a global graph.
type App struct {
	names   Namer
	planets Planter
}

func NewApp(names Namer, planets Planter) *App {
	return &App{names: names, planets: planets}
}

func (a *App) Render(id uint64) string {
	return fmt.Sprintf("%s is from %s", a.names.Name(id), a.planets.Planet(id))
}

func main() {
	transport := http.DefaultTransport

	// Each dependency is constructed explicitly and passed down the chain.
	// A DI framework would generate this wiring; Wire (see wire/ example) does so at compile time.
	nameAPI := NewNameAPI(transport)
	planetAPI := NewPlanetAPI(transport)
	app := NewApp(nameAPI, planetAPI)

	fmt.Println(app.Render(42))
}
