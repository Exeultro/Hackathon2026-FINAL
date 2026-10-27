const cache = new Map<string, Promise<string | null>>()
const queues = new Map<string, Promise<unknown>>()
const videos = new Map<string, Promise<HTMLVideoElement>>()

function once(el: HTMLVideoElement, event: string, ms = 6000): Promise<void> {
  return new Promise((resolve, reject) => {
    const t = setTimeout(() => reject(new Error('timeout')), ms)
    el.addEventListener(event, () => (clearTimeout(t), resolve()), { once: true })
    el.addEventListener('error', () => (clearTimeout(t), reject(new Error('video error'))), { once: true })
  })
}

function video(url: string): Promise<HTMLVideoElement> {
  if (!videos.has(url)) {
    const v = Object.assign(document.createElement('video'), { crossOrigin: 'anonymous', muted: true, preload: 'auto', src: url })
    videos.set(url, once(v, 'loadeddata').then(() => v))
  }
  return videos.get(url)!
}

async function grab(url: string, t: number): Promise<string | null> {
  try {
    const v = await video(url)
    v.currentTime = Math.min(t + 0.5, Math.max(0, v.duration - 0.1))
    await once(v, 'seeked')
    const c = Object.assign(document.createElement('canvas'), { width: 320, height: 180 })
    c.getContext('2d')!.drawImage(v, 0, 0, c.width, c.height)
    return c.toDataURL('image/jpeg', 0.75)
  } catch {
    // CORS-tainted canvas or unreadable video: caller falls back to the lecture thumbnail
    return null
  }
}

// Frame of a video at time t, as a data URL; seeks on one shared <video> per URL, one at a time
export function frameAt(url: string, t: number): Promise<string | null> {
  const key = `${url}#${Math.floor(t)}`
  if (!cache.has(key)) {
    const job = (queues.get(url) ?? Promise.resolve()).then(() => grab(url, t))
    queues.set(url, job.catch(() => null))
    cache.set(key, job)
  }
  return cache.get(key)!
}
