const FLAG = /\p{Regional_Indicator}{2}/u

export const GENERIC_FLAG = '🌐'

export function parseServerName(name: string) {
  const flag = name.match(FLAG)?.[0]
  const title = (flag ? name.replace(flag, ' ') : name).replace(/\s+/g, ' ').trim()
  return { flag: flag ?? GENERIC_FLAG, title: title || name }
}

export function countryFlag(code: string) {
  return /^[A-Za-z]{2}$/.test(code)
    ? String.fromCodePoint(...[...code.toUpperCase()].map((c) => 0x1f1e6 + c.charCodeAt(0) - 65))
    : ''
}
