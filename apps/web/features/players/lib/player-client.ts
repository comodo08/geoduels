import { apiFetch, readError } from "../../../lib/http";
import type { RuntimeConfig } from "../../../lib/runtime-config";
import type { PlayerMatchesPage, PublicPlayerProfile } from "../types";

export async function requestPlayerProfile(
  config: RuntimeConfig,
  userId: string,
): Promise<PublicPlayerProfile> {
  const resp = await apiFetch(config, `/v1/players/${encodeURIComponent(userId)}`);
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load player profile"));
  }
  return resp.json();
}

export async function requestPlayerMatches(
  config: RuntimeConfig,
  userId: string,
  limit = 20,
  cursor = "",
): Promise<PlayerMatchesPage> {
  const query = new URLSearchParams({ limit: String(limit) });
  if (cursor) query.set("cursor", cursor);
  const resp = await apiFetch(
    config,
    `/v1/players/${encodeURIComponent(userId)}/matches?${query.toString()}`,
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load match history"));
  }
  return resp.json();
}
