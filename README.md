<div align="center">

# 🗄️&nbsp;&nbsp;go-uhrp-storage-server

**Content-addressed UHRP file hosting for BSV, with BRC-31 authentication and BRC-29 payments.**

<br/>

<a href="https://github.com/bsv-blockchain/go-uhrp-storage-server/releases"><img src="https://img.shields.io/github/release-pre/bsv-blockchain/go-uhrp-storage-server?include_prereleases&style=flat-square&logo=github&color=black" alt="Release"></a>
<a href="https://golang.org/"><img src="https://img.shields.io/github/go-mod/go-version/bsv-blockchain/go-uhrp-storage-server?style=flat-square&logo=go&color=00ADD8" alt="Go Version"></a>
<a href="https://github.com/bsv-blockchain/go-uhrp-storage-server/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue?style=flat-square" alt="License"></a>

<br/>

<table align="center" border="0">
  <tr>
    <td align="right">
       <code>CI / CD</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/bsv-blockchain/go-uhrp-storage-server/actions"><img src="https://img.shields.io/github/actions/workflow/status/bsv-blockchain/go-uhrp-storage-server/fortress.yml?branch=main&label=build&logo=github&style=flat-square" alt="Build"></a>
       <a href="https://github.com/bsv-blockchain/go-uhrp-storage-server/actions"><img src="https://img.shields.io/github/last-commit/bsv-blockchain/go-uhrp-storage-server?style=flat-square&logo=git&logoColor=white&label=last%20update" alt="Last Commit"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Quality</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://codecov.io/gh/bsv-blockchain/go-uhrp-storage-server"><img src="https://codecov.io/gh/bsv-blockchain/go-uhrp-storage-server/branch/main/graph/badge.svg?style=flat-square" alt="Coverage"></a>
    </td>
  </tr>

  <tr>
    <td align="right">
       <code>Security</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://scorecard.dev/viewer/?uri=github.com/bsv-blockchain/go-uhrp-storage-server"><img src="https://api.scorecard.dev/projects/github.com/bsv-blockchain/go-uhrp-storage-server/badge?style=flat-square" alt="Scorecard"></a>
       <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/policy-active-success?style=flat-square&logo=security&logoColor=white" alt="Security"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Community</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/bsv-blockchain/go-uhrp-storage-server/graphs/contributors"><img src="https://img.shields.io/github/contributors/bsv-blockchain/go-uhrp-storage-server?style=flat-square&color=orange" alt="Contributors"></a>
       <a href="https://deepwiki.com/bsv-blockchain/go-uhrp-storage-server"><img src="https://deepwiki.com/badge.svg" alt="Ask DeepWiki"></a>
    </td>
  </tr>
</table>

</div>

<br/>
<br/>

<div align="center">

### <code>Project Navigation</code>

</div>

<table align="center">
  <tr>
    <td align="center" width="33%">
       📖&nbsp;<a href="#-overview"><code>Overview</code></a>
    </td>
    <td align="center" width="33%">
       📦&nbsp;<a href="#-installation"><code>Installation</code></a>
    </td>
    <td align="center" width="33%">
       🚀&nbsp;<a href="#-quick-start"><code>Quick&nbsp;Start</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       📚&nbsp;<a href="#-documentation"><code>Documentation</code></a>
    </td>
    <td align="center">
       🏗️&nbsp;<a href="#-architecture"><code>Architecture</code></a>
    </td>
    <td align="center">
       🧪&nbsp;<a href="#-examples--tests"><code>Examples&nbsp;&&nbsp;Tests</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       ⚡&nbsp;<a href="#-benchmarks"><code>Benchmarks</code></a>
    </td>
    <td align="center">
       🛠️&nbsp;<a href="#-code-standards"><code>Code&nbsp;Standards</code></a>
    </td>
    <td align="center">
       🤖&nbsp;<a href="#-ai-usage--assistant-guidelines"><code>AI&nbsp;Usage</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       🤝&nbsp;<a href="#-contributing"><code>Contributing</code></a>
    </td>
    <td align="center">
       👥&nbsp;<a href="#-maintainers"><code>Maintainers</code></a>
    </td>
    <td align="center">
       📝&nbsp;<a href="#-license"><code>License</code></a>
    </td>
  </tr>
</table>
<br/>

## 📖 Overview

**go-uhrp-storage-server** is a Go reimplementation of the [BSV Lite Storage Server](https://github.com/bsv-blockchain/lite-storage-server): a UHRP (Universal Hash Resolution Protocol) content-addressed file host. Clients pay in BSV to keep a file online for a chosen retention period, and the server advertises every file on the UHRP overlay network so anyone can resolve its SHA-256 hash to a download URL. The HTTP API matches the TypeScript server, so `@bsv/sdk`'s `StorageUploader` works against it unchanged.

### Core Capabilities
- **Paid Uploads**: `/quote` prices storage in satoshis from a USD price per GB-month and the live BSV/USD rate; `/upload` returns an HMAC-signed presigned URL that `/put` verifies before it accepts a single byte
- **Authentication & Payments**: BRC-31 mutual authentication and BRC-29 payments through `go-bsv-middleware`, backed by a `go-wallet-toolbox` wallet
- **UHRP Advertisements**: PushDrop tokens in the server wallet's `uhrp advertisements` basket, broadcast to the `tm_uhrp` overlay topic and renewable through `/renew`
- **Discovery**: `/list` and `/find` read the caller's active advertisements back from the wallet
- **Static CDN**: Uploaded files are served from `/cdn/<objectID>` with a MIME type detected from the extension or content; lookups are confined to the CDN directory and directories are never listed

### Key Dependencies

| TypeScript Original               | Go Equivalent                                                   |
|-----------------------------------|-----------------------------------------------------------------|
| `@bsv/sdk`                        | `github.com/bsv-blockchain/go-sdk`                              |
| `@bsv/auth-express-middleware`    | `github.com/bsv-blockchain/go-bsv-middleware` (auth)            |
| `@bsv/payment-express-middleware` | `github.com/bsv-blockchain/go-bsv-middleware` (payment)         |
| `@bsv/wallet-toolbox`             | `github.com/bsv-blockchain/go-wallet-toolbox`                   |
| Express.js                        | `github.com/go-chi/chi/v5`                                      |
| `dotenv`                          | `github.com/joho/godotenv`                                      |

### Differences from the Original
1. **Storage**: Local filesystem (`PUBLIC_DIR/cdn`) instead of Google Cloud Storage
2. **Notifier**: The GCS Cloud Function notifier is not included (it is a separate deployment artifact)

<br/>

## 📦 Installation

**go-uhrp-storage-server** requires a [supported release of Go](https://golang.org/doc/devel/release.html#policy) (see [go.mod](go.mod) for the exact version). No C toolchain is needed: the server builds with `CGO_ENABLED=0`.

Run it as a server by building from source (see [Quick Start](#-quick-start)) or with Docker. The pricing and UHRP URL helpers are importable on their own:

```shell script
go get -u github.com/bsv-blockchain/go-uhrp-storage-server
```

- [`pkg/pricing`](pkg/pricing): satoshi pricing for a file size and retention period, with a WhatsOnChain BSV/USD rate provider
- [`pkg/uhrp`](pkg/uhrp): UHRP URLs (Base58Check of the SHA-256 hash, identical to `@bsv/sdk`'s `StorageUtils`)

<br/>

## 🚀 Quick Start

### Prerequisites

- **Go 1.27+** (see [go.mod](go.mod) for the exact version)
- A 64-character hex **server private key** for the server's identity
- A **wallet storage server** URL for the server wallet (only for the paid, authenticated routes)
- **Docker** (optional), for a containerized server

### Build from Source

```bash
git clone https://github.com/bsv-blockchain/go-uhrp-storage-server.git
cd go-uhrp-storage-server
go build -o uhrp-storage-server ./cmd/server
```

### Configure

The server reads environment variables and also loads a `.env` file from the working directory if one exists (real environment variables win). [.env.example](.env.example) lists them:

```bash
cp .env.example .env
# Set SERVER_PRIVATE_KEY, WALLET_STORAGE_URL and HOSTING_DOMAIN
```

`SERVER_PRIVATE_KEY` and `WALLET_STORAGE_URL` together enable the wallet. Without them the server still starts with `/quote` and the CDN, and the wallet routes (`/upload`, `/list`, `/renew`, `/find`) are not mounted. Set `HOSTING_DOMAIN` to the public host clients upload to. See the [configuration reference](#-documentation) for every variable.

### Run

```bash
./uhrp-storage-server
```

Or try it with a throwaway identity (quotes and the CDN only):

```bash
SERVER_PRIVATE_KEY=$(openssl rand -hex 32) HOSTING_DOMAIN=localhost:8080 ./uhrp-storage-server
```

The API listens on port `8080` by default, the Swagger UI is served at `/swagger/index.html`, and uploaded files live in `public/cdn` under the working directory (override with `PUBLIC_DIR`).

### Docker

```bash
docker build -t go-uhrp-storage-server .
docker run -p 8080:8080 --env-file .env go-uhrp-storage-server
```

Or with Docker Compose, which reads `.env` and keeps uploaded files in a named volume:

```bash
docker compose up
```

Pushes to `main` and tagged releases also publish an image to the GitHub Container Registry as `ghcr.io/bsv-blockchain/go-uhrp-storage-server`.

<br/>

## 📚 Documentation

- **API Reference**: Browse the endpoints in the Swagger UI at `/swagger/index.html`, or read the [OpenAPI spec](docs/swagger.yaml)
- **Package Docs**: Dive into the godocs at [pkg.go.dev/github.com/bsv-blockchain/go-uhrp-storage-server](https://pkg.go.dev/github.com/bsv-blockchain/go-uhrp-storage-server)
- **Usage Examples**: Run the Go clients in the [examples directory](examples), built on go-sdk's `storage.Uploader`
- **Test Suite**: Review the unit, fuzz and example tests (powered by [`testify`](https://github.com/stretchr/testify)) and the [Jest integration tests](test-client)

### API Endpoints

| Method | Path                | Auth                | Description                                                         |
|--------|---------------------|---------------------|---------------------------------------------------------------------|
| `POST` | `/quote`            | none                | Price a file size and retention period in satoshis                  |
| `PUT`  | `/put`              | presigned HMAC      | Upload the file body to the presigned URL issued by `/upload`       |
| `POST` | `/upload`           | BRC-31 + BRC-29     | Pay for an upload and receive its presigned URL                     |
| `GET`  | `/list`             | BRC-31              | List the caller's active (non-expired) advertisements               |
| `GET`  | `/find`             | BRC-31              | Find the metadata of one of the caller's files by `uhrpUrl`         |
| `POST` | `/renew`            | BRC-31 + BRC-29     | Pay to extend a file's retention and re-advertise it                |
| `GET`  | `/cdn/{objectID}`   | none                | Download an uploaded file                                           |
| `GET`  | `/swagger/*`        | none                | Swagger UI and the generated OpenAPI spec                           |

The server's own errors are JSON in one shape, for example `{"status":"error","code":"ERR_ROUTE_NOT_FOUND","description":"Route not found."}` for an unknown route.

<br/>

<details>
<summary><strong><code>Configuration Reference</code></strong></summary>
<br/>

| Variable              | Default          | Description                                                                                          |
|-----------------------|------------------|------------------------------------------------------------------------------------------------------|
| `SERVER_PRIVATE_KEY`  | unset            | Hex-encoded private key for the server identity; required for the wallet                             |
| `WALLET_STORAGE_URL`  | unset            | Remote wallet storage server for the server wallet; required for the wallet                          |
| `HOSTING_DOMAIN`      | unset            | Public host (and port) clients upload to and download from, e.g. `files.example.com`; required for uploads. `localhost…` URLs use `http://` and skip advertising |
| `HTTP_PORT`           | `8080`           | HTTP listen port                                                                                     |
| `BSV_NETWORK`         | `mainnet`        | `testnet` selects testnet; any other value selects mainnet                                            |
| `PRICE_PER_GB_MO`     | `0.03`           | Price in USD per GB (10⁹ bytes) per 30-day month                                                      |
| `MIN_HOSTING_MINUTES` | `0`              | Shortest retention period an upload or quote may ask for                                             |
| `PUBLIC_DIR`          | `public`         | Directory holding the `cdn` folder of uploaded files, relative to the working directory              |
| `SLAP_TRACKERS`       | 4 BSVB trackers  | Comma-separated SLAP trackers used to find `tm_uhrp` overlay hosts                                   |
| `LOG_LEVEL`           | `info`           | `debug`, `info`, `warn` or `error`                                                                   |
| `LOG_FORMAT`          | `json`           | `json` or `text`                                                                                     |

The default trackers are `overlay-us-1`, `overlay-eu-1` and `overlay-ap-1` at `bsvb.tech`, and `users.bapp.dev`.

</details>

<details>
<summary><strong><code>Upload Flow</code></strong></summary>
<br/>

1. **Quote** (optional): `POST /quote` with `{"fileSize": <bytes>, "retentionPeriod": <minutes>}` returns `{"quote": <satoshis>}`.
2. **Upload**: `POST /upload` with the same body, over BRC-31 auth with a BRC-29 payment for the quoted amount. The response carries a presigned `uploadURL`: `/put?fileSize=…&objectID=…&expiry=…&uploader=…&hmac=…`, HMAC-signed by the server wallet. Files may be up to 11 GB.
3. **Put**: `PUT` the file body to `uploadURL`. The server verifies the HMAC first, reads at most the authorized number of bytes, refuses a size mismatch or an existing object, and stores the file at `/cdn/<objectID>`.
4. **Advertise**: The server mints a 1-satoshi PushDrop advertisement holding the file's hash, URL, expiry and size, and broadcasts it to the `tm_uhrp` overlay topic. Admission is best effort: a host that refuses it is logged, and the file stays stored and served.
5. **Manage**: `GET /list`, `GET /find?uhrpUrl=…` and `POST /renew` (with `{"uhrpUrl": …, "additionalMinutes": …}`) work on the caller's advertisements. A renewal spends the old token and mints one with the later expiry.

On a `localhost` `HOSTING_DOMAIN` the file is stored but not advertised, and `/put` answers `500 No advertising in localhost.`

</details>

<details>
<summary><strong><code>Pricing</code></strong></summary>
<br/>

```
satoshis = max(10, floor(size_GB × retention_months × PRICE_PER_GB_MO × 10⁸ / BSV_USD))
```

`size_GB` is `fileSize / 10⁹` and `retention_months` is `retentionPeriod / 43,200` (30-day months). The BSV/USD rate comes from WhatsOnChain; a failed lookup is retried up to three times, and a rate fetched in the last 15 minutes is reused while the API is down. A quote may cover at most 69 million minutes (about 130 years).

</details>

<details>
<summary><strong><code>Development Build Commands</code></strong></summary>
<br/>

Get the [MAGE-X](https://github.com/mrz1836/mage-x) build tool for development:
```shell script
go install github.com/mrz1836/mage-x/cmd/magex@latest
```

View all build commands

```bash script
magex help
```

Regenerate the Swagger docs after changing a handler's annotations:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs --parseInternal
magex format:fix
```

</details>

<details>
<summary><strong>Repository Features</strong></summary>
<br/>

This repository includes 25+ built-in features covering CI/CD, security, code quality, developer experience, and community tooling.

**[View the full Repository Features list →](.github/docs/repository-features.md)**

</details>

<details>
<summary><strong><code>Releases & Docker Images</code></strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined releases to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.

Then create and push a new Git tag using:

```bash
magex version:bump push=true bump=patch branch=main
```

The [Docker workflow](.github/workflows/docker-publish.yml) builds the [Dockerfile](Dockerfile) on every pull request to `main`, and publishes the image to `ghcr.io/bsv-blockchain/go-uhrp-storage-server`:

| Trigger            | Image tags                    |
|--------------------|-------------------------------|
| Push to `main`     | `latest`, `<commit-sha>`      |
| `v*` release tag   | `<tag>` (e.g. `v1.2.3`), `<commit-sha>` |
| Pull request       | Built only, not pushed        |

</details>

<details>
<summary><strong><code>Pre-commit Hooks</code></strong></summary>
<br/>

Set up the Go-Pre-commit System to run the same formatting, linting, and tests defined in [AGENTS.md](.github/AGENTS.md) before every commit:

```bash
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest
go-pre-commit install
```

The system is configured via modular environment files in [`.github/env/`](.github/env/README.md) and provides 17x faster execution than traditional Python-based pre-commit hooks. See the [complete documentation](http://github.com/mrz1836/go-pre-commit) for details.

</details>

<details>
<summary><strong>GitHub Workflows</strong></summary>
<br/>

All workflows are driven by modular configuration in [`.github/env/`](.github/env/README.md); no YAML editing required.

**[View all workflows and the control center →](.github/docs/workflows.md)**

</details>

<details>
<summary><strong><code>Updating Dependencies</code></strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
magex deps:update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any tools managed by [MAGE-X](https://github.com/mrz1836/mage-x). It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<br/>

## 🏗️ Architecture

```
cmd/server/            - Entry point: configuration, wallet initialization, file store, HTTP server
internal/
  config/              - Environment variable loading
  logger/              - slog setup (level and JSON/text format)
  server/              - Router, CORS, route registration
    handlers/          - quote, upload, put, list, find, renew, and request pricing for payments
    middlewares/       - Identity-key extraction and CDN file serving
    responses/         - JSON and error responses
  storage/             - Local file store under PUBLIC_DIR/cdn (os.Root-confined)
  wallet/              - Server wallet: upload HMACs and PushDrop UHRP advertisements
pkg/
  pricing/             - Satoshi pricing and the WhatsOnChain BSV/USD rate provider
  uhrp/                - UHRP URL encoding and decoding
docs/                  - Generated OpenAPI (Swagger) spec
examples/              - Go clients for upload, list, find and renew
public/cdn/            - Uploaded files (default PUBLIC_DIR)
test-client/           - Jest integration tests (TypeScript)
```

A request passes through CORS, then either a public route (`/quote`, `/put`, `/cdn/*`, `/swagger/*`) or the BRC-31 auth, identity-key and BRC-29 payment middleware in front of the wallet routes.

<br/>

## 🧪 Examples & Tests

All unit, fuzz and example tests run via [GitHub Actions](https://github.com/bsv-blockchain/go-uhrp-storage-server/actions) and use [Go version 1.27.x](https://go.dev/doc/go1.27). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
magex test
```

Run all tests with race detector (slower):
```bash script
magex test:race
```

Run the fuzz tests, which cover UHRP URL parsing and round trips, price calculation, and the quote endpoint:
```bash script
magex test:fuzz
```

<details>
<summary><strong><code>Go Examples</code></strong></summary>
<br/>

Each [example](examples) authenticates with a `go-wallet-toolbox` wallet and talks to a running server through go-sdk's `storage.Uploader`. They use `UHRP_SERVER_URL` (default `http://localhost:8080`) and `WALLET_STORAGE_URL` (default `http://localhost:8100`) for the client wallet's storage.

| Example                                   | What it does                                          |
|-------------------------------------------|-------------------------------------------------------|
| [upload](examples/upload/upload.go)       | Quote, upload, find and list a file                   |
| [list](examples/list/list.go)             | List the caller's active uploads                      |
| [find_file](examples/find_file/find_file.go) | Look up a file's metadata by UHRP URL              |
| [renew](examples/renew/renew.go)          | Upload a file, then extend its retention              |

```bash
go run ./examples/upload
go run ./examples/find_file uhrp://<url>
```

</details>

<details>
<summary><strong><code>Jest Integration Tests</code></strong></summary>
<br/>

The `test-client/` directory holds integration tests that run against a live server with `@bsv/sdk`:

```bash
# Terminal 1: Start the server
SERVER_PRIVATE_KEY=$(openssl rand -hex 32) HTTP_PORT=8081 HOSTING_DOMAIN=localhost:8081 go run ./cmd/server

# Terminal 2: Run the tests
cd test-client
npm ci
npm run test:integration
```

**Always run (11):** quote pricing (valid requests, scaling with size and retention), input validation (missing fields, negative sizes, absurd retention periods), the JSON 404 for unknown routes, and UHRP URL encoding and decoding.

**Auth-gated (5):** upload and PUT with UHRP URL verification, list, find and renew. They need a wallet-enabled server on a public HTTPS origin, because `StorageUploader` refuses any other storage URL:

```bash
UHRP_AUTH_ENABLED=true UHRP_HOST=https://files.example.com npm run test:integration
```

`npm run test:diagnose` walks the public overlay network (SLAP trackers, `ls_uhrp` hosts, downloads) for one UHRP URL (`UHRP_URL`) to show where a lookup breaks.

</details>

<br/>

## ⚡ Benchmarks

Run the Go benchmarks:

```bash script
magex bench
```

<br/>

### Benchmark Results

| Benchmark                                                   | Iterations  |  ns/op | B/op  | allocs/op |
|-------------------------------------------------------------|------------:|-------:|------:|----------:|
| [CalculatorGetPrice](pkg/pricing/pricing_test.go)           | 155,853,988 |   7.68 |     0 |         0 |
| [HashData](pkg/uhrp/uhrp_benchmark_test.go) (1 KiB)         |   2,500,934 |  473.4 |     0 |         0 |
| [GetURLForHash](pkg/uhrp/uhrp_benchmark_test.go)            |     568,406 |  2,037 |   760 |        56 |
| [GetHashFromURL](pkg/uhrp/uhrp_benchmark_test.go)           |     628,782 |  1,879 |   360 |        10 |
| [QuoteHandler](internal/server/handlers/quote_test.go)      |     405,746 |  3,489 | 6,677 |        28 |

> Pricing excludes the exchange-rate lookup (a fixed-rate provider), and the quote handler benchmark covers JSON decoding, validation, pricing and the response.
> Performance benchmarks for the core functions in this server, executed on an Apple M1 Max (ARM64).

<br/>

## 🛠️ Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br/>

## 🤖 AI Usage & Assistant Guidelines
Read the [AI Usage & Assistant Guidelines](.github/tech-conventions/ai-compliance.md) for details on how AI is used in this project and how to interact with AI assistants.

<br/>

## 👥 Maintainers
| [<img src="https://github.com/pawellewandowski98.png" height="50" alt="Paweł" />](https://github.com/pawellewandowski98) | [<img src="https://github.com/sirdeggen.png" height="50" alt="Deggen" />](https://github.com/sirdeggen) | [<img src="https://github.com/mrz1836.png" height="50" width="50" alt="MrZ" />](https://github.com/mrz1836) |
|:------------------------------------------------------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------:|
|                                       [Paweł](https://github.com/pawellewandowski98)                                       |                                 [Deggen](https://github.com/sirdeggen)                                  |                                      [MrZ](https://github.com/mrz1836)                                      |

<br/>

## 🤝 Contributing
View the [contributing guidelines](.github/CONTRIBUTING.md) and please follow the [code of conduct](.github/CODE_OF_CONDUCT.md).

### How can I help?
All kinds of contributions are welcome :raised_hands:!
The most basic way to show your support is to star :star2: the project, or to raise issues :speech_balloon:.

[![Stars](https://img.shields.io/github/stars/bsv-blockchain/go-uhrp-storage-server?label=Please%20like%20us&style=social&v=1)](https://github.com/bsv-blockchain/go-uhrp-storage-server/stargazers)

<br/>

## 📝 License

[![License](https://img.shields.io/badge/license-Apache--2.0-blue?style=flat&logo=springsecurity&logoColor=white)](LICENSE)
