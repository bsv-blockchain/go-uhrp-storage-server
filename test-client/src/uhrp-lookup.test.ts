/**
 * UHRP lookup diagnostics against the in-memory mock overlay: SLAP discovery, the ls_uhrp lookup,
 * the SDK's LookupResolver and the hash-checked download, with no network at all.
 * The same functions walk the real overlay in uhrp-debug.test.ts (UHRP_LIVE=true).
 */
import { HTTPSOverlayLookupFacilitator, LookupResolver } from '@bsv/sdk'

import {
  discoverUhrpHosts,
  downloadVerified,
  findAdvertisements,
  resolveAdvertisements,
  usableDownloadURLs,
} from './diagnostics/uhrp-lookup'
import {
  type HostedFile,
  MOCK_CDN,
  MOCK_DOWN_TRACKER,
  MOCK_LOOKUP_HOST,
  MOCK_TRACKER,
  MockOverlay,
} from './support/mock-overlay'

const TIMEOUT_MS = 1000
const content = new TextEncoder().encode('Hello from the mock overlay!')

describe('UHRP lookup diagnostics (mock overlay)', () => {
  const now = Math.floor(Date.now() / 1000)
  let overlay: MockOverlay
  let current: HostedFile
  let expired: HostedFile

  beforeAll(async () => {
    overlay = new MockOverlay()
    await overlay.advertiseLookupService(MOCK_LOOKUP_HOST, 'ls_uhrp')
    await overlay.advertiseLookupService('https://other.uhrp.test', 'ls_other')
    overlay.addMalformedSlapOutput('ls_uhrp')

    // The same file advertised twice: once until an hour from now, once until an hour ago.
    current = await overlay.hostFile('current', content, now + 3600)
    expired = await overlay.hostFile('expired', content, now - 3600)
  })

  describe('SLAP discovery', () => {
    test('finds the verified ls_uhrp host and reports trackers and outputs it could not use', async () => {
      const { hosts, failures } = await discoverUhrpHosts(overlay.fetch, [MOCK_TRACKER, MOCK_DOWN_TRACKER], TIMEOUT_MS)

      expect(hosts).toEqual([MOCK_LOOKUP_HOST])
      expect(failures).toEqual(
        expect.arrayContaining([
          { url: MOCK_DOWN_TRACKER, error: 'HTTP 503 Service Unavailable' },
          { url: MOCK_TRACKER, error: expect.stringMatching(/^output 1: /) },
        ])
      )
      expect(failures).toHaveLength(2)
    })

    test('reports a tracker that cannot be reached', async () => {
      const { hosts, failures } = await discoverUhrpHosts(overlay.fetch, ['https://gone.uhrp.test'], TIMEOUT_MS)

      expect(hosts).toEqual([])
      expect(failures).toEqual([{ url: 'https://gone.uhrp.test', error: expect.stringContaining('fetch failed') }])
    })
  })

  describe('ls_uhrp lookup', () => {
    test('returns every advertisement for the file and flags the expired one', async () => {
      const { ads, failures } = await findAdvertisements(overlay.fetch, [MOCK_LOOKUP_HOST], current.uhrpUrl, now, TIMEOUT_MS)

      expect(failures).toEqual([])
      expect(ads).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ cdnUrl: current.cdnUrl, expired: false, hashMatches: true }),
          expect.objectContaining({ cdnUrl: expired.cdnUrl, expired: true, hashMatches: true }),
        ])
      )
      expect(usableDownloadURLs(ads)).toEqual([current.cdnUrl])
    })

    test('returns nothing for a file no one hosts', async () => {
      const unhosted = 'XUT6PqWb3GP3LR7dmBMCJwZ3oo5g1iGCF3CrpzyuJCemkGu1WGoq'
      const { ads, failures } = await findAdvertisements(overlay.fetch, [MOCK_LOOKUP_HOST], unhosted, now, TIMEOUT_MS)

      expect(ads).toEqual([])
      expect(failures).toEqual([])
    })
  })

  describe('SDK LookupResolver', () => {
    test('discovers the host through SLAP, verifies the tokens and returns both advertisements', async () => {
      const resolver = new LookupResolver({
        slapTrackers: [MOCK_TRACKER, MOCK_DOWN_TRACKER],
        facilitator: new HTTPSOverlayLookupFacilitator(overlay.fetch),
      })

      const { ads, failures } = await resolveAdvertisements(resolver, current.uhrpUrl, now)

      expect(failures).toEqual([])
      expect(ads.map(ad => ad.cdnUrl).sort()).toEqual([current.cdnUrl, expired.cdnUrl].sort())
      expect(usableDownloadURLs(ads)).toEqual([current.cdnUrl])
      expect(overlay.requests).toEqual(expect.arrayContaining([`${MOCK_TRACKER}/lookup`, `${MOCK_LOOKUP_HOST}/lookup`]))
    })
  })

  describe('download', () => {
    test('downloads the file and verifies its hash', async () => {
      const { verifiedUrl, attempts } = await downloadVerified(overlay.fetch, [current.cdnUrl], current.uhrpUrl, TIMEOUT_MS)

      expect(verifiedUrl).toBe(current.cdnUrl)
      expect(attempts).toEqual([{ url: current.cdnUrl, status: 200, bytes: content.length, hashMatch: true }])
    })

    test('skips a host serving different bytes and a missing file, then uses the next URL', async () => {
      const tampered = await overlay.hostFile('tampered', content, now + 3600, new TextEncoder().encode('not the file'))
      const missing = `${MOCK_CDN}/cdn/missing`

      const { verifiedUrl, attempts } = await downloadVerified(
        overlay.fetch,
        [tampered.cdnUrl, missing, current.cdnUrl],
        current.uhrpUrl,
        TIMEOUT_MS
      )

      expect(verifiedUrl).toBe(current.cdnUrl)
      expect(attempts).toEqual([
        expect.objectContaining({ url: tampered.cdnUrl, status: 200, hashMatch: false }),
        { url: missing, status: 404 },
        expect.objectContaining({ url: current.cdnUrl, hashMatch: true }),
      ])
    })

    test('reports no verified URL when every download fails', async () => {
      const { verifiedUrl, attempts } = await downloadVerified(
        overlay.fetch,
        ['https://gone.uhrp.test/cdn/x'],
        current.uhrpUrl,
        TIMEOUT_MS
      )

      expect(verifiedUrl).toBeUndefined()
      expect(attempts).toEqual([{ url: 'https://gone.uhrp.test/cdn/x', error: expect.stringContaining('fetch failed') }])
    })
  })
})
