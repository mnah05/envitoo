import { useCallback, useState } from "react";

const STORAGE_KEY = "envitoo.onboardingSeen";

function readFlag(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

function writeFlag() {
  try {
    localStorage.setItem(STORAGE_KEY, "1");
  } catch {
    // Storage unavailable (private mode) — just show again next visit.
  }
}

/**
 * First-run onboarding flag. Shown until the user dismisses it once;
 * shared across dashboard pages via localStorage.
 */
export function useOnboarding() {
  const [seen, setSeen] = useState(readFlag);

  const dismiss = useCallback(() => {
    writeFlag();
    setSeen(true);
  }, []);

  return { show: !seen, dismiss };
}
