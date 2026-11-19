/**
 * Jest setup file that fails any HTTP request leaving this machine, so a test that forgets to mock
 * an external service fails loudly instead of quietly reaching the internet.
 *
 * Allowed: loopback hosts, and the origin of UHRP_HOST when you point the integration tests at a
 * server of your own. UHRP_LIVE=true lifts the guard for the live overlay diagnostic.
 */
import http from 'node:http'
import https from 'node:https'

const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '::1', '[::1]'])

function allowedOrigins(): Set<string> {
  const origins = new Set<string>()
  const host = process.env.UHRP_HOST
  if (host !== undefined && host !== '') origins.add(new URL(host).origin)
  return origins
}

/** Throws unless url points at loopback or an explicitly allowed origin. */
export function assertLocalRequest(url: URL): void {
  if (LOOPBACK_HOSTS.has(url.hostname) || allowedOrigins().has(url.origin)) return
  throw new Error(
    `Blocked a live network request to ${url.origin}: tests must mock external hosts ` +
      '(set UHRP_LIVE=true to run the live overlay diagnostic).'
  )
}

/** The URL an http(s).request or get call is aimed at, whichever argument form it uses. */
function requestURL(protocol: string, target: unknown): URL {
  if (typeof target === 'string' || target instanceof URL) return new URL(target)
  const options = (target ?? {}) as http.RequestOptions
  let hostname = options.hostname ?? options.host ?? 'localhost'
  // `host` may carry a port ("example.com:443"); a bare IPv6 address has more than one colon.
  if (options.hostname == null && hostname.split(':').length === 2) hostname = hostname.split(':')[0]
  if (hostname.includes(':') && !hostname.startsWith('[')) hostname = `[${hostname}]`
  const port = options.port != null ? `:${options.port}` : ''
  return new URL(`${protocol}//${hostname}${port}`)
}

function guardModule(mod: typeof http | typeof https, protocol: string): void {
  for (const method of ['request', 'get'] as const) {
    const original = mod[method] as (...args: unknown[]) => http.ClientRequest
    Object.defineProperty(mod, method, {
      configurable: true,
      writable: true,
      value: (...args: unknown[]) => {
        assertLocalRequest(requestURL(protocol, args[0]))
        return original.apply(mod, args)
      },
    })
  }
}

if (process.env.UHRP_LIVE !== 'true') {
  const realFetch = globalThis.fetch
  globalThis.fetch = async (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
    assertLocalRequest(new URL(input instanceof Request ? input.url : input))
    return await realFetch(input, init)
  }

  guardModule(http, 'http:')
  guardModule(https, 'https:')
}
