import { primaryGroup, resolveNode } from '../lib/groups'
import { useApp } from './app'

export const useActiveProfile = () => useApp((s) => s.profiles.find((p) => p.id === s.activeProfileId))

export const useViewedProfile = () => useApp((s) => s.profiles.find((p) => p.id === s.viewedProfileId))

export const useActiveNode = () =>
  useApp((s) => {
    const groups = s.activeProfileId ? s.groups[s.activeProfileId] : undefined
    const primary = groups && primaryGroup(groups)
    return groups && primary ? resolveNode(groups, primary.name) : undefined
  })

export const useTunnelMode = () => useApp((s) => s.settings?.tunnelMode ?? 'tun')
export const useRoutingMode = () => useApp((s) => s.settings?.routingMode ?? 'rule')
