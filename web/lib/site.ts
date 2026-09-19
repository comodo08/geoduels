import { getRuntimeConfig } from './runtime-config';

export function getSiteURL() {
  return getRuntimeConfig().siteURL;
}
