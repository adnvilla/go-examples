// Functional options pattern: configure a struct without exporting its fields
// or multiplying constructor overloads.
// Each option is a function that modifies the internal config, so new options
// are added without breaking existing callers.
package main

import (
	"fmt"
	"time"
)

type serverConfig struct {
	host         string
	port         int
	readTimeout  time.Duration
	writeTimeout time.Duration
	maxConns     int
}

// Option is a function that applies one setting to serverConfig.
type Option func(*serverConfig)

func WithHost(host string) Option {
	return func(c *serverConfig) { c.host = host }
}

func WithPort(port int) Option {
	return func(c *serverConfig) { c.port = port }
}

func WithReadTimeout(d time.Duration) Option {
	return func(c *serverConfig) { c.readTimeout = d }
}

func WithWriteTimeout(d time.Duration) Option {
	return func(c *serverConfig) { c.writeTimeout = d }
}

func WithMaxConns(n int) Option {
	return func(c *serverConfig) { c.maxConns = n }
}

// Server holds the resolved configuration.
type Server struct {
	cfg serverConfig
}

// NewServer applies options on top of sensible defaults.
// Callers only specify what differs from the defaults.
func NewServer(opts ...Option) *Server {
	cfg := serverConfig{
		host:         "0.0.0.0",
		port:         8080,
		readTimeout:  5 * time.Second,
		writeTimeout: 10 * time.Second,
		maxConns:     100,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Server{cfg: cfg}
}

func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.host, s.cfg.port)
}

func main() {
	// Minimal — all defaults.
	s1 := NewServer()
	fmt.Printf("default server: addr=%s maxConns=%d\n", s1.Addr(), s1.cfg.maxConns)

	// Override only what's needed.
	s2 := NewServer(
		WithPort(9090),
		WithReadTimeout(30*time.Second),
		WithMaxConns(500),
	)
	fmt.Printf("custom server:  addr=%s readTimeout=%s maxConns=%d\n",
		s2.Addr(), s2.cfg.readTimeout, s2.cfg.maxConns)
}
