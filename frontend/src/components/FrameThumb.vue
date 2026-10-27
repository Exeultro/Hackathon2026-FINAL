<script setup lang="ts">
import { ref, watchEffect } from 'vue'
import { frameAt } from '@/frames'
import { timecode } from '@/format'

const props = defineProps<{ video?: string | null; t: number; fallback?: string | null }>()
const src = ref<string | null>(null)

watchEffect(async () => {
  src.value = null
  const { video, t } = props
  if (!video) return
  const frame = await frameAt(video, t)
  if (props.video === video && props.t === t) src.value = frame
})
</script>

<template>
  <div class="frame">
    <img v-if="src || fallback" :src="src ?? fallback ?? undefined" alt="" />
    <span class="at">{{ timecode(t) }}</span>
  </div>
</template>

<style scoped>
.frame {
  position: relative;
  aspect-ratio: 16 / 9;
  border-radius: 8px;
  overflow: hidden;
  background: linear-gradient(135deg, #2b3350, #1b2030);
}
img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.at {
  position: absolute;
  right: 5px;
  bottom: 5px;
  padding: 0 5px;
  border-radius: 4px;
  background: rgb(0 0 0 / 70%);
  color: #fff;
  font: 600 11.5px ui-monospace, monospace;
}
</style>
