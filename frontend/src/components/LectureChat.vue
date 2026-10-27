<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { api, ApiError } from '@/api'
import type { ChatMessage, SearchHit } from '@/api'
import { markdown, timecode } from '@/format'
import Icon from './Icon.vue'

const props = defineProps<{ lectureId: string; suggestions: string[] }>()
const emit = defineEmits<{ seek: [t: number] }>()

interface Turn extends ChatMessage {
  sources?: SearchHit[]
  error?: boolean
}

const turns = ref<Turn[]>([])
const q = ref('')
const thinking = ref(false)
const log = ref<HTMLElement | null>(null)
const form = ref<HTMLElement | null>(null)

async function scroll() {
  await nextTick()
  log.value?.scrollTo({ top: log.value.scrollHeight, behavior: 'smooth' })
  form.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
}

async function ask(text = q.value) {
  const question = text.trim()
  if (question.length < 2 || thinking.value) return
  const history = turns.value.filter((t) => !t.error).map(({ role, content }) => ({ role, content }))
  turns.value.push({ role: 'user', content: question })
  q.value = ''
  thinking.value = true
  void scroll()
  try {
    const r = await api.ask({ question, lecture_id: props.lectureId, history: history.slice(-20) })
    turns.value.push({ role: 'assistant', content: r.answer, sources: r.sources })
  } catch (e) {
    turns.value.push({ role: 'assistant', content: e instanceof ApiError ? e.message : 'Не получилось ответить, попробуйте ещё раз', error: true })
  } finally {
    thinking.value = false
    void scroll()
  }
}

function onCite(e: MouseEvent, t: Turn) {
  const n = (e.target as HTMLElement).closest<HTMLElement>('[data-cite]')?.dataset.cite
  const src = n ? t.sources?.[Number(n) - 1] : undefined
  if (src) emit('seek', src.start)
}
</script>

<template>
  <div class="chat">
    <div ref="log" class="log">
      <div v-if="!turns.length" class="hello">
        <p class="muted">Задайте вопрос по лекции. Ответ будет со ссылками на моменты видео.</p>
        <div class="sugg">
          <button v-for="s in suggestions" :key="s" @click="ask(s)">{{ s }}</button>
        </div>
      </div>
      <div v-for="(t, i) in turns" :key="i" class="turn" :class="t.role">
        <div v-if="t.role === 'user'" class="bubble">{{ t.content }}</div>
        <div v-else class="answer" :class="{ err: t.error }">
          <div class="md" @click="onCite($event, t)" v-html="markdown(t.content)" />
          <div v-if="t.sources?.length" class="sources">
            <button v-for="(s, k) in t.sources" :key="k" class="src" @click="emit('seek', s.start)">
              <span class="cite">{{ k + 1 }}</span>
              <span class="tc">{{ timecode(s.start) }}</span>
              <span class="snip">{{ s.text }}</span>
            </button>
          </div>
        </div>
      </div>
      <div v-if="thinking" class="turn assistant thinking"><span class="spin" /> Ищу в лекции и формулирую ответ…</div>
    </div>
    <form ref="form" class="ask" @submit.prevent="ask()">
      <input v-model="q" class="input" placeholder="Спросите что-нибудь по лекции" :disabled="thinking" />
      <button class="btn primary" :disabled="thinking || q.trim().length < 2" title="Спросить"><Icon name="send" :size="16" /></button>
    </form>
  </div>
</template>

<style scoped>
.chat {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.log {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 420px;
  overflow-y: auto;
}
.hello p {
  margin: 0 0 10px;
}
.sugg {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.sugg button {
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
  cursor: pointer;
  text-align: left;
}
.sugg button:hover {
  border-color: var(--accent);
}
.turn.user {
  align-self: flex-end;
  max-width: 80%;
}
.bubble {
  padding: 8px 12px;
  border-radius: 12px 12px 4px 12px;
  background: var(--accent);
  color: var(--on-accent);
}
.answer {
  max-width: 92%;
}
.answer.err {
  color: var(--err);
}
.answer .md :deep(p:last-child),
.answer .md :deep(ul:last-child) {
  margin-bottom: 0;
}
.sources {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 8px;
}
.src {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  text-align: left;
  cursor: pointer;
  min-width: 0;
}
.src:hover {
  border-color: var(--accent);
}
.snip {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.thinking {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 14px;
}
.ask {
  display: flex;
  gap: 8px;
}
.ask .btn {
  height: 38px;
}
</style>
