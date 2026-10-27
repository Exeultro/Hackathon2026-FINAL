import { reactive, watch } from 'vue'

export interface Settings {
  theme: 'system' | 'light' | 'dark'
  captionSize: 's' | 'm' | 'l'
  speed: number
  autoscroll: boolean
  terms: boolean
  searchLimit: number
}

const KEY = 'lectures.settings.v1'
const DEFAULTS: Settings = { theme: 'system', captionSize: 'm', speed: 1, autoscroll: true, terms: true, searchLimit: 20 }

function load(): Settings {
  try {
    return { ...DEFAULTS, ...JSON.parse(localStorage.getItem(KEY) ?? '{}') }
  } catch {
    return { ...DEFAULTS }
  }
}

export const settings = reactive(load())

const CAPTION_PX = { s: '15px', m: '19px', l: '24px' }

function apply() {
  const root = document.documentElement
  if (settings.theme === 'system') delete root.dataset.theme
  else root.dataset.theme = settings.theme
  root.style.setProperty('--cap-size', CAPTION_PX[settings.captionSize])
}

watch(
  settings,
  () => {
    apply()
    try {
      localStorage.setItem(KEY, JSON.stringify(settings))
    } catch {
      /* storage unavailable: settings live for this session only */
    }
  },
  { deep: true },
)
apply()
