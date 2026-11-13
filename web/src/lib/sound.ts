// Сигнал тревоги без аудиофайлов: два тона через WebAudio.
// Браузер разрешает звук только после первого действия пользователя — до этого playAlarm молчит.
let ctx: AudioContext | null = null

export function unlockAudio(): void {
  if (ctx) return
  try {
    ctx = new AudioContext()
  } catch {
    ctx = null
  }
}

export function playAlarm(): void {
  if (!ctx || ctx.state !== 'running') return
  const start = ctx.currentTime
  for (const [i, freq] of [880, 660].entries()) {
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.type = 'square'
    osc.frequency.value = freq
    const t = start + i * 0.16
    gain.gain.setValueAtTime(0.0001, t)
    gain.gain.exponentialRampToValueAtTime(0.06, t + 0.01)
    gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.14)
    osc.connect(gain).connect(ctx.destination)
    osc.start(t)
    osc.stop(t + 0.15)
  }
}
