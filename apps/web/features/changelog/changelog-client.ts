import type { RuntimeConfig } from "../../lib/runtime-config";
import type { ChangelogPost } from "./types";

export async function requestChangelogPosts(config: RuntimeConfig) {
  const resp = await fetch(`${config.apiURL}/v1/content/changelog`);
  if (!resp.ok) {
    throw new Error("Failed to load changelog");
  }
  return resp.json() as Promise<{ posts: ChangelogPost[] }>;
}

export async function requestChangelogPost(config: RuntimeConfig, slug: string) {
  const resp = await fetch(`${config.apiURL}/v1/content/changelog/${encodeURIComponent(slug)}`);
  if (resp.status === 404) {
    return null;
  }
  if (!resp.ok) {
    throw new Error("Failed to load changelog post");
  }
  return resp.json() as Promise<ChangelogPost>;
}
