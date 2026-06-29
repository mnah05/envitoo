import { useEffect } from "react";
import Lenis from "lenis";

/**
 * Initializes Lenis smooth scrolling for the document. No-op when the user
 * prefers reduced motion. Only intended for the marketing/discover surfaces —
 * avoid wrapping dashboard pages that need native scroll anchoring.
 */
export function useSmoothScroll(enabled = true) {
  useEffect(() => {
    if (!enabled) return;
    const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (prefersReduced) return;

    const lenis = new Lenis({ duration: 1.1, smoothWheel: true });
    let raf = 0;
    function loop(time: number) {
      lenis.raf(time);
      raf = requestAnimationFrame(loop);
    }
    raf = requestAnimationFrame(loop);
    return () => {
      cancelAnimationFrame(raf);
      lenis.destroy();
    };
  }, [enabled]);
}
