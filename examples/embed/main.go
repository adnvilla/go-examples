// //go:embed compiles static files into the binary at build time.
// The embedded FS behaves like io/fs.FS, so it works with http.FileServer,
// template parsing, and any io/fs-aware API.
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed static/hello.txt
var helloText string

//go:embed static/config.json
var configJSON []byte

//go:embed static
var staticFS embed.FS

type appConfig struct {
	Version      string            `json:"version"`
	FeatureFlags map[string]bool   `json:"feature_flags"`
}

func main() {
	// Embed a single file as a string.
	fmt.Println("=== embedded string ===")
	fmt.Print(helloText)

	// Embed a single file as bytes, decode as JSON.
	fmt.Println("=== embedded JSON ===")
	var cfg appConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		panic(err)
	}
	fmt.Printf("version: %s\n", cfg.Version)
	for flag, enabled := range cfg.FeatureFlags {
		fmt.Printf("  %s: %v\n", flag, enabled)
	}

	// Embed a whole directory as an fs.FS.
	fmt.Println("=== embedded directory ===")
	fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := staticFS.ReadFile(path)
		preview := strings.TrimSpace(string(data))
		if len(preview) > 40 {
			preview = preview[:40]
		}
		fmt.Printf("%s (%d bytes): %s\n", path, len(data), preview)
		return nil
	})
}
