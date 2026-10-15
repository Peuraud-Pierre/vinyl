// Bandeau de mise à jour de Vinyl : « Vinyl 1.0.3 est disponible » → téléchargement → redémarrage ; en
// cas d'échec, lien vers la page de la release.
export interface UpdateInfo { available: boolean; version: string; notes: string; dev: boolean; }
export interface BannerDeps {
  install(): Promise<boolean>; // false : plus rien à installer
  openRelease(version: string): void;
}

export function mountUpdateBanner(el: HTMLElement, deps: BannerDeps) {
  let version = "";
  let installing = false;
  el.hidden = true;
  const render = (html: string) => {
    el.innerHTML = `<div class="glass flex items-center gap-3 rounded-full px-4 py-2 text-[13px]">${html}</div>`;
    el.hidden = false;
  };
  const fail = () => {
    installing = false;
    render(`<span>La mise à jour a échoué.</span><button data-release class="cursor-pointer font-semibold underline">Voir la release</button>`);
    el.querySelector<HTMLButtonElement>("[data-release]")!.onclick = () => deps.openRelease(version);
  };
  return {
    show(info: UpdateInfo) {
      if (installing) return;
      if (!info.available) { el.hidden = true; return; }
      version = info.version;
      render(`<span>Vinyl ${info.version.replace(/[<>&"]/g, "")} est disponible</span><button data-update class="btn-dark cursor-pointer !px-3 !py-1 text-[13px]">Mettre à jour</button>`);
      el.querySelector<HTMLButtonElement>("[data-update]")!.onclick = () => {
        installing = true;
        render(`<span>Téléchargement…</span>`);
        deps.install().then((started) => {
          if (!started) { installing = false; el.hidden = true; }
        }, fail);
      };
    },
    progress(written: number, total: number) {
      if (installing && total > 0) render(`<span>Téléchargement… ${Math.round((written / total) * 100)} %</span>`);
    },
    // Wails signale aussi les échecs de vérification : on ne réagit que pendant une installation.
    installFailed() { if (installing) fail(); },
  };
}
