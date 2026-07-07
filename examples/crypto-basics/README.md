# Crypto Basics

**Category:** security
**Difficulty:** Intermediate

## Objective

Show the four crypto building blocks every backend eventually needs, using only the standard library: `sha256` content fingerprints, HMAC message authentication with constant-time verification, `crypto/rand` for unpredictable tokens, and AES-GCM authenticated encryption — including the demonstration that tampering with one bit of ciphertext makes decryption *fail* instead of returning garbage.

## Concepts Covered

- `sha256.Sum256` — deterministic, one-way fingerprints with the avalanche effect (one changed character rewrites the digest), and why fast hashes are **not** for passwords
- `hmac.New(sha256.New, key)` + `hmac.Equal` — signing a message with a shared secret and verifying in constant time (no timing side channel)
- `crypto/rand.Read` — the unpredictable source for session ids, API keys, and nonces (`math/rand` is for simulations, never secrets)
- AES-256-GCM via `cipher.NewGCM` — confidentiality and integrity in one primitive; `Seal`/`Open`, the prepended-nonce convention, and authenticated rejection of tampered ciphertexts
- Key hygiene signposting: demo key derived from a string with a loud comment; real keys come from a KDF (argon2/scrypt) or a secrets manager

## Prerequisites

- Go 1.25+
- No external services or environment variables required

## Project Structure

```
crypto-basics/
├── go.mod
├── main.go
├── Makefile
└── README.md
```

## How to Run

```bash
make run
# or
go run .
```

## Expected Output

Digest and tag lines are deterministic; the random token and the GCM nonce differ every run (marked below):

```
--- sha256: content fingerprints ---
sha256("the quick brown fox") = 9ecb36561341d18eb65484e833efea61edc74b84cf5e6ae1b81c63533e25fc8f
sha256("the quick brown fix") = 08b6e30107c526b3fcd635de9b011e0ee90c46190168c3e578532eb7c61f69e5

--- hmac-sha256: authenticate a message ---
message: {"amount": 100, "to": "alice"}
tag:     e39792c118e73302f7e5aa38b2c1304d76152332f39125fe05826d1489e97e29
genuine message accepted: true
tampered message accepted: false

--- crypto/rand: unpredictable tokens ---
32-byte token, base64url encoded (differs every run): <varies>

--- aes-gcm: authenticated encryption ---
plaintext 16 bytes -> sealed 44 bytes (12 nonce + 16 ciphertext + 16 tag)
decrypted matches original: true
tampered ciphertext rejected: cipher: message authentication failed
```

## Code Walkthrough

- `hashing` fingerprints two strings that differ by one character — the completely unrelated digests are the avalanche effect, which is what makes hashes useful as content addresses and integrity checks. The comment says the quiet part loudly: sha256 is *fast*, and fast is exactly wrong for passwords, where the attacker's cost per guess is the defense (use bcrypt/scrypt/argon2).
- `messageAuthentication` shows why HMAC exists: the tag proves the message came from someone holding the key *and* wasn't modified. Verification recomputes the tag and compares with `hmac.Equal` — constant-time, so response timing doesn't leak how many leading bytes matched. The tampered `{"amount": 9999}` message is rejected while the genuine one passes.
- `randomTokens` reads 32 bytes from `crypto/rand` and base64url-encodes them — the standard recipe for session ids and API keys. The error check on `rand.Read` matters in principle (an unreadable entropy source is a stop-the-world problem, not one to paper over).
- `authenticatedEncryption` runs the full AES-GCM lifecycle: derive a (demo) 32-byte key, generate a fresh random nonce, `Seal`, then `Open`. Passing `nonce` as `Seal`'s first argument prepends it to the output — the standard way to ship nonce+ciphertext+tag as one blob, since the nonce isn't secret, it just must never repeat under the same key. Flipping one ciphertext bit makes `Open` return `cipher: message authentication failed` — GCM authenticates before it decrypts, which is why modern designs use AEAD instead of bare CBC/CTR plus hope.

## Common Pitfalls

- **Hashing passwords with sha256 (even salted).** GPUs try billions of sha256 guesses per second. Password storage needs a deliberately slow, memory-hard KDF: argon2id or scrypt (`golang.org/x/crypto`), or bcrypt.
- **Comparing secrets with `==` or `bytes.Equal`.** Both short-circuit on the first differing byte, leaking match length through timing. Use `hmac.Equal` / `subtle.ConstantTimeCompare` for tags, tokens, and API keys.
- **`math/rand` for anything secret.** It's seeded and predictable by design. `crypto/rand` is the only source for keys, tokens, and nonces.
- **Reusing a GCM nonce under the same key.** Two messages sealed with the same key+nonce break the scheme catastrophically (keystream reuse, forgeable tags). Random 12-byte nonces are fine for bounded message counts; high-volume systems use counters or rotate keys.
- **Encrypting without authentication.** Bare AES-CTR/CBC returns *something* for tampered input, and downstream code parses it. AEAD modes like GCM fail closed — prefer them always; there is essentially no reason to hand-roll encrypt-then-MAC today.
- **Hardcoding keys in source.** The demo derives its key from a string and says so in a comment; production keys live in a KMS/secrets manager and reach the process as bytes, not literals.

## References

- [crypto/sha256 package docs](https://pkg.go.dev/crypto/sha256)
- [crypto/hmac package docs](https://pkg.go.dev/crypto/hmac)
- [crypto/cipher package docs — AEAD](https://pkg.go.dev/crypto/cipher#AEAD)
- [crypto/rand package docs](https://pkg.go.dev/crypto/rand)
- [golang.org/x/crypto/argon2](https://pkg.go.dev/golang.org/x/crypto/argon2) — for the password-hashing case this example deliberately excludes

## Next Steps

- [errors](../errors/) — the wrapping discipline used on every failure path here
- [io-readers-writers](../io-readers-writers/) — `io.TeeReader` + `sha256` for hashing streams in one pass
- [http-server](../http-server/) — where HMAC-signed payloads (webhooks) typically arrive
