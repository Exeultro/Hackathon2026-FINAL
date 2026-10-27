import { reactive } from 'vue'
import { api, ApiError, session } from '@/api'
import { clearSession, setSession, setUser } from '@/api/session'
import { toast } from '@/ui'

export const authDialog = reactive({
  open: false,
  reason: '',
  then: null as null | (() => void),
})

// true if signed in; otherwise opens the sign-in dialog and runs `then` after success
export function requireLogin(reason: string, then?: () => void): boolean {
  if (session.user) return true
  Object.assign(authDialog, { open: true, reason, then: then ?? null })
  return false
}

export function closeAuth(done: boolean) {
  const then = done ? authDialog.then : null
  Object.assign(authDialog, { open: false, reason: '', then: null })
  then?.()
}

export async function login(email: string, password: string) {
  setSession(await api.login({ email, password }))
}

export async function register(name: string, email: string, password: string) {
  setSession(await api.register({ name, email, password }))
}

export async function logout() {
  const refresh = session.refresh
  clearSession()
  await api.logout(refresh).catch(() => {
    /* the token just expires */
  })
}

export async function checkSession() {
  if (!session.user) return
  try {
    setUser(await api.me())
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) clearSession()
  }
}

// For protected actions: a 401 means the session is gone, ask to sign in again
export function actionFailed(e: unknown, reason: string) {
  if (e instanceof ApiError && e.status === 401) requireLogin(reason)
  else toast(e instanceof ApiError ? e.message : 'Не получилось, проверьте связь с сервером')
}
