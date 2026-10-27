import { reactive } from 'vue'
import { api } from '@/api'
import type { Bookmark, WatchProgress } from '@/api'
import { toast } from '@/ui'

// Personal data from the API (scoped by X-Client-Id), cached for synchronous reads in views
export const local = reactive({
  bookmarks: [] as Bookmark[],
  watched: {} as Record<string, WatchProgress>,
})

export async function loadPersonal() {
  const [bookmarks, progress] = await Promise.all([api.listBookmarks(), api.listProgress()])
  local.bookmarks = bookmarks
  local.watched = Object.fromEntries(progress.map((p) => [p.lecture_id, p]))
}

export async function addBookmark(lectureId: string, t: number): Promise<Bookmark> {
  local.bookmarks.push(await api.createBookmark({ lecture_id: lectureId, t: Math.floor(t), note: '' }))
  return local.bookmarks[local.bookmarks.length - 1]
}

export async function removeBookmark(id: string) {
  await api.deleteBookmark(id)
  local.bookmarks = local.bookmarks.filter((b) => b.id !== id)
}

export function bookmarksOf(lectureId: string): Bookmark[] {
  return local.bookmarks.filter((b) => b.lecture_id === lectureId).sort((a, b) => a.t - b.t)
}

const pending = new Map<string, number>()

// Debounced so typing sends one PATCH per pause; now=true flushes (on blur)
export function saveNote(b: Bookmark, now = false) {
  clearTimeout(pending.get(b.id))
  const send = () => {
    pending.delete(b.id)
    api.updateBookmark(b.id, { note: b.note }).catch(() => toast('Заметка не сохранилась, проверьте связь с сервером'))
  }
  if (now) send()
  else pending.set(b.id, window.setTimeout(send, 800))
}

export function saveWatched(lectureId: string, position: number) {
  if (position < 3) return
  local.watched[lectureId] = { lecture_id: lectureId, position: Math.floor(position), updated_at: new Date().toISOString() }
  api.saveProgress(lectureId, Math.floor(position)).catch(() => {
    /* progress is best-effort */
  })
}

export async function clearPersonal() {
  await Promise.all(local.bookmarks.map((b) => api.deleteBookmark(b.id)))
  await api.clearProgress()
  local.bookmarks = []
  local.watched = {}
}
