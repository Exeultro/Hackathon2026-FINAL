<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api, session } from '@/api'
import type { Course, Lecture } from '@/api'
import { actionFailed, requireLogin } from '@/auth'
import { local } from '@/bookmarks'
import { EXAMPLE_QUERIES } from '@/examples'
import { duration, plural, stageName, timecode } from '@/format'
import Icon from '@/components/Icon.vue'

const router = useRouter()
const q = ref('')

const lectures = ref<Lecture[]>([])
const courses = ref<Course[]>([])
const course = ref('')
const loading = ref(true)
const loadError = ref(false)
const adding = ref(false)
const newCourse = ref('')
let timer = 0

const busy = (l: Lecture) => l.status !== 'ready' && l.status !== 'failed'
const shown = computed(() => lectures.value.filter((l) => !course.value || l.course_id === course.value))
const courseTitle = (id: string | null | undefined) => courses.value.find((c) => c.id === id)?.title
const ready = computed(() => lectures.value.filter((l) => l.status === 'ready'))
const hours = computed(() => duration(ready.value.reduce((a, l) => a + (l.duration ?? 0), 0)))
const watchedShare = (l: Lecture) => {
  const w = local.watched[l.id]
  return w && l.duration ? Math.min(1, w.position / l.duration) : 0
}
const continueList = computed(() =>
  ready.value
    .filter((l) => local.watched[l.id] && watchedShare(l) < 0.97)
    .sort((a, b) => local.watched[b.id].updated_at.localeCompare(local.watched[a.id].updated_at))
    .slice(0, 3),
)

function search(text = q.value) {
  if (text.trim().length >= 2) router.push({ path: '/search', query: { q: text.trim() } })
}

async function load() {
  clearTimeout(timer)
  try {
    ;[lectures.value, courses.value] = await Promise.all([api.listLectures(), api.listCourses()])
    loadError.value = false
  } catch {
    loadError.value = true
    timer = window.setTimeout(load, 5000)
    return
  } finally {
    loading.value = false
  }
  if (lectures.value.some(busy)) timer = window.setTimeout(load, 2500)
}

async function addCourse() {
  const title = newCourse.value.trim()
  if (!title) return
  let c: Course
  try {
    c = await api.createCourse(title)
  } catch (e) {
    actionFailed(e, 'Чтобы создать курс, войдите')
    return
  }
  courses.value.push(c)
  course.value = c.id
  newCourse.value = ''
  adding.value = false
}

function startCourse() {
  if (requireLogin('Чтобы создавать курсы, войдите', startCourse)) adding.value = true
}

async function remove(l: Lecture) {
  if (!requireLogin('Чтобы удалять лекции, войдите', () => remove(l))) return
  if (!confirm(`Удалить «${l.title}»? Видео, текст и индекс удалятся.`)) return
  try {
    await api.deleteLecture(l.id)
  } catch (e) {
    actionFailed(e, 'Чтобы удалять лекции, войдите')
    return
  }
  await load()
}

function hue(id: string) {
  let h = 0
  for (const ch of id) h = (h * 31 + ch.charCodeAt(0)) % 360
  return h
}

onMounted(load)
// Private lectures and courses depend on who is signed in
watch(() => session.user?.id, () => {
  course.value = ''
  load()
})
onUnmounted(() => clearTimeout(timer))
</script>

<template>
  <div class="page">
    <section class="hero">
      <h1>Найдите нужный момент в любой лекции</h1>
      <p class="sub">Ищем по смыслу, а не по словам: опишите своими словами, что хотите найти.</p>
      <form class="big-search" @submit.prevent="search()">
        <Icon name="search" :size="20" class="lens" />
        <input v-model="q" class="input" placeholder="Например: как доказать теорему Пифагора" data-search />
        <button class="btn primary" :disabled="q.trim().length < 2">Найти</button>
      </form>
      <div class="examples">
        <button v-for="e in EXAMPLE_QUERIES" :key="e" @click="search(e)">{{ e }}</button>
      </div>
      <div v-if="!loading" class="stats">
        <span><b>{{ plural(ready.length, ['лекция', 'лекции', 'лекций']) }}</b> в индексе</span>
        <span v-if="hours"><b>{{ hours }}</b> видео</span>
        <span><b>{{ plural(courses.length, ['курс', 'курса', 'курсов']) }}</b></span>
        <span><b>{{ plural(local.bookmarks.length, ['закладка', 'закладки', 'закладок']) }}</b></span>
      </div>
    </section>

    <section v-if="continueList.length" class="continue">
      <h2>Продолжить просмотр</h2>
      <div class="row">
        <RouterLink v-for="l in continueList" :key="l.id" :to="`/lectures/${l.id}?t=${local.watched[l.id].position}`" class="mini">
          <div class="mini-thumb" :style="{ '--h': hue(l.id) }">
            <img v-if="l.thumbnail_url" :src="l.thumbnail_url" alt="" />
            <i class="watched" :style="{ width: watchedShare(l) * 100 + '%' }" />
          </div>
          <div>
            <b>{{ l.title }}</b>
            <span class="muted small">с {{ timecode(local.watched[l.id].position) }}</span>
          </div>
        </RouterLink>
      </div>
    </section>

    <div class="head">
      <h2>Все лекции</h2>
      <div class="chips">
        <button :class="{ on: !course }" @click="course = ''">Все</button>
        <button v-for="c in courses" :key="c.id" :class="{ on: course === c.id }" @click="course = c.id">
          {{ c.title }} <span class="n">{{ c.lecture_count }}</span>
        </button>
        <form v-if="adding" class="add" @submit.prevent="addCourse">
          <input v-model="newCourse" class="input" placeholder="Название курса" autofocus @blur="!newCourse && (adding = false)" />
        </form>
        <button v-else class="ghost" @click="startCourse"><Icon name="plus" :size="15" /> Курс</button>
      </div>
    </div>

    <div v-if="loading" class="empty"><span class="spin" style="display: inline-block" /></div>
    <div v-else-if="loadError && !lectures.length" class="empty">
      <p>Сервер не отвечает, пробуем снова.</p>
      <button class="btn" @click="load">Повторить</button>
    </div>
    <div v-else-if="!shown.length" class="empty">
      <p>Здесь пока пусто.</p>
      <p>Добавьте видеолекцию файлом или ссылкой на YouTube - кнопка сверху справа.</p>
    </div>

    <div v-else class="grid">
      <RouterLink v-for="l in shown" :key="l.id" :to="`/lectures/${l.id}`" class="card">
        <div class="thumb" :style="{ '--h': hue(l.id) }">
          <img v-if="l.thumbnail_url" :src="l.thumbnail_url" alt="" />
          <span v-else class="ph"><Icon name="play" :size="26" /></span>
          <span v-if="l.duration && !busy(l)" class="dur">{{ duration(l.duration) }}</span>
          <i v-if="watchedShare(l)" class="watched" :style="{ width: watchedShare(l) * 100 + '%' }" />
          <span v-if="l.source !== 'upload'" class="src">{{ l.source === 'youtube' ? 'YouTube' : l.source === 'vimeo' ? 'Vimeo' : 'Ссылка' }}</span>
        </div>
        <div class="body">
          <h3>{{ l.title }}</h3>
          <span v-if="courseTitle(l.course_id)" class="muted small">{{ courseTitle(l.course_id) }}</span>
          <div v-if="busy(l)" class="proc">
            <div class="bar"><i :style="{ width: l.progress * 100 + '%' }" /></div>
            <span class="small muted"><span class="spin tiny" /> {{ stageName(l.status) }} · {{ Math.round(l.progress * 100) }}%</span>
          </div>
          <span v-else-if="l.status === 'failed'" class="small err">Ошибка обработки</span>
          <span v-else-if="l.summary_status === 'processing' || l.summary_status === 'pending'" class="small muted">
            <Icon name="sparkle" :size="13" /> Готовим конспект…
          </span>
        </div>
        <button v-if="l.mine" class="del btn ghost" title="Удалить" @click.prevent="remove(l)"><Icon name="trash" :size="16" /></button>
      </RouterLink>
    </div>
  </div>
</template>

<style scoped>
.page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px 20px 40px;
}
.hero {
  padding: 44px 24px 32px;
  margin-bottom: 28px;
  border-radius: 18px;
  text-align: center;
  background:
    radial-gradient(120% 140% at 0% 0%, var(--accent-soft) 0%, transparent 55%),
    radial-gradient(120% 140% at 100% 100%, var(--accent-soft) 0%, transparent 50%),
    var(--surface);
  border: 1px solid var(--border);
}
.hero h1 {
  font-size: clamp(24px, 3.2vw, 34px);
  letter-spacing: -0.01em;
}
.sub {
  margin: 10px auto 22px;
  max-width: 560px;
  color: var(--muted);
}
.big-search {
  position: relative;
  display: flex;
  gap: 8px;
  max-width: 680px;
  margin: 0 auto;
}
.big-search .input {
  height: 52px;
  padding-left: 46px;
  border-radius: 12px;
  font-size: 16px;
  box-shadow: var(--shadow);
}
.big-search .btn {
  height: 52px;
  padding: 0 22px;
  border-radius: 12px;
}
.big-search .lens {
  position: absolute;
  left: 16px;
  top: 16px;
  color: var(--faint);
}
.examples {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
}
.examples button {
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  font-size: 13.5px;
  cursor: pointer;
}
.examples button:hover {
  border-color: var(--accent);
  color: var(--accent-ink);
}
.stats {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px 22px;
  margin-top: 22px;
  color: var(--muted);
  font-size: 14px;
}
.stats b {
  color: var(--text);
}
h2 {
  font-size: 19px;
}
.continue {
  margin-bottom: 28px;
}
.continue h2 {
  margin-bottom: 12px;
}
.row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.mini {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}
.mini:hover {
  border-color: var(--accent);
}
.mini > div:last-child {
  display: flex;
  flex-direction: column;
  min-width: 0;
  font-size: 14px;
}
.mini-thumb {
  position: relative;
  flex-shrink: 0;
  width: 112px;
  aspect-ratio: 16 / 9;
  border-radius: 8px;
  overflow: hidden;
  background: linear-gradient(135deg, hsl(var(--h) 45% 32%), hsl(calc(var(--h) + 40) 50% 22%));
}
.mini-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.watched {
  position: absolute;
  left: 0;
  bottom: 0;
  height: 3px;
  background: var(--accent);
}
.head {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 20px;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.chips button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  cursor: pointer;
}
.chips button.on {
  background: var(--text);
  border-color: var(--text);
  color: var(--bg);
}
.chips button.ghost {
  border-style: dashed;
  color: var(--muted);
}
.n {
  font-size: 12px;
  opacity: 0.65;
}
.add .input {
  height: 32px;
  width: 220px;
  border-radius: 16px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 18px;
}
.card {
  position: relative;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  overflow: hidden;
  transition: box-shadow 0.15s, transform 0.15s;
}
.card:hover {
  box-shadow: 0 6px 20px rgb(16 24 40 / 10%);
  transform: translateY(-1px);
}
.thumb {
  position: relative;
  aspect-ratio: 16 / 9;
  background: linear-gradient(135deg, hsl(var(--h) 45% 32%), hsl(calc(var(--h) + 40) 50% 22%));
  display: grid;
  place-items: center;
}
.thumb img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.ph {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: rgb(255 255 255 / 16%);
  color: #fff;
}
.dur,
.src {
  position: absolute;
  bottom: 8px;
  padding: 1px 7px;
  border-radius: 5px;
  background: rgb(0 0 0 / 65%);
  color: #fff;
  font-size: 12px;
}
.dur {
  right: 8px;
}
.src {
  left: 8px;
}
.body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px 14px;
}
h3 {
  font-size: 15px;
  font-weight: 600;
  padding-right: 24px;
}
.small {
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.err {
  color: var(--err);
}
.proc {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 2px;
}
.tiny {
  width: 11px;
  height: 11px;
  border-width: 1.5px;
}
.del {
  position: absolute;
  right: 6px;
  bottom: 8px;
  padding: 0 8px;
  height: 30px;
  color: var(--faint);
  opacity: 0;
}
.card:hover .del {
  opacity: 1;
}
.del:hover {
  color: var(--err);
}
@media (hover: none) {
  .del {
    opacity: 1;
  }
}
</style>
