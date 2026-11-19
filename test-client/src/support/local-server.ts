/**
 * Runs the Go UHRP server for the integration tests with nothing outside this machine involved:
 * no wallet, and a local stub answering its BSV/USD exchange-rate requests.
 */
import { type ChildProcess, execFile, spawn } from 'node:child_process'
import { once } from 'node:events'
import { mkdtemp, rm } from 'node:fs/promises'
import { createServer, type Server } from 'node:http'
import type { AddressInfo } from 'node:net'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'

/** BSV/USD rate the stub exchange-rate endpoint reports. */
export const STUB_BSV_USD_RATE = 50

/** PRICE_PER_GB_MO the local server runs with. */
export const PRICE_PER_GB_MONTH = 0.03

const REPO_ROOT = path.resolve(__dirname, '..', '..', '..')
const READY_TIMEOUT_MS = 15_000

export interface LocalServer {
  /** Base URL of the server, e.g. http://127.0.0.1:54321. */
  url: string
  /** How many times the server has asked the stub for the exchange rate. */
  rateRequests: () => number
  stop: () => Promise<void>
}

async function listen(server: Server): Promise<number> {
  server.listen(0, '127.0.0.1')
  await once(server, 'listening')
  return (server.address() as AddressInfo).port
}

async function freePort(): Promise<number> {
  const probe = createServer()
  const port = await listen(probe)
  probe.close()
  await once(probe, 'close')
  return port
}

async function waitUntilReady(url: string, child: ChildProcess, output: () => string): Promise<void> {
  const deadline = Date.now() + READY_TIMEOUT_MS
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`UHRP server exited with code ${child.exitCode}:\n${output()}`)
    try {
      await fetch(`${url}/health-probe`)
      return
    } catch {
      await new Promise(resolve => setTimeout(resolve, 100))
    }
  }
  throw new Error(`UHRP server did not start within ${READY_TIMEOUT_MS} ms:\n${output()}`)
}

/** Builds the Go server and starts it on a free loopback port. */
export async function startLocalServer(): Promise<LocalServer> {
  const workDir = await mkdtemp(path.join(tmpdir(), 'uhrp-server-test-'))
  const binary = path.join(workDir, process.platform === 'win32' ? 'uhrp-server.exe' : 'uhrp-server')
  await promisify(execFile)('go', ['build', '-o', binary, './cmd/server'], { cwd: REPO_ROOT })

  let rateRequests = 0
  const rateServer = createServer((_req, res) => {
    rateRequests++
    res.writeHead(200, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ currency: 'USD', rate: STUB_BSV_USD_RATE, time: Math.floor(Date.now() / 1000) }))
  })
  const rateURL = `http://127.0.0.1:${await listen(rateServer)}/exchangerate`

  const port = await freePort()
  const url = `http://127.0.0.1:${port}`

  // A minimal environment, run from an empty directory: no .env file or inherited variable can
  // give the server a wallet or point it at a real network service.
  const child = spawn(binary, [], {
    cwd: workDir,
    env: {
      PATH: process.env.PATH,
      HTTP_PORT: String(port),
      HOSTING_DOMAIN: `localhost:${port}`,
      EXCHANGE_RATE_URL: rateURL,
      PRICE_PER_GB_MO: String(PRICE_PER_GB_MONTH),
      SERVER_PRIVATE_KEY: '',
      WALLET_STORAGE_URL: '',
      SLAP_TRACKERS: `http://127.0.0.1:${port}`,
      PUBLIC_DIR: path.join(workDir, 'public'),
      LOG_FORMAT: 'text',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  let output = ''
  child.stdout?.on('data', (chunk: Buffer) => (output += chunk.toString()))
  child.stderr?.on('data', (chunk: Buffer) => (output += chunk.toString()))

  const stop = async (): Promise<void> => {
    if (child.exitCode === null && child.signalCode === null) {
      const exited = once(child, 'exit')
      child.kill('SIGTERM')
      await exited
    }
    rateServer.close()
    await rm(workDir, { recursive: true, force: true })
  }

  try {
    await waitUntilReady(url, child, () => output)
  } catch (e) {
    await stop()
    throw e
  }

  return { url, rateRequests: () => rateRequests, stop }
}
