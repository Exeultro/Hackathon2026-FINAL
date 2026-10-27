<script setup lang="ts">
import type { OutlineItem } from '@/api'
import { timecode } from '@/format'

defineProps<{ items: OutlineItem[]; time: number; depth?: number }>()
const emit = defineEmits<{ seek: [t: number] }>()
</script>

<template>
  <ol class="tree" :class="{ sub: depth }">
    <li v-for="(it, i) in items" :key="i">
      <button class="row" :class="{ now: time >= it.start && time < it.end }" @click="emit('seek', it.start)">
        <span class="tc">{{ timecode(it.start) }}</span>
        <span class="t">
          <b>{{ it.title }}</b>
          <span v-if="it.summary" class="muted">{{ it.summary }}</span>
        </span>
      </button>
      <OutlineTree v-if="it.children.length" :items="it.children" :time="time" :depth="(depth ?? 0) + 1" @seek="emit('seek', $event)" />
    </li>
  </ol>
</template>

<style scoped>
.tree {
  list-style: none;
  margin: 0;
  padding: 0;
}
.tree.sub {
  margin-left: 26px;
  padding-left: 10px;
  border-left: 2px solid var(--border);
}
.row {
  display: flex;
  gap: 10px;
  width: 100%;
  padding: 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  text-align: left;
  cursor: pointer;
}
.row:hover {
  background: var(--surface-2);
}
.row.now {
  background: var(--accent-soft);
}
.t {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.sub b {
  font-weight: 500;
}
.t .muted {
  font-size: 13.5px;
}
</style>
