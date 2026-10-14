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
