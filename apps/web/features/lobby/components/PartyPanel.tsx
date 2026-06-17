import { forwardRef } from "react";
import { Copy, Crown, Loader2, LogOut, Map as MapIcon, Play, UserMinus, UserPlus } from "lucide-react";
import { motion } from "framer-motion";
import type { MatchConfig, GameRuleset } from "../../matchmaking/lib/queue-client";
import type { LobbyRuntimeStatus } from "../controllers/lobby-controller";
import type { LobbySnapshot, LobbyTeamId, PartyMode } from "../lib/lobby-client";
import {
  CLOCK_OPTIONS,
  PRESSURE_OPTIONS,
  lobbyTeamLabel,
  lobbyTeamPillClass,
  lobbyTeamTextClass,
} from "../lib/lobby-ui";
import type { PartyPanelState } from "../hooks/usePartyPanelState";
import {
  LobbyActionButton,
  LobbyActionLink,
  LobbyFieldLabel,
  LobbyInset,
  LobbyLoadingState,
  LobbyPanel,
  LobbyPill,
  LobbySelect,
} from "./lobby-primitives";

type PrivateLobbyView = {
  status: LobbyRuntimeStatus;
  snapshot: LobbySnapshot | null;
  inviteCode: string;
  isMember: boolean;
  isOwner: boolean;
  busy: boolean;
  error: string;
};

type PartyPanelProps = {
  authError: string;
  authLoading: boolean;
  inviteCopied: boolean;
  privateLobby: PrivateLobbyView;
  setMapPickerOpen: (open: boolean) => void;
  state: PartyPanelState;
  userId: string;
  joinInviteLobby: (inviteCode?: string) => Promise<boolean>;
  leavePrivateLobby: () => Promise<void>;
  kickLobbyMember: (userId: string) => Promise<void>;
  transferLobbyOwner: (userId: string) => Promise<void>;
  startPrivateLobby: () => Promise<void>;
  switchPrivateLobbyTeam: (teamId: LobbyTeamId) => Promise<void>;
};

const panelMotion = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -14 },
  transition: { duration: 0.22, ease: "easeOut" },
} as const;

export const PartyPanel = forwardRef<HTMLDivElement, PartyPanelProps>(function PartyPanel({
  authError,
  authLoading,
  inviteCopied,
  joinInviteLobby,
  kickLobbyMember,
  leavePrivateLobby,
  privateLobby,
  setMapPickerOpen,
  startPrivateLobby,
  state,
  switchPrivateLobbyTeam,
  transferLobbyOwner,
  userId,
}, ref) {
  const {
    activeMatchId,
    canStart,
    clockOn,
    config,
    copyInvite,
    currentMember,
    loading,
    matchInProgress,
    members,
    mode,
    pressureOn,
    pressureSeconds,
    roundSeconds,
    saveConfig,
    saveMode,
  } = state;

  return (
    <motion.div
      ref={ref}
      key="private-lobby"
      {...panelMotion}
      className="w-full max-w-[980px] pointer-events-auto"
    >
      <LobbyPanel radius="3xl" density="none" className="overflow-hidden">
        <div className="relative min-h-[220px] p-5 sm:p-7">
          <div className="absolute inset-0 pointer-events-none bg-[linear-gradient(180deg,rgba(42,209,143,0.16)_0%,rgba(10,23,26,0.74)_100%)]" />
          <div className="relative z-10 flex flex-col gap-5">
            <PartyHeader
              inviteCopied={inviteCopied}
              matchInProgress={matchInProgress}
              onCopyInvite={copyInvite}
              onLeave={() => void leavePrivateLobby()}
              privateLobby={privateLobby}
            />

            {privateLobby.inviteCode ? (
              <LobbyInset>
                <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
                  Invite Code
                </p>
                <p className="mt-1 font-mono text-[26px] font-black tracking-[0.18em] text-white">
                  {privateLobby.inviteCode}
                </p>
              </LobbyInset>
            ) : null}

            {matchInProgress ? (
              <LobbyInset tone="subtle" className="border-[#77f0be]/20 bg-[#22d385]/10">
                <p className="text-[11px] font-black uppercase tracking-[0.14em] text-[#77f0be]">
                  Game In Progress
                </p>
                <p className="mt-1 text-sm font-semibold text-[#d8f7e9]">
                  {!privateLobby.isMember
                    ? "Join the lobby now and you will be ready for the next game."
                    : currentMember?.inActiveMatch
                      ? "You are part of this game and can reconnect whenever you are ready."
                      : "You joined after this game started and will be able to play in the next one."}
                </p>
                {currentMember?.inActiveMatch && activeMatchId ? (
                  <LobbyActionLink
                    href={`/match/${encodeURIComponent(activeMatchId)}`}
                    size="md"
                    className="mt-3"
                  >
                    <Play size={16} fill="currentColor" />
                    Reconnect to Game
                  </LobbyActionLink>
                ) : null}
              </LobbyInset>
            ) : null}

            {privateLobby.snapshot ? (
              <PartySettings
                busy={privateLobby.busy}
                clockOn={clockOn}
                config={config}
                isOwner={privateLobby.isOwner}
                mode={mode}
                pressureOn={pressureOn}
                pressureSeconds={pressureSeconds}
                roundSeconds={roundSeconds}
                saveConfig={saveConfig}
                saveMode={saveMode}
                setMapPickerOpen={setMapPickerOpen}
                snapshot={privateLobby.snapshot}
              />
            ) : null}

            {loading ? (
              <LobbyInset>
                <LobbyLoadingState className="min-h-[64px] justify-center">
                  <Loader2 className="animate-spin text-[#77f0be]" size={18} />
                  Connecting to lobby
                </LobbyLoadingState>
              </LobbyInset>
            ) : !privateLobby.isMember ? (
              <LobbyInset>
                <LobbyActionButton
                  type="button"
                  onClick={() => void joinInviteLobby()}
                  disabled={privateLobby.busy || authLoading}
                  size="lg"
                  className="w-full"
                >
                  {privateLobby.busy ? <Loader2 className="animate-spin" size={18} /> : <UserPlus size={18} />}
                  Join Party
                </LobbyActionButton>
                {authError ? (
                  <p className="mt-3 text-center text-xs font-semibold text-red-300">{authError}</p>
                ) : null}
              </LobbyInset>
            ) : null}

            {privateLobby.snapshot ? (
              <PartyMemberList
                busy={privateLobby.busy}
                isOwner={privateLobby.isOwner}
                members={members}
                mode={mode}
                snapshot={privateLobby.snapshot}
                switchPrivateLobbyTeam={switchPrivateLobbyTeam}
                transferLobbyOwner={transferLobbyOwner}
                kickLobbyMember={kickLobbyMember}
                userId={userId}
              />
            ) : null}

            {privateLobby.isOwner && privateLobby.snapshot?.state === "open" ? (
              <LobbyActionButton
                type="button"
                onClick={() => void startPrivateLobby()}
                disabled={!canStart || privateLobby.busy}
                size="lg"
                className="w-full"
              >
                {privateLobby.busy ? <Loader2 className="animate-spin" size={18} /> : <Play size={18} fill="currentColor" />}
                {mode === "team_duel" ? "Start Team Duel" : mode === "free_for_all" ? "Start FFA" : "Start Duel"}
              </LobbyActionButton>
            ) : privateLobby.isMember && privateLobby.snapshot?.state === "open" ? (
              <LobbyInset className="text-center text-sm font-semibold text-[#a9bfd4]">
                Waiting for the leader to start.
              </LobbyInset>
            ) : null}
          </div>
        </div>
      </LobbyPanel>
    </motion.div>
  );
});

function PartyHeader({
  inviteCopied,
  matchInProgress,
  onCopyInvite,
  onLeave,
  privateLobby,
}: {
  inviteCopied: boolean;
  matchInProgress: boolean;
  onCopyInvite: () => void;
  onLeave: () => void;
  privateLobby: PrivateLobbyView;
}) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <span className="mb-2 block text-[12px] font-black uppercase tracking-[0.16em] text-[#77f0be]">
          CUSTOM
        </span>
        <h2 className="text-[34px] font-black leading-tight tracking-tight text-white sm:text-[42px]">
          Private Party
        </h2>
        <p className="mt-2 max-w-[48ch] text-[14px] leading-6 text-[#a9bfd4]">
          {matchInProgress
            ? "A game is in progress. Friends can still join the lobby and wait for the next one."
            : "Share the invite, wait for one opponent, then the leader starts the match."}
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {privateLobby.inviteCode ? (
          <LobbyActionButton type="button" variant="secondary" onClick={onCopyInvite}>
            <Copy className="text-[#77f0be]" size={16} />
            {inviteCopied ? "Copied" : "Copy Invite"}
          </LobbyActionButton>
        ) : null}
        {privateLobby.isMember && !(privateLobby.isOwner && matchInProgress) ? (
          <LobbyActionButton
            type="button"
            variant="secondary"
            onClick={onLeave}
            disabled={privateLobby.busy}
          >
            <LogOut size={16} />
            Leave
          </LobbyActionButton>
        ) : null}
      </div>
    </div>
  );
}

function PartySettings({
  busy,
  clockOn,
  config,
  isOwner,
  mode,
  pressureOn,
  pressureSeconds,
  roundSeconds,
  saveConfig,
  saveMode,
  setMapPickerOpen,
  snapshot,
}: {
  busy: boolean;
  clockOn: boolean;
  config: MatchConfig;
  isOwner: boolean;
  mode: PartyMode;
  pressureOn: boolean;
  pressureSeconds: number;
  roundSeconds: number;
  saveConfig: (patch: MatchConfig) => void;
  saveMode: (mode: PartyMode) => void;
  setMapPickerOpen: (open: boolean) => void;
  snapshot: LobbySnapshot;
}) {
  if (!(isOwner && snapshot.state === "open")) {
    return (
      <LobbyInset>
        <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
          Game Settings
        </p>
        <p className="mt-1 text-sm font-semibold text-white">
          {snapshot.mapName || config.mapId || "Official Map"} · {config.ruleset === "nmpz" ? "NMPZ" : "Moving"} ·{" "}
          {clockOn ? `${roundSeconds}s clock` : "Infinite clock"} ·{" "}
          {pressureOn ? `${pressureSeconds}s pressure` : "No pressure"}
        </p>
      </LobbyInset>
    );
  }

  return (
    <LobbyInset>
      <div className="grid w-full gap-3 sm:grid-cols-5">
        <label className="grid gap-1.5">
          <LobbyFieldLabel>Mode</LobbyFieldLabel>
          <LobbySelect
            value={mode}
            onChange={(event) => saveMode(event.target.value as PartyMode)}
            disabled={busy}
          >
            <option value="duel">Duel</option>
            <option value="team_duel">Team Duel</option>
            <option value="free_for_all">Free For All</option>
          </LobbySelect>
        </label>
        <label className="grid gap-1.5">
          <LobbyFieldLabel>Map</LobbyFieldLabel>
          <LobbyActionButton
            type="button"
            variant="secondary"
            size="md"
            onClick={() => setMapPickerOpen(true)}
            disabled={busy}
            className="min-w-0 justify-center normal-case tracking-normal"
          >
            <MapIcon className="shrink-0 text-[#77f0be]" size={14} />
            <span className="truncate">
              {config.mapName || config.mapId || "Select map"}
            </span>
          </LobbyActionButton>
        </label>
        <label className="grid gap-1.5">
          <LobbyFieldLabel>Rules</LobbyFieldLabel>
          <LobbySelect
            value={config.ruleset || "moving"}
            onChange={(event) => saveConfig({ ruleset: event.target.value as GameRuleset })}
            disabled={busy}
          >
            <option value="moving">Moving</option>
            <option value="nmpz">NMPZ</option>
          </LobbySelect>
        </label>
        <label className="grid gap-1.5">
          <LobbyFieldLabel>Clock</LobbyFieldLabel>
          <LobbySelect
            value={clockOn ? String(roundSeconds) : "infinite"}
            onChange={(event) => {
              const value = event.target.value;
              saveConfig(
                value === "infinite"
                  ? { roundTimerMode: "none", roundTimeLimitMs: undefined }
                  : { roundTimerMode: "fixed", roundTimeLimitMs: Number(value) * 1000 },
              );
            }}
            disabled={busy}
          >
            {CLOCK_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </LobbySelect>
        </label>
        <label className="grid gap-1.5">
          <LobbyFieldLabel>Pressure</LobbyFieldLabel>
          <LobbySelect
            value={pressureOn ? "15" : "none"}
            onChange={(event) => {
              const value = event.target.value;
              saveConfig({ pressureTimeLimitMs: value === "15" ? 15000 : undefined });
            }}
            disabled={busy}
          >
            {PRESSURE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </LobbySelect>
        </label>
      </div>
    </LobbyInset>
  );
}

function PartyMemberList({
  busy,
  isOwner,
  kickLobbyMember,
  members,
  mode,
  snapshot,
  switchPrivateLobbyTeam,
  transferLobbyOwner,
  userId,
}: {
  busy: boolean;
  isOwner: boolean;
  kickLobbyMember: (userId: string) => Promise<void>;
  members: LobbySnapshot["members"];
  mode: PartyMode;
  snapshot: LobbySnapshot;
  switchPrivateLobbyTeam: (teamId: LobbyTeamId) => Promise<void>;
  transferLobbyOwner: (userId: string) => Promise<void>;
  userId: string;
}) {
  return (
    <div className="grid gap-3">
      {members.map((member) => {
        const isLeader = member.userId === snapshot.ownerUserId;
        const isSelf = member.userId === userId;
        const presenceStatus = member.presenceStatus || (member.connected ? "online" : "offline");
        const lobbyStatus =
          presenceStatus === "online" ? "Online" : presenceStatus === "away" ? "Reconnecting" : "Offline";
        return (
          <LobbyInset
            key={member.userId}
            className={`flex min-h-[72px] flex-col gap-3 sm:flex-row sm:items-center sm:justify-between ${
              presenceStatus === "offline" ? "opacity-50" : ""
            }`}
          >
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <p className="truncate text-[16px] font-extrabold text-white">
                  {member.displayName || member.userId}
                </p>
                {isLeader ? (
                  <LobbyPill tone="success">
                    <Crown className="mr-1" size={12} />
                    Leader
                  </LobbyPill>
                ) : null}
              </div>
              <p className="mt-1 text-[12px] font-semibold text-[#a9bfd4]">
                {isSelf ? `You · ${lobbyStatus}` : lobbyStatus}
              </p>
              {mode === "team_duel" ? (
                <p className="mt-1 text-[12px] font-semibold uppercase tracking-[0.12em]">
                  <span className={lobbyTeamTextClass(member.teamId)}>{lobbyTeamLabel(member.teamId)}</span>
                </p>
              ) : null}
            </div>
            {mode === "team_duel" && isSelf && snapshot.state === "open" ? (
              <div className="flex gap-2">
                {(["a", "b"] as const).map((teamId) => (
                  <button
                    key={teamId}
                    type="button"
                    onClick={() => void switchPrivateLobbyTeam(teamId)}
                    disabled={busy || (member.teamId || "a") === teamId}
                    className={`inline-flex min-h-[36px] items-center rounded-[10px] px-3 text-[11px] font-extrabold uppercase tracking-[0.08em] transition disabled:opacity-50 ${lobbyTeamPillClass(
                      teamId,
                      (member.teamId || "a") === teamId,
                    )}`}
                  >
                    {lobbyTeamLabel(teamId)}
                  </button>
                ))}
              </div>
            ) : null}
            {isOwner && snapshot.state === "open" && !isSelf ? (
              <div className="flex flex-wrap gap-2">
                <LobbyActionButton
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => void transferLobbyOwner(member.userId)}
                  disabled={busy}
                >
                  <Crown size={14} />
                  Make Leader
                </LobbyActionButton>
                <LobbyActionButton
                  type="button"
                  variant="danger"
                  size="sm"
                  onClick={() => void kickLobbyMember(member.userId)}
                  disabled={busy}
                >
                  <UserMinus size={14} />
                  Kick
                </LobbyActionButton>
              </div>
            ) : null}
          </LobbyInset>
        );
      })}
    </div>
  );
}
