import type { Config } from 'jest'

// TypeScript 7 is the native compiler and no longer ships the JS compiler API that ts-jest
// needs, so @swc/jest strips the types and `npm run typecheck` (tsc --noEmit) checks them.
const config: Config = {
  testEnvironment: 'node',
  testTimeout: 30000,
  roots: ['<rootDir>/src'],
  // Fails any request that leaves this machine unless UHRP_LIVE=true (see the file).
  setupFiles: ['<rootDir>/src/support/network-guard.ts'],
  transform: {
    '^.+\\.ts$': [
      '@swc/jest',
      {
        jsc: { parser: { syntax: 'typescript' }, target: 'es2022' },
        module: { type: 'commonjs' },
      },
    ],
  },
}

export default config
