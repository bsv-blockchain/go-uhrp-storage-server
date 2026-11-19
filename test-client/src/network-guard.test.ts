/**
 * The network guard (src/support/network-guard.ts) must stop requests that leave this machine.
 */
import http from 'node:http'
import https from 'node:https'

const guardTest = process.env.UHRP_LIVE === 'true' ? test.skip : test

describe('network guard', () => {
  guardTest('blocks fetch to an external host before any connection is made', async () => {
    await expect(fetch('https://example.com/')).rejects.toThrow(/Blocked a live network request to https:\/\/example\.com/)
  })

  guardTest('blocks node:http and node:https requests to an external host', () => {
    expect(() => https.get('https://example.com/')).toThrow(/Blocked a live network request/)
    expect(() => http.request({ host: 'example.com', port: 80, path: '/' })).toThrow(/Blocked a live network request/)
  })

  guardTest('allows loopback', async () => {
    // Nothing listens on port 9 (discard), so the request is refused rather than blocked.
    await expect(fetch('http://127.0.0.1:9/')).rejects.toThrow(/fetch failed/)
  })
})
