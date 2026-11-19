# CLAUDE.md

## 📦 Project Summary

**go-uhrp-storage-server** is a Go reimplementation of the [BSV Lite Storage Server](https://github.com/bsv-blockchain/lite-storage-server): a UHRP (Universal Hash Resolution Protocol) content-addressed file host. Clients pay in BSV to store a file for a chosen retention period, and the server advertises each file on the UHRP overlay network so anyone can resolve its hash to a download URL.

### Core Capabilities
- **Paid Uploads**: `/quote` prices storage in satoshis (USD per GB-month, converted at the live BSV/USD rate); `/upload` returns an HMAC-signed presigned URL that `/put` verifies before writing the file
- **Authentication & Payments**: BRC-31 mutual authentication and BRC-29 payments through `go-bsv-middleware`, backed by a `go-wallet-toolbox` wallet
- **UHRP Advertisements**: PushDrop tokens in the `uhrp advertisements` basket, broadcast to the `tm_uhrp` overlay topic and renewable via `/renew`
- **Discovery**: `/list` and `/find` read the caller's active advertisements back from the wallet
- **Static CDN**: Uploaded files are served from `/cdn/*` with MIME types detected by extension or content

### Layout
`cmd/server` (entry point), `internal/` (config, logger, HTTP server and handlers, file storage, wallet provider), `pkg/` (pricing and UHRP URL helpers), `docs/` (generated Swagger spec), `test-client/` (Jest integration tests).

## 🤖 Welcome, Claude

This repository uses **`AGENTS.md`** as the single source of truth for:

* Coding conventions (naming, formatting, commenting, testing)
* Contribution workflows (branch prefixes, commit message style, PR templates)
* Release, CI, and dependency‑management policies
* Security reporting and governance links

> **TL;DR:** **Read `AGENTS.md` first.**
> All technical or procedural questions are answered there.

### Quick Checklist for Claude

1. **Study `AGENTS.md`**
   Make sure every automated change or suggestion respects those rules.
2. **Follow branch‑prefix and commit‑message standards**
   They drive Mergify auto‑labeling and CI gates.
3. **Never tag releases**
4. **Pass CI**
   Run `go fmt`, `goimports`, `go vet`, `staticcheck`, and `golangci‑lint` locally before opening a PR.

If you encounter conflicting guidance elsewhere, `AGENTS.md` wins.
Questions or ambiguities? Open a discussion or ping a maintainer instead of guessing.

Happy hacking!
