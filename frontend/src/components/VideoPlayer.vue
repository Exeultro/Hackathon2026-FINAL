<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Plyr from 'plyr'
import type { Playback } from '@/api'
import { settings } from '@/settings'

const props = defineProps<{ playback: Playback; subtitles?: string | null; caption?: string }>()
const emit = defineEmits<{ time: [t: number] }>()

const el = ref<HTMLElement | null>(null)
let player: Plyr | null = null
let ready = false
let pending: { t: number; play: boolean } | null = null

function apply(t: number, play: boolean) {
  if (!player) return
  // Land just past a cue boundary: at exactly start == previous end both cues are shown
  player.currentTime = t + 0.05
  emit('time', t)
  if (play) void player.play()
}

function seek(t: number, play = true) {
  if (ready) apply(t, play)
  else pending = { t, play }
}

onMounted(() => {
  player = new Plyr(el.value!, {
    // Embeds get our own caption overlay; active captions would also turn on YouTube auto-subs
    captions: { active: props.playback.type === 'file', language: 'ru', update: true },
    i18n: { captions: 'Субтитры', disabled: 'Выкл', enabled: 'Вкл', play: 'Играть', pause: 'Пауза', settings: 'Настройки', speed: 'Скорость', normal: 'Обычная' },
    speed: { selected: settings.speed, options: [0.75, 1, 1.25, 1.5, 1.75, 2] },
    storage: { enabled: false },
    seekTime: 5,
    keyboard: { focused: true, global: false },
  })
  // Plyr ignores currentTime until duration is known
  const onReady = () => {
    if (ready || !player?.duration) return
    ready = true
    if (pending) apply(pending.t, pending.play)
    pending = null
  }
  player.on('ready', onReady)
  player.on('loadedmetadata', onReady)
  player.on('canplay', onReady)
  player.on('timeupdate', () => emit('time', player!.currentTime))
  player.on('seeked', () => emit('time', player!.currentTime))
})

onBeforeUnmount(() => player?.destroy())

defineExpose({ seek })
</script>

<template>
  <div class="player">
    <video v-if="playback.type === 'file'" ref="el" playsinline preload="metadata" crossorigin="anonymous">
      <source :src="playback.url" />
      <track v-if="subtitles" kind="captions" label="Русский" srclang="ru" :src="subtitles" default />
    </video>
    <div v-else ref="el" :data-plyr-provider="playback.type" :data-plyr-embed-id="playback.provider_id ?? playback.url" />
    <div v-if="playback.type !== 'file' && caption" class="caption">{{ caption }}</div>
  </div>
</template>

<style scoped>
.player {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  background: #000;
  aspect-ratio: 16 / 9;
}
.player :deep(.plyr) {
  height: 100%;
}
.player :deep(video) {
  width: 100%;
  height: 100%;
}
.caption {
  position: absolute;
  left: 50%;
  bottom: 56px;
  transform: translateX(-50%);
  max-width: 86%;
  padding: 4px 10px;
  border-radius: 6px;
  background: rgb(0 0 0 / 72%);
  color: #fff;
  font-size: var(--cap-size);
  text-align: center;
  pointer-events: none;
}
</style>
