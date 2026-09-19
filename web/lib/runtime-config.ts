declare global {
  interface Window {
    __GEODUELS_CONFIG__?: Partial<WindowRuntimeConfig>;
  }
}

export type WindowRuntimeConfig = {
  NEXT_PUBLIC_SITE_URL: string;
  NEXT_PUBLIC_QUEUE_URL: string;
  NEXT_PUBLIC_REALTIME_URL: string;
  NEXT_PUBLIC_API_URL: string;
  NEXT_PUBLIC_GOOGLE_CLIENT_ID: string;
  NEXT_PUBLIC_GOOGLE_ALLOWED_ORIGINS: string;
  NEXT_PUBLIC_DISCORD_CLIENT_ID: string;
  NEXT_PUBLIC_TURNSTILE_SITE_KEY: string;
  NEXT_PUBLIC_GOOGLE_EMBED_KEY: string;
  NEXT_PUBLIC_APP_VERSION: string;
};

export type RuntimeConfig = {
  siteURL: string;
  queueURL: string;
  realtimeBaseURL: string;
  apiURL: string;
  googleClientId: string;
  googleAllowedOrigins: string[];
  discordClientId: string;
  turnstileSiteKey: string;
  googleEmbedKey: string;
  appVersion: string;
  roundDurationMs: number;
  maxHP: number;
  queueHeartbeatIntervalMs: number;
  socketHeartbeatIntervalMs: number;
  socketStaleAfterMs: number;
  connectionErrorMessage: string;
  gameConnectionErrorMessage: string;
};

let browserRuntimeConfig: RuntimeConfig | null = null;

function splitOrigins(value: string) {
  return value
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean);
}

// Dynamic process.env[name] so Next does not inline NEXT_PUBLIC_* at image build.
function envString(name: string): string {
  if (typeof process === 'undefined' || !process.env) return '';
  const value = process.env[name];
  return typeof value === 'string' ? value : '';
}

function readWindowRuntimeConfig(source?: Partial<WindowRuntimeConfig>) {
  const runtimeSource = source ?? (typeof window !== 'undefined' ? window.__GEODUELS_CONFIG__ : undefined) ?? {};
  const runtimeEntries = Object.entries(runtimeSource).filter(([, value]) => {
    if (value === undefined || value === '') return false;
    if (typeof value === 'string' && value.startsWith('REPLACE_WITH_')) return false;
    return true;
  });
  return Object.fromEntries(runtimeEntries) as Partial<WindowRuntimeConfig>;
}

export function createRuntimeConfig(source?: Partial<WindowRuntimeConfig>): RuntimeConfig {
  const defaults: WindowRuntimeConfig = {
    NEXT_PUBLIC_SITE_URL: envString('NEXT_PUBLIC_SITE_URL') || 'http://localhost:3000',
    NEXT_PUBLIC_QUEUE_URL: envString('NEXT_PUBLIC_QUEUE_URL') || 'http://localhost:8090',
    NEXT_PUBLIC_REALTIME_URL: envString('NEXT_PUBLIC_REALTIME_URL') || 'http://localhost:8092',
    NEXT_PUBLIC_API_URL: envString('NEXT_PUBLIC_API_URL'),
    NEXT_PUBLIC_GOOGLE_CLIENT_ID: envString('NEXT_PUBLIC_GOOGLE_CLIENT_ID'),
    NEXT_PUBLIC_GOOGLE_ALLOWED_ORIGINS: envString('NEXT_PUBLIC_GOOGLE_ALLOWED_ORIGINS'),
    NEXT_PUBLIC_DISCORD_CLIENT_ID: envString('NEXT_PUBLIC_DISCORD_CLIENT_ID'),
    NEXT_PUBLIC_TURNSTILE_SITE_KEY: envString('NEXT_PUBLIC_TURNSTILE_SITE_KEY'),
    NEXT_PUBLIC_GOOGLE_EMBED_KEY: envString('NEXT_PUBLIC_GOOGLE_EMBED_KEY') || 'NO_KEY_DEFINED',
    NEXT_PUBLIC_APP_VERSION:
      envString('NEXT_PUBLIC_APP_VERSION') || (envString('NEXT_PUBLIC_GIT_SHA') || 'dev').slice(0, 12)
  };
  const publicRuntimeConfig = {
    ...defaults,
    ...readWindowRuntimeConfig(source)
  };
  const config: RuntimeConfig = {
    siteURL: publicRuntimeConfig.NEXT_PUBLIC_SITE_URL.replace(/\/$/, ''),
    queueURL: publicRuntimeConfig.NEXT_PUBLIC_QUEUE_URL,
    realtimeBaseURL: publicRuntimeConfig.NEXT_PUBLIC_REALTIME_URL,
    apiURL: publicRuntimeConfig.NEXT_PUBLIC_API_URL,
    googleClientId: publicRuntimeConfig.NEXT_PUBLIC_GOOGLE_CLIENT_ID,
    googleAllowedOrigins: splitOrigins(publicRuntimeConfig.NEXT_PUBLIC_GOOGLE_ALLOWED_ORIGINS),
    discordClientId: publicRuntimeConfig.NEXT_PUBLIC_DISCORD_CLIENT_ID,
    turnstileSiteKey: publicRuntimeConfig.NEXT_PUBLIC_TURNSTILE_SITE_KEY,
    googleEmbedKey: publicRuntimeConfig.NEXT_PUBLIC_GOOGLE_EMBED_KEY,
    appVersion: publicRuntimeConfig.NEXT_PUBLIC_APP_VERSION,
    roundDurationMs: 45_000,
    maxHP: 6_000,
    queueHeartbeatIntervalMs: 10_000,
    socketHeartbeatIntervalMs: 20_000,
    socketStaleAfterMs: 35_000,
    connectionErrorMessage: 'Connection error',
    gameConnectionErrorMessage: 'Connection lost. Reconnecting...'
  };
  if (process.env.NODE_ENV !== 'production') {
    Object.freeze(config.googleAllowedOrigins);
    Object.freeze(config);
  }
  return config;
}

export function getRuntimeConfig(): RuntimeConfig {
  if (typeof window === 'undefined') {
    return createRuntimeConfig();
  }
  if (!browserRuntimeConfig) {
    browserRuntimeConfig = createRuntimeConfig();
  }
  return browserRuntimeConfig;
}

export function normalizeHTTPBase(value: string): string {
  if (!value) return '';
  if (value.startsWith('ws://')) return `http://${value.slice(5)}`;
  if (value.startsWith('wss://')) return `https://${value.slice(6)}`;
  return value;
}

export function normalizeWSBase(value: string): string {
  if (!value) return '';
  if (value.startsWith('http://')) return `ws://${value.slice(7)}`;
  if (value.startsWith('https://')) return `wss://${value.slice(8)}`;
  return value;
}
