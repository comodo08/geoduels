import { useRuntimeConfig } from './runtime-config-context';

export function useSiteURL() {
  return useRuntimeConfig().siteURL;
}
