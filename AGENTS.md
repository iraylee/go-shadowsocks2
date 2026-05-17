# AGENTS.md

## Build & Test

```sh
go build ./...          # local build (single platform)
make -j all             # cross-platform: linux/amd64,arm64; darwin/amd64,arm64; windows/amd64,386
make -j test            # cross-platform test (same 6 combos)
go test ./...           # local test (single platform) — prefer this during development
```

No lint/vet step in CI or Makefile.

## Architecture

Single binary (`main.go`) that runs in **client** mode (`-c`) or **server** mode (`-s`). Both modes share the same cipher stack.

### Package map

| Package | Role |
|---|---|
| `main` (root) | CLI, TCP/UDP relay, SOCKS local, plugin launcher |
| `core/` | Cipher registry, key derivation (MD5 KDF) |
| `shadowaead/` | AEAD stream/packet wrappers (AES-GCM, ChaCha20-Poly1305) |
| `socks/` | SOCKS5 handshake and address encoding |
| `internal/` | Bloom filter for replay-attack mitigation (salt filter) |
| `nfutil/` | Linux-only: `SO_ORIGINAL_DST` for netfilter TCP redirect |
| `pfutil/` | Darwin-only: PF NAT lookup for TCP redirect |
| `model/` | **Dead code** — GORM/Postgres models not imported by main |

### OS-specific compilation

- `tcp_linux.go` — `redirLocal`, `redir6Local` via netfilter
- `tcp_darwin.go` — `redirLocal` via PF; `redir6Local` panics
- `tcp_other.go` — both functions are no-ops (build tag: `// +build !linux,!darwin`, **old-style**)

`nfutil/` uses `syscall.Socketcall` on Linux 386 for ABI compatibility (`socketcall_linux_386.go`).

### Password & key

Precedence: `-key` (base64url) > `-password` > `$SS_PASSWORD` (silent env fallback).

If no explicit key is given, an MD5-based KDF derives the key from the password.

## Environment variables

| Variable | Purpose |
|---|---|
| `SS_PASSWORD` | Password fallback (both client and server) |
| `SHADOWSOCKS_SF_CAPACITY` | Replay filter capacity (default `1e6`). `0` disables. |
| `SHADOWSOCKS_SF_FPR` | Replay filter false-positive rate (default `1e-6`) |
| `SHADOWSOCKS_SF_SLOT` | Replay filter slot count (default `10`) |

## Quirks

- The binary **always** starts an HTTP server on `:888` with a `/log` JSON endpoint (connection stats) in both modes.
- Default cipher is `AEAD_CHACHA20_POLY1305`. Cipher names are case-insensitive.
- Cipher aliases: `CHACHA20-IETF-POLY1305`, `AES-128-GCM`, `AES-256-GCM`.
- SIP003 plugins: searched in cwd first, then `$PATH`. Plugin env vars (`SS_REMOTE_HOST`, etc.) are set automatically.
- `model/` is not wired into main; changes there won't affect the binary.
