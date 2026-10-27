import type { TermMatcher } from './terms'

export function timecode(sec: number): string {
  const s = Math.floor(sec)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const ss = String(s % 60).padStart(2, '0')
  return h ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${m}:${ss}`
}

export function plural(n: number, forms: [string, string, string]): string {
  const m10 = n % 10
  const m100 = n % 100
  const f = m10 === 1 && m100 !== 11 ? 0 : m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14) ? 1 : 2
  return `${n} ${forms[f]}`
}

export function duration(sec: number | null | undefined): string {
  if (!sec) return ''
  const m = Math.round(sec / 60)
  return m < 60 ? `${m} мин` : `${Math.floor(m / 60)} ч ${m % 60} мин`
}

const STAGES: Record<string, string> = {
  queued: 'В очереди',
  downloading: 'Скачиваем видео',
  extracting_audio: 'Извлекаем звук',
  transcribing: 'Распознаём речь',
  indexing: 'Строим поисковый индекс',
  ready: 'Готово',
  failed: 'Ошибка',
}

export const stageName = (s: string) => STAGES[s] ?? s

export function escape(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)
}

function withTerms(s: string, mark?: TermMatcher): string {
  if (!mark) return escape(s)
  let out = ''
  let at = 0
  for (const m of mark(s)) {
    out += escape(s.slice(at, m.from)) + `<span class="term" data-term="${escape(m.term.term)}">${escape(s.slice(m.from, m.to))}</span>`
    at = m.to
  }
  return out + escape(s.slice(at))
}

function inline(s: string, mark?: TermMatcher): string {
  return withTerms(s, mark)
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    .replace(/\[(\d+)\]/g, '<button class="cite" data-cite="$1">$1</button>')
}

// Minimal Markdown: headings, paragraphs, "-"/"1." lists (one nesting level), bold, italic, [n] citations, optional key-term spans
export function markdown(md: string, mark?: TermMatcher): string {
  const out: string[] = []
  let para: string[] = []
  let list: 'ul' | 'ol' | null = null
  const flushPara = () => {
    if (para.length) out.push(`<p>${para.join('<br>')}</p>`)
    para = []
  }
  const closeList = () => {
    if (list) out.push(`</${list}>`)
    list = null
  }
  for (const raw of md.split('\n')) {
    const line = raw.trimEnd()
    const h = /^\s*#{1,6}\s+(.*)$/.exec(line)
    const li = /^(\s*)(?:[-*+]|(\d+)[.)])\s+(.*)$/.exec(line)
    if (!line.trim()) {
      flushPara()
      closeList()
    } else if (h) {
      flushPara()
      closeList()
      out.push(`<h4>${inline(h[1]!, mark)}</h4>`)
    } else if (li) {
      flushPara()
      const kind = li[2] && li[1]!.length < 2 ? 'ol' : 'ul'
      const sub = li[1]!.length >= 2
      if (list !== kind) {
        closeList()
        out.push(kind === 'ol' ? `<ol start="${li[2]}">` : '<ul>')
        list = kind
      }
      out.push(`<li${sub ? ' class="sub"' : ''}>${inline(li[3]!, mark)}</li>`)
    } else {
      closeList()
      para.push(inline(line.trim(), mark))
    }
  }
  flushPara()
  closeList()
  return out.join('')
}
