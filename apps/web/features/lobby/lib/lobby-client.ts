import type { RuntimeConfig } from "../../../lib/runtime-config";
import { normalizeHTTPBase, normalizeWSBase } from "../../../lib/runtime-config";
import type { AuthSessionSnapshot } from "../../auth/session";
import type { MatchConfig } from "../../matchmaking/lib/queue-client";
import type { PlayerBadgeInfo } from "../../../components/ui/PlayerBadge";

export type PartyMode = "duel" | "team_duel" | "free_for_all";
export type LobbyTeamId = "a" | "b";

export type LobbyMember = {
  userId: string;
  displayName: string;
  avatarUrl?: string;
  isGuest?: boolean;
  isAdmin?: boolean;
  selectedBadge?: PlayerBadgeInfo | null;
  teamId?: LobbyTeamId | "";
  role: string;
  ready?: boolean;
  connected?: boolean;
  presenceStatus?: "online" | "away" | "offline";
  inActiveMatch?: boolean;
};

export type LobbySnapshot = {
  id: string;
  inviteCode: string;
  ownerUserId: string;
  state: "open" | "in_match" | "started" | "closed" | "expired";
  mode: PartyMode;
  mapScope: string;
  mapName?: string;
  mapLocationCount?: number;
  config?: MatchConfig;
  activeMatchId?: string;
  lastMatchId?: string;
  startedMatchId?: string;
  members: LobbyMember[];
};

export type LobbyPatch = {
  revision: number;
  state?: LobbySnapshot["state"];
  ownerUserId?: string;
  mode?: PartyMode;
  config?: MatchConfig;
  activeMatchId?: string;
  lastMatchId?: string;
  startedMatchId?: string;
  upsertMembers?: LobbyMember[];
  removeMemberIds?: string[];
};

export type LobbyAssignment = {
  matchId: string;
  mode?: string;
  config?: MatchConfig;
  node: string;
  ticket: string;
  wsPath: string;
  sourcePartyId?: string;
  sourcePartyInviteCode?: string;
};

export type LobbyEvent =
  | { type: "party_snapshot"; lobby: LobbySnapshot }
  | { type: "party_patch"; patch: LobbyPatch }
  | { type: "match_assigned"; assignment: LobbyAssignment }
  | { type: "party_error"; message: string };

function authHeaders(accessToken: string) {
  return { Authorization: `Bearer ${accessToken}` };
}

function lobbyHTTPBase(config: RuntimeConfig) {
  return normalizeHTTPBase(config.queueURL).replace(/\/$/, "");
}

function lobbyWSTarget(config: RuntimeConfig, lobbyId: string, accessToken: string) {
  return `${normalizeWSBase(config.queueURL).replace(/\/$/, "")}/parties/${encodeURIComponent(lobbyId)}/ws?accessToken=${encodeURIComponent(accessToken)}`;
}

export async function createLobby(config: RuntimeConfig, accessToken: string, mode: PartyMode = "duel", matchConfig?: MatchConfig): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties`, {
    method: "POST",
    headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
    body: JSON.stringify({ mode, config: matchConfig }),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Party unavailable");
  return resp.json();
}

export async function updateLobbySettings(config: RuntimeConfig, lobbyId: string, accessToken: string, matchConfig: MatchConfig, mode?: PartyMode): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/settings`, {
    method: "PATCH",
    headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
    body: JSON.stringify({ config: matchConfig, mode }),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not update party settings");
  return resp.json();
}

export async function updateLobbyTeam(config: RuntimeConfig, lobbyId: string, accessToken: string, teamId: LobbyTeamId): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/team`, {
    method: "PATCH",
    headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
    body: JSON.stringify({ teamId }),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not switch team");
  return resp.json();
}

export function applyLobbyPatch(lobby: LobbySnapshot | null, patch: LobbyPatch): LobbySnapshot | null {
  if (!lobby) return lobby;
  const next: LobbySnapshot = {
    ...lobby,
    state: patch.state ?? lobby.state,
    ownerUserId: patch.ownerUserId ?? lobby.ownerUserId,
    mode: patch.mode ?? lobby.mode,
    config: patch.config ?? lobby.config,
    activeMatchId: patch.activeMatchId ?? lobby.activeMatchId,
    lastMatchId: patch.lastMatchId ?? lobby.lastMatchId,
    startedMatchId: patch.startedMatchId ?? lobby.startedMatchId,
    members: lobby.members,
  };
  if (patch.upsertMembers?.length || patch.removeMemberIds?.length) {
    const removed = new Set(patch.removeMemberIds || []);
    const members = new Map(next.members.filter((member) => !removed.has(member.userId)).map((member) => [member.userId, member]));
    for (const member of patch.upsertMembers || []) {
      members.set(member.userId, { ...(members.get(member.userId) || {} as LobbyMember), ...member });
    }
    next.members = Array.from(members.values());
  }
  return next;
}

export async function fetchLobby(config: RuntimeConfig, code: string, signal?: AbortSignal): Promise<LobbySnapshot | null> {
  const target = `${lobbyHTTPBase(config)}/parties/${encodeURIComponent(code)}`;
  const resp = signal ? await fetch(target, { signal }) : await fetch(target);
  if (resp.status === 404) return null;
  if (!resp.ok) throw new Error("Party unavailable");
  return resp.json();
}

export async function touchLobbyPresence(config: RuntimeConfig, lobbyId: string, accessToken: string, signal?: AbortSignal): Promise<void> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/presence`, {
    method: "POST",
    headers: authHeaders(accessToken),
    signal,
  });
  if (!resp.ok) throw new Error("Party presence unavailable");
}

export async function joinLobby(config: RuntimeConfig, code: string, accessToken: string): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(code)}/join`, {
    method: "POST",
    headers: authHeaders(accessToken),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not join party");
  return resp.json();
}

export async function leaveLobby(config: RuntimeConfig, lobbyId: string, accessToken: string): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/leave`, {
    method: "POST",
    headers: authHeaders(accessToken),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not leave party");
  return resp.json();
}

export async function kickLobbyMember(config: RuntimeConfig, lobbyId: string, accessToken: string, userId: string): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/kick`, {
    method: "POST",
    headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
    body: JSON.stringify({ userId }),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not kick player");
  return resp.json();
}

export async function transferLobbyOwner(config: RuntimeConfig, lobbyId: string, accessToken: string, userId: string): Promise<LobbySnapshot> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/transfer-owner`, {
    method: "POST",
    headers: { ...authHeaders(accessToken), "Content-Type": "application/json" },
    body: JSON.stringify({ userId }),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not transfer leader");
  return resp.json();
}

export async function startLobby(config: RuntimeConfig, lobbyId: string, accessToken: string): Promise<LobbyAssignment> {
  const resp = await fetch(`${lobbyHTTPBase(config)}/parties/${encodeURIComponent(lobbyId)}/start`, {
    method: "POST",
    headers: authHeaders(accessToken),
  });
  if (!resp.ok) throw new Error((await resp.text()) || "Could not start party");
  const data = await resp.json();
  return data.assignment;
}

export async function streamLobby(
  config: RuntimeConfig,
  session: AuthSessionSnapshot,
  lobbyId: string,
  signal: AbortSignal,
  onEvent: (event: LobbyEvent) => void,
) {
  const target = lobbyWSTarget(config, lobbyId, session.accessToken);
  await new Promise<void>((resolve, reject) => {
    let settled = false;
    const ws = new WebSocket(target);
    const cleanup = () => signal.removeEventListener("abort", abort);
    const settle = (fn: typeof resolve | typeof reject, value?: unknown) => {
      if (settled) return;
      settled = true;
      cleanup();
      fn(value as never);
    };
    const abort = () => {
      ws.close();
      settle(reject, new DOMException("Aborted", "AbortError"));
    };
    signal.addEventListener("abort", abort, { once: true });
    ws.onerror = () => settle(reject, new Error("Party connection failed"));
    ws.onclose = () => {
      if (signal.aborted) settle(reject, new DOMException("Aborted", "AbortError"));
      else settle(resolve);
    };
    ws.onmessage = (evt) => {
      let msg: any;
      try {
        msg = JSON.parse(String(evt.data));
      } catch {
        settle(reject, new Error("Party connection failed"));
        return;
      }
      const payload = msg?.payload ?? {};
      if (msg?.type === "party_snapshot") onEvent({ type: "party_snapshot", lobby: payload as LobbySnapshot });
      if (msg?.type === "party_patch") onEvent({ type: "party_patch", patch: payload as LobbyPatch });
      if (msg?.type === "match_assigned") onEvent({ type: "match_assigned", assignment: payload as LobbyAssignment });
      if (msg?.type === "party_error") onEvent({ type: "party_error", message: payload?.message || "Party unavailable" });
    };
  });
}
