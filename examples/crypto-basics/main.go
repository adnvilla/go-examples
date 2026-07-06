// Demonstrates the crypto building blocks every backend eventually needs,
// stdlib only: sha256 content hashing, HMAC message authentication with
// constant-time verification, crypto/rand for unpredictable tokens, and
// AES-GCM authenticated encryption — including what happens when a
// ciphertext is tampered with.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	hashing()
	messageAuthentication()
	if err := randomTokens(); err != nil {
		return err
	}
	return authenticatedEncryption()
}

// hashing shows sha256 as a content fingerprint: deterministic, one-way, and
// avalanche — a one-character change rewrites the whole digest. NOT for
// passwords (those need a slow KDF: bcrypt/scrypt/argon2).
func hashing() {
	fmt.Println("--- sha256: content fingerprints ---")
	digest := sha256.Sum256([]byte("the quick brown fox"))
	fmt.Printf("sha256(\"the quick brown fox\") = %x\n", digest)

	changed := sha256.Sum256([]byte("the quick brown fix"))
	fmt.Printf("sha256(\"the quick brown fix\") = %x\n", changed)
}

// messageAuthentication signs a message with HMAC-SHA256 and verifies it with
// hmac.Equal — which compares in constant time, so attackers can't binary-
// search the tag byte by byte through response timings.
func messageAuthentication() {
	fmt.Println("\n--- hmac-sha256: authenticate a message ---")
	key := []byte("shared secret between the two parties")
	message := []byte(`{"amount": 100, "to": "alice"}`)

	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message) // hash.Hash writes never return an error
	tag := mac.Sum(nil)
	fmt.Printf("message: %s\ntag:     %x\n", message, tag)

	fmt.Println("genuine message accepted:", verify(key, message, tag))
	fmt.Println("tampered message accepted:", verify(key, []byte(`{"amount": 9999, "to": "mallory"}`), tag))
}

// verify recomputes the tag and compares in constant time.
func verify(key, message, tag []byte) bool {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message)
	return hmac.Equal(tag, mac.Sum(nil))
}

// randomTokens draws from crypto/rand — the unpredictable source for session
// ids, API keys, and nonces. math/rand is for simulations, never for secrets.
func randomTokens() error {
	fmt.Println("\n--- crypto/rand: unpredictable tokens ---")
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return fmt.Errorf("reading random bytes: %w", err)
	}
	fmt.Printf("32-byte token, base64url encoded (differs every run): %s\n",
		base64.RawURLEncoding.EncodeToString(token))
	return nil
}

// authenticatedEncryption encrypts with AES-256-GCM: confidentiality AND
// integrity in one primitive. The random nonce is prepended to the ciphertext
// (it's not secret, it just must never repeat for the same key); tampering
// with a single byte makes Open fail instead of returning garbage.
func authenticatedEncryption() error {
	fmt.Println("\n--- aes-gcm: authenticated encryption ---")
	// Demo only: a real key comes from a KDF (argon2/scrypt) or a secrets
	// manager, never from a string in the source.
	key := sha256.Sum256([]byte("derive real keys with a KDF, not like this"))
	plaintext := []byte("attack at dawn!!")

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generating nonce: %w", err)
	}

	// Seal appends ciphertext+tag to its first argument; passing the nonce
	// there is the standard trick to ship nonce and ciphertext as one blob.
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	fmt.Printf("plaintext %d bytes -> sealed %d bytes (12 nonce + %d ciphertext + 16 tag)\n",
		len(plaintext), len(sealed), len(plaintext))

	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	decrypted, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decrypting: %w", err)
	}
	fmt.Printf("decrypted matches original: %t\n", string(decrypted) == string(plaintext))

	// Flip one bit and GCM's authentication rejects the whole message.
	ciphertext[0] ^= 0x01
	if _, err := gcm.Open(nil, nonce, ciphertext, nil); err != nil {
		fmt.Printf("tampered ciphertext rejected: %v\n", err)
		return nil
	}
	return errors.New("tampered ciphertext was accepted — GCM should have rejected it")
}
