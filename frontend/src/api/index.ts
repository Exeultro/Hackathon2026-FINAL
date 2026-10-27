import { httpApi } from './httpApi'
import { mockApi } from './mockApi'
import { clearSession, session, setSession } from './session'
import { ApiError } from './types'
import type { Api } from './types'

// VITE_API_URL empty -> in-browser mock (VITE_API_URL= npm run dev)
const raw: Api = import.meta.env.VITE_API_URL ? httpApi : mockApi
export const isMock = !import.meta.env.VITE_API_URL

const NO_RETRY = new Set<string | symbol>(['register', 'login', 'refresh', 'logout'])
let renewing: Promise<void> | null = null

// One refresh for all requests that hit an expired access token at once
function renew(): Promise<void> {
  renewing ??= raw
    .refresh(session.refresh)
    .then(setSession, (e) => {
      clearSession()
      throw e
    })
    .finally(() => (renewing = null))
  return renewing
}

// On 401 refresh the token pair once and repeat the call
export const api = new Proxy(raw, {
  get(target, key) {
    const fn = target[key as keyof Api] as (...args: unknown[]) => Promise<unknown>
    if (typeof fn !== 'function' || NO_RETRY.has(key)) return fn
    return async (...args: unknown[]) => {
      try {
        return await fn.apply(target, args)
      } catch (e) {
        if (!(e instanceof ApiError && e.status === 401 && e.code === 'unauthorized' && session.refresh)) throw e
        await renew()
        return fn.apply(target, args)
      }
    }
  },
}) as Api

export { session } from './session'
export * from './types'
