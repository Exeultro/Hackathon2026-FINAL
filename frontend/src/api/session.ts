import { reactive } from 'vue'
import type { AuthResponse, User } from './types'

const KEY = 'lectures.auth.v1'

interface Session {
  user: User | null
  access: string
  refresh: string
}

function load(): Session {
  try {
    const s = JSON.parse(localStorage.getItem(KEY) ?? 'null')
    if (s?.user && s.access && s.refresh) return s
  } catch {
    /* storage unavailable */
  }
  return { user: null, access: '', refresh: '' }
}

function save() {
  try {
    localStorage.setItem(KEY, JSON.stringify(session))
  } catch {
    /* session lives until reload */
  }
}

export const session = reactive<Session>(load())

export function setSession(r: AuthResponse) {
  Object.assign(session, { user: r.user, access: r.access_token, refresh: r.refresh_token })
  save()
}

export function setUser(user: User) {
  session.user = user
  save()
}

export function clearSession() {
  Object.assign(session, { user: null, access: '', refresh: '' })
  save()
}
