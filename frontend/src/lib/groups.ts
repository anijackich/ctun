import type { ProxyGroup, ProxyNode } from './types'

export const DIRECT = 'DIRECT'
const REJECT = ['REJECT', 'REJECT-DROP']

export const isReject = (target: string) => REJECT.includes(target)

export const isGroupNode = (node: ProxyNode) => node.protocol === ''

export const isServer = (node: ProxyNode) => node.protocol !== '' && node.protocol !== node.name

export const primaryGroup = (groups: ProxyGroup[]) =>
  groups.find((g) => g.type === 'select') ?? groups[0]

export function resolveNode(groups: ProxyGroup[], groupName: string): ProxyNode | undefined {
  const group = groups.find((g) => g.name === groupName)
  if (!group) return undefined
  const nested = groups.find((g) => g.name === group.now)
  return nested ? resolveNode(groups, nested.name) : group.nodes.find((n) => n.name === group.now)
}
