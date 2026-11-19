/**
 * The network hops StorageDownloader.download() makes for a UHRP URL, one function per hop, so a
 * diagnostic can show which hop breaks. Every function takes the `fetch` (or resolver) to use: the
 * tests hand them the mock overlay, and the live diagnostic hands them the real network.
 */
import {
  decodeAndVerifyUHRPAdvertisement,
  Hash,
  type LookupResolver,
  OverlayAdminTokenTemplate,
  StorageUtils,
  Transaction,
  Utils,
} from '@bsv/sdk'

export type Fetch = (input: string | URL | Request, init?: RequestInit) => Promise<Response>

/** One output of an overlay lookup answer. */
export interface LookupOutput {
  beef: number[]
  outputIndex: number
}

/** A host that did not answer, or an output that could not be used, and why. */
export interface Failure {
  url: string
  error: string
}

/** A verified UHRP advertisement and whether StorageDownloader would use it. */
export interface Advertisement {
  source: string
  cdnUrl: string
  expiryTime: number
  expired: boolean
  /** Whether the advertised hash is the one the UHRP URL names. */
  hashMatches: boolean
}

/** One download attempt for a UHRP URL. */
export interface DownloadAttempt {
  url: string
  status?: number
  bytes?: number
  hashMatch?: boolean
  error?: string
}

function errorMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

function lockingScriptOf(output: LookupOutput) {
  const script = Transaction.fromBEEF(output.beef).outputs[output.outputIndex]?.lockingScript
  if (script == null) throw new Error(`BEEF has no output ${output.outputIndex}`)
  return script
}

/** Posts a lookup question to an overlay host and returns the outputs of its answer. */
export async function lookup(
  fetchFn: Fetch,
  host: string,
  question: { service: string; query: unknown },
  timeoutMs: number
): Promise<LookupOutput[]> {
  // No X-Aggregation header: hosts then answer with a JSON output list, not the binary aggregate.
  const res = await fetchFn(`${host}/lookup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(question),
    signal: AbortSignal.timeout(timeoutMs),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`.trim())

  const answer = (await res.json()) as { type?: string; outputs?: LookupOutput[] }
  if (answer.type !== 'output-list') throw new Error(`answer type is ${String(answer.type)}, not output-list`)
  return answer.outputs ?? []
}

/** Asks every SLAP tracker, in parallel, for the hosts of ls_uhrp whose SLAP token verifies. */
export async function discoverUhrpHosts(
  fetchFn: Fetch,
  trackers: string[],
  timeoutMs: number
): Promise<{ hosts: string[]; failures: Failure[] }> {
  const hosts = new Set<string>()
  const failures: Failure[] = []

  await Promise.all(
    trackers.map(async tracker => {
      let outputs: LookupOutput[]
      try {
        outputs = await lookup(fetchFn, tracker, { service: 'ls_slap', query: { service: 'ls_uhrp' } }, timeoutMs)
      } catch (e) {
        failures.push({ url: tracker, error: errorMessage(e) })
        return
      }

      for (const [i, output] of outputs.entries()) {
        try {
          const ad = await OverlayAdminTokenTemplate.decodeAndVerify(lockingScriptOf(output), 'SLAP')
          if (ad.topicOrService === 'ls_uhrp') hosts.add(ad.domain)
        } catch (e) {
          failures.push({ url: tracker, error: `output ${i}: ${errorMessage(e)}` })
        }
      }
    })
  )

  return { hosts: [...hosts].sort(), failures }
}

/** Verifies a UHRP advertisement token the way StorageDownloader does and reports what it says. */
export async function decodeAdvertisement(
  output: LookupOutput,
  source: string,
  uhrpUrl: string,
  now: number
): Promise<Advertisement> {
  const token = await decodeAndVerifyUHRPAdvertisement(lockingScriptOf(output))
  return {
    source,
    cdnUrl: token.hostedFileLocation,
    expiryTime: token.expiryTime,
    expired: token.expiryTime < now,
    hashMatches: Utils.toHex(token.hash) === Utils.toHex(StorageUtils.getHashFromURL(uhrpUrl)),
  }
}

async function decodeAll(
  outputs: LookupOutput[],
  source: string,
  uhrpUrl: string,
  now: number
): Promise<{ ads: Advertisement[]; failures: Failure[] }> {
  const ads: Advertisement[] = []
  const failures: Failure[] = []

  for (const [i, output] of outputs.entries()) {
    try {
      ads.push(await decodeAdvertisement(output, source, uhrpUrl, now))
    } catch (e) {
      failures.push({ url: source, error: `output ${i}: ${errorMessage(e)}` })
    }
  }

  return { ads, failures }
}

/** Asks every ls_uhrp host, in parallel, for the advertisements of uhrpUrl. */
export async function findAdvertisements(
  fetchFn: Fetch,
  hosts: string[],
  uhrpUrl: string,
  now: number,
  timeoutMs: number
): Promise<{ ads: Advertisement[]; failures: Failure[] }> {
  const results = await Promise.all(
    hosts.map(async host => {
      try {
        const outputs = await lookup(fetchFn, host, { service: 'ls_uhrp', query: { uhrpUrl } }, timeoutMs)
        return await decodeAll(outputs, host, uhrpUrl, now)
      } catch (e) {
        return { ads: [], failures: [{ url: host, error: errorMessage(e) }] }
      }
    })
  )

  return { ads: results.flatMap(r => r.ads), failures: results.flatMap(r => r.failures) }
}

/** Resolves uhrpUrl through the SDK's LookupResolver: SLAP discovery, then the ls_uhrp lookup. */
export async function resolveAdvertisements(
  resolver: LookupResolver,
  uhrpUrl: string,
  now: number
): Promise<{ ads: Advertisement[]; failures: Failure[] }> {
  const answer = await resolver.query({ service: 'ls_uhrp', query: { uhrpUrl } })
  return await decodeAll(answer.outputs, 'LookupResolver', uhrpUrl, now)
}

/** The download URLs StorageDownloader would try: unexpired, matching advertisements, deduplicated. */
export function usableDownloadURLs(ads: Advertisement[]): string[] {
  return [...new Set(ads.filter(ad => !ad.expired && ad.hashMatches).map(ad => ad.cdnUrl))]
}

/** Downloads from each URL in turn until one serves content whose SHA-256 matches uhrpUrl. */
export async function downloadVerified(
  fetchFn: Fetch,
  urls: string[],
  uhrpUrl: string,
  timeoutMs: number
): Promise<{ verifiedUrl?: string; attempts: DownloadAttempt[] }> {
  const expected = Utils.toHex(StorageUtils.getHashFromURL(uhrpUrl))
  const attempts: DownloadAttempt[] = []

  for (const url of urls) {
    try {
      const res = await fetchFn(url, { redirect: 'error', signal: AbortSignal.timeout(timeoutMs) })
      if (!res.ok) {
        attempts.push({ url, status: res.status })
        continue
      }

      const data = new Uint8Array(await res.arrayBuffer())
      const hashMatch = Utils.toHex(Hash.sha256(Array.from(data))) === expected
      attempts.push({ url, status: res.status, bytes: data.length, hashMatch })
      if (hashMatch) return { verifiedUrl: url, attempts }
    } catch (e) {
      attempts.push({ url, error: errorMessage(e) })
    }
  }

  return { attempts }
}
