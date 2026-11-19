/**
 * Integration tests for the Go UHRP Storage Server
 * Uses @bsv/sdk StorageUploader where possible, plus direct fetch for lower-level endpoints.
 *
 * Based on: https://fast.brc.dev/?snippet=upload
 * Reference: https://github.com/bsv-blockchain/ts-sdk/blob/master/src/storage/StorageUploader.ts
 *
 * By default the suite builds the Go server and runs it on a free loopback port, with no wallet and
 * a local stub for its BSV/USD exchange rate, so it needs Go but no network. Set UHRP_HOST to test
 * a server you started yourself instead.
 *
 * Run: npm run test:integration
 */

import { PrivateKey, ProtoWallet, StorageUploader, StorageUtils } from '@bsv/sdk'
import { createHash } from 'crypto'

import { type LocalServer, PRICE_PER_GB_MONTH, STUB_BSV_USD_RATE, startLocalServer } from './support/local-server'

const EXTERNAL_SERVER_URL = process.env.UHRP_HOST
const localTest = EXTERNAL_SERVER_URL ? test.skip : test

let SERVER_URL = EXTERNAL_SERVER_URL ?? ''
let localServer: LocalServer | undefined

beforeAll(async () => {
  if (EXTERNAL_SERVER_URL) return
  localServer = await startLocalServer()
  SERVER_URL = localServer.url
}, 120_000)

afterAll(async () => {
  await localServer?.stop()
})

// Create a ProtoWallet for authenticated requests
const userKey = PrivateKey.fromRandom()
const userWallet = new ProtoWallet(userKey)

// StorageUploader (@bsv/sdk 2.8+) only accepts a public HTTPS origin, so it is created on first
// use by the auth tests. Everything else calls the server directly and also runs against a local
// http:// server.
let uploaderInstance: StorageUploader | undefined
function uploader(): StorageUploader {
  uploaderInstance ??= new StorageUploader({
    storageURL: SERVER_URL,
    wallet: userWallet as any,
  })
  return uploaderInstance
}

// Helper: make a direct (non-authed) JSON POST
async function postJSON(path: string, body: object): Promise<any> {
  const res = await fetch(`${SERVER_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return { status: res.status, data: await res.json() }
}

// Helper: make a direct (non-authed) GET
async function getJSON(path: string): Promise<any> {
  const res = await fetch(`${SERVER_URL}${path}`, { method: 'GET' })
  return { status: res.status, data: await res.json() }
}

describe('Go UHRP Storage Server — Integration Tests', () => {
  // ═══════════════════════════════════════════════════════════════════
  // Quote endpoint (POST /quote) — no auth required
  // ═══════════════════════════════════════════════════════════════════

  describe('POST /quote', () => {
    test('should return a price quote for a valid file', async () => {
      const { status, data } = await postJSON('/quote', {
        fileSize: 1024,
        retentionPeriod: 60, // 1 hour
      })

      expect(status).toBe(200)
      expect(data).toHaveProperty('quote')
      expect(typeof data.quote).toBe('number')
      expect(data.quote).toBeGreaterThanOrEqual(0)
    })

    test('should return higher or equal quote for larger files', async () => {
      const { data: small } = await postJSON('/quote', {
        fileSize: 1024,
        retentionPeriod: 1440, // 1 day
      })
      const { data: large } = await postJSON('/quote', {
        fileSize: 1024 * 1024 * 100, // 100 MB
        retentionPeriod: 1440,
      })

      expect(large.quote).toBeGreaterThanOrEqual(small.quote)
    })

    test('should return higher or equal quote for longer retention', async () => {
      const { data: short } = await postJSON('/quote', {
        fileSize: 1024 * 1024, // 1 MB
        retentionPeriod: 60, // 1 hour
      })
      const { data: long } = await postJSON('/quote', {
        fileSize: 1024 * 1024,
        retentionPeriod: 525600 * 10, // 10 years
      })

      expect(long.quote).toBeGreaterThanOrEqual(short.quote)
    })

    localTest('should price with the exchange rate from the stub', async () => {
      const fileSize = 5_000_000_000 // 5 GB
      const retentionPeriod = 43_200 * 3 // 3 months

      const { status, data } = await postJSON('/quote', { fileSize, retentionPeriod })

      // The server's formula, in the same float64 operations: USD price times satoshis per USD.
      const usd = (fileSize / 1e9) * (retentionPeriod / 43_200) * PRICE_PER_GB_MONTH
      const expected = Math.max(10, Math.floor(usd * (1 / (STUB_BSV_USD_RATE / 1e8))))

      expect(status).toBe(200)
      expect(data.quote).toBe(expected)
    })

    localTest('should fetch the exchange rate once and serve later quotes from its cache', async () => {
      for (let i = 0; i < 3; i++) {
        const { status } = await postJSON('/quote', { fileSize: 1024 * (i + 1), retentionPeriod: 60 })
        expect(status).toBe(200)
      }

      expect(localServer?.rateRequests()).toBe(1)
    })

    test('should reject missing fileSize', async () => {
      const { status, data } = await postJSON('/quote', {
        retentionPeriod: 60,
      })

      expect(status).toBe(400)
      expect(data).toHaveProperty('code')
    })

    test('should reject missing retentionPeriod', async () => {
      const { status, data } = await postJSON('/quote', {
        fileSize: 1024,
      })

      expect(status).toBe(400)
      expect(data).toHaveProperty('code')
    })

    test('should reject negative fileSize', async () => {
      const { status } = await postJSON('/quote', {
        fileSize: -100,
        retentionPeriod: 60,
      })

      expect(status).toBe(400)
    })

    test('should reject absurdly long retention (>69M minutes)', async () => {
      const { status, data } = await postJSON('/quote', {
        fileSize: 1024,
        retentionPeriod: 70_000_000,
      })

      expect(status).toBe(400)
      expect(data.code).toBe('ERR_INVALID_RETENTION_PERIOD')
    })
  })

  // ═══════════════════════════════════════════════════════════════════
  // Upload flow (POST /upload → PUT /put) — requires auth
  // ═══════════════════════════════════════════════════════════════════

  // Upload, List, Find and Renew need BRC-31 auth, so the server must run with a wallet
  // (SERVER_PRIVATE_KEY and WALLET_STORAGE_URL), and StorageUploader needs it on a public
  // HTTPS origin: run them with UHRP_AUTH_ENABLED=true UHRP_HOST=https://your-host.
  const authTest = process.env.UHRP_AUTH_ENABLED === 'true' ? test : test.skip

  describe('Upload + Put flow', () => {
    authTest('should get upload URL via StorageUploader and PUT a file', async () => {
      const fileContent = Buffer.from('Hello, UHRP world!')
      const fileData = new Uint8Array(fileContent)

      // publishFile uses authFetch for /upload, then direct fetch for PUT
      const result = await uploader().publishFile({
        file: { data: fileData, type: 'text/plain' },
        retentionPeriod: 60, // 1 hour
      })

      expect(result.published).toBe(true)
      expect(result.uhrpURL).toBeDefined()
      expect(typeof result.uhrpURL).toBe('string')
      expect(result.uhrpURL.length).toBeGreaterThan(0)

      // Verify the UHRP URL decodes to the correct SHA-256 hash
      const expectedHash = createHash('sha256').update(fileContent).digest()
      const urlHash = StorageUtils.getHashFromURL(result.uhrpURL)
      expect(Buffer.from(urlHash)).toEqual(expectedHash)
    })

    authTest('should upload a binary file and produce valid UHRP URL', async () => {
      const binaryData = new Uint8Array(256)
      for (let i = 0; i < 256; i++) binaryData[i] = i

      const result = await uploader().publishFile({
        file: { data: binaryData, type: 'application/octet-stream' },
        retentionPeriod: 120,
      })

      expect(result.published).toBe(true)
      expect(StorageUtils.isValidURL(result.uhrpURL)).toBe(true)
    })
  })

  // ═══════════════════════════════════════════════════════════════════
  // List uploads (GET /list) — requires auth
  // ═══════════════════════════════════════════════════════════════════

  describe('GET /list', () => {
    authTest('should list uploads for the authenticated user', async () => {
      const uploads = await uploader().listUploads()

      expect(Array.isArray(uploads)).toBe(true)
      // After previous upload tests, we should have at least some entries
      // (depends on wallet integration; may be empty if wallet stubs return [])
    })
  })

  // ═══════════════════════════════════════════════════════════════════
  // Find file (GET /find) — requires auth
  // ═══════════════════════════════════════════════════════════════════

  describe('GET /find', () => {
    authTest('should return error for a non-existent UHRP URL', async () => {
      await expect(
        uploader().findFile('XUT6PqWb3GP3LR7dmBMCJwZ3oo5g1iGCF3CrpzyuJCemkGu1WGoq')
      ).rejects.toThrow()
    })
  })

  // ═══════════════════════════════════════════════════════════════════
  // Renew (POST /renew) — requires auth
  // ═══════════════════════════════════════════════════════════════════

  describe('POST /renew', () => {
    authTest('should return error when renewing a non-existent file', async () => {
      await expect(
        uploader().renewFile('XUT6PqWb3GP3LR7dmBMCJwZ3oo5g1iGCF3CrpzyuJCemkGu1WGoq', 60)
      ).rejects.toThrow()
    })
  })

  // ═══════════════════════════════════════════════════════════════════
  // 404 handler
  // ═══════════════════════════════════════════════════════════════════

  describe('Unknown routes', () => {
    test('should return 404 JSON for unknown routes', async () => {
      const { status, data } = await getJSON('/nonexistent')

      expect(status).toBe(404)
      expect(data.code).toBe('ERR_ROUTE_NOT_FOUND')
    })
  })

  // ═══════════════════════════════════════════════════════════════════
  // UHRP URL utilities (from @bsv/sdk StorageUtils)
  // ═══════════════════════════════════════════════════════════════════

  describe('StorageUtils (UHRP URL encoding)', () => {
    test('should produce a valid UHRP URL from file data', () => {
      const data = Buffer.from('test file content')
      const url = StorageUtils.getURLForFile(Array.from(data))

      expect(typeof url).toBe('string')
      expect(url.length).toBeGreaterThan(0)
      expect(StorageUtils.isValidURL(url)).toBe(true)
    })

    test('should round-trip: file → URL → hash matches SHA-256', () => {
      const data = Buffer.from('round trip test data')
      const url = StorageUtils.getURLForFile(Array.from(data))
      const hash = StorageUtils.getHashFromURL(url)
      const expected = createHash('sha256').update(data).digest()

      expect(Buffer.from(hash)).toEqual(expected)
    })

    test('should reject invalid UHRP URLs', () => {
      expect(StorageUtils.isValidURL('notavalidurl')).toBe(false)
      expect(StorageUtils.isValidURL('')).toBe(false)
    })
  })
})
