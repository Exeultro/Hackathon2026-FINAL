import type { Lecture, OutlineItem, Summary } from '@/api'
import { duration, escape, markdown, timecode } from '@/format'

const link = (l: Lecture, t: number) => `${location.origin}/lectures/${l.id}?t=${Math.floor(t)}`

function outlineMd(l: Lecture, items: OutlineItem[], depth = 0): string {
  return items
    .map((it) => {
      const line = `${'  '.repeat(depth)}- [${timecode(it.start)}](${link(l, it.start)}) ${it.title}${it.summary ? ` - ${it.summary}` : ''}`
      return it.children.length ? line + '\n' + outlineMd(l, it.children, depth + 1) : line
    })
    .join('\n')
}

export function summaryMarkdown(l: Lecture, s: Summary): string {
  const terms = s.key_terms
    .map((t) => `- **${t.term}**${t.definition ? ` - ${t.definition}` : ''} (${t.occurrences.map(timecode).join(', ')})`)
    .join('\n')
  return [
    `# ${l.title}`,
    duration(l.duration) ? `_Длительность: ${duration(l.duration)}_` : '',
    '## Конспект',
    s.summary,
    '## Оглавление',
    outlineMd(l, s.outline),
    s.key_terms.length ? '## Ключевые термины\n\n' + terms : '',
  ]
    .filter(Boolean)
    .join('\n\n')
}

export function downloadMarkdown(l: Lecture, s: Summary) {
  const blob = new Blob([summaryMarkdown(l, s)], { type: 'text/markdown;charset=utf-8' })
  const a = Object.assign(document.createElement('a'), {
    href: URL.createObjectURL(blob),
    download: `${l.title.replace(/[\\/:*?"<>|]+/g, ' ').trim()}.md`,
  })
  a.click()
  setTimeout(() => URL.revokeObjectURL(a.href), 1000)
}

function outlineHtml(items: OutlineItem[]): string {
  return (
    '<ul>' +
    items
      .map(
        (it) =>
          `<li><b class="tc">${timecode(it.start)}</b> ${escape(it.title)}${it.summary ? ` <span class="m">- ${escape(it.summary)}</span>` : ''}${it.children.length ? outlineHtml(it.children) : ''}</li>`,
      )
      .join('') +
    '</ul>'
  )
}

export function printSummary(l: Lecture, s: Summary) {
  const w = window.open('', '_blank')
  if (!w) return false
  const terms = s.key_terms
    .map((t) => `<li><b>${escape(t.term)}</b>${t.definition ? ` - ${escape(t.definition)}` : ''} <span class="m">(${t.occurrences.map(timecode).join(', ')})</span></li>`)
    .join('')
  w.document.write(`<!doctype html><html lang="ru"><head><meta charset="utf-8"><title>${escape(l.title)}</title><style>
    body{font:15px/1.55 system-ui,'Noto Sans',sans-serif;max-width:720px;margin:32px auto;padding:0 20px;color:#111}
    h1{font-size:22px}h2{font-size:17px;margin-top:26px;border-bottom:1px solid #ddd;padding-bottom:4px}
    .tc{font-family:ui-monospace,monospace;font-size:13px;color:#4338ca}.m{color:#666}li{margin:3px 0}
  </style></head><body>
    <h1>${escape(l.title)}</h1>${duration(l.duration) ? `<p class="m">Длительность: ${duration(l.duration)}</p>` : ''}
    <h2>Конспект</h2>${markdown(s.summary)}
    <h2>Оглавление</h2>${outlineHtml(s.outline)}
    ${terms ? `<h2>Ключевые термины</h2><ul>${terms}</ul>` : ''}
  </body></html>`)
  w.document.close()
  w.focus()
  w.print()
  return true
}
