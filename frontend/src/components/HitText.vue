<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ text: string; highlights: number[][] }>()

const parts = computed(() => {
  const out: { s: string; mark: boolean }[] = []
  let at = 0
  for (const [from, to] of [...props.highlights].sort((a, b) => a[0] - b[0])) {
    if (from < at) continue
    if (from > at) out.push({ s: props.text.slice(at, from), mark: false })
    out.push({ s: props.text.slice(from, to), mark: true })
    at = to
  }
  out.push({ s: props.text.slice(at), mark: false })
  return out
})
</script>

<template>
  <span><template v-for="(p, i) in parts" :key="i"><mark v-if="p.mark">{{ p.s }}</mark><template v-else>{{ p.s }}</template></template></span>
</template>
