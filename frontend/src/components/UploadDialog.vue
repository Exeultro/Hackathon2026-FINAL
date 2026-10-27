<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api, ApiError } from '@/api'
import type { Course, Language } from '@/api'
import Icon from './Icon.vue'

const emit = defineEmits<{ close: []; done: [id: string] }>()

const tab = ref<'file' | 'url'>('file')
const file = ref<File | null>(null)
const url = ref('')
const title = ref('')
const courseId = ref('')
const language = ref<Language>('auto')
const courses = ref<Course[]>([])
const busy = ref(false)
const progress = ref(0)
const error = ref('')
const drag = ref(false)

const ACCEPT = ['video/mp4', 'video/webm']
const canSubmit = computed(() => !busy.value && (tab.value === 'file' ? !!file.value : url.value.trim().length > 0))

function pick(f: File | undefined) {
  error.value = ''
  if (!f) return
  if (!ACCEPT.includes(f.type) && !/\.(mp4|webm)$/i.test(f.name)) {
    error.value = 'Нужен файл MP4 или WebM'
    return
  }
  file.value = f
}

function onDrop(e: DragEvent) {
  drag.value = false
  pick(e.dataTransfer?.files[0])
}

async function submit() {
  if (!canSubmit.value) return
  busy.value = true
  error.value = ''
  progress.value = 0
  const input = { title: title.value.trim(), course_id: courseId.value, language: language.value }
  try {
    const lec =
      tab.value === 'file'
        ? await api.uploadLecture(file.value!, input, (p) => (progress.value = p))
        : await api.importLecture(url.value.trim(), input)
    emit('done', lec.id)
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Не получилось отправить, попробуйте ещё раз'
    busy.value = false
  }
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && !busy.value) emit('close')
}

onMounted(async () => {
  window.addEventListener('keydown', onKey)
  courses.value = await api.listCourses()
})
onUnmounted(() => window.removeEventListener('keydown', onKey))

const mb = (n: number) => (n / 1024 / 1024).toFixed(1) + ' МБ'
</script>

<template>
  <div class="veil" @mousedown.self="!busy && emit('close')">
    <form class="dialog" @submit.prevent="submit">
      <div class="head">
        <h2>Новая лекция</h2>
        <button type="button" class="btn ghost icon" :disabled="busy" @click="emit('close')"><Icon name="x" /></button>
      </div>

      <div class="tabs">
        <button type="button" :class="{ on: tab === 'file' }" :disabled="busy" @click="tab = 'file'">
          <Icon name="file" :size="16" /> Файл
        </button>
        <button type="button" :class="{ on: tab === 'url' }" :disabled="busy" @click="tab = 'url'">
          <Icon name="link" :size="16" /> Ссылка
        </button>
      </div>

      <label
        v-if="tab === 'file'"
        class="drop"
        :class="{ drag, has: file }"
        @dragover.prevent="drag = true"
        @dragleave="drag = false"
        @drop.prevent="onDrop"
      >
        <input type="file" accept="video/mp4,video/webm,.mp4,.webm" hidden :disabled="busy" @change="pick(($event.target as HTMLInputElement).files?.[0])" />
        <template v-if="file">
          <Icon name="file" :size="22" />
          <b>{{ file.name }}</b>
          <span class="muted">{{ mb(file.size) }} · нажмите, чтобы выбрать другой</span>
        </template>
        <template v-else>
          <Icon name="upload" :size="22" />
          <b>Перетащите видео сюда или нажмите</b>
          <span class="muted">MP4 или WebM, до 2 ГБ</span>
        </template>
      </label>

      <div v-else class="field">
        <input v-model="url" class="input" type="url" placeholder="https://www.youtube.com/watch?v=..." :disabled="busy" autofocus />
        <span class="hint">YouTube, Vimeo или прямая ссылка на MP4/WebM</span>
      </div>

      <div class="field">
        <span class="label">Название</span>
        <input v-model="title" class="input" :placeholder="tab === 'file' ? 'Как у файла' : 'Как на площадке'" :disabled="busy" />
      </div>

      <div class="row">
        <div class="field">
          <span class="label">Курс</span>
          <select v-model="courseId" class="input" :disabled="busy">
            <option value="">Без курса</option>
            <option v-for="c in courses" :key="c.id" :value="c.id">{{ c.title }}</option>
          </select>
        </div>
        <div class="field">
          <span class="label">Язык речи</span>
          <select v-model="language" class="input" :disabled="busy">
            <option value="auto">Определить</option>
            <option value="ru">Русский</option>
            <option value="en">English</option>
          </select>
        </div>
      </div>

      <div v-if="busy && tab === 'file'" class="up">
        <div class="bar"><i :style="{ width: progress * 100 + '%' }" /></div>
        <span class="muted">Загружаем… {{ Math.round(progress * 100) }}%</span>
      </div>
      <p v-if="error" class="error">{{ error }}</p>

      <div class="foot">
        <span class="muted small">Дальше сервер сам распознает речь, построит индекс и конспект</span>
        <button class="btn primary" :disabled="!canSubmit">
          <span v-if="busy" class="spin" />
          {{ tab === 'file' ? 'Загрузить' : 'Импортировать' }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.veil {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgb(10 12 18 / 45%);
}
.dialog {
  width: min(520px, 100%);
  max-height: calc(100vh - 32px);
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px;
  border-radius: 14px;
  background: var(--surface);
  box-shadow: 0 20px 50px rgb(0 0 0 / 25%);
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.head h2 {
  font-size: 18px;
}
.icon {
  padding: 0 8px;
}
.tabs {
  display: flex;
  gap: 4px;
  padding: 4px;
  border-radius: 10px;
  background: var(--surface-2);
}
.tabs button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}
.tabs button.on {
  background: var(--surface);
  color: var(--text);
  box-shadow: var(--shadow);
  font-weight: 600;
}
.drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 26px 16px;
  border: 1.5px dashed var(--border);
  border-radius: 12px;
  text-align: center;
  color: var(--muted);
  cursor: pointer;
  word-break: break-all;
}
.drop b {
  color: var(--text);
  font-weight: 600;
}
.drop:hover,
.drop.drag {
  border-color: var(--accent);
  background: var(--accent-soft);
}
.drop.has {
  border-style: solid;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex: 1;
  min-width: 0;
}
.label {
  font-size: 13px;
  font-weight: 600;
  color: var(--muted);
}
.hint {
  font-size: 12.5px;
  color: var(--faint);
}
.row {
  display: flex;
  gap: 12px;
}
.up {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}
.error {
  margin: 0;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--err-soft);
  color: var(--err);
  font-size: 14px;
}
.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.small {
  font-size: 12.5px;
}
</style>
