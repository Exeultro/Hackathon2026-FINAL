import { demoCourses, demoLectures } from './mockData'
import type { DemoLecture } from './mockData'
import { session } from './session'
import { ApiError } from './types'
import type { Api, AuthResponse, Bookmark, Course, Lecture, LectureStatus, Playback, SearchHit, SearchQuery, Segment, Summary, User, WatchProgress } from './types'

const KEY = 'lectures.mock.v1'

interface StoredLecture {
  id: string
  title: string
  course_id: string | null
  source: Lecture['source']
  source_url: string | null
  created_at: string
  template: string
  provider_id?: string
  fail?: boolean
}

interface State {
  lectures: StoredLecture[]
  courses: { id: string; title: string; created_at: string }[]
  bookmarks: (Omit<Bookmark, 'lecture_title'> & { owner?: string })[]
  progress: (WatchProgress & { owner?: string })[]
  users: (User & { password: string })[]
  refresh: { token: string; user_id: string; expires: number }[]
}

const LONG_AGO = '2026-10-20T09:00:00Z'
const blobUrls = new Map<string, string>()

function seed(): State {
  return {
    lectures: demoLectures.map((d) => ({
      id: d.id,
      title: d.title,
      course_id: d.course_id,
      source: 'upload',
      source_url: null,
      created_at: LONG_AGO,
      template: d.id,
    })),
    courses: demoCourses.map((c) => ({ ...c, created_at: LONG_AGO })),
    bookmarks: [],
    progress: [],
    users: [],
    refresh: [],
  }
}

function load(): State {
  try {
    const s = JSON.parse(localStorage.getItem(KEY) ?? 'null')
    if (s?.lectures) return { bookmarks: [], progress: [], users: [], refresh: [], ...s }
  } catch {
    /* fall through to seed */
  }
  return seed()
}

const state = load()

function save() {
  localStorage.setItem(KEY, JSON.stringify(state))
}

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms))
const rid = (prefix: string) => prefix + '_' + Math.random().toString(36).slice(2, 8)

// Fake pipeline: stage durations in seconds since created_at
const PIPELINE: [LectureStatus, number][] = [
  ['queued', 1],
  ['downloading', 3],
  ['extracting_audio', 1.5],
  ['transcribing', 6],
  ['indexing', 2],
]
const SUMMARY_SECONDS = 6

const ACCESS_TTL = 15 * 60
const REFRESH_TTL = 30 * 24 * 3600

const unauthorized = () => new ApiError(401, 'unauthorized', 'Нужно войти')

// Mock access token: "mock.<user id>.<expiry ms>"; the real one is a JWT
function currentUser(): string | null {
  if (!session.access) return null
  const [, id, exp] = session.access.split('.')
  if (!state.users.some((u) => u.id === id) || Date.now() > Number(exp)) throw unauthorized()
  return id
}

function requireUser(): string {
  const id = currentUser()
  if (!id) throw unauthorized()
  return id
}

const owner = () => currentUser() ?? 'anon'
const mine = (x: { owner?: string }) => (x.owner ?? 'anon') === owner()

function issue(u: State['users'][number]): AuthResponse {
  const refresh = rid('rt') + rid('')
  state.refresh.push({ token: refresh, user_id: u.id, expires: Date.now() + REFRESH_TTL * 1000 })
  save()
  const { password: _, ...user } = u
  return {
    access_token: `mock.${u.id}.${Date.now() + ACCESS_TTL * 1000}`,
    refresh_token: refresh,
    token_type: 'Bearer',
    expires_in: ACCESS_TTL,
    user: { ...user, telegram: user.telegram ?? null },
  }
}

function template(s: StoredLecture): DemoLecture {
  return demoLectures.find((d) => d.id === s.template) ?? demoLectures[0]
}

function stages(s: StoredLecture) {
  return PIPELINE.filter(([st]) => st !== 'downloading' || s.source !== 'upload')
}

function view(s: StoredLecture): Lecture {
  const t = template(s)
  const elapsed = (Date.now() - Date.parse(s.created_at)) / 1000
  const st = stages(s)
  const total = st.reduce((a, [, d]) => a + d, 0)
  let status: LectureStatus = 'ready'
  let acc = 0
  for (const [name, d] of st) {
    if (elapsed < acc + d) {
      status = name
      break
    }
    acc += d
  }
  let error: string | null = null
  if (s.fail && (status === 'transcribing' || status === 'indexing' || status === 'ready')) {
    status = 'failed'
    error = 'Не удалось скачать видео: ролик недоступен или удалён'
  }
  const ready = status === 'ready'
  const after = (stage: LectureStatus) => ready || st.findIndex(([n]) => n === status) > st.findIndex(([n]) => n === stage)
  const sumElapsed = elapsed - total
  return {
    id: s.id,
    title: s.title,
    course_id: s.course_id,
    mine: true,
    status,
    progress: status === 'failed' ? 0 : Math.min(1, elapsed / total),
    error,
    summary_status: !ready ? 'pending' : sumElapsed < SUMMARY_SECONDS ? 'processing' : 'ready',
    source: s.source,
    source_url: s.source_url,
    playback: status === 'queued' || status === 'downloading' || status === 'failed' ? null : playback(s, t),
    duration: after('extracting_audio') ? t.duration : null,
    language: after('transcribing') ? 'ru' : null,
    thumbnail_url: s.id === t.id ? t.video.replace(/\.mp4$/, '.jpg') : null,
    created_at: s.created_at,
  }
}

function playback(s: StoredLecture, t: DemoLecture): Playback {
  if (s.source === 'youtube' || s.source === 'vimeo') return { type: s.source, url: s.source_url!, provider_id: s.provider_id }
  if (s.source === 'url') return { type: 'file', url: s.source_url! }
  return { type: 'file', url: blobUrls.get(s.id) ?? t.video }
}

function find(id: string): StoredLecture {
  const s = state.lectures.find((l) => l.id === id)
  if (!s) throw new ApiError(404, 'not_found', 'Лекция не найдена')
  return s
}

function requireReady(id: string): DemoLecture {
  const s = find(id)
  if (view(s).status !== 'ready') throw new ApiError(409, 'not_ready', 'Лекция ещё обрабатывается')
  return template(s)
}

function segments(t: DemoLecture): Segment[] {
  return t.lines.map(([start, text], i) => ({ id: i, start, end: t.lines[i + 1]?.[0] ?? t.duration, text }))
}

function vtt(segs: Segment[]): string {
  const ts = (x: number) => new Date(x * 1000).toISOString().slice(11, 23)
  return 'WEBVTT\n\n' + segs.map((s) => `${s.id + 1}\n${ts(s.start)} --> ${ts(s.end)}\n${s.text}\n`).join('\n')
}

// --- fake "semantic" search: prefix stemming + a tiny synonym table ---

const STOP = new Set(
  'и в во на не что как это так про для по из от до или а но же ли бы то там тут где когда почему зачем какой какая какие такое такой объяснение объясни расскажи рассказ лекция лекции момент где-то нужно надо есть был была'.split(
    ' ',
  ),
)
const SYNONYMS: Record<string, string[]> = {
  пифаг: ['гипотенуз', 'катет'],
  длинн: ['гипотенуз'],
  скорос: ['производн'],
  наклон: ['касательн', 'угловой'],
  крутиз: ['касательн', 'круче'],
  доказ: ['площад'],
  формул: ['квадрат', 'формулиров'],
  расстоя: ['координат'],
  правил: ['дифференцир'],
  пример: ['например'],
  треуго: ['треугольник'],
}

const WORD = /[\p{L}\d]+/gu

function words(text: string): { w: string; from: number; to: number }[] {
  return [...text.toLowerCase().matchAll(WORD)].map((m) => ({ w: m[0], from: m.index!, to: m.index! + m[0].length }))
}

function same(a: string, b: string): boolean {
  let i = 0
  while (i < a.length && i < b.length && a[i] === b[i]) i++
  return i >= Math.min(5, a.length, b.length) && Math.min(a.length, b.length) >= 3
}

interface Concept {
  word: string
  syn: string[]
}

function concepts(q: string): Concept[] {
  return words(q)
    .map((x) => x.w)
    .filter((w) => w.length >= 2 && !STOP.has(w))
    .map((w) => ({ word: w, syn: Object.entries(SYNONYMS).find(([k]) => w.startsWith(k))?.[1] ?? [] }))
}

function score(cs: Concept[], text: string): { score: number; highlights: number[][] } {
  const ws = words(text)
  let sum = 0
  const highlights: number[][] = []
  for (const c of cs) {
    const direct = ws.filter((x) => same(x.w, c.word))
    if (direct.length) {
      sum += 1
      for (const x of direct) highlights.push([x.from, x.to])
    } else if (ws.some((x) => c.syn.some((s) => x.w.startsWith(s)))) sum += 0.8
  }
  highlights.sort((a, b) => a[0] - b[0])
  return { score: cs.length ? sum / cs.length : 0, highlights }
}

function searchSync(q: SearchQuery): SearchHit[] {
  const cs = concepts(q.q)
  const pool = state.lectures
    .filter((s) => (!q.lecture_id || s.id === q.lecture_id) && (!q.course_id || s.course_id === q.course_id))
    .filter((s) => view(s).status === 'ready')
  const hits: SearchHit[] = []
  for (const s of pool) {
    const segs = segments(template(s))
    const found: SearchHit[] = []
    for (let i = 0; i < segs.length; i++) {
      const chunk = segs.slice(i, i + 2)
      const text = chunk.map((x) => x.text).join(' ')
      const r = score(cs, text)
      if (r.score <= 0 || score(cs, chunk[0].text).score <= 0) continue
      const start = chunk[0].start
      found.push({
        lecture: { id: s.id, title: s.title, thumbnail_url: view(s).thumbnail_url },
        start,
        end: chunk[chunk.length - 1].end,
        text,
        score: Math.round((0.35 + 0.6 * r.score - i * 0.0005) * 1000) / 1000,
        highlights: r.highlights,
        deep_link: `/lectures/${s.id}?t=${Math.floor(start)}`,
      })
    }
    found.sort((a, b) => b.score - a.score)
    for (const h of found) if (!hits.some((x) => x.lecture.id === h.lecture.id && x.start < h.end && h.start < x.end)) hits.push(h)
  }
  hits.sort((a, b) => b.score - a.score)
  return hits.slice(0, q.limit ?? 10)
}

function bestSentence(text: string, cs: Concept[]): string {
  const sentences = text.match(/[^.!?]+[.!?]?/g)?.map((x) => x.trim()) ?? [text]
  return sentences.reduce((best, x) => (score(cs, x).score > score(cs, best).score ? x : best))
}

export const mockApi: Api = {
  async listLectures(q = {}) {
    await wait(150)
    return state.lectures
      .filter((s) => !q.course_id || s.course_id === q.course_id)
      .map(view)
      .sort((a, b) => b.created_at.localeCompare(a.created_at))
  },

  async getLecture(id) {
    await wait(80)
    return view(find(id))
  },

  async uploadLecture(file, input, onProgress) {
    requireUser()
    for (let i = 1; i <= 10; i++) {
      await wait(80)
      onProgress?.(i / 10)
    }
    const s: StoredLecture = {
      id: rid('lec'),
      title: input.title || file.name.replace(/\.[^.]+$/, ''),
      course_id: input.course_id || null,
      source: 'upload',
      source_url: null,
      created_at: new Date().toISOString(),
      template: demoLectures[0].id,
    }
    blobUrls.set(s.id, URL.createObjectURL(file))
    state.lectures.push(s)
    save()
    return view(s)
  },

  async importLecture(url, input) {
    await wait(300)
    let u: URL
    try {
      u = new URL(url)
    } catch {
      throw new ApiError(422, 'bad_url', 'Это не похоже на ссылку', { url: 'Неверная ссылка' })
    }
    const yt = u.hostname.includes('youtu') ? (u.searchParams.get('v') ?? u.pathname.split('/').pop()) : null
    const vimeo = u.hostname.includes('vimeo') ? u.pathname.split('/').filter(Boolean).pop() : null
    const s: StoredLecture = {
      id: rid('lec'),
      title: input.title || (yt ? 'Лекция с YouTube' : vimeo ? 'Лекция с Vimeo' : decodeURIComponent(u.pathname.split('/').pop() || u.hostname)),
      course_id: input.course_id || null,
      source: yt ? 'youtube' : vimeo ? 'vimeo' : 'url',
      source_url: url,
      created_at: new Date().toISOString(),
      template: demoLectures[0].id,
      provider_id: yt ?? vimeo ?? undefined,
      fail: url.includes('fail'),
    }
    state.lectures.push(s)
    save()
    return view(s)
  },

  async updateLecture(id, patch) {
    await wait(100)
    requireUser()
    const s = find(id)
    if (patch.title !== undefined) s.title = patch.title
    if (patch.course_id !== undefined) s.course_id = patch.course_id
    save()
    return view(s)
  },

  async deleteLecture(id) {
    await wait(100)
    requireUser()
    find(id)
    state.lectures = state.lectures.filter((l) => l.id !== id)
    state.bookmarks = state.bookmarks.filter((b) => b.lecture_id !== id)
    state.progress = state.progress.filter((p) => p.lecture_id !== id)
    save()
  },

  async retryLecture(id) {
    await wait(100)
    const s = find(id)
    if (view(s).status !== 'failed') throw new ApiError(409, 'not_failed', 'Лекция не в ошибке')
    s.fail = false
    s.created_at = new Date().toISOString()
    save()
    return view(s)
  },

  async getTranscript(id) {
    await wait(120)
    return { lecture_id: id, language: 'ru', segments: segments(requireReady(id)) }
  },

  async subtitlesUrl(id) {
    const t = requireReady(id)
    return URL.createObjectURL(new Blob([vtt(segments(t))], { type: 'text/vtt' }))
  },

  async getSummary(id): Promise<Summary> {
    await wait(150)
    const t = requireReady(id)
    if (view(find(id)).summary_status !== 'ready') throw new ApiError(409, 'not_ready', 'Конспект ещё готовится')
    const segs = segments(t)
    return {
      lecture_id: id,
      summary: t.summary,
      outline: t.outline,
      key_terms: t.terms.map(([term, definition, stems]) => ({
        term,
        definition,
        occurrences: segs.filter((s) => words(s.text).some((x) => stems.some((p) => x.w.startsWith(p)))).map((s) => s.start),
      })),
    }
  },

  async search(q) {
    const t0 = performance.now()
    await wait(250)
    if (q.q.trim().length < 2) throw new ApiError(422, 'validation', 'Запрос слишком короткий', { q: 'Минимум 2 символа' })
    if (q.lecture_id) find(q.lecture_id)
    const hits = searchSync(q)
    return { query: q.q, took_ms: Math.round(performance.now() - t0), hits }
  },

  async ask(req) {
    await wait(1200)
    const sources = searchSync({ q: req.question, lecture_id: req.lecture_id, course_id: req.course_id, limit: 3 })
    if (!sources.length)
      return { answer: 'В лекции не нашлось ответа на этот вопрос. Попробуйте спросить иначе.', sources: [] }
    return {
      answer: 'Вот что говорится в лекции:\n\n' + sources.map((h, i) => `- ${bestSentence(h.text, concepts(req.question))} [${i + 1}]`).join('\n'),
      sources,
    }
  },

  async listCourses() {
    await wait(100)
    return state.courses.map(courseView)
  },

  async createCourse(title) {
    await wait(100)
    requireUser()
    if (!title.trim()) throw new ApiError(422, 'validation', 'Пустое название', { title: 'Обязательное поле' })
    const c = { id: rid('crs'), title: title.trim(), created_at: new Date().toISOString() }
    state.courses.push(c)
    save()
    return courseView(c)
  },

  async updateCourse(id, title) {
    await wait(100)
    requireUser()
    const c = state.courses.find((x) => x.id === id)
    if (!c) throw new ApiError(404, 'not_found', 'Курс не найден')
    c.title = title.trim()
    save()
    return courseView(c)
  },

  async deleteCourse(id) {
    await wait(100)
    requireUser()
    state.courses = state.courses.filter((x) => x.id !== id)
    for (const l of state.lectures) if (l.course_id === id) l.course_id = null
    save()
  },

  async listBookmarks(lectureId) {
    await wait(80)
    return state.bookmarks
      .filter((b) => mine(b) && (!lectureId || b.lecture_id === lectureId))
      .map(bookmarkView)
      .sort((a, b) => a.lecture_id.localeCompare(b.lecture_id) || a.t - b.t)
  },

  async createBookmark(body) {
    await wait(80)
    find(body.lecture_id)
    const now = new Date().toISOString()
    const b = { id: rid('bm'), owner: owner(), lecture_id: body.lecture_id, t: body.t, note: body.note ?? '', created_at: now, updated_at: now }
    state.bookmarks.push(b)
    save()
    return bookmarkView(b)
  },

  async updateBookmark(id, patch) {
    await wait(60)
    const b = state.bookmarks.find((x) => x.id === id && mine(x))
    if (!b) throw new ApiError(404, 'not_found', 'Закладка не найдена')
    if (patch.note !== undefined) b.note = patch.note
    if (patch.t !== undefined) b.t = patch.t
    b.updated_at = new Date().toISOString()
    save()
    return bookmarkView(b)
  },

  async deleteBookmark(id) {
    await wait(60)
    if (!state.bookmarks.some((x) => x.id === id && mine(x))) throw new ApiError(404, 'not_found', 'Закладка не найдена')
    state.bookmarks = state.bookmarks.filter((x) => x.id !== id)
    save()
  },

  async listProgress() {
    await wait(60)
    return state.progress.filter(mine).map(({ owner: _, ...p }) => p).sort((a, b) => b.updated_at.localeCompare(a.updated_at))
  },

  async saveProgress(lectureId, position) {
    find(lectureId)
    const p = { lecture_id: lectureId, position, updated_at: new Date().toISOString() }
    const o = owner()
    state.progress = [...state.progress.filter((x) => !(mine(x) && x.lecture_id === lectureId)), { ...p, owner: o }]
    save()
    return p
  },

  async clearProgress() {
    await wait(60)
    state.progress = state.progress.filter((x) => !mine(x))
    save()
  },

  async register(body) {
    await wait(300)
    const email = body.email.trim().toLowerCase()
    const name = body.name.trim()
    const details: Record<string, string> = {}
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) details.email = 'Неверный email'
    if (body.password.length < 8 || body.password.length > 64) details.password = 'От 8 до 64 символов'
    if (!name || name.length > 64) details.name = 'От 1 до 64 символов'
    if (Object.keys(details).length) throw new ApiError(422, 'validation', 'Проверьте поля', details)
    if (state.users.some((u) => u.email === email)) throw new ApiError(409, 'email_taken', 'Этот email уже зарегистрирован')
    const u = { id: rid('usr'), email, name, password: body.password, created_at: new Date().toISOString(), telegram: null }
    state.users.push(u)
    return issue(u)
  },

  async login(body) {
    await wait(300)
    const u = state.users.find((x) => x.email === body.email.trim().toLowerCase() && x.password === body.password)
    if (!u) throw new ApiError(401, 'invalid_credentials', 'Неверный email или пароль')
    return issue(u)
  },

  async refresh(refreshToken) {
    await wait(100)
    const rt = state.refresh.find((x) => x.token === refreshToken)
    state.refresh = state.refresh.filter((x) => x.token !== refreshToken && x.expires > Date.now())
    const u = rt && rt.expires > Date.now() && state.users.find((x) => x.id === rt.user_id)
    save()
    if (!u) throw unauthorized()
    return issue(u)
  },

  async logout(refreshToken) {
    state.refresh = state.refresh.filter((x) => x.token !== refreshToken)
    save()
  },

  async me() {
    await wait(60)
    const id = requireUser()
    const { password: _, ...user } = state.users.find((u) => u.id === id)!
    return { ...user, telegram: user.telegram ?? null }
  },

  // No bot in the mock: the link "completes" a few seconds after the code is issued
  async telegramCode() {
    await wait(150)
    const id = requireUser()
    const code = rid('')
    setTimeout(() => {
      const u = state.users.find((x) => x.id === id)
      if (u) u.telegram = { id: 100000001, username: 'demo_student' }
      save()
    }, 4000)
    return { code, bot_url: 'https://t.me/poisklkbot?start=link_' + code, expires_in: 600 }
  },

  async unlinkTelegram() {
    await wait(100)
    const u = state.users.find((x) => x.id === requireUser())!
    u.telegram = null
    save()
  },

  async system() {
    return {
      offline: true,
      models: { asr: 'faster-whisper large-v3-turbo', embeddings: 'multilingual-e5-base', llm: 'mock (без бэкенда)' },
    }
  },
}

export function resetMock() {
  Object.assign(state, seed())
  save()
}

function bookmarkView({ owner: _, ...b }: State['bookmarks'][number]): Bookmark {
  return { ...b, lecture_title: state.lectures.find((l) => l.id === b.lecture_id)?.title ?? 'Удалённая лекция' }
}

function courseView(c: State['courses'][number]): Course {
  return { ...c, mine: true, lecture_count: state.lectures.filter((l) => l.course_id === c.id).length }
}
