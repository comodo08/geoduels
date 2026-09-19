#!/bin/sh
set -eu

# Project container env into the browser config. Do not invent localhost or
# product-origin defaults here — those live in web/lib/runtime-config.ts so
# `next dev` and the image stay aligned. Empty values are ignored at runtime.
cat > /app/public/runtime-config.js <<EOF
window.__GEODUELS_CONFIG__ = {
  NEXT_PUBLIC_SITE_URL: "${NEXT_PUBLIC_SITE_URL:-}",
  NEXT_PUBLIC_QUEUE_URL: "${NEXT_PUBLIC_QUEUE_URL:-}",
  NEXT_PUBLIC_REALTIME_URL: "${NEXT_PUBLIC_REALTIME_URL:-}",
  NEXT_PUBLIC_API_URL: "${NEXT_PUBLIC_API_URL:-}",
  NEXT_PUBLIC_GOOGLE_CLIENT_ID: "${NEXT_PUBLIC_GOOGLE_CLIENT_ID:-}",
  NEXT_PUBLIC_GOOGLE_ALLOWED_ORIGINS: "${NEXT_PUBLIC_GOOGLE_ALLOWED_ORIGINS:-}",
  NEXT_PUBLIC_DISCORD_CLIENT_ID: "${NEXT_PUBLIC_DISCORD_CLIENT_ID:-}",
  NEXT_PUBLIC_TURNSTILE_SITE_KEY: "${NEXT_PUBLIC_TURNSTILE_SITE_KEY:-}",
  NEXT_PUBLIC_GOOGLE_EMBED_KEY: "${NEXT_PUBLIC_GOOGLE_EMBED_KEY:-}",
  NEXT_PUBLIC_APP_VERSION: "${NEXT_PUBLIC_APP_VERSION:-${NEXT_PUBLIC_GIT_SHA:-}}"
};
EOF

exec "$@"
