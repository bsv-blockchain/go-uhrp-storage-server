# go-uhrp-storage-server

A Go reimplementation of the [BSV Lite Storage Server](https://github.com/bsv-blockchain/lite-storage-server) — a UHRP (Universal Hash Resolution Protocol) content-addressed file storage server with BSV authentication and payment integration.

## API Endpoints

All endpoints match the original TypeScript implementation:

### Pre-Auth Routes (no authentication required)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/quote` | Get a storage price quote for a file size and retention period |
| `PUT` | `/put` | Upload a file via presigned URL (HMAC-verified) |

### Post-Auth Routes (require BRC-31 authentication + BRC-29 payments)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/upload` | Request an upload URL for a file (returns presigned URL with HMAC) |
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
  handlers/         - HTTP route handlers (quote, upload, put, list, find, renew)
  pricing/          - Storage price calculation (USD→BSV→sats via WhatOnChain)
  storage/          - Local filesystem storage
  uhrp/             - UHRP URL encoding/decoding
  wallet/           - BSV wallet provider using go-wallet-toolbox
public/cdn/         - Uploaded file storage
test-client/        - Jest integration tests (TypeScript)
```

## Dependencies

| Go Package | Replaces (TypeScript) |
|-----------|----------------------|
| `github.com/bsv-blockchain/go-sdk` | `@bsv/sdk` |
| `github.com/bsv-blockchain/go-bsv-middleware` | `@bsv/auth-express-middleware` + `@bsv/payment-express-middleware` |
| `github.com/bsv-blockchain/go-wallet-toolbox` | `@bsv/wallet-toolbox` |
| `github.com/go-chi/chi/v5` | `express` |
| `github.com/joho/godotenv` | `dotenv` |

## Wallet Integration

Uses `go-wallet-toolbox` for BSV wallet functionality:

- **HMAC creation** on `/upload` — secures presigned upload URLs
- **HMAC verification** on `/put` — validates upload authorization
- **PushDrop advertisements** — on-chain UHRP file advertisements in the `uhrp advertisements` basket
- **ListOutputs** — queries advertisements for `/list` and `/find`
- **Advertisement renewal** — redeems + re-creates PushDrop tokens with updated expiry

Wallet initialization requires `SERVER_PRIVATE_KEY` and `WALLET_STORAGE_URL`. Without these, the server starts with wallet features disabled (quote and static file serving still work).

## Authentication & Payment Flow

- **Authentication**: BRC-31 mutual authentication via `go-bsv-middleware` auth middleware
- **Payments**: BRC-29 payment protocol via `go-bsv-middleware` payment middleware
- **UHRP**: Content-addressed storage using SHA-256 hashes encoded as Base58Check URLs
- **PushDrop**: Advertisement tokens stored on-chain, tagged with uploader identity key

> **Note**: Auth middleware is defined but not yet wired into the router pending full wallet initialization. The endpoint handlers are ready — once middleware is enabled, authenticated routes will work end-to-end.

## Setup

```bash
cp .env.example .env
# Edit .env with your SERVER_PRIVATE_KEY and WALLET_STORAGE_URL
go run ./cmd/server
```

## Docker

```bash
docker build -t uhrp-storage-server .
docker run -p 8080:8080 --env-file .env uhrp-storage-server
```

## Testing

### Go unit tests

```bash
go test ./...
```

Tests cover pricing calculations, UHRP URL encoding/decoding, file storage operations, and quote handler validation.

### Jest integration tests

The `test-client/` directory contains 16 integration tests using `@bsv/sdk` `StorageUploader` and `StorageUtils`, running against a live local server.

```bash
# Terminal 1: Start the server
SERVER_PRIVATE_KEY=$(openssl rand -hex 32) HTTP_PORT=8081 HOSTING_DOMAIN=localhost:8081 go run ./cmd/server

# Terminal 2: Run tests
cd test-client
npm install
npx jest --verbose
```

**Active tests (11):**
- Quote pricing (valid requests, scaling with size/retention, edge cases)
- Input validation (missing fields, negative sizes, absurd retention periods)
- 404 handler for unknown routes
- UHRP URL encoding/decoding (round-trip hash verification, invalid URL rejection)

**Skipped tests (5, pending auth middleware):**
- File upload + PUT flow with UHRP URL verification
- List uploads for authenticated user
- Find non-existent file
- Renew non-existent file

Set `UHRP_AUTH_ENABLED=true` to enable the auth-dependent tests once `/.well-known/auth` is wired up.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PRIVATE_KEY` | (required) | Hex-encoded private key for server identity |
| `WALLET_STORAGE_URL` | `` | Remote wallet storage URL (required for wallet features) |
| `HTTP_PORT` | `8080` | HTTP listen port |
| `HOSTING_DOMAIN` | `localhost:8080` | Public domain for presigned upload URLs |
| `BSV_NETWORK` | `mainnet` | BSV network (`mainnet`, `testnet`) |
| `PRICE_PER_GB_MO` | `0.03` | Base price per GB per month in USD |
| `MIN_HOSTING_MINUTES` | `0` | Minimum retention period in minutes |

## Differences from the Original

1. **Router**: Uses `chi` instead of Express.js — same routing semantics
2. **Storage**: Local filesystem instead of Google Cloud Storage
3. **Auth middleware**: Defined but not yet wired into router (handlers are ready)
4. **Notifier**: The GCS Cloud Function notifier is not included (separate deployment artifact)

## License

See LICENSE.txt
