<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { api, isMock, session } from '@/api'
import type { SystemInfo } from '@/api'
import { resetMock } from '@/api/mockApi'
import { setUser } from '@/api/session'
import { actionFailed, requireLogin } from '@/auth'
import { clearPersonal } from '@/bookmarks'
import { settings } from '@/settings'
import { toast } from '@/ui'

const sys = ref<SystemInfo | null>(null)
const sysError = ref(false)

const THEMES = [
  ['system', 'Как в системе'],
  ['light', 'Светлая'],
  ['dark', 'Тёмная'],
] as const
const SIZES = [
  ['s', 'Мелкий'],
  ['m', 'Средний'],
  ['l', 'Крупный'],
] as const
const SPEEDS = [0.75, 1, 1.25, 1.5, 1.75, 2]
const LIMITS = [10, 20, 50]
const KEYS = [
  ['/', 'Перейти к поиску'],
  ['B', 'Закладка на текущий момент'],
  ['Пробел или K', 'Пауза / воспроизведение (фокус на плеере)'],
  ['← →', 'Назад / вперёд на 5 секунд'],
  ['C', 'Включить / выключить субтитры'],
  ['F', 'Полный экран'],
]

async function clearLocal() {
  if (!confirm('Удалить все закладки, заметки и историю просмотра?')) return
  try {
    await clearPersonal()
    toast('Закладки и история удалены')
  } catch {
    toast('Не удалось удалить, проверьте связь с сервером')
  }
}

function resetDemo() {
  if (!confirm('Вернуть демо-лекции в исходное состояние? Загруженные вами лекции пропадут.')) return
  resetMock()
  location.reload()
}

const tgBusy = ref(false)
const tgWaiting = ref(false)
const tgUrl = ref('')
let tgTimer = 0
let tgUntil = 0

async function linkTelegram() {
  // Opened before the await, otherwise the popup blocker eats it
  const win = isMock ? null : window.open('', '_blank')
  tgBusy.value = true
  try {
    const c = await api.telegramCode()
    tgUrl.value = c.bot_url
    if (win) win.location.href = c.bot_url
    tgUntil = Date.now() + c.expires_in * 1000
    tgWaiting.value = true
    pollTelegram()
  } catch (e) {
    win?.close()
    actionFailed(e, 'Чтобы привязать Telegram, войдите')
  } finally {
    tgBusy.value = false
  }
}

async function pollTelegram() {
  clearTimeout(tgTimer)
  try {
    const u = await api.me()
    setUser(u)
    if (u.telegram) {
      tgWaiting.value = false
      toast('Telegram привязан')
      return
    }
  } catch {
    /* try again on the next tick */
  }
  if (tgWaiting.value && Date.now() < tgUntil) tgTimer = window.setTimeout(pollTelegram, 2500)
  else tgWaiting.value = false
}

async function unlinkTelegram() {
  if (!confirm('Отвязать Telegram? Закладки останутся в аккаунте, в боте их больше не будет.')) return
  try {
    await api.unlinkTelegram()
    setUser({ ...session.user!, telegram: null })
  } catch (e) {
    actionFailed(e, 'Чтобы отвязать Telegram, войдите')
  }
}

onUnmounted(() => clearTimeout(tgTimer))

onMounted(async () => {
  try {
    sys.value = await api.system()
  } catch {
    sysError.value = true
  }
})
</script>

<template>
  <div class="page">
    <h1>Настройки</h1>

    <section>
      <h2>Внешний вид</h2>
      <div class="row">
        <span class="label">Тема</span>
        <div class="seg">
          <button v-for="[v, t] in THEMES" :key="v" :class="{ on: settings.theme === v }" @click="settings.theme = v">{{ t }}</button>
        </div>
      </div>
      <div class="row">
        <span class="label">Размер субтитров</span>
        <div class="seg">
          <button v-for="[v, t] in SIZES" :key="v" :class="{ on: settings.captionSize === v }" @click="settings.captionSize = v">{{ t }}</button>
        </div>
      </div>
      <div class="preview"><span>Квадрат гипотенузы равен сумме квадратов катетов.</span></div>
    </section>

    <section>
      <h2>Просмотр</h2>
      <div class="row">
        <span class="label">Скорость по умолчанию</span>
        <select v-model.number="settings.speed" class="input narrow">
          <option v-for="s in SPEEDS" :key="s" :value="s">{{ s === 1 ? 'Обычная' : s + '×' }}</option>
        </select>
      </div>
      <label class="row">
        <span class="label">
          Автопрокрутка текста
          <small>Текст лекции сам прокручивается за видео</small>
        </span>
        <input v-model="settings.autoscroll" type="checkbox" class="switch" />
      </label>
      <label class="row">
        <span class="label">
          Подсветка терминов
          <small>Ключевые понятия подчёркнуты в тексте и конспекте</small>
        </span>
        <input v-model="settings.terms" type="checkbox" class="switch" />
      </label>
    </section>

    <section>
      <h2>Поиск</h2>
      <div class="row">
        <span class="label">Сколько фрагментов показывать</span>
        <div class="seg">
          <button v-for="n in LIMITS" :key="n" :class="{ on: settings.searchLimit === n }" @click="settings.searchLimit = n">{{ n }}</button>
        </div>
      </div>
    </section>

    <section>
      <h2>Модели</h2>
      <p v-if="sysError" class="muted">Сервер не ответил.</p>
      <template v-else-if="sys">
        <p class="offline" :class="{ on: sys.offline }">
          {{ sys.offline ? 'Всё работает локально, интернет не нужен' : 'Часть моделей работает через внешние API' }}
        </p>
        <div class="row"><span class="label">Распознавание речи</span><code>{{ sys.models.asr }}</code></div>
        <div class="row"><span class="label">Семантический поиск</span><code>{{ sys.models.embeddings }}</code></div>
        <div class="row"><span class="label">Конспект и ответы</span><code>{{ sys.models.llm }}</code></div>
      </template>
      <span v-else class="spin" />
    </section>

    <section>
      <h2>Горячие клавиши</h2>
      <div v-for="[k, t] in KEYS" :key="k" class="row">
        <span class="label">{{ t }}</span>
        <span class="keys"><kbd v-for="part in k.split(' или ')" :key="part">{{ part }}</kbd></span>
      </div>
    </section>

    <section>
      <h2>Telegram</h2>
      <div v-if="!session.user" class="row">
        <span class="label">
          Бот для поиска по лекциям
          <small>Войдите, чтобы привязать его: закладки в боте и на сайте станут общими</small>
        </span>
        <button class="btn" @click="requireLogin('Чтобы привязать Telegram, войдите')">Войти</button>
      </div>
      <div v-else-if="session.user.telegram" class="row">
        <span class="label">
          Привязан {{ session.user.telegram.username ? '@' + session.user.telegram.username : 'аккаунт ' + session.user.telegram.id }}
          <small>Закладки в боте и на сайте общие</small>
        </span>
        <button class="btn" @click="unlinkTelegram">Отвязать</button>
      </div>
      <div v-else class="row">
        <span class="label">
          Бот для поиска по лекциям
          <small v-if="tgWaiting">
            Нажмите «Запустить» в Telegram, ждём бота…
            <a v-if="tgUrl && !isMock" :href="tgUrl" target="_blank" rel="noopener">Открыть ещё раз</a>
          </small>
          <small v-else>Закладки в боте и на сайте станут общими</small>
        </span>
        <button class="btn primary" :disabled="tgBusy || tgWaiting" @click="linkTelegram">
          <span v-if="tgBusy || tgWaiting" class="spin" />
          Привязать Telegram
        </button>
      </div>
    </section>

    <section>
      <h2>Данные</h2>
      <p class="muted small">
        Закладки, заметки и история просмотра {{ session.user ? 'привязаны к аккаунту' : 'привязаны к этому браузеру' }}. Настройки интерфейса
        хранятся только в браузере.
      </p>
      <div class="actions">
        <button class="btn" @click="clearLocal">Удалить закладки и историю</button>
        <button v-if="isMock" class="btn" @click="resetDemo">Сбросить демо-данные</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page {
  max-width: 720px;
  margin: 0 auto;
  padding: 24px 20px 48px;
}
h1 {
  font-size: 24px;
  margin-bottom: 18px;
}
section {
  margin-bottom: 16px;
  padding: 16px 18px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}
h2 {
  font-size: 15px;
  margin-bottom: 8px;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 44px;
  padding: 6px 0;
  border-top: 1px solid var(--border);
}
h2 + .row,
.offline + .row {
  border-top: 0;
}
.label {
  display: flex;
  flex-direction: column;
}
.label small {
  color: var(--muted);
  font-size: 12.5px;
}
.seg {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 3px;
  border-radius: 9px;
  background: var(--surface-2);
}
.seg button {
  height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}
.seg button.on {
  background: var(--surface);
  color: var(--text);
  font-weight: 600;
  box-shadow: var(--shadow);
}
.narrow {
  width: auto;
  height: 34px;
}
.preview {
  display: grid;
  place-items: end center;
  height: 90px;
  margin-top: 6px;
  padding-bottom: 12px;
  border-radius: 10px;
  background: #1b2230;
}
.preview span {
  padding: 2px 8px;
  background: rgb(0 0 0 / 75%);
  color: #fff;
  font-size: var(--cap-size);
}
.switch {
  appearance: none;
  position: relative;
  flex-shrink: 0;
  width: 40px;
  height: 22px;
  margin: 0;
  border-radius: 11px;
  background: var(--border);
  cursor: pointer;
  transition: background 0.15s;
}
.switch::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.15s;
}
.switch:checked {
  background: var(--accent);
}
.switch:checked::after {
  transform: translateX(18px);
}
label.row {
  cursor: pointer;
}
code {
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--surface-2);
  font-size: 13px;
  text-align: right;
}
.offline {
  margin: 0 0 4px;
  color: var(--warn);
  font-weight: 600;
}
.offline.on {
  color: var(--ok);
}
.keys {
  display: flex;
  gap: 4px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.small {
  font-size: 13.5px;
  margin: 0 0 10px;
}
</style>
