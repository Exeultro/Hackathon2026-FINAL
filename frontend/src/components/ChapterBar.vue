<script setup lang="ts">
import { computed } from 'vue'
import type { OutlineItem } from '@/api'
import { timecode } from '@/format'

const props = defineProps<{ chapters: OutlineItem[]; duration: number; time: number; pins?: number[] }>()
const emit = defineEmits<{ seek: [t: number] }>()

const items = computed(() =>
  props.chapters.map((c) => ({
    ...c,
    share: (c.end - c.start) / props.duration,
    now: props.time >= c.start && props.time < c.end,
    fill: Math.min(1, Math.max(0, (props.time - c.start) / (c.end - c.start))),
  })),
)
</script>

<template>
  <div class="chapters">
    <button
      v-for="p in pins"
      :key="p"
      class="pin"
      :style="{ left: (p / duration) * 100 + '%' }"
      :title="`Закладка ${timecode(p)}`"
      @click="emit('seek', p)"
    />
    <div class="strip">
      <button
        v-for="(c, i) in items"
        :key="i"
        :class="{ now: c.now }"
        :style="{ flexGrow: c.share }"
        :title="`${timecode(c.start)} ${c.title}`"
        @click="emit('seek', c.start)"
      >
        <i :style="{ width: c.fill * 100 + '%' }" />
        <span>{{ c.title }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.chapters {
  position: relative;
  margin-top: 10px;
}
.pin {
  position: absolute;
  z-index: 1;
  top: -5px;
  width: 10px;
  height: 10px;
  margin-left: -5px;
  padding: 0;
  border: 2px solid var(--surface);
  border-radius: 50% 50% 50% 0;
  transform: rotate(-45deg);
  background: var(--accent);
  cursor: pointer;
}
.strip {
  display: flex;
  gap: 3px;
}
.strip button {
  position: relative;
  flex-basis: 0;
  min-width: 0;
  height: 30px;
  padding: 0 8px;
  border: 0;
  border-radius: 6px;
  background: var(--surface-2);
  color: var(--muted);
  font-size: 12.5px;
  text-align: left;
  overflow: hidden;
  cursor: pointer;
}
.strip button:hover {
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--accent);
}
.strip i {
  position: absolute;
  inset: 0 auto 0 0;
  background: var(--accent-soft);
}
.strip button.now {
  color: var(--accent-ink);
  font-weight: 600;
}
.strip span {
  position: relative;
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
