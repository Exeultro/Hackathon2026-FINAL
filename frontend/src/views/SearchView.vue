<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, ApiError, session } from '@/api'
import type { Course, Lecture, SearchHit, SearchResponse } from '@/api'
import { timecode } from '@/format'
import HitText from '@/components/HitText.vue'
import { EXAMPLE_QUERIES } from '@/examples'
import { settings } from '@/settings'
import FrameThumb from '@/components/FrameThumb.vue'
import HeatStrip from '@/components/HeatStrip.vue'

const route = useRoute()
const router = useRouter()
const res = ref<SearchResponse | null>(null)
const loading = ref(false)
const error = ref('')
const courses = ref<Course[]>([])
const lectures = ref<Lecture[]>([])

const durationOf = (id: string) => lectures.value.find((l) => l.id === id)?.duration ?? 0
const videoOf = (id: string) => {
  const p = lectures.value.find((l) => l.id === id)?.playback
  return p?.type === 'file' ? p.url : null
}
const sameLecture = (h: SearchHit) => res.value?.hits.filter((x) => x.lecture.id === h.lecture.id) ?? []

const q = computed(() => String(route.query.q ?? '').trim())
const courseId = computed(() => String(route.query.course_id ?? ''))


async function run() {
  error.value = ''
  if (q.value.length < 2) {
    res.value = null
    return
  }
  loading.value = true
  try {
    res.value = await api.search({ q: q.value, course_id: courseId.value || undefined, limit: settings.searchLimit })
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Поиск не ответил, попробуйте ещё раз'
    res.value = null
  } finally {
    loading.value = false
  }
}

function setCourse(id: string) {
  router.replace({ query: { ...route.query, course_id: id || undefined } })
}

function open(h: SearchHit) {
  const u = new URL(h.deep_link, location.origin)
  router.push({ path: u.pathname, query: { ...Object.fromEntries(u.searchParams), q: q.value } })
}

watch([q, courseId, () => session.user?.id], run)
onMounted(async () => {
  void run()
  ;[courses.value, lectures.value] = await Promise.all([api.listCourses(), api.listLectures()])
})
</script>

<template>
  <div class="page">
    <div v-if="!q" class="intro">
      <h1>Поиск по смыслу</h1>
      <p class="muted">Опишите своими словами, что ищете. Найдём фрагменты, где об этом говорят, даже если слова другие.</p>
      <div class="examples">
        <RouterLink v-for="e in EXAMPLE_QUERIES" :key="e" :to="{ path: '/search', query: { q: e } }">{{ e }}</RouterLink>
      </div>
    </div>

    <template v-else>
      <div class="head">
        <h1>«{{ q }}»</h1>
        <div class="scope">
          <span class="muted">Где искать:</span>
          <select class="input" :value="courseId" @change="setCourse(($event.target as HTMLSelectElement).value)">
            <option value="">Все лекции</option>
            <option v-for="c in courses" :key="c.id" :value="c.id">{{ c.title }}</option>
          </select>
        </div>
      </div>

      <p v-if="res && !loading" class="meta muted">
        {{ res.hits.length ? `Нашли ${res.hits.length} фрагм. за ${res.took_ms} мс` : 'Ничего не нашли. Попробуйте сформулировать иначе.' }}
      </p>
      <p v-if="error" class="meta err">{{ error }}</p>
      <div v-if="loading" class="empty"><span class="spin" style="display: inline-block" /></div>

      <ol v-else-if="res" class="hits">
        <li v-for="(h, i) in res.hits" :key="i" class="hit" @click="open(h)">
          <FrameThumb class="thumb" :video="videoOf(h.lecture.id)" :t="h.start" :fallback="h.lecture.thumbnail_url" />
          <div class="body">
          <div class="top">
            <button class="tc" @click.stop="open(h)">▶ {{ timecode(h.start) }}</button>
            <span class="lec">{{ h.lecture.title }}</span>
            <span class="score" :title="`Релевантность ${Math.round(h.score * 100)}%`">
              <i :style="{ width: h.score * 100 + '%' }" />
            </span>
          </div>
          <p class="text"><HitText :text="h.text" :highlights="h.highlights" /></p>
          <span v-if="!h.highlights.length" class="semantic">найдено по смыслу</span>
          <HeatStrip
            v-if="durationOf(h.lecture.id)"
            class="where"
            compact
            :hits="sameLecture(h)"
            :duration="durationOf(h.lecture.id)"
            :active="h.start"
          />
          </div>
        </li>
      </ol>
    </template>
  </div>
</template>

<style scoped>
.page {
  max-width: 860px;
  margin: 0 auto;
  padding: 24px 20px 40px;
}
.intro {
  padding: 48px 0;
  text-align: center;
}
.intro h1 {
  font-size: 26px;
  margin-bottom: 8px;
}
.examples {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 20px;
}
.examples a {
  padding: 7px 14px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--surface);
}
.examples a:hover {
  border-color: var(--accent);
  color: var(--accent-ink);
}
.head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.head h1 {
  font-size: 22px;
  overflow-wrap: anywhere;
}
.scope {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}
.scope .input {
  width: auto;
  height: 34px;
}
.meta {
  margin: 8px 0 16px;
  font-size: 14px;
}
.err {
  color: var(--err);
}
.hits {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.hit {
  display: flex;
  gap: 14px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  cursor: pointer;
}
.hit:hover {
  border-color: var(--accent);
}
.thumb {
  flex-shrink: 0;
  width: 150px;
  align-self: flex-start;
}
.body {
  flex: 1;
  min-width: 0;
}
.where {
  margin-top: 10px;
}
@media (max-width: 560px) {
  .thumb {
    display: none;
  }
}
.top {
  display: flex;
  align-items: center;
  gap: 10px;
}
.lec {
  flex: 1;
  min-width: 0;
  font-size: 13.5px;
  font-weight: 600;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.score {
  width: 48px;
  height: 5px;
  border-radius: 3px;
  background: var(--surface-2);
  overflow: hidden;
  flex-shrink: 0;
}
.score i {
  display: block;
  height: 100%;
  background: var(--ok);
}
.text {
  margin: 8px 0 0;
}
.semantic {
  display: inline-block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--accent-ink);
}
</style>
