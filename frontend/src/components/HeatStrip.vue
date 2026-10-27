<script setup lang="ts">
import { computed } from 'vue'
import { timecode } from '@/format'

const props = defineProps<{
  hits: { start: number; end: number; score: number }[]
  duration: number
  active?: number
  time?: number
  compact?: boolean
}>()
const emit = defineEmits<{ seek: [t: number] }>()

const marks = computed(() => {
  const top = Math.max(...props.hits.map((h) => h.score), 0.01)
  const low = Math.min(...props.hits.map((h) => h.score))
  return props.hits.map((h) => ({
    ...h,
    left: (h.start / props.duration) * 100,
    width: Math.max(((h.end - h.start) / props.duration) * 100, 0.8),
    heat: top === low ? 1 : 0.3 + (0.7 * (h.score - low)) / (top - low),
  }))
})
</script>

<template>
  <div class="heat" :class="{ compact }">
    <button
      v-for="(m, i) in marks"
      :key="i"
      class="mark"
      :class="{ active: active === m.start }"
      :style="{ left: m.left + '%', width: m.width + '%', opacity: m.heat }"
      :title="`${timecode(m.start)} · релевантность ${Math.round(m.score * 100)}%`"
      @click.stop="emit('seek', m.start)"
    />
    <i v-if="time !== undefined" class="cursor" :style="{ left: (time / duration) * 100 + '%' }" />
  </div>
</template>

<style scoped>
.heat {
  position: relative;
  height: 14px;
  border-radius: 7px;
  background: var(--surface-2);
}
.heat.compact {
  height: 6px;
  border-radius: 3px;
}
.mark {
  position: absolute;
  top: 0;
  bottom: 0;
  padding: 0;
  border: 0;
  border-radius: 3px;
  background: linear-gradient(180deg, #f59e0b, #ef4444);
  cursor: pointer;
}
.mark:hover {
  outline: 2px solid var(--text);
  outline-offset: 1px;
  opacity: 1 !important;
}
.compact .mark {
  background: var(--faint);
  cursor: default;
}
.compact .mark.active {
  background: linear-gradient(180deg, #f59e0b, #ef4444);
  opacity: 1 !important;
}
.cursor {
  position: absolute;
  top: -3px;
  bottom: -3px;
  width: 2px;
  margin-left: -1px;
  border-radius: 1px;
  background: var(--accent);
  pointer-events: none;
}
</style>
