import { session } from './session'
import { ApiError } from './types'
import type { Api, LectureInput } from './types'

const BASE = (import.meta.env.VITE_API_URL as string).replace(/\/$/, '')
const CLIENT_KEY = 'lectures.client.v1'
const TOKEN_ISSUING = new Set(['/auth/register', '/auth/login', '/auth/refresh', '/auth/logout'])

// Anonymous per-browser id that scopes bookmarks and watch progress on the server
function clientId(): string {
  let id = ''
  try {
    id = localStorage.getItem(CLIENT_KEY) ?? ''
  } catch {
    /* storage unavailable */
  }
  if (!id) {
    id = crypto.randomUUID()
    try {
      localStorage.setItem(CLIENT_KEY, id)
    } catch {
      /* id lives for this session only */
    }
  }
  return id
}

function authHeaders(): Record<string, string> {
  const h: Record<string, string> = { 'X-Client-Id': clientId() }
  if (session.access) h.Authorization = 'Bearer ' + session.access
  return h
}

async function fail(res: Response): Promise<never> {
  let body: { code?: string; message?: string; details?: Record<string, unknown> } = {}
  try {
    body = await res.json()
  } catch {
    /* non-JSON error body */
  }
  throw new ApiError(res.status, body.code ?? 'http_' + res.status, body.message ?? res.statusText, body.details)
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers = authHeaders()
  // An expired access token must not break refresh or login
  if (TOKEN_ISSUING.has(path)) delete headers.Authorization
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) await fail(res)
  return (res.status === 204 ? undefined : await res.json()) as T
}

function qs(params: Record<string, string | number | undefined>): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '') p.set(k, String(v))
  const s = p.toString()
  return s ? '?' + s : ''
}

// XHR instead of fetch: fetch has no upload progress
function upload<T>(path: string, form: FormData, onProgress?: (share: number) => void): Promise<T> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', BASE + path)
    for (const [k, v] of Object.entries(authHeaders())) xhr.setRequestHeader(k, v)
    xhr.upload.onprogress = (e) => e.lengthComputable && onProgress?.(e.loaded / e.total)
    xhr.onerror = () => reject(new ApiError(0, 'network', 'Сервер недоступен'))
    xhr.onload = () => {
      let body: Record<string, unknown> = {}
      try {
        body = JSON.parse(xhr.responseText)
      } catch {
        /* empty */
      }
      if (xhr.status >= 200 && xhr.status < 300) resolve(body as T)
      else reject(new ApiError(xhr.status, String(body.code ?? 'http_' + xhr.status), String(body.message ?? xhr.statusText)))
    }
    xhr.send(form)
  })
}

function withInput(input: LectureInput) {
  return { title: input.title || undefined, course_id: input.course_id || undefined, language: input.language }
}

export const httpApi: Api = {
  listLectures: (q = {}) => req('GET', '/lectures' + qs(q)),
  getLecture: (id) => req('GET', `/lectures/${id}`),
  uploadLecture(file, input, onProgress) {
    const form = new FormData()
    form.set('file', file)
    for (const [k, v] of Object.entries(withInput(input))) if (v) form.set(k, v)
    return upload('/lectures', form, onProgress)
  },
  importLecture: (url, input) => req('POST', '/lectures/import', { url, ...withInput(input) }),
  updateLecture: (id, patch) => req('PATCH', `/lectures/${id}`, patch),
  deleteLecture: (id) => req('DELETE', `/lectures/${id}`),
  retryLecture: (id) => req('POST', `/lectures/${id}/retry`),
  getTranscript: (id) => req('GET', `/lectures/${id}/transcript`),
  // <track> sends no headers, and subtitles of a private lecture need them
  subtitlesUrl: async (id) => {
    const res = await fetch(`${BASE}/lectures/${id}/subtitles.vtt`, { headers: authHeaders() })
    if (!res.ok) await fail(res)
    return URL.createObjectURL(await res.blob())
  },
  getSummary: (id) => req('GET', `/lectures/${id}/summary`),
  search: (q) => req('GET', '/search' + qs({ ...q })),
  ask: (body) => req('POST', '/ask', body),
  listCourses: () => req('GET', '/courses'),
  createCourse: (title) => req('POST', '/courses', { title }),
  updateCourse: (id, title) => req('PATCH', `/courses/${id}`, { title }),
  deleteCourse: (id) => req('DELETE', `/courses/${id}`),
  system: () => req('GET', '/system'),
  listBookmarks: (lectureId) => req('GET', '/bookmarks' + qs({ lecture_id: lectureId })),
  createBookmark: (body) => req('POST', '/bookmarks', body),
  updateBookmark: (id, patch) => req('PATCH', `/bookmarks/${id}`, patch),
  deleteBookmark: (id) => req('DELETE', `/bookmarks/${id}`),
  listProgress: () => req('GET', '/progress'),
  saveProgress: (lectureId, position) => req('PUT', `/lectures/${lectureId}/progress`, { position }),
  clearProgress: () => req('DELETE', '/progress'),
  register: (body) => req('POST', '/auth/register', body),
  login: (body) => req('POST', '/auth/login', body),
  refresh: (refreshToken) => req('POST', '/auth/refresh', { refresh_token: refreshToken }),
  logout: (refreshToken) => req('POST', '/auth/logout', { refresh_token: refreshToken }),
  me: () => req('GET', '/auth/me'),
  telegramCode: () => req('POST', '/auth/telegram/code'),
  unlinkTelegram: () => req('DELETE', '/auth/telegram'),
}
