---
title: Encryption
type: docs
prev: docs/security/csrf
next: docs/session
weight: 104
sidebar:
  open: true
---

Sometimes you have to store something you must be able to read back, but
nobody else should — an API token for a third-party service, say. Hashing is
no good here, because hashing is one-way and you need the value again.
**Encryption** is the tool: scramble it with a key, and only something holding
that key can unscramble it.

{{< callout type="warning" >}}
**Passwords are not this.** Never encrypt a password — hash it. If you can
decrypt it, so can whoever steals your database and your key. See
[Authentication](/docs/security/authentication).
{{< /callout >}}

The `encryption` package gives you AES-256-GCM, which also detects tampering:
if an encrypted value is altered, decrypting fails rather than returning
something subtly wrong. It takes its key from `APP_KEY`.

## Configuration

The app key is generated during project creation and stored in `.env`:

```shell
APP_KEY=base64:abc123... (32 bytes base64-encoded)
```

You can regenerate the app key:

```shell
lemmego appkey
```

This replaces the `APP_KEY` value in your `.env` file with a new 32-byte random key.

## Basic Usage

```go
import "github.com/lemmego/api/encryption"
```

### Encrypt

```go
ciphertext, err := encryption.Encrypt([]byte("sensitive data"))
// Returns base64-encoded ciphertext
```

### Decrypt

```go
plaintext, err := encryption.Decrypt(ciphertext)
// Returns []byte
```

### Encrypt/Decrypt Strings

```go
encrypted, err := encryption.Get().EncryptString("sensitive data")
decrypted, err := encryption.Get().DecryptString(encrypted)
```

## Manual Initialization

```go
key := []byte("32-bytes-exactly-for-aes-256-key!")
enc, err := encryption.NewEncrypter(key)

ciphertext, err := enc.Encrypt([]byte("data"))
plaintext, err := enc.Decrypt(ciphertext)
```

## How It Works

1. **AES-256-GCM** — Authenticated encryption with associated data (AEAD)
2. **Random Nonce** — A 12-byte random nonce is generated for each encryption operation
3. **Nonce Prepend** — The nonce is prepended to the ciphertext
4. **Base64 Encoding** — The combined nonce+ciphertext is base64-encoded for transport

```
Encrypt(plaintext):
  nonce = random(12 bytes)
  ciphertext = AES-256-GCM.Seal(nonce, plaintext)
  return base64(nonce + ciphertext)

Decrypt(ciphertext):
  raw = base64_decode(ciphertext)
  nonce = raw[:12]
  ciphertext = raw[12:]
  return AES-256-GCM.Open(nonce, ciphertext)
```

## Key Requirements

- The key must be exactly **32 bytes** for AES-256
- The `APP_KEY` in `.env` is base64-encoded before use
- Never commit the real `APP_KEY` to version control

## Use Cases

- Encrypting sensitive user data (PII, payment info)
- Encrypting API tokens stored in the database
- Secure cookie values
- Encrypting data at rest

## Next steps

- [Environment variables](/docs/appendix/#environment-variables) — where
  `APP_KEY` is set and what it must look like.
- [Session](/docs/session) — data encryption already applies to.
