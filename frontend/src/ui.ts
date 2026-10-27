import { reactive } from 'vue'

export const ui = reactive({
  tip: null as null | { text: string; x: number; y: number },
  toast: '',
})

export function showTip(el: Element, text: string) {
  const r = el.getBoundingClientRect()
  const x = Math.min(Math.max(r.left + r.width / 2, 150), window.innerWidth - 150)
  ui.tip = { text, x, y: r.top }
}

export function hideTip() {
  ui.tip = null
}

let timer = 0
export function toast(text: string) {
  ui.toast = text
  clearTimeout(timer)
  timer = window.setTimeout(() => (ui.toast = ''), 2600)
}

export async function copy(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    const ta = Object.assign(document.createElement('textarea'), { value: text })
    document.body.append(ta)
    ta.select()
    const ok = document.execCommand('copy')
    ta.remove()
    return ok
  }
}
