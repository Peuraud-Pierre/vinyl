let ctx: AudioContext | null = null;

// Court craquement de vinyle généré (bruit filtré + quelques « pops »), sans fichier audio.
export function playCrackle(): void {
  ctx ??= new AudioContext();
  const sr = ctx.sampleRate;
  const len = Math.floor(sr * 0.45);
  const buf = ctx.createBuffer(1, len, sr);
  const data = buf.getChannelData(0);
  for (let i = 0; i < len; i++) {
    const fade = 1 - i / len;
    let v = (Math.random() * 2 - 1) * 0.05 * fade;
    if (Math.random() < 0.0009) v += (Math.random() < 0.5 ? -1 : 1) * 0.6 * fade;
    data[i] = v;
  }
  const src = ctx.createBufferSource();
  src.buffer = buf;
  const filter = ctx.createBiquadFilter();
  filter.type = "bandpass";
  filter.frequency.value = 2500;
  filter.Q.value = 0.7;
  src.connect(filter).connect(ctx.destination);
  src.start();
}

// Son de scratch généré : bruit filtré dont le volume et la hauteur suivent la vitesse du disque à la main.
export function startScratch(): { speed(degPerMs: number): void; stop(): void } {
  ctx ??= new AudioContext();
  const c = ctx;
  const buf = c.createBuffer(1, c.sampleRate, c.sampleRate);
  const data = buf.getChannelData(0);
  for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
  const src = c.createBufferSource();
  src.buffer = buf;
  src.loop = true;
  const filter = c.createBiquadFilter();
  filter.type = "bandpass";
  filter.Q.value = 2.5;
  filter.frequency.value = 400;
  const gain = c.createGain();
  gain.gain.value = 0;
  src.connect(filter).connect(gain).connect(c.destination);
  src.start();
  return {
    speed(v) {
      const s = Math.min(1, Math.abs(v) / 1.2); // 1,2 °/ms : geste rapide
      const t = c.currentTime;
      gain.gain.setTargetAtTime(s * 0.35, t, 0.02);
      filter.frequency.setTargetAtTime(300 + s * 2600, t, 0.02);
    },
    stop() {
      gain.gain.setTargetAtTime(0, c.currentTime, 0.03);
      src.stop(c.currentTime + 0.2);
    },
  };
}
