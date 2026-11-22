import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Profile } from '../lib/types'

const api = vi.hoisted(() => ({
  import: vi.fn(),
  importFile: vi.fn(),
  groups: vi.fn(),
}))

vi.mock('../lib/backend', () => ({ backend: {}, privilege: {}, profiles: api }))

const { useApp } = await import('./app')

const profile = { id: 'p1', name: 'Config' } as Profile

describe('imports', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    api.groups.mockResolvedValue([])
    useApp.setState({ viewedProfileId: undefined, notice: undefined, importing: false })
  })

  it('opens the imported profile', async () => {
    api.import.mockResolvedValue(profile)
    const pending = useApp.getState().importText('vless://x')
    expect(useApp.getState().importing).toBe(true)
    await pending
    expect(useApp.getState()).toMatchObject({ viewedProfileId: 'p1', importing: false, notice: undefined })
  })

  it('reports a failed import without the error prefixes', async () => {
    api.import.mockRejectedValue(new Error('profile: no servers found'))
    await useApp.getState().importText('{"foo": 1}')
    expect(useApp.getState()).toMatchObject({ notice: 'no servers found', importing: false })
  })

  it('ignores a cancelled file dialog', async () => {
    api.importFile.mockResolvedValue(undefined)
    await useApp.getState().importFile()
    expect(useApp.getState()).toMatchObject({ viewedProfileId: undefined, notice: undefined })
  })
})
