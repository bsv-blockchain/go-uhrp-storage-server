/**
 * Live UHRP lookup diagnostic against the public mainnet overlay.
 *
 * Walks every network hop that StorageDownloader.download() makes so we can see exactly where the
 * lookup breaks for a given UHRP URL. It reaches real SLAP trackers, overlay hosts and CDNs, so it
 * only runs when asked to; uhrp-lookup.test.ts covers the same code against a mock overlay.
 *
 * Run:
 *   npm run test:diagnose
 *   UHRP_URL=XUSzUk... npm run test:diagnose
 */

import { DEFAULT_SLAP_TRACKERS, LookupResolver, StorageUtils, Utils } from '@bsv/sdk'

import {
  type Advertisement,
  discoverUhrpHosts,
  downloadVerified,
  findAdvertisements,
  resolveAdvertisements,
  usableDownloadURLs,
} from './diagnostics/uhrp-lookup'

const UHRP_URL = process.env.UHRP_URL ?? 'XUSzUkfq8SSqLQEn2LL98gcxBF6MwTCzuxPrnuYwiRQpi6fp7W6U'
const TIMEOUT_MS = 8000

const liveDescribe = process.env.UHRP_LIVE === 'true' ? describe : describe.skip

function logAds(ads: Advertisement[]): void {
  const now = Math.floor(Date.now() / 1000)
  for (const ad of ads) {
    console.log(`  ${ad.source}: ${JSON.stringify({ ...ad, source: undefined, expiresIn: ad.expiryTime - now })}`)
  }
}

liveDescribe('UHRP lookup diagnostics (live mainnet overlay)', () => {
  let lsUhrpHosts: string[] = []

  test('UHRP URL is valid and decodes to a hash', () => {
    console.log('\nUHRP URL:', UHRP_URL)
    expect(StorageUtils.isValidURL(UHRP_URL)).toBe(true)
    const hash = StorageUtils.getHashFromURL(UHRP_URL)
    console.log('SHA-256 (hex):', Utils.toHex(hash))
    expect(hash.length).toBe(32)
  })

  test('SLAP trackers name ls_uhrp hosts', async () => {
    console.log('\nDefault SLAP trackers:', DEFAULT_SLAP_TRACKERS)
    const { hosts, failures } = await discoverUhrpHosts(fetch, DEFAULT_SLAP_TRACKERS, TIMEOUT_MS)
    lsUhrpHosts = hosts

    console.log('ls_uhrp hosts found:', hosts)
    if (failures.length > 0) console.log('Failures:', failures)
    // Not asserting: an empty list IS the finding (uploads succeed, but downloads always fail).
    if (hosts.length === 0) console.warn('⚠️  NO ls_uhrp HOSTS found via any SLAP tracker.')
  }, 30_000)

  test('ls_uhrp hosts return advertisements for the UHRP URL', async () => {
    if (lsUhrpHosts.length === 0) {
      console.log('Skipping: no ls_uhrp hosts discovered in the previous step')
      return
    }

    const now = Math.floor(Date.now() / 1000)
    const { ads, failures } = await findAdvertisements(fetch, lsUhrpHosts, UHRP_URL, now, TIMEOUT_MS)
    logAds(ads)
    if (failures.length > 0) console.log('Failures:', failures)
    console.log('Usable download URLs:', usableDownloadURLs(ads))
  }, 30_000)

  test('LookupResolver (SDK) resolves the file and a download verifies its hash', async () => {
    const now = Math.floor(Date.now() / 1000)
    let ads: Advertisement[]
    try {
      const result = await resolveAdvertisements(new LookupResolver({ networkPreset: 'mainnet' }), UHRP_URL, now)
      ads = result.ads
      if (result.failures.length > 0) console.log('Failures:', result.failures)
    } catch (e) {
      console.log('resolver.query threw:', e instanceof Error ? e.message : e)
      return
    }

    logAds(ads)
    const urls = usableDownloadURLs(ads)
    if (urls.length === 0) {
      console.warn('⚠️  No unexpired advertisement: StorageDownloader throws "No one currently hosts this file!"')
      return
    }

    const { verifiedUrl, attempts } = await downloadVerified(fetch, urls, UHRP_URL, TIMEOUT_MS)
    console.log('Download attempts:', attempts)
    expect(verifiedUrl).toBeDefined()
  }, 60_000)
})
