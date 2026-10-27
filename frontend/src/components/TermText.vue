<script setup lang="ts">
import { computed } from 'vue'
import type { KeyTerm } from '@/api'
import type { TermMatcher } from '@/terms'
import { hideTip, showTip } from '@/ui'

const props = defineProps<{ text: string; matcher?: TermMatcher | null }>()
const emit = defineEmits<{ term: [term: KeyTerm] }>()

const parts = computed(() => {
  const out: { s: string; term?: KeyTerm }[] = []
  let at = 0
  for (const m of props.matcher?.(props.text) ?? []) {
    if (m.from > at) out.push({ s: props.text.slice(at, m.from) })
    out.push({ s: props.text.slice(m.from, m.to), term: m.term })
    at = m.to
  }
  out.push({ s: props.text.slice(at) })
  return out
})

function tip(e: MouseEvent, t: KeyTerm) {
  showTip(e.currentTarget as Element, t.definition ? `${t.term}: ${t.definition}` : t.term)
}
</script>

<template>
  <span><template v-for="(p, i) in parts" :key="i"><span
        v-if="p.term"
        class="term"
        @mouseenter="tip($event, p.term)"
        @mouseleave="hideTip"
        @click.stop="hideTip(), emit('term', p.term)"
      >{{ p.s }}</span><template v-else>{{ p.s }}</template></template></span>
</template>
