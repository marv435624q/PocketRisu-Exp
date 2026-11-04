import { describe, expect, it } from 'vitest'
import path from 'node:path'
import pkg from './cloudflared-utils.cjs'

const { findTermuxCloudflared, createQuickTunnelUrlReader } = pkg as {
    findTermuxCloudflared: (env: Record<string, string>, exists: (file: string) => boolean) => string | null
    createQuickTunnelUrlReader: () => (chunk: string) => string | null
}

describe('Termux cloudflared resolution', () => {
    it('uses the package under PREFIX, not a project-local Linux binary', () => {
        const prefix = '/data/data/com.termux/files/usr'
        const expected = path.join(prefix, 'bin', 'cloudflared')
        const checked: string[] = []
        expect(findTermuxCloudflared({
            PREFIX: prefix,
            PATH: '/project/bin:/data/data/com.termux/files/usr/bin',
        }, file => { checked.push(file); return file === expected })).toBe(expected)
        expect(checked).toEqual([expected])
    })

    it('does not fall back to a bundled Linux binary or other PATH entry', () => {
        expect(findTermuxCloudflared({
            PREFIX: '/termux',
            PATH: '/project/bin:/usr/bin',
        }, file => file === '/project/bin/cloudflared')).toBeNull()
    })

    it('requires a Termux PREFIX to identify the package location', () => {
        expect(findTermuxCloudflared({
            PATH: '/project/bin',
        }, () => true)).toBeNull()
    })
})

describe('Quick Tunnel log URL reader', () => {
    it('handles a URL split across stderr chunks', () => {
        const read = createQuickTunnelUrlReader()
        expect(read('2026 INF Visit https://my-')).toBeNull()
        expect(read('quick-tunnel.trycloudflare.')).toBeNull()
        expect(read('com to connect')).toBe('https://my-quick-tunnel.trycloudflare.com')
    })

    it('does not accept unrelated URLs', () => {
        const read = createQuickTunnelUrlReader()
        expect(read('https://example.com')).toBeNull()
        expect(read('https://my-tunnel.cloudflare.com')).toBeNull()
    })

    it('still finds a URL after verbose logs', () => {
        const read = createQuickTunnelUrlReader()
        expect(read('x'.repeat(10000))).toBeNull()
        expect(read('https://sample.trycloudflare.com')).toBe('https://sample.trycloudflare.com')
    })
})
