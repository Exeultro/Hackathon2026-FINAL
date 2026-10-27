<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ApiError } from '@/api'
import { authDialog, closeAuth, login, register } from '@/auth'
import Icon from './Icon.vue'

const ERRORS: Record<string, string> = {
  invalid_credentials: 'Неверный email или пароль',
  email_taken: 'Этот email уже зарегистрирован',
}

const tab = ref<'login' | 'register'>('login')
const name = ref('')
const email = ref('')
const password = ref('')
const busy = ref(false)
const error = ref('')
const fields = ref<Record<string, string>>({})

const canSubmit = computed(
  () => !busy.value && email.value.trim() !== '' && password.value !== '' && (tab.value === 'login' || name.value.trim() !== ''),
)

function switchTab(t: 'login' | 'register') {
  tab.value = t
  error.value = ''
  fields.value = {}
}

async function submit() {
  if (!canSubmit.value) return
  busy.value = true
  error.value = ''
  fields.value = {}
  try {
    if (tab.value === 'login') await login(email.value.trim(), password.value)
    else await register(name.value.trim(), email.value.trim(), password.value)
    closeAuth(true)
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.code === 'validation' && e.details) {
        fields.value = Object.fromEntries(Object.entries(e.details).map(([k, v]) => [k, String(v)]))
        if (!['name', 'email', 'password'].some((k) => k in fields.value)) error.value = 'Проверьте поля'
      } else error.value = ERRORS[e.code] ?? (e.status >= 500 ? 'Сервер недоступен, попробуйте ещё раз' : e.message)
    } else error.value = 'Сервер недоступен, попробуйте ещё раз'
    busy.value = false
  }
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && !busy.value) closeAuth(false)
}

onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="veil" @mousedown.self="!busy && closeAuth(false)">
    <form class="dialog" novalidate @submit.prevent="submit">
      <div class="head">
        <h2>{{ tab === 'login' ? 'Вход' : 'Регистрация' }}</h2>
        <button type="button" class="btn ghost icon" :disabled="busy" @click="closeAuth(false)"><Icon name="x" /></button>
      </div>
      <p v-if="authDialog.reason" class="reason">{{ authDialog.reason }}</p>

      <div class="tabs">
        <button type="button" :class="{ on: tab === 'login' }" :disabled="busy" @click="switchTab('login')">Вход</button>
        <button type="button" :class="{ on: tab === 'register' }" :disabled="busy" @click="switchTab('register')">Регистрация</button>
      </div>

      <label v-if="tab === 'register'" class="field">
        <span class="label">Имя</span>
        <input v-model="name" class="input" autocomplete="name" maxlength="64" :disabled="busy" autofocus />
        <span v-if="fields.name" class="bad">{{ fields.name }}</span>
      </label>
      <label class="field">
        <span class="label">Email</span>
        <input v-model="email" class="input" type="email" autocomplete="email" maxlength="254" :disabled="busy" :autofocus="tab === 'login'" />
        <span v-if="fields.email" class="bad">{{ fields.email }}</span>
      </label>
      <label class="field">
        <span class="label">Пароль</span>
        <input
          v-model="password"
          class="input"
          type="password"
          :autocomplete="tab === 'login' ? 'current-password' : 'new-password'"
          maxlength="64"
          :disabled="busy"
        />
        <span v-if="fields.password" class="bad">{{ fields.password }}</span>
        <span v-else-if="tab === 'register'" class="hint">Не короче 8 символов</span>
      </label>

      <p v-if="error" class="error">{{ error }}</p>

      <button class="btn primary wide" :disabled="!canSubmit">
        <span v-if="busy" class="spin" />
        {{ tab === 'login' ? 'Войти' : 'Зарегистрироваться' }}
      </button>
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
  width: min(400px, 100%);
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
.reason {
  margin: -6px 0 0;
  color: var(--muted);
  font-size: 14px;
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
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
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
.bad {
  font-size: 12.5px;
  color: var(--err);
}
.error {
  margin: 0;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--err-soft);
  color: var(--err);
  font-size: 14px;
}
.wide {
  justify-content: center;
  height: 40px;
}
</style>
