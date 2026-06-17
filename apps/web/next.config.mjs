import bundleAnalyzer from '@next/bundle-analyzer';
import { fileURLToPath } from 'node:url';

const appRoot = fileURLToPath(new URL('./', import.meta.url));

const withBundleAnalyzer = bundleAnalyzer({
  enabled: process.env.ANALYZE === 'true'
});

const localAPIURL = process.env.API_PROXY_URL || process.env.BETA_PROXY_API_URL || 'http://localhost:8080';

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  experimental: {
    webpackBuildWorker: false
  },
  output: "standalone",
  outputFileTracingRoot: appRoot,
  async headers() {
    return [
      {
        source: "/runtime-config.js",
        headers: [
          {
            key: "Cache-Control",
            value: "no-store"
          }
        ]
      },
      {
        source: "/:path*.v:version(\\d+).:ext(jpg|jpeg|png|webp|avif|svg|ico|ogg|mp3|woff|woff2)",
        headers: [
          {
            key: "Cache-Control",
            value: "public, max-age=31536000, immutable"
          }
        ]
      },
      {
        // Avoid storing route documents so clients always fetch the latest
        // document after a deploy, while keeping static assets cacheable.
        source: "/((?!_next/static|_next/image|.*\\..*).*)",
        headers: [
          {
            key: "Cache-Control",
            value: "no-store"
          }
        ]
      }
    ];
  },
  async rewrites() {
    if (process.env.NODE_ENV === 'production' && !process.env.API_PROXY_URL && !process.env.BETA_PROXY_API_URL) {
      return [];
    }
    return [
      {
        source: "/v1/:path*",
        destination: `${localAPIURL.replace(/\/$/, "")}/v1/:path*`
      }
    ];
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "lh3.googleusercontent.com",
        pathname: "/**"
      }
    ]
  }
};

export default withBundleAnalyzer(nextConfig);
