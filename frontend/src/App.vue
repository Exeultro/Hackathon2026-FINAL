<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, isMock, session } from '@/api'
import type { SystemInfo } from '@/api'
import { authDialog, checkSession, logout, requireLogin } from '@/auth'
import AuthDialog from '@/components/AuthDialog.vue'
import Icon from '@/components/Icon.vue'
import UploadDialog from '@/components/UploadDialog.vue'
import { loadPersonal, local } from '@/bookmarks'
import { ui } from '@/ui'

const route = useRoute()
const router = useRouter()
const q = ref('')
const uploadOpen = ref(false)
const sys = ref<SystemInfo | null>(null)

watch(
  () => route.query.q,
  (v) => {
    if (route.path === '/search') q.value = String(v ?? '')
  },
  { immediate: true },
)

function submit() {
  const text = q.value.trim()
  if (text.length >= 2) router.push({ path: '/search', query: { ...(route.path === '/search' ? route.query : {}), q: text } })
}

function openUpload() {
  if (requireLogin('Чтобы добавлять лекции, войдите или зарегистрируйтесь', openUpload)) uploadOpen.value = true
}

async function signOut() {
  if (!confirm('Выйти из аккаунта?')) return
  await logout()
}

function reloadPersonal() {
  loadPersonal().catch(() => {
    /* personal data stays empty if the server is down */
  })
}

// Bookmarks and progress belong to the account when signed in, to the browser otherwise
watch(() => session.user?.id, reloadPersonal)

function uploaded(id: string) {
  uploadOpen.value = false
  router.push(`/lectures/${id}`)
}

// "/" focuses the search box, like on most sites
function onKey(e: KeyboardEvent) {
  if (e.key !== '/' || e.ctrlKey || e.metaKey || (e.target as HTMLElement).closest('input, textarea, select, [contenteditable]')) return
  const box = [...document.querySelectorAll<HTMLInputElement>('input[data-search]')].find((el) => el.offsetParent)
  if (!box) return
  e.preventDefault()
  box.focus()
}

onUnmounted(() => window.removeEventListener('keydown', onKey))
onMounted(async () => {
  window.addEventListener('keydown', onKey)
  await checkSession()
  reloadPersonal()
  try {
    sys.value = await api.system()
  } catch {
    sys.value = null
  }
})
</script>

<template>
  <header class="top">
    <RouterLink to="/" class="logo">
      <span class="mark"><Icon name="play" :size="14" /></span>
      Лекторий
    </RouterLink>
    <form v-if="route.path !== '/'" class="search" @submit.prevent="submit">
      <Icon name="search" class="lens" />
      <input v-model="q" class="input" type="search" placeholder="Поиск по всем лекциям" data-search />
    </form>
    <span class="grow" />
    <span
      v-if="sys?.offline"
      class="offline"
      :title="`Распознавание: ${sys.models.asr}\nПоиск: ${sys.models.embeddings}\nКонспект и ответы: ${sys.models.llm}`"
    >
      <Icon name="wifi" :size="15" /> Локально
    </span>
    <span v-if="isMock" class="mock" title="Бэкенд не подключён, данные в браузере">мок</span>
    <RouterLink to="/bookmarks" class="nav" title="Закладки">
      <Icon name="bookmark" :size="17" /><span class="label">Закладки</span>
      <span v-if="local.bookmarks.length" class="count">{{ local.bookmarks.length }}</span>
    </RouterLink>
    <RouterLink to="/settings" class="nav" title="Настройки"><Icon name="gear" :size="17" /></RouterLink>
    <button v-if="session.user" class="nav user" :title="`${session.user.email}\nНажмите, чтобы выйти`" @click="signOut">
      <Icon name="user" :size="17" /><span class="label">{{ session.user.name }}</span><Icon name="logout" :size="15" class="out" />
    </button>
    <button v-else class="nav user" @click="requireLogin('')"><Icon name="user" :size="17" /><span class="label">Войти</span></button>
    <button class="btn primary" @click="openUpload"><Icon name="upload" :size="16" /> Добавить лекцию</button>
  </header>
  <main>
    <RouterView />
  </main>
  <UploadDialog v-if="uploadOpen" @close="uploadOpen = false" @done="uploaded" />
  <AuthDialog v-if="authDialog.open" />
  <div v-if="ui.tip" class="tip" :style="{ left: ui.tip.x + 'px', top: ui.tip.y + 'px' }">{{ ui.tip.text }}</div>
  <Transition name="toast"><div v-if="ui.toast" class="toast">{{ ui.toast }}</div></Transition>
</template>

<style scoped>
.top {
  position: sticky;
  top: 0;
  z-index: 20;
  height: var(--header);
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 20px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}
.logo {
  display: flex;
  align-items: center;
  gap: 9px;
  font-weight: 700;
  font-size: 17px;
  flex-shrink: 0;
}
.mark {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--accent);
  color: var(--on-accent);
}
.search {
  position: relative;
  flex: 1;
  max-width: 640px;
}
.search .input {
  padding-left: 38px;
  background: var(--surface-2);
  border-color: transparent;
}
.search .input:focus {
  background: var(--surface);
}
.lens {
  position: absolute;
  left: 12px;
  top: 10px;
  color: var(--faint);
}
.offline {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
  color: var(--ok);
  cursor: help;
  white-space: nowrap;
}
.mock {
  font-size: 12px;
  padding: 2px 7px;
  border-radius: 6px;
  background: var(--surface-2);
  color: var(--muted);
}
.grow {
  flex: 1 0 0;
}
.nav {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 10px;
  border-radius: 8px;
  color: var(--muted);
  white-space: nowrap;
}
.nav:hover,
.nav.router-link-active {
  background: var(--surface-2);
  color: var(--text);
}
.user {
  border: 0;
  background: none;
  font: inherit;
  cursor: pointer;
}
.user .label {
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.out {
  color: var(--faint);
}
.count {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--accent);
  color: var(--on-accent);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
}
.tip {
  position: fixed;
  z-index: 60;
  max-width: 280px;
  padding: 7px 10px;
  border-radius: 8px;
  background: var(--text);
  color: var(--bg);
  font-size: 13px;
  line-height: 1.4;
  transform: translate(-50%, calc(-100% - 8px));
  pointer-events: none;
}
.toast {
  position: fixed;
  z-index: 60;
  left: 50%;
  bottom: 24px;
  transform: translateX(-50%);
  padding: 10px 16px;
  border-radius: 10px;
  background: var(--text);
  color: var(--bg);
  font-size: 14px;
  box-shadow: 0 8px 24px rgb(0 0 0 / 20%);
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translate(-50%, 8px);
}
@media (max-width: 760px) {
  .top {
    gap: 10px;
    padding: 0 12px;
  }
  .logo {
    font-size: 0;
    gap: 0;
  }
  .offline,
  .mock,
  .nav .label,
  .search + .grow {
    display: none;
  }
  .search .input {
    padding-right: 6px;
  }
  .nav {
    padding: 0 6px;
  }
  .btn.primary {
    font-size: 0;
    gap: 0;
    padding: 0 10px;
  }
}
</style>
