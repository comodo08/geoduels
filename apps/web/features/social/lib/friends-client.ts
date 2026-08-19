import type { RuntimeConfig } from "../../../lib/runtime-config";
import { apiFetch, authHeaders, readError } from "../../../lib/http";
import type { PlayerBadgeInfo } from "../../../components/ui/PlayerBadge";

export type SocialPresenceStatus = "online" | "away" | "offline";

export type FriendRow = {
  userId: string;
  displayName: string;
  avatarUrl?: string;
  isGuest?: boolean;
  selectedBadge?: PlayerBadgeInfo | null;
};

export type PlayerSearchResult = FriendRow;

async function parseOrThrow<T>(resp: Response, fallback: string): Promise<T> {
  if (!resp.ok) {
    throw new Error(await readError(resp, fallback));
  }
  return resp.json() as Promise<T>;
}

export async function searchPlayers(
  config: RuntimeConfig,
  accessToken: string,
  query: string,
  signal?: AbortSignal,
): Promise<PlayerSearchResult[]> {
  const resp = await apiFetch(
    config,
    `/v1/users/search?q=${encodeURIComponent(query)}`,
    { headers: authHeaders(accessToken), signal },
  );
  const data = await parseOrThrow<{ players: PlayerSearchResult[] }>(
    resp,
    "Search failed",
  );
  return data.players || [];
}

export async function listFriends(
  config: RuntimeConfig,
  accessToken: string,
  signal?: AbortSignal,
): Promise<FriendRow[]> {
  const resp = await apiFetch(config, "/v1/me/friends", {
    headers: authHeaders(accessToken),
    signal,
  });
  const data = await parseOrThrow<{ friends: FriendRow[] }>(resp, "Could not load friends");
  return data.friends || [];
}

export async function listFriendRequests(
  config: RuntimeConfig,
  accessToken: string,
  signal?: AbortSignal,
): Promise<{ incoming: FriendRow[]; outgoing: FriendRow[] }> {
  const resp = await apiFetch(config, "/v1/me/friends/requests", {
    headers: authHeaders(accessToken),
    signal,
  });
  const data = await parseOrThrow<{ incoming: FriendRow[]; outgoing: FriendRow[] }>(
    resp,
    "Could not load requests",
  );
  return { incoming: data.incoming || [], outgoing: data.outgoing || [] };
}

export async function sendFriendRequest(
  config: RuntimeConfig,
  accessToken: string,
  targetUserId: string,
): Promise<void> {
  const resp = await apiFetch(config, "/v1/me/friends/requests", {
    method: "POST",
    headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
    body: JSON.stringify({ targetUserId }),
  });
  await parseOrThrow(resp, "Could not send request");
}

export async function acceptFriendRequest(
  config: RuntimeConfig,
  accessToken: string,
  requesterId: string,
): Promise<void> {
  const resp = await apiFetch(
    config,
    `/v1/me/friends/requests/${encodeURIComponent(requesterId)}/accept`,
    { method: "POST", headers: authHeaders(accessToken) },
  );
  await parseOrThrow(resp, "Could not accept request");
}

export async function declineFriendRequest(
  config: RuntimeConfig,
  accessToken: string,
  requesterId: string,
): Promise<void> {
  const resp = await apiFetch(
    config,
    `/v1/me/friends/requests/${encodeURIComponent(requesterId)}/decline`,
    { method: "POST", headers: authHeaders(accessToken) },
  );
  await parseOrThrow(resp, "Could not decline request");
}

export async function removeFriend(
  config: RuntimeConfig,
  accessToken: string,
  friendUserId: string,
): Promise<void> {
  const resp = await apiFetch(
    config,
    `/v1/me/friends/${encodeURIComponent(friendUserId)}`,
    { method: "DELETE", headers: authHeaders(accessToken) },
  );
  await parseOrThrow(resp, "Could not remove friend");
}

export async function sendPartyInvite(
  config: RuntimeConfig,
  accessToken: string,
  partyId: string,
  friendUserId: string,
): Promise<void> {
  const resp = await apiFetch(
    config,
    `/v1/parties/${encodeURIComponent(partyId)}/invite-friend`,
    {
      method: "POST",
      headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
      body: JSON.stringify({ friendUserId }),
    },
  );
  await parseOrThrow(resp, "Could not invite friend");
}
