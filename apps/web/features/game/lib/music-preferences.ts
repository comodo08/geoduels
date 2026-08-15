export const MUSIC_MUTED_STORAGE_KEY = 'geoduels.musicMuted';

function safeLocalStorage(): Storage | null {
  if (typeof window === 'undefined') return null;
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

export function isMusicMuted(): boolean {
  return safeLocalStorage()?.getItem(MUSIC_MUTED_STORAGE_KEY) === 'true';
}

export function setMusicMuted(muted: boolean): void {
  safeLocalStorage()?.setItem(MUSIC_MUTED_STORAGE_KEY, String(muted));
}
