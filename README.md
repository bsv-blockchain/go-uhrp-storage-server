# go-uhrp-storage-server

A Go reimplementation of the [BSV Lite Storage Server](https://github.com/bsv-blockchain/lite-storage-server) — a UHRP (Universal Hash Resolution Protocol) content-addressed file storage server with BSV authentication and payment integration.

## API Endpoints

All endpoints match the original TypeScript implementation:

### Pre-Auth Routes (no authentication required)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/quote` | Get a storage price quote for a file size and retention period |
| `PUT` | `/put` | Upload a file via presigned URL (HMAC-verified) |

### Post-Auth Routes (require BRC-103/104 authentication + BRC-29 payments)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/upload` | Request an upload URL for a file (returns presigned URL) |
| `GET` | `/list` | List all UHRP advertisements for the authenticated user |
| `POST` | `/renew` | Extend the retention period of an existing file |
| `GET` | `/find` | Find metadata for a file by UHRP URL |

### Static Files

| Path | Description |
|------|-------------|
| `/cdn/*` | Serves uploaded files with correct MIME types |

## Architecture

```
cmd/server/         - Main server entry point
internal/
  config/           - Environment configuration
  handlers/         - HTTP route handlers
  pricing/          - Storage price calculation (USD→BSV→sats)
  storage/          - Local filesystem storage
  uhrp/             - UHRP URL encoding/decoding
  wallet/           - BSV wallet provider (singleton)
public/cdn/         - Uploaded file storage
```

## Dependencies

| Go Package | Replaces (TypeScript) |
|-----------|----------------------|
| `github.com/bsv-blockchain/go-sdk` | `@bsv/sdk` |
| `github.com/bsv-blockchain/go-bsv-middleware` | `@bsv/auth-express-middleware` + `@bsv/payment-express-middleware` |
| `github.com/go-chi/chi/v5` | `express` |
| `github.com/joho/godotenv` | `dotenv` |

## Authentication & Payment Flow

- **Authentication**: BRC-103/104 mutual authentication via `go-bsv-middleware` auth middleware
- **Payments**: BRC-29 payment protocol via `go-bsv-middleware` payment middleware
- **UHRP**: Content-addressed storage using SHA-256 hashes (`uhrp://<hex-sha256>`)
- **PushDrop**: Advertisement tokens stored on-chain in the `uhrp advertisements` basket
- **SHIP**: Advertisements broadcast via SHIPBroadcaster to `tm_uhrp` topic

## Setup

```bash
cp .env.example .env
# Edit .env with your SERVER_PRIVATE_KEY and other settings
go run ./cmd/server
```

## Docker

```bash
docker build -t uhrp-storage-server .
docker run -p 8080:8080 --env-file .env uhrp-storage-server
```

## Testing

```bash
go test ./...
```

## Implementation Notes

### What's Fully Implemented
- Complete API surface matching all 6 endpoints
- Price calculation with live BSV exchange rate from WhatOnChain
- Local filesystem file storage with CDN serving
- MIME type detection (magic bytes + extension mapping)
- CORS middleware matching the original
- UHRP URL encoding/decoding
- Base58 object identifier generation
- Input validation matching the original error codes

### What Requires Wallet Integration
The following features require a running BSV wallet (via `go-wallet-toolbox` or equivalent):

- **HMAC creation/verification** on `/upload` and `/put` routes
- **UHRP advertisement creation** (PushDrop token creation + SHIP broadcast)
- **Advertisement listing** (`wallet.ListOutputs` for `/list` and `/find`)
- **Advertisement renewal** (PushDrop token redemption + re-creation for `/renew`)
- **Auth middleware** activation (requires non-nil `wallet.Interface`)
- **Payment middleware** activation (requires non-nil `wallet.Interface`)

These are marked with `// TODO` comments in the source. Once `go-wallet-toolbox-client` is available as a Go package, these can be wired in by initializing the wallet in `internal/wallet/wallet.go` and enabling the middleware in `cmd/server/main.go`.

### Differences from Reference
1. **Router**: Uses `chi` instead of Express.js — same routing semantics
2. **Wallet initialization**: Deferred until `go-wallet-toolbox-client` Go package is available
3. **Notifier**: The GCS Cloud Function notifier (`notifier.js`) is not reimplemented as it's a separate deployment artifact for Google Cloud Storage triggers

## License

See LICENSE.txt
