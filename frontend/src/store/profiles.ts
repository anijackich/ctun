import type { StateCreator } from 'zustand'
import { profiles as profileApi } from '../lib/backend'
import { errorMessage } from '../lib/errors'
import { isServer } from '../lib/groups'
import type { NewProfile, Profile, ProxyGroup } from '../lib/types'
import type { AppState } from './app'

export interface ProfilesSlice {
  profiles: Profile[]
  activeProfileId?: string
  viewedProfileId?: string
  groups: Record<string, ProxyGroup[]>
  measuring: Record<string, string[]>
  importing: boolean

  viewProfile: (id: string) => void
  loadGroups: (id: string) => Promise<void>
  addProfile: (draft: NewProfile) => Promise<void>
  editProfile: (draft: NewProfile & { id: string }) => Promise<void>
  importText: (text: string) => Promise<void>
  importFile: () => Promise<void>
  refreshProfile: (id: string) => Promise<void>
  removeProfile: (id: string) => Promise<void>
  selectProxy: (profileId: string, group: string, name: string) => void
  testLatency: (profileId: string, group: string) => Promise<void>
  attachProfiles: () => Promise<() => void>
}

export const createProfilesSlice: StateCreator<AppState, [], [], ProfilesSlice> = (set, get) => {
  const showError = (err: unknown) => set({ notice: errorMessage(err) })

  const trackImport = (task: Promise<void>) => {
    set({ importing: true, notice: undefined })
    return task.catch(showError).finally(() => set({ importing: false }))
  }

  const openProfile = (profile: Profile) => {
    set({ viewedProfileId: profile.id, notice: undefined })
    void get().loadGroups(profile.id)
  }

  const setGroups = (id: string, groups: ProxyGroup[]) => set((s) => ({ groups: { ...s.groups, [id]: groups } }))

  const applyProfiles = (profiles: Profile[]) => {
    const active = profiles.find((p) => p.active)?.id
    set((s) => ({
      profiles,
      activeProfileId: active,
      viewedProfileId: profiles.some((p) => p.id === s.viewedProfileId) ? s.viewedProfileId : (active ?? profiles[0]?.id),
    }))
    for (const id of new Set([active, get().viewedProfileId])) if (id) void get().loadGroups(id)
  }

  return {
    profiles: [],
    groups: {},
    measuring: {},
    importing: false,

    viewProfile: (viewedProfileId) => {
      set({ viewedProfileId })
      void get().loadGroups(viewedProfileId)
    },

    loadGroups: (id) => profileApi.groups(id).then((groups) => setGroups(id, groups), showError),

    addProfile: async (draft) => openProfile(await profileApi.add(draft)),

    editProfile: async (draft) => {
      await profileApi.edit(draft)
      await get().loadGroups(draft.id)
    },

    importText: (text) => trackImport(profileApi.import(text).then(openProfile)),

    importFile: () => profileApi.importFile().then((p) => p && openProfile(p), showError),

    refreshProfile: async (id) => {
      await profileApi.refresh(id)
      await get().loadGroups(id)
    },

    removeProfile: (id) => profileApi.remove(id),

    selectProxy: (profileId, group, name) => {
      setGroups(
        profileId,
        (get().groups[profileId] ?? []).map((g) => (g.name === group ? { ...g, now: name } : g)),
      )
      void profileApi
        .select(profileId, group, name)
        .then(() => get().status === 'running' && get().refreshExit(), showError)
        .finally(() => void get().loadGroups(profileId))
    },

    testLatency: async (profileId, group) => {
      const servers =
        get()
          .groups[profileId]?.find((g) => g.name === group)
          ?.nodes.filter(isServer)
          .map((n) => n.name) ?? []
      set((s) => ({ measuring: { ...s.measuring, [profileId]: servers } }))
      try {
        setGroups(profileId, await profileApi.testLatency(profileId, group))
      } catch (err) {
        showError(err)
      } finally {
        set((s) => ({ measuring: { ...s.measuring, [profileId]: [] } }))
      }
    },

    attachProfiles: async () => {
      const offs = [
        profileApi.onChange(applyProfiles),
        profileApi.onLatency(({ profileId, name, delay }) =>
          set((s) => ({
            measuring: { ...s.measuring, [profileId]: (s.measuring[profileId] ?? []).filter((n) => n !== name) },
            groups: {
              ...s.groups,
              [profileId]: (s.groups[profileId] ?? []).map((g) => ({
                ...g,
                nodes: g.nodes.map((n) => (n.name === name ? { ...n, delay } : n)),
              })),
            },
          })),
        ),
      ]
      applyProfiles(await profileApi.list())
      return () => offs.forEach((off) => off())
    },
  }
}
