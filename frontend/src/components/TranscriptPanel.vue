<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { api, ApiError } from '@/api'
import type { KeyTerm, SearchHit, Segment } from '@/api'
import { settings } from '@/settings'
import type { TermMatcher } from '@/terms'
import { timecode } from '@/format'
import HitText from './HitText.vue'
import Icon from './Icon.vue'
import TermText from './TermText.vue'

const props = defineProps<{ lectureId: string; segments: Segment[]; time: number; initialQuery?: string; matcher?: TermMatcher | null }>()
const emit = defineEmits<{ seek: [t: number]; term: [term: KeyTerm]; hits: [found: { q: string; hits: SearchHit[] } | null] }>()

const list = ref<HTMLElement | null>(null)
const q = ref(props.initialQuery ?? '')
const hits = ref<SearchHit[] | null>(null)
const searching = ref(false)
const error = ref('')
let userScrollUntil = 0

const current = computed(() => {
  const segs = props.segments
  let lo = 0
  let hi = segs.length - 1
  let found = -1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (segs[mid].start <= props.time + 0.05) {
      found = mid
      lo = mid + 1
    } else hi = mid - 1
  }
  return found
})

async function follow(behavior: ScrollBehavior) {
  const i = current.value
  if (i < 0 || hits.value || !settings.autoscroll || Date.now() < userScrollUntil) return
  await nextTick()
  const row = list.value?.querySelector<HTMLElement>(`[data-i="${i}"]`)
  if (!row || !list.value) return
  const box = list.value.getBoundingClientRect()
  const r = row.getBoundingClientRect()
  if (r.top < box.top + 40 || r.bottom > box.bottom - 40)
    list.value.scrollTo({ top: row.offsetTop - list.value.clientHeight / 3, behavior })
}

watch(current, () => follow('smooth'))
onMounted(() => follow('auto'))

async function search() {
  const text = q.value.trim()
  error.value = ''
  if (text.length < 2) {
    hits.value = null
    return
  }
  searching.value = true
  try {
    hits.value = (await api.search({ q: text, lecture_id: props.lectureId, limit: settings.searchLimit })).hits
    emit('hits', { q: text, hits: hits.value })
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Поиск не ответил'
  } finally {
    searching.value = false
  }
}

function clear() {
  q.value = ''
  hits.value = null
  error.value = ''
  emit('hits', null)
}

if (q.value) void search()
</script>

<template>
  <aside class="panel">
    <form class="find" @submit.prevent="search">
      <Icon name="search" class="lens" :size="16" />
      <input v-model="q" class="input" placeholder="Найти в этой лекции по смыслу" @keydown.esc="clear" />
      <button v-if="hits || q" type="button" class="x" title="Сбросить" @click="clear"><Icon name="x" :size="15" /></button>
    </form>

    <div v-if="searching" class="empty"><span class="spin" style="display: inline-block" /></div>
    <div v-else-if="hits" class="list">
      <p class="note muted">
        {{ hits.length ? `Фрагменты по запросу «${q}»` : 'Ничего не нашли' }}
        <button class="link" @click="clear">к полному тексту</button>
      </p>
      <button v-for="(h, i) in hits" :key="i" class="hit" @click="emit('seek', h.start)">
        <span class="tc">{{ timecode(h.start) }}</span>
        <span><HitText :text="h.text" :highlights="h.highlights" /></span>
      </button>
    </div>
    <p v-if="error" class="note err">{{ error }}</p>

    <div v-show="!hits && !searching" ref="list" class="list" @wheel.passive="userScrollUntil = Date.now() + 4000" @touchmove.passive="userScrollUntil = Date.now() + 4000">
      <button
        v-for="(s, i) in segments"
        :key="s.id"
        :data-i="i"
        class="seg"
        :class="{ now: i === current }"
        @click="emit('seek', s.start)"
      >
        <span class="tc">{{ timecode(s.start) }}</span>
        <TermText :text="s.text" :matcher="matcher" @term="emit('term', $event)" />
      </button>
    </div>
  </aside>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  overflow: hidden;
}
.find {
  position: relative;
  padding: 10px;
  border-bottom: 1px solid var(--border);
}
.find .input {
  padding: 0 34px;
}
.lens {
  position: absolute;
  left: 22px;
  top: 21px;
  color: var(--faint);
}
.x {
  position: absolute;
  right: 18px;
  top: 18px;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--faint);
  cursor: pointer;
}
.x:hover {
  background: var(--surface-2);
}
.list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 6px;
  position: relative;
}
.seg,
.hit {
  display: flex;
  gap: 10px;
  width: 100%;
  padding: 7px 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  text-align: left;
  line-height: 1.45;
  cursor: pointer;
}
.seg:hover,
.hit:hover {
  background: var(--surface-2);
}
.seg .tc,
.hit .tc {
  margin-top: 1px;
  background: transparent;
  color: var(--faint);
}
.seg.now {
  background: var(--accent-soft);
}
.seg.now .tc {
  color: var(--accent-ink);
}
.hit {
  margin-bottom: 4px;
  border: 1px solid var(--border);
}
.hit .tc {
  background: var(--accent-soft);
  color: var(--accent-ink);
}
.note {
  margin: 4px 6px 10px;
  font-size: 13px;
}
.err {
  color: var(--err);
}
.link {
  border: 0;
  padding: 0;
  margin-left: 6px;
  background: none;
  color: var(--accent-ink);
  cursor: pointer;
  text-decoration: underline;
}
</style>
