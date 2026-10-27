<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api, ApiError } from '@/api'
import type { KeyTerm, Lecture, LectureStatus, SearchHit, Segment, Summary } from '@/api'
import { addBookmark, bookmarksOf, removeBookmark, saveNote, saveWatched } from '@/bookmarks'
import { downloadMarkdown, printSummary } from '@/exportSummary'
import { duration, markdown, stageName, timecode } from '@/format'
import { settings } from '@/settings'
import { termMatcher } from '@/terms'
import { copy, hideTip, showTip, toast } from '@/ui'
import ChapterBar from '@/components/ChapterBar.vue'
import HeatStrip from '@/components/HeatStrip.vue'
import Icon from '@/components/Icon.vue'
import LectureChat from '@/components/LectureChat.vue'
import OutlineTree from '@/components/OutlineTree.vue'
import TranscriptPanel from '@/components/TranscriptPanel.vue'
import VideoPlayer from '@/components/VideoPlayer.vue'

const route = useRoute()
const id = computed(() => String(route.params.id))

const lecture = ref<Lecture | null>(null)
const segments = ref<Segment[] | null>(null)
const subtitles = ref<string | null>(null)
const summary = ref<Summary | null>(null)
const notFound = ref(false)
const time = ref(0)
const tab = ref<'summary' | 'outline' | 'terms' | 'notes' | 'chat'>('summary')
const player = ref<InstanceType<typeof VideoPlayer> | null>(null)
const focusTerm = ref('')
const found = ref<{ q: string; hits: SearchHit[] } | null>(null)
const focusNote = ref('')
let timer = 0
let lastSaved = 0

const notes = computed(() => (lecture.value ? bookmarksOf(lecture.value.id) : []))

const matcher = computed(() => (settings.terms && summary.value && segments.value ? termMatcher(summary.value.key_terms, segments.value.map((s) => s.text)) : null))
const segmentAt = (t: number) => segments.value?.find((s) => s.start <= t && t < s.end)?.text ?? ''

const ready = computed(() => lecture.value?.status === 'ready')
const caption = computed(() => segments.value?.find((s) => s.start <= time.value && time.value < s.end)?.text ?? '')
const suggestions = computed(() => [
  ...(summary.value?.key_terms.slice(0, 3).map((t) => `Что такое ${t.term}?`) ?? []),
  ...(summary.value?.outline.slice(1, 2).map((o) => `Расскажи подробнее: ${o.title.toLowerCase()}`) ?? []),
])

const STAGES: LectureStatus[] = ['queued', 'downloading', 'extracting_audio', 'transcribing', 'indexing']
const stages = computed(() => STAGES.filter((s) => s !== 'downloading' || lecture.value?.source !== 'upload'))
const stageAt = computed(() => stages.value.indexOf(lecture.value?.status as LectureStatus))

async function poll() {
  clearTimeout(timer)
  let l: Lecture
  try {
    l = await api.getLecture(id.value)
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound.value = true
    else timer = window.setTimeout(poll, 3000)
    return
  }
  const becameReady = l.status === 'ready' && lecture.value?.status !== 'ready'
  const summaryReady = l.summary_status === 'ready' && !summary.value
  lecture.value = l
  if (becameReady) await loadContent()
  if (summaryReady) summary.value = await api.getSummary(l.id).catch(() => null)
  const waiting = (l.status !== 'ready' && l.status !== 'failed') || (l.status === 'ready' && (l.summary_status === 'pending' || l.summary_status === 'processing'))
  if (waiting) timer = window.setTimeout(poll, l.status === 'ready' ? 3000 : 2000)
}

async function loadContent() {
  const [t, sub] = await Promise.all([api.getTranscript(id.value), api.subtitlesUrl(id.value).catch(() => null)])
  subtitles.value = sub
  segments.value = t.segments
  const start = Number(route.query.t)
  if (start >= 0 && route.query.t !== undefined) {
    time.value = start
    await nextTick()
    player.value?.seek(start, false)
  }
}

function seek(t: number) {
  player.value?.seek(t)
}

async function openTerm(t: KeyTerm) {
  tab.value = 'terms'
  focusTerm.value = t.term
  await nextTick()
  document.querySelector(`[data-term-item="${CSS.escape(t.term)}"]`)?.scrollIntoView({ block: 'center', behavior: 'smooth' })
}

function termEvent(e: MouseEvent) {
  const el = (e.target as HTMLElement).closest<HTMLElement>('.term')
  const t = el && summary.value?.key_terms.find((k) => k.term === el.dataset.term)
  if (e.type === 'mouseout') hideTip()
  else if (t && e.type === 'mouseover') showTip(el, t.definition ? `${t.term}: ${t.definition}` : t.term)
  else if (t && e.type === 'click') {
    hideTip()
    void openTerm(t)
  }
}

async function share() {
  if (!lecture.value) return
  const t = Math.floor(time.value)
  const ok = await copy(`${location.origin}/lectures/${lecture.value.id}?t=${t}`)
  toast(ok ? `Ссылка на момент ${timecode(t)} скопирована` : 'Не удалось скопировать ссылку')
}

async function mark() {
  if (!lecture.value) return
  let b
  try {
    b = await addBookmark(lecture.value.id, time.value)
  } catch {
    toast('Не удалось поставить закладку')
    return
  }
  tab.value = 'notes'
  focusNote.value = b.id
  toast(`Закладка на ${timecode(b.t)}: можно дописать заметку`)
  void nextTick(() => document.querySelector<HTMLTextAreaElement>(`[data-note="${b.id}"]`)?.focus())
}

// B (or the same key on the Russian layout) bookmarks the current moment
function onKey(e: KeyboardEvent) {
  if (e.ctrlKey || e.metaKey || e.altKey || (e.target as HTMLElement).closest('input, textarea, select, [contenteditable]')) return
  if (e.code === 'KeyB' && ready.value && segments.value) {
    e.preventDefault()
    void mark()
  }
}

function print() {
  if (lecture.value && summary.value && !printSummary(lecture.value, summary.value)) toast('Браузер заблокировал окно печати')
}

async function retry() {
  if (!lecture.value) return
  lecture.value = await api.retryLecture(lecture.value.id)
  void poll()
}

watch(
  id,
  () => {
    lecture.value = segments.value = summary.value = subtitles.value = found.value = null
    notFound.value = false
    time.value = 0
    void poll()
  },
  { immediate: true },
)

watch(time, (t) => {
  if (lecture.value?.duration && Math.abs(t - lastSaved) >= 5) {
    lastSaved = t
    saveWatched(lecture.value.id, t)
  }
})

watch(
  () => route.query.t,
  (t) => {
    if (t !== undefined && segments.value) player.value?.seek(Number(t), false)
  },
)

onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => {
  clearTimeout(timer)
  window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <div v-if="notFound" class="empty">
    <p>Лекция не найдена. Возможно, её удалили.</p>
    <RouterLink to="/" class="btn">К списку лекций</RouterLink>
  </div>

  <div v-else-if="!lecture" class="empty"><span class="spin" style="display: inline-block" /></div>

  <div v-else-if="!ready" class="proc-wrap">
    <RouterLink to="/" class="back"><Icon name="back" :size="16" /> Все лекции</RouterLink>
    <div class="proc">
      <h1>{{ lecture.title }}</h1>
      <template v-if="lecture.status === 'failed'">
        <p class="fail">{{ lecture.error || 'Не удалось обработать лекцию' }}</p>
        <button class="btn primary" @click="retry"><Icon name="refresh" :size="16" /> Попробовать снова</button>
      </template>
      <template v-else>
        <p class="muted">Обрабатываем видео. Можно закрыть страницу, обработка продолжится.</p>
        <div class="bar big"><i :style="{ width: lecture.progress * 100 + '%' }" /></div>
        <ol class="steps">
          <li v-for="(s, i) in stages" :key="s" :class="{ done: i < stageAt, now: i === stageAt }">
            <span class="dot"><span v-if="i === stageAt" class="spin" /><template v-else-if="i < stageAt">✓</template></span>
            {{ stageName(s) }}
          </li>
        </ol>
      </template>
    </div>
  </div>

  <div v-else class="layout">
    <section class="main">
      <div class="title">
        <RouterLink to="/" class="back"><Icon name="back" :size="16" /></RouterLink>
        <h1>{{ lecture.title }}</h1>
        <span class="muted small">{{ duration(lecture.duration) }}</span>
        <button class="btn" title="Добавить закладку на текущий момент (клавиша B)" @click="mark"><Icon name="bookmark" :size="15" /> <span class="share-text">Отметить</span></button>
        <button class="btn" title="Скопировать ссылку на текущий момент видео" @click="share"><Icon name="link" :size="15" /> <span class="share-text">Поделиться моментом</span></button>
      </div>

      <div class="player-box">
        <VideoPlayer
          v-if="lecture.playback && segments"
          :key="lecture.id"
          ref="player"
          :playback="lecture.playback"
          :subtitles="subtitles"
          :caption="caption"
          @time="time = $event"
        />
        <div v-if="found && lecture.duration" class="heat-wrap">
          <span class="heat-label">
            {{ found.hits.length ? `Где говорят о «${found.q}»` : `«${found.q}»: в этой лекции не нашли` }}
          </span>
          <HeatStrip v-if="found.hits.length" :hits="found.hits" :duration="lecture.duration" :time="time" @seek="seek" />
        </div>
        <ChapterBar
          v-if="summary?.outline.length && lecture.duration"
          :chapters="summary.outline"
          :duration="lecture.duration"
          :time="time"
          :pins="notes.map((n) => n.t)"
          @seek="seek"
        />
      </div>

      <nav class="tabs">
        <button :class="{ on: tab === 'summary' }" @click="tab = 'summary'">Конспект</button>
        <button :class="{ on: tab === 'outline' }" @click="tab = 'outline'">Оглавление</button>
        <button :class="{ on: tab === 'terms' }" @click="tab = 'terms'">Термины</button>
        <button :class="{ on: tab === 'notes' }" @click="tab = 'notes'">Заметки<span v-if="notes.length" class="badge">{{ notes.length }}</span></button>
        <button :class="{ on: tab === 'chat' }" @click="tab = 'chat'"><Icon name="sparkle" :size="15" /> Вопрос-ответ</button>
      </nav>

      <div class="tab">
        <div v-if="tab === 'notes'" class="notes">
          <p v-if="!notes.length" class="muted">
            Отмечайте важные моменты кнопкой «Отметить» или клавишей <kbd>B</kbd> во время просмотра. Все закладки собраны на странице «Закладки».
          </p>
          <div v-for="n in notes" :key="n.id" class="note" :class="{ fresh: focusNote === n.id }">
            <button class="tc" @click="seek(n.t)">{{ timecode(n.t) }}</button>
            <div class="note-body">
              <span class="muted said">{{ segmentAt(n.t) }}</span>
              <textarea
                v-model="n.note"
                :data-note="n.id"
                rows="2"
                placeholder="Заметка к моменту…"
                @input="saveNote(n)"
                @blur="saveNote(n, true)"
              />
            </div>
            <button class="btn ghost del" title="Удалить закладку" @click="removeBookmark(n.id)"><Icon name="trash" :size="15" /></button>
          </div>
        </div>
        <div v-else-if="tab !== 'chat' && !summary" class="pending">
          <template v-if="lecture.summary_status === 'failed'">
            <span class="fail">Конспект не получился.</span>
            <button class="btn" @click="retry"><Icon name="refresh" :size="16" /> Повторить</button>
          </template>
          <template v-else><span class="spin" /> Нейросеть пишет конспект и оглавление, это займёт несколько минут…</template>
        </div>
        <template v-else-if="tab === 'summary' && summary">
          <div class="md summary" @mouseover="termEvent" @mouseout="termEvent" @click="termEvent" v-html="markdown(summary.summary, matcher ?? undefined)" />
          <div class="export">
            <button class="btn" @click="downloadMarkdown(lecture, summary)"><Icon name="download" :size="15" /> Скачать .md</button>
            <button class="btn" @click="print"><Icon name="print" :size="15" /> Печать / PDF</button>
          </div>
        </template>
        <OutlineTree v-else-if="tab === 'outline' && summary" :items="summary.outline" :time="time" @seek="seek" />
        <ul v-else-if="tab === 'terms' && summary" class="terms">
          <li v-for="t in summary.key_terms" :key="t.term" :data-term-item="t.term" :class="{ focus: focusTerm === t.term }">
            <div>
              <b>{{ t.term }}</b>
              <span v-if="t.definition" class="muted"> — {{ t.definition }}</span>
            </div>
            <div class="occ">
              <button v-for="o in t.occurrences" :key="o" class="tc" :title="segmentAt(o)" @click="seek(o)">{{ timecode(o) }}</button>
            </div>
          </li>
        </ul>
        <LectureChat v-show="tab === 'chat'" :lecture-id="lecture.id" :suggestions="suggestions" @seek="seek" />
      </div>
    </section>

    <TranscriptPanel
      v-if="segments"
      :key="lecture.id"
      class="side"
      :lecture-id="lecture.id"
      :segments="segments"
      :time="time"
      :initial-query="typeof route.query.q === 'string' ? route.query.q : ''"
      :matcher="matcher"
      @hits="found = $event"
      @seek="seek"
      @term="openTerm"
    />
  </div>
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 400px;
  gap: 16px;
  height: calc(100vh - var(--header));
  padding: 16px 20px;
}
.main {
  min-height: 0;
  overflow-y: auto;
  padding-right: 4px;
}
.side {
  min-height: 0;
}
.title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.title h1 {
  flex: 1;
  min-width: 0;
  font-size: 19px;
}
.back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--muted);
  padding: 4px;
  border-radius: 6px;
}
.back:hover {
  background: var(--surface-2);
  color: var(--text);
}
.small {
  font-size: 13px;
  white-space: nowrap;
}
.player-box {
  max-width: calc((100vh - var(--header) - 230px) * 16 / 9);
  min-width: min(100%, 480px);
}
.tabs {
  display: flex;
  gap: 2px;
  margin: 16px 0 0;
  border-bottom: 1px solid var(--border);
  overflow-x: auto;
  overflow-y: hidden;
}
.tabs button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 9px 14px;
  border: 0;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  background: none;
  color: var(--muted);
  cursor: pointer;
  white-space: nowrap;
}
.tabs button.on {
  color: var(--text);
  border-bottom-color: var(--accent);
  font-weight: 600;
}
.tab {
  padding: 16px 2px 24px;
}
.summary {
  max-width: 760px;
  line-height: 1.6;
}
.pending {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--muted);
}
.terms {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 760px;
}
.terms li {
  padding: 8px 10px;
  margin: -8px -10px;
  border-radius: 10px;
  transition: background 0.6s;
}
.terms li.focus {
  background: var(--accent-soft);
}
.badge {
  margin-left: 6px;
  padding: 0 6px;
  border-radius: 9px;
  background: var(--accent-soft);
  color: var(--accent-ink);
  font-size: 12px;
  font-weight: 700;
}
.heat-wrap {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-top: 10px;
}
.heat-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--muted);
}
.notes {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 760px;
}
.note {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}
.note.fresh {
  border-color: var(--accent);
}
.note-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.said {
  font-size: 13px;
}
.note textarea {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  resize: vertical;
  outline: none;
}
.note textarea:focus {
  border-color: var(--accent);
}
.note .del {
  height: 30px;
  padding: 0 8px;
  color: var(--faint);
}
.export {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}
.occ {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 5px;
}
.fail {
  color: var(--err);
}
.proc-wrap {
  max-width: 560px;
  margin: 0 auto;
  padding: 24px 20px;
}
.proc {
  margin-top: 14px;
  padding: 24px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--surface);
}
.proc h1 {
  font-size: 20px;
  margin-bottom: 6px;
}
.bar.big {
  height: 8px;
  margin: 16px 0;
}
.steps {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.steps li {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--faint);
}
.steps li.done {
  color: var(--muted);
}
.steps li.now {
  color: var(--text);
  font-weight: 600;
}
.dot {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1.5px solid var(--border);
  font-size: 12px;
  color: var(--ok);
}
.done .dot {
  border-color: var(--ok);
}
.now .dot {
  border: 0;
}
@media (max-width: 1000px) {
  .layout {
    display: flex;
    flex-direction: column;
    height: auto;
    padding: 12px;
  }
  .main {
    overflow: visible;
  }
  .player-box {
    max-width: none;
    min-width: 0;
  }
  .side {
    height: 70vh;
  }
  .share-text {
    display: none;
  }
}
</style>
