function decodeBase64URL(value: string): string {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
  const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4);
  if (typeof window !== 'undefined' && typeof window.atob === 'function') {
    return window.atob(padded);
  }
  return '';
}

export function decodeAccessTokenExpiry(accessToken: string): number {
  if (!accessToken) return 0;
  const parts = accessToken.split('.');
  if (parts.length < 2) return 0;
  try {
    const payload = JSON.parse(decodeBase64URL(parts[1] || ''));
    return typeof payload?.exp === 'number' ? payload.exp * 1000 : 0;
  } catch {
    return 0;
  }
}
