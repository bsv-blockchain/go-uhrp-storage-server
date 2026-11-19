/**
 * An in-memory stand-in for the UHRP overlay network: a SLAP tracker, an ls_uhrp lookup host and a
 * CDN, all reached through `MockOverlay.fetch`, so lookups and downloads run without the network.
 *
 * The tokens are real. SLAP advertisements are minted with the SDK's OverlayAdminTokenTemplate, and
 * UHRP advertisements carry the same PushDrop fields, protocol and key the Go server signs with, so
 * the SDK's own signature checks run against them. Every origin sits under `.test`, a TLD reserved
 * for testing (RFC 6761) that never resolves, so nothing can leak to a real host by accident.
 */
import {
  type LockingScript,
  OverlayAdminTokenTemplate,
  PrivateKey,
  ProtoWallet,
  PushDrop,
  StorageUtils,
  Transaction,
  UHRP_ADVERTISEMENT_KEY_ID,
  UHRP_ADVERTISEMENT_PROTOCOL,
  Utils,
  type WalletInterface,
} from '@bsv/sdk'

/** SLAP tracker that knows which hosts run which lookup services. */
export const MOCK_TRACKER = 'https://tracker.uhrp.test'
/** SLAP tracker that is always down (HTTP 503). */
export const MOCK_DOWN_TRACKER = 'https://down-tracker.uhrp.test'
/** Overlay host running the ls_uhrp lookup service. */
export const MOCK_LOOKUP_HOST = 'https://lookup.uhrp.test'
/** CDN serving the hosted files under /cdn/<id>. */
export const MOCK_CDN = 'https://cdn.uhrp.test'

const MOCK_ORIGINS = new Set([MOCK_TRACKER, MOCK_DOWN_TRACKER, MOCK_LOOKUP_HOST, MOCK_CDN])

/** One output of an overlay lookup answer. */
export interface LookupOutput {
  beef: number[]
  outputIndex: number
}

/** A file published on the mock overlay. */
export interface HostedFile {
  uhrpUrl: string
  cdnUrl: string
}

interface LookupQuestion {
  service?: string
  query?: { service?: string; uhrpUrl?: string }
}

/** Wraps a locking script in a one-satoshi output, the shape overlay hosts answer with. */
function toLookupOutput(lockingScript: LockingScript): LookupOutput {
  const tx = new Transaction()
  tx.addOutput({ lockingScript, satoshis: 1 })
  return { beef: tx.toBEEF(), outputIndex: 0 }
}

function outputList(outputs: LookupOutput[]): Response {
  return Response.json({ type: 'output-list', outputs })
}

/** The hex SHA-256 a UHRP URL names, or undefined when it is not a UHRP URL. */
function hashOf(uhrpUrl: string | undefined): string | undefined {
  if (uhrpUrl === undefined || !StorageUtils.isValidURL(uhrpUrl)) return undefined
  return Utils.toHex(StorageUtils.getHashFromURL(uhrpUrl))
}

export class MockOverlay {
  /** Every URL requested through `fetch`, in order. */
  readonly requests: string[] = []

  private readonly hostWallet = new ProtoWallet(PrivateKey.fromRandom())
  private readonly slapOutputs: Array<{ service: string; output: LookupOutput }> = []
  private readonly advertisements = new Map<string, LookupOutput[]>()
  private readonly files = new Map<string, Uint8Array<ArrayBuffer>>()

  /** Publishes a SLAP token on the mock tracker naming `domain` as a host of lookup `service`. */
  async advertiseLookupService(domain: string, service: string): Promise<void> {
    // The template only calls getPublicKey and createSignature, which ProtoWallet implements.
    const wallet = this.hostWallet as unknown as WalletInterface
    const lockingScript = await new OverlayAdminTokenTemplate(wallet).lock('SLAP', domain, service)
    this.slapOutputs.push({ service, output: toLookupOutput(lockingScript) })
  }

  /** Adds an output that is not a transaction at all to the tracker's answers for `service`. */
  addMalformedSlapOutput(service: string): void {
    this.slapOutputs.push({ service, output: { beef: [1, 2, 3], outputIndex: 0 } })
  }

  /**
   * Advertises `data` on the ls_uhrp host until `expiryTime` (Unix seconds) and serves `served`
   * (the file itself unless a test wants a tampered copy) at MOCK_CDN/cdn/<id>.
   */
  async hostFile(id: string, data: Uint8Array, expiryTime: number, served: Uint8Array = data): Promise<HostedFile> {
    const uhrpUrl = StorageUtils.getURLForFile(Array.from(data))
    const cdnUrl = `${MOCK_CDN}/cdn/${id}`
    const { publicKey } = await this.hostWallet.getPublicKey({ identityKey: true })

    // The Go server's field order: host identity key, content hash, URL, expiry, content length.
    const fields = [
      Utils.toArray(publicKey, 'hex'),
      StorageUtils.getHashFromURL(uhrpUrl),
      Utils.toArray(cdnUrl, 'utf8'),
      new Utils.Writer().writeVarIntNum(expiryTime).toArray(),
      new Utils.Writer().writeVarIntNum(data.length).toArray(),
    ]
    const lockingScript = await new PushDrop(this.hostWallet).lock(
      fields,
      UHRP_ADVERTISEMENT_PROTOCOL,
      UHRP_ADVERTISEMENT_KEY_ID,
      'anyone',
      true,
      true,
      'before'
    )

    const hash = Utils.toHex(StorageUtils.getHashFromURL(uhrpUrl))
    this.advertisements.set(hash, [...(this.advertisements.get(hash) ?? []), toLookupOutput(lockingScript)])
    this.files.set(new URL(cdnUrl).pathname, new Uint8Array(served))

    return { uhrpUrl, cdnUrl }
  }

  /** A `fetch` answered by the mock overlay. Any other host fails the way an unresolvable name does. */
  readonly fetch = async (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
    const url = new URL(input instanceof Request ? input.url : input)
    this.requests.push(url.href)

    if (!MOCK_ORIGINS.has(url.origin)) {
      throw new TypeError(`fetch failed: ${url.host} is not part of the mock overlay`)
    }

    if (url.origin === MOCK_DOWN_TRACKER) {
      return new Response('unavailable', { status: 503, statusText: 'Service Unavailable' })
    }

    if (url.pathname === '/lookup' && init?.method === 'POST') {
      return this.lookup(url.origin, JSON.parse(String(init.body)) as LookupQuestion)
    }

    if (url.origin === MOCK_CDN && (init?.method ?? 'GET') === 'GET') {
      const file = this.files.get(url.pathname)
      if (file === undefined) return new Response('not found', { status: 404, statusText: 'Not Found' })
      return new Response(file, { headers: { 'Content-Type': 'application/octet-stream' } })
    }

    return new Response('not found', { status: 404, statusText: 'Not Found' })
  }

  private lookup(origin: string, question: LookupQuestion): Response {
    if (origin === MOCK_TRACKER && question.service === 'ls_slap') {
      return outputList(this.slapOutputs.filter(s => s.service === question.query?.service).map(s => s.output))
    }

    if (origin === MOCK_LOOKUP_HOST && question.service === 'ls_uhrp') {
      const hash = hashOf(question.query?.uhrpUrl)
      return outputList(hash === undefined ? [] : (this.advertisements.get(hash) ?? []))
    }

    return Response.json({ status: 'error', description: `No lookup service ${String(question.service)} here` }, { status: 400 })
  }
}
