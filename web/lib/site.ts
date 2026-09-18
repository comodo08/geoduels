const DEFAULT_SITE_URL = 'https://geoduels.io';

function normalizeSiteURL(value?: string) {
  if (!value) return DEFAULT_SITE_URL;
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

export function getSiteURL() {
  return normalizeSiteURL(process.env.NEXT_PUBLIC_SITE_URL);
}
