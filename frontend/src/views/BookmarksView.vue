<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api'
import type { Bookmark, Lecture } from '@/api'
import { local, removeBookmark } from '@/bookmarks'
import { timecode } from '@/format'
import FrameThumb from '@/components/FrameThumb.vue'
import Icon from '@/components/Icon.vue'

const filter = ref('')
const lectures = ref<Lecture[]>([])
const lectureOf = (id: string) => lectures.value.find((l) => l.id === id)
const videoOf = (id: string) => {
  const p = lectureOf(id)?.playback
  return p?.type === 'file' ? p.url : null
}

onMounted(async () => {
  lectures.value = await api.listLectures().catch(() => [])
})

const groups = computed(() => {
  const f = filter.value.trim().toLowerCase()
  const list = local.bookmarks.filter((b) => !f || b.note.toLowerCase().includes(f) || b.lecture_title.toLowerCase().includes(f))
  const map = new Map<string, { title: string; items: Bookmark[] }>()
  for (const b of list) {
    if (!map.has(b.lecture_id)) map.set(b.lecture_id, { title: b.lecture_title, items: [] })
    map.get(b.lecture_id)!.items.push(b)
  }
  for (const g of map.values()) g.items.sort((a, b) => a.t - b.t)
  return [...map.entries()].map(([id, g]) => ({ id, ...g }))
})

function exportAll() {
  const md = groups.value
    .map(
      (g) =>
        `## ${g.title}\n\n` +
        g.items.map((b) => `- [${timecode(b.t)}](${location.origin}/lectures/${g.id}?t=${b.t})${b.note ? ' ' + b.note.replace(/\n+/g, ' ') : ''}`).join('\n'),
    )
    .join('\n\n')
  const a = Object.assign(document.createElement('a'), {
    href: URL.createObjectURL(new Blob(['# Мои закладки\n\n' + md], { type: 'text/markdown;charset=utf-8' })),
    download: 'Закладки.md',
  })
  a.click()
  setTimeout(() => URL.revokeObjectURL(a.href), 1000)
}
</script>

<template>
  <div class="page">
    <div class="head">
      <h1>Закладки</h1>
      <div v-if="local.bookmarks.length" class="tools">
        <input v-model="filter" class="input" placeholder="Найти в заметках" />
        <button class="btn" @click="exportAll"><Icon name="download" :size="15" /> Скачать .md</button>
      </div>
    </div>

    <div v-if="!local.bookmarks.length" class="empty">
      <Icon name="bookmark" :size="32" />
      <p>Закладок пока нет.</p>
      <p>Во время просмотра нажмите «Отметить» или клавишу <kbd>B</kbd>, чтобы сохранить момент и дописать к нему заметку.</p>
    </div>
    <p v-else-if="!groups.length" class="empty">Ничего не нашли.</p>

    <section v-for="g in groups" :key="g.id" class="group">
      <RouterLink :to="`/lectures/${g.id}`" class="lec">{{ g.title }}</RouterLink>
      <div v-for="b in g.items" :key="b.id" class="item">
        <RouterLink :to="`/lectures/${g.id}?t=${b.t}`" class="shot">
          <FrameThumb :video="videoOf(g.id)" :t="b.t" :fallback="lectureOf(g.id)?.thumbnail_url" />
        </RouterLink>
        <span v-if="b.note" class="note">{{ b.note }}</span>
        <span v-else class="note muted">без заметки</span>
        <button class="btn ghost del" title="Удалить" @click="removeBookmark(b.id)"><Icon name="trash" :size="15" /></button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page {
  max-width: 860px;
  margin: 0 auto;
  padding: 24px 20px 40px;
}
.head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}
h1 {
  font-size: 24px;
}
.tools {
  display: flex;
  gap: 8px;
}
.tools .input {
  width: 240px;
  height: 36px;
}
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.empty p {
  margin: 4px 0;
  max-width: 460px;
}
.group {
  margin-bottom: 18px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}
.lec {
  display: block;
  margin-bottom: 8px;
  font-weight: 700;
}
.lec:hover {
  color: var(--accent-ink);
}
.item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-top: 1px solid var(--border);
}
.shot {
  flex-shrink: 0;
  width: 112px;
}
.note {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.del {
  height: 28px;
  padding: 0 8px;
  color: var(--faint);
}
.del:hover {
  color: var(--err);
}
@media (max-width: 560px) {
  .tools,
  .tools .input {
    width: 100%;
  }
}
</style>
