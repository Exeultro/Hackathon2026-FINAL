import type { KeyTerm } from '@/api'

export interface TermMatch {
  from: number
  to: number
  term: KeyTerm
}

export type TermMatcher = (text: string) => TermMatch[]

const WORD = /[\p{L}\d]+/gu

function tokens(text: string) {
  return [...text.toLowerCase().matchAll(WORD)].map((m) => ({ w: m[0], from: m.index!, to: m.index! + m[0].length }))
}

// Crude Russian stem: drop the ending so "катетами" matches "катет"
function stem(w: string): string {
  return w.length <= 4 ? w : w.slice(0, w.length - (w.length > 6 ? 2 : 1))
}

function find(toks: ReturnType<typeof tokens>, i: number, stems: string[]): boolean {
  return stems.every((s, j) => toks[i + j]?.w.startsWith(s))
}

// Multi-word terms match as a phrase; if the phrase never occurs in the lecture, fall back to its first word
export function termMatcher(terms: KeyTerm[], corpus: string[]): TermMatcher {
  const all = corpus.map(tokens)
  const patterns = terms
    .map((term) => {
      const stems = tokens(term.term).map((t) => stem(t.w))
      const phrase = all.some((toks) => toks.some((_, i) => find(toks, i, stems)))
      return { term, stems: phrase ? stems : stems.slice(0, 1) }
    })
    .filter((p) => p.stems.length)
    .sort((a, b) => b.stems.length - a.stems.length)

  return (text) => {
    const toks = tokens(text)
    const out: TermMatch[] = []
    for (let i = 0; i < toks.length; i++) {
      const p = patterns.find((x) => find(toks, i, x.stems))
      if (!p) continue
      out.push({ from: toks[i].from, to: toks[i + p.stems.length - 1].to, term: p.term })
      i += p.stems.length - 1
    }
    return out
  }
}
