// Demonstrates the io.Reader/io.Writer contract and the stdlib combinators
// (io.Copy, io.LimitReader, bufio, io.TeeReader, io.MultiWriter, io.Pipe) that
// let small stream pieces compose into pipelines without loading everything
// into memory — the foundation under files, sockets, HTTP bodies, and gzip.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	demos := []func() error{
		readInChunks,
		copyStreams,
		bufferedLines,
		teeWhileReading,
		writeOnceToMany,
		pipeAcrossGoroutines,
	}
	for _, demo := range demos {
		if err := demo(); err != nil {
			return err
		}
	}
	return nil
}

// readInChunks shows the raw io.Reader contract: Read fills a caller-owned
// buffer, returns how many bytes it wrote, and signals exhaustion with io.EOF.
// The n > 0 bytes must be processed *before* looking at the error — a reader
// may return data and io.EOF in the same call.
func readInChunks() error {
	fmt.Println("--- manual Read loop (the io.Reader contract) ---")
	reader := strings.NewReader("the quick brown fox")
	buf := make([]byte, 8)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			fmt.Printf("read %d bytes: %q\n", n, buf[:n])
		}
		if err == io.EOF {
			fmt.Println("reached EOF")
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// copyStreams moves bytes between a reader and a writer without a hand-written
// loop; io.LimitReader wraps a reader so downstream code can't read past a cap.
func copyStreams() error {
	fmt.Println("\n--- io.Copy and io.LimitReader ---")
	var dst bytes.Buffer
	n, err := io.Copy(&dst, strings.NewReader("streams compose"))
	if err != nil {
		return err
	}
	fmt.Printf("copied %d bytes: %q\n", n, dst.String())

	dst.Reset()
	limited := io.LimitReader(strings.NewReader("only the first 9 bytes survive"), 9)
	if _, err := io.Copy(&dst, limited); err != nil {
		return err
	}
	fmt.Printf("limited copy: %q\n", dst.String())
	return nil
}

// bufferedLines wraps a reader in bufio.Scanner, which batches small reads and
// splits the stream into lines — the standard way to consume line-oriented input.
func bufferedLines() error {
	fmt.Println("\n--- bufio.Scanner (buffered line reading) ---")
	scanner := bufio.NewScanner(strings.NewReader("alpha\nbeta\ngamma\n"))
	line := 1
	for scanner.Scan() {
		fmt.Printf("line %d: %s\n", line, scanner.Text())
		line++
	}
	return scanner.Err()
}

// teeWhileReading observes a stream as it is consumed: io.TeeReader copies
// every byte read from the source into a writer — here a sha256 hash — so one
// pass over the data yields both the payload and its checksum.
func teeWhileReading() error {
	fmt.Println("\n--- io.TeeReader (observe a stream while consuming it) ---")
	hasher := sha256.New()
	tee := io.TeeReader(strings.NewReader("hash me as I stream"), hasher)
	payload, err := io.ReadAll(tee)
	if err != nil {
		return err
	}
	fmt.Printf("consumed: %q\n", payload)
	fmt.Printf("sha256:   %x\n", hasher.Sum(nil))
	return nil
}

// writeOnceToMany fans a single write out to several sinks with io.MultiWriter —
// the writer-side dual of TeeReader.
func writeOnceToMany() error {
	fmt.Println("\n--- io.MultiWriter (one write, several sinks) ---")
	var first, second bytes.Buffer
	hasher := sha256.New()
	sink := io.MultiWriter(&first, &second, hasher)
	if _, err := fmt.Fprint(sink, "fan out this write"); err != nil {
		return err
	}
	fmt.Printf("buffer 1: %q\n", first.String())
	fmt.Printf("buffer 2: %q\n", second.String())
	fmt.Printf("sha256:   %x\n", hasher.Sum(nil))
	return nil
}

// pipeAcrossGoroutines connects writer-shaped code (json.Encoder wants an
// io.Writer) to reader-shaped code (io.ReadAll wants an io.Reader) without an
// intermediate buffer. io.Pipe blocks each Write until the other side Reads,
// so producer and consumer MUST run in different goroutines.
func pipeAcrossGoroutines() error {
	fmt.Println("\n--- io.Pipe (connect writer-shaped code to reader-shaped code) ---")
	type event struct {
		Name string `json:"name"`
		Seq  int    `json:"seq"`
	}

	pr, pw := io.Pipe()
	go func() {
		// CloseWithError(nil) closes normally; a non-nil error surfaces as the
		// reading side's Read error instead of being silently dropped.
		pw.CloseWithError(json.NewEncoder(pw).Encode(event{Name: "ping", Seq: 1}))
	}()

	payload, err := io.ReadAll(pr)
	if err != nil {
		return err
	}
	fmt.Printf("received: %s", payload)
	return nil
}
