import { resolvePublicConfig } from '../config/public-runtime-config.mjs';
import { createRuntimeConfig, type PublicConfig } from './runtime-config';

export function readServerConfig() {
  if (typeof window !== 'undefined') throw new Error('Environment configuration is server-only.');
  return createRuntimeConfig(resolvePublicConfig(process.env, process.env.NODE_ENV === 'development') as PublicConfig);
}
