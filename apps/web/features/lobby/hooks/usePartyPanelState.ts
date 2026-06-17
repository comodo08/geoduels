import type { MatchConfig } from "../../matchmaking/lib/queue-client";
import type { LobbyRuntimeStatus } from "../controllers/lobby-controller";
import type { LobbySnapshot, PartyMode } from "../lib/lobby-client";

type PrivateLobbyView = {
  status: LobbyRuntimeStatus;
  snapshot: LobbySnapshot | null;
  inviteCode: string;
  isMember: boolean;
  isOwner: boolean;
  busy: boolean;
  error: string;
};

type UsePartyPanelStateInput = {
  privateLobby: PrivateLobbyView;
  userId: string;
  updateSettings: (config: MatchConfig, mode?: PartyMode) => Promise<void>;
  setInviteCopied: (copied: boolean) => void;
};

const defaultLobbyConfig: MatchConfig = {
  ruleset: "moving",
  roundTimerMode: "none",
  pressureTimeLimitMs: 15000,
};

export function usePartyPanelState({
  privateLobby,
  userId,
  updateSettings,
  setInviteCopied,
}: UsePartyPanelStateInput) {
  const inviteURL =
    typeof window !== "undefined" && privateLobby.inviteCode
      ? `${window.location.origin}/party/${privateLobby.inviteCode}`
      : "";
  const loading =
    !privateLobby.snapshot &&
    ["creating", "joining", "connecting", "reconnecting"].includes(privateLobby.status);
  const active = !!privateLobby.snapshot || privateLobby.status !== "idle";
  const members = privateLobby.snapshot?.members || [];
  const activeMatchId = privateLobby.snapshot?.activeMatchId || privateLobby.snapshot?.startedMatchId || "";
  const matchInProgress =
    privateLobby.snapshot?.state === "in_match" || privateLobby.snapshot?.state === "started";
  const currentMember = members.find((member) => member.userId === userId);
  const config = privateLobby.snapshot?.config || defaultLobbyConfig;
  const mode = privateLobby.snapshot?.mode || "duel";
  const clockOn = config.roundTimerMode === "fixed";
  const pressureOn =
    (typeof config.pressureTimeLimitMs === "number" && config.pressureTimeLimitMs > 0) ||
    config.roundTimerMode === "pressure";
  const roundSeconds = Math.round((config.roundTimeLimitMs || 45000) / 1000);
  const pressureSeconds = pressureOn ? Math.round((config.pressureTimeLimitMs || 15000) / 1000) : 0;
  const missingMembers = members.filter(
    (member) => (member.presenceStatus || (member.connected ? "online" : "offline")) !== "online",
  );
  const teamACount = members.filter((member) => (member.teamId || "a") === "a").length;
  const teamBCount = members.filter((member) => member.teamId === "b").length;
  const canStart =
    privateLobby.isOwner &&
    privateLobby.snapshot?.state === "open" &&
    ((mode === "duel" && members.length === 2) ||
      (mode === "team_duel" &&
        members.length >= 2 &&
        members.length <= 8 &&
        teamACount > 0 &&
        teamBCount > 0) ||
      (mode === "free_for_all" && members.length >= 2 && members.length <= 8)) &&
    missingMembers.length === 0;

  const saveConfig = (patch: MatchConfig) => {
    const next: MatchConfig = {
      ...config,
      ...patch,
    };
    if (next.roundTimerMode !== "fixed") {
      next.roundTimerMode = "none";
      next.roundTimeLimitMs = undefined;
    } else {
      next.roundTimeLimitMs = Math.max(10000, Math.min(120000, next.roundTimeLimitMs || 45000));
    }
    if ((next.pressureTimeLimitMs || 0) !== 15000) {
      next.pressureTimeLimitMs = undefined;
    }
    void updateSettings(next);
  };

  const saveMode = (nextMode: PartyMode) => {
    void updateSettings(config, nextMode);
  };

  const copyInvite = () => {
    if (!inviteURL) return;
    void navigator.clipboard?.writeText(inviteURL);
    setInviteCopied(true);
    if (typeof window !== "undefined") {
      window.setTimeout(() => setInviteCopied(false), 1600);
    }
  };

  return {
    active,
    activeMatchId,
    canStart,
    clockOn,
    config,
    copyInvite,
    currentMember,
    inviteURL,
    loading,
    matchInProgress,
    members,
    missingMembers,
    mode,
    pressureOn,
    pressureSeconds,
    roundSeconds,
    saveConfig,
    saveMode,
    teamACount,
    teamBCount,
  };
}

export type PartyPanelState = ReturnType<typeof usePartyPanelState>;
