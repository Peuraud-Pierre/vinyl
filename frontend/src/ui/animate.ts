// Joue une animation CSS d'arrivée puis retire sa classe : avec « fill both », l'état final (filtre,
// transformation) resterait sinon appliqué pour toujours. Dans WebKit, un filtre sur un parent fait
// dessiner la 3D à plat (la pile de disques de la platine passait de côté).
export function animateOnce(el: Element | null | undefined, cls: string): void {
  if (!el) return;
  el.classList.add(cls);
  const done = (e: Event) => {
    if (e.target !== el) return; // animation d'un enfant
    el.classList.remove(cls);
    el.removeEventListener("animationend", done);
  };
  el.addEventListener("animationend", done);
}
