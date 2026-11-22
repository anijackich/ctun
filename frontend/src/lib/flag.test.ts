import { describe, expect, it } from 'vitest'
import { GENERIC_FLAG, parseServerName } from './flag'

describe('parseServerName', () => {
  it.each([
    ['Germany 🇩🇪', { flag: '🇩🇪', title: 'Germany' }],
    ['🇳🇱 Amsterdam-1', { flag: '🇳🇱', title: 'Amsterdam-1' }],
    ['Germany 92 🇩🇪 → [Whitelists] CDN', { flag: '🇩🇪', title: 'Germany 92 → [Whitelists] CDN' }],
    ['Auto 3', { flag: GENERIC_FLAG, title: 'Auto 3' }],
    ['🤖 AI', { flag: GENERIC_FLAG, title: '🤖 AI' }],
  ])('%s', (name, expected) => {
    expect(parseServerName(name)).toEqual(expected)
  })
})
