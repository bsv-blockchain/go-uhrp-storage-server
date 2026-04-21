/**
 * UHRP lookup diagnostic.
 *
 * Walks every network hop that StorageDownloader.download() makes so we can
 * see exactly where the lookup breaks for a given UHRP URL.
 *
 * Run:
 *   npx jest uhrp-debug --verbose
 *   UHRP_URL=XUSzUk... npx jest uhrp-debug --verbose
 */

import {
  DEFAULT_SLAP_TRACKERS,
  LookupResolver,
  PushDrop,
  OverlayAdminTokenTemplate,
  Transaction,
  Utils,
  StorageUtils,
  Hash,
} from '@bsv/sdk'

const UHRP_URL = process.env.UHRP_URL ?? 'XUSzUkfq8SSqLQEn2LL98gcxBF6MwTCzuxPrnuYwiRQpi6fp7W6U'
const TIMEOUT_MS = 8000

// ─── helpers ────────────────────────────────────────────────────────────────

async function post(url: string, body: unknown): Promise<unknown> {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), TIMEOUT_MS)
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Aggregation': 'yes' },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    })
    if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`)
    return await res.json()
  } finally {
    clearTimeout(timer)
  }
}

// ─── tests ───────────────────────────────────────────────────────────────────

describe('UHRP lookup diagnostics', () => {
  let lsUhrpHosts: string[] = []
  let downloadURLs: string[] = []

  test('UHRP URL is valid and decodes to a hash', () => {
    console.log('\nUHRP URL:', UHRP_URL)
    expect(StorageUtils.isValidURL(UHRP_URL)).toBe(true)
    const hash = StorageUtils.getHashFromURL(UHRP_URL)
    console.log('SHA-256 (hex):', Utils.toHex(hash))
    expect(hash.length).toBe(32)
  })

  test('SLAP trackers respond to ls_uhrp query', async () => {
    console.log('\nDefault SLAP trackers:', DEFAULT_SLAP_TRACKERS)

    for (const tracker of DEFAULT_SLAP_TRACKERS) {
      console.log(`\n  → ${tracker}/lookup`)
      try {
        const raw = await post(`${tracker}/lookup`, {
          service: 'ls_slap',
          query: { service: 'ls_uhrp' },
        }) as any

        console.log('    type:', raw?.type)
        const outputs: any[] = raw?.outputs ?? []
        console.log('    outputs:', outputs.length)

        for (const output of outputs) {
          try {
            const tx = Transaction.fromBEEF(output.beef)
            const script = tx.outputs[output.outputIndex]?.lockingScript
            if (!script) continue
            const parsed = OverlayAdminTokenTemplate.decode(script)
            console.log('    token:', JSON.stringify({ protocol: parsed.protocol, service: parsed.topicOrService, domain: parsed.domain }))
            if (parsed.topicOrService === 'ls_uhrp' && parsed.protocol === 'SLAP') {
              if (!lsUhrpHosts.includes(parsed.domain)) lsUhrpHosts.push(parsed.domain)
            }
          } catch (e: any) {
            console.log('    parse error:', e.message)
          }
        }
      } catch (e: any) {
        console.log('    FAILED:', e.message)
      }
    }

    console.log('\nls_uhrp hosts found:', lsUhrpHosts)
    // Not asserting — we just want to see what's there.
    // If this is empty that IS the problem.
    if (lsUhrpHosts.length === 0) {
      console.warn('\n⚠️  NO ls_uhrp HOSTS found via any SLAP tracker.')
      console.warn('   This means the overlay lookup network is not aware of any UHRP service.')
      console.warn('   Uploads succeed but downloads (and advertisement visibility) will always fail.')
    }
  }, 30_000)

  test('ls_uhrp hosts return advertisement for the UHRP URL', async () => {
    if (lsUhrpHosts.length === 0) {
      console.log('Skipping — no ls_uhrp hosts discovered in previous step')
      return
    }

    const currentTime = Math.floor(Date.now() / 1000)

    for (const host of lsUhrpHosts) {
      console.log(`\n  → ${host}/lookup (ls_uhrp)`)
      try {
        const raw = await post(`${host}/lookup`, {
          service: 'ls_uhrp',
          query: { uhrpUrl: UHRP_URL },
        }) as any

        console.log('    type:', raw?.type)
        const outputs: any[] = raw?.outputs ?? []
        console.log('    outputs:', outputs.length)

        for (const output of outputs) {
          try {
            const tx = Transaction.fromBEEF(output.beef)
            const { fields } = PushDrop.decode(tx.outputs[output.outputIndex].lockingScript)

            const expiryTime = new Utils.Reader(fields[3]).readVarIntNum()
            const cdnUrl = Utils.toUTF8(fields[2])
            const expired = expiryTime < currentTime

            console.log('    ad:', JSON.stringify({ cdnUrl, expiryTime, expired, expiresIn: expiryTime - currentTime }))

            if (!expired) downloadURLs.push(cdnUrl)
          } catch (e: any) {
            console.log('    decode error:', e.message)
          }
        }
      } catch (e: any) {
        console.log('    FAILED:', e.message)
      }
    }

    console.log('\nValid download URLs:', downloadURLs)
  }, 30_000)

  test('download URLs actually serve the file with matching hash', async () => {
    if (downloadURLs.length === 0) {
      console.log('Skipping — no valid download URLs from previous step')
      return
    }

    const expectedHex = Utils.toHex(StorageUtils.getHashFromURL(UHRP_URL))

    for (const url of downloadURLs) {
      console.log(`\n  → GET ${url}`)
      try {
        const ctrl = new AbortController()
        const timer = setTimeout(() => ctrl.abort(), TIMEOUT_MS)
        const res = await fetch(url, { signal: ctrl.signal })
        clearTimeout(timer)

        console.log('    status:', res.status)
        console.log('    Content-Type:', res.headers.get('Content-Type'))

        if (!res.ok) { console.log('    not ok, skip'); continue }

        const data = new Uint8Array(await res.arrayBuffer())
        console.log('    bytes:', data.length)

        const hashStream = new Hash.SHA256()
        hashStream.update(Array.from(data))
        const actualHex = Utils.toHex(hashStream.digest())
        console.log('    expected hash:', expectedHex)
        console.log('    actual hash:  ', actualHex)
        console.log('    hash match:', actualHex === expectedHex)
        expect(actualHex).toBe(expectedHex)
        return // first working host is enough
      } catch (e: any) {
        console.log('    FAILED:', e.message)
      }
    }
  }, 30_000)

  test('LookupResolver (SDK) resolves and decodes advertisements', async () => {
    const resolver = new LookupResolver({ networkPreset: 'mainnet' })

    let answer: any
    try {
      answer = await resolver.query({ service: 'ls_uhrp', query: { uhrpUrl: UHRP_URL } })
    } catch (e: any) {
      console.log('resolver.query threw:', e.message)
      return
    }

    console.log('\nanswer type:', answer.type)
    const outputs: any[] = answer.outputs ?? []
    console.log('answer outputs:', outputs.length)

    if (!outputs.length) {
      console.warn('⚠️  resolver returned 0 outputs — StorageDownloader will throw "No one currently hosts this file!"')
      return
    }

    const currentTime = Math.floor(Date.now() / 1000)

    for (let i = 0; i < outputs.length; i++) {
      const output = outputs[i]
      try {
        const tx = Transaction.fromBEEF(output.beef)
        const { fields } = PushDrop.decode(tx.outputs[output.outputIndex].lockingScript)
        const expiryTime = new Utils.Reader(fields[3]).readVarIntNum()
        const cdnUrl = Utils.toUTF8(fields[2])
        const expired = expiryTime < currentTime
        console.log(`  [${i}] cdnUrl=${cdnUrl} expiry=${expiryTime} expired=${expired} expiresIn=${expiryTime - currentTime}s`)
        if (!expired) downloadURLs.push(cdnUrl)
      } catch (e: any) {
        console.log(`  [${i}] decode error: ${e.message}`)
      }
    }

    console.log('\nValid (non-expired) download URLs:', downloadURLs)
  }, 30_000)

  test('download URLs serve file with matching hash', async () => {
    if (downloadURLs.length === 0) {
      console.log('No valid download URLs — skipping')
      return
    }

    const expectedHex = Utils.toHex(StorageUtils.getHashFromURL(UHRP_URL))
    console.log('\nExpected hash:', expectedHex)

    for (const url of downloadURLs) {
      console.log(`\n  → GET ${url}`)
      try {
        const ctrl = new AbortController()
        const timer = setTimeout(() => ctrl.abort(), TIMEOUT_MS)
        const res = await fetch(url, { signal: ctrl.signal })
        clearTimeout(timer)

        console.log('    status:', res.status, res.statusText)
        console.log('    Content-Type:', res.headers.get('Content-Type'))

        if (!res.ok) { console.log('    not ok, trying next'); continue }

        const data = new Uint8Array(await res.arrayBuffer())
        console.log('    bytes:', data.length)

        const hashStream = new Hash.SHA256()
        hashStream.update(Array.from(data))
        const actualHex = Utils.toHex(hashStream.digest())
        console.log('    hash match:', actualHex === expectedHex)
        expect(actualHex).toBe(expectedHex)
        return
      } catch (e: any) {
        console.log('    FAILED:', e.message)
      }
    }

    throw new Error('All download URLs failed')
  }, 30_000)
})
