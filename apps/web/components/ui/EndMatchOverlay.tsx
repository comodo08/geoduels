import dynamic from "next/dynamic";
import { motion, AnimatePresence } from "framer-motion";
import { Flag, RotateCcw, List, LogOut } from "lucide-react";
import { useState, useMemo } from "react";
import {
  PlayerIdentityCard,
  PlayerIdentityRow,
  type ParticipantIdentityView,
} from "./PlayerIdentity";
import AppModalShell from "./AppModalShell";
import type { PlayerBadgeInfo } from "./PlayerBadge";
import type { RoundResult } from "./types";

const GuessMap = dynamic(() => import("../GuessMap"), { ssr: false });

type Props = {
  onLeaveGame: () => void;
  onPlayAgain?: () => Promise<string> | void;
  mode: EndMatchMode;
  outcome?: "win" | "lose" | "draw";
  selfName: string;
  opponentName?: string;
  opponentUserId?: string;
  selfElo?: number;
  opponentElo?: number;
  selfEloDelta?: number;
  opponentEloDelta?: number;
  selfHP: number;
  oppHP?: number;
  selfAvatarUrl?: string;
  oppAvatarUrl?: string;
  selfFallback: string;
  oppFallback?: string;
  selfAvatarColor?: string;
  oppAvatarColor?: string;
  selfIsAdmin: boolean;
  opponentIsAdmin?: boolean;
  selfSelectedBadge?: PlayerBadgeInfo | null;
  opponentSelectedBadge?: PlayerBadgeInfo | null;
  totalScore: number;
  roundResults: RoundResult[];
  resultPlayerNames: Record<string, string | undefined>;
  resultPlayerAvatars: Record<string, string | undefined>;
  resultPlayerFallbacks: Record<string, string | undefined>;
  resultPlayerBorderColors?: Record<string, string | undefined>;
  participantsById?: Record<string, ParticipantIdentityView>;
  selfParticipant?: ParticipantIdentityView;
  opponentParticipant?: ParticipantIdentityView;
  onReportPlayer?: (
    reportedUserId: string,
    category?: string,
    reason?: string,
  ) => Promise<void> | void;
  asPage?: boolean;
};

function formatGuessTime(ms?: number) {
  if (typeof ms !== "number" || !Number.isFinite(ms) || ms <= 0) return "";
  return `${(ms / 1000).toFixed(1)}s`;
}

type EndMatchMode = "duel" | "singleplayer" | "team_duel" | "free_for_all";

export default function EndMatchOverlay({
  onLeaveGame,
  mode,
  outcome,
  selfName,
  opponentName,
  opponentUserId,
  selfElo,
  opponentElo,
  selfEloDelta,
  opponentEloDelta,
  selfHP,
  oppHP,
  selfAvatarUrl,
  oppAvatarUrl,
  selfFallback,
  oppFallback,
  selfAvatarColor,
  oppAvatarColor,
  selfIsAdmin,
  opponentIsAdmin,
  selfSelectedBadge,
  opponentSelectedBadge,
  totalScore,
  roundResults,
  resultPlayerNames,
  resultPlayerAvatars,
  resultPlayerFallbacks,
  resultPlayerBorderColors,
  participantsById = {},
  selfParticipant,
  opponentParticipant,
  onReportPlayer,
  onPlayAgain,
  asPage = false,
}: Props) {
  const [reportedUserIds, setReportedUserIds] = useState<Record<string, boolean>>({});
  const [reportBusyUserId, setReportBusyUserId] = useState("");
  const [reportError, setReportError] = useState("");
  const [reportCategory, setReportCategory] = useState("cheating");
  const [reportReason, setReportReason] = useState("");
  const [pendingReport, setPendingReport] = useState<{ userId: string; name: string; } | null>(null);
  const [playAgainBusy, setPlayAgainBusy] = useState(false);
  const [showBreakdown, setShowBreakdown] = useState(false);

  const totalRounds = roundResults.length;
  const hasRoundResults = totalRounds > 0;
  const playerIds = Object.keys(roundResults[0]?.players || {});
  const selfPlayerId =
    playerIds.find((id) => resultPlayerNames[id] === selfName) ||
    playerIds[0] ||
    "self";
  const opponentPlayerId = playerIds.find(
    (id) => id !== selfPlayerId && resultPlayerNames[id] === opponentName,
  ) || playerIds.find((id) => id !== selfPlayerId) || "opp";

  const backLabel = mode === "singleplayer" ? "Back To Home" : "Back To Lobby";
  const showPlayAgain = mode === "singleplayer" && !!onPlayAgain;

  const isDuelsMode = mode === 'duel' || mode === 'team_duel';
  const isFFAMode = mode === 'free_for_all';
  const isSPMode = mode === 'singleplayer';

  const playerScores = useMemo(() => {
    const totals: Record<string, { score: number; time: number; distance: number }> = {};
    for (const round of roundResults) {
      for (const [pid, p] of Object.entries(round.players)) {
        if (!totals[pid]) totals[pid] = { score: 0, time: 0, distance: 0 };
        totals[pid].score += p.score;
        totals[pid].time += p.guessMs || 0;
        totals[pid].distance += p.distanceKm;
      }
    }
    const arr = Object.entries(totals).map(([id, stats]) => ({
      id,
      name: resultPlayerNames[id] || 'Unknown',
      avatar: resultPlayerAvatars[id],
      fallback: resultPlayerFallbacks[id] || '?',
      participant: participantsById[id] || {
        kind: "player" as const,
        id,
        name: resultPlayerNames[id] || 'Unknown',
        avatarUrl: resultPlayerAvatars[id],
        avatarFallback: resultPlayerFallbacks[id] || '?',
      },
      ...stats
    }));
    arr.sort((a, b) => b.score - a.score);
    return arr;
  }, [roundResults, resultPlayerNames, resultPlayerAvatars, resultPlayerFallbacks]);

  const ffaWinner = playerScores.length > 0 ? playerScores[0] : null;
  const myIndex = playerScores.findIndex(p => p.id === selfPlayerId);
  const myPlacement = myIndex >= 0 ? myIndex + 1 : 0;
  const placementText = myPlacement === 1 ? '1st' : myPlacement === 2 ? '2nd' : myPlacement === 3 ? '3rd' : `${myPlacement}th`;
  const outcomeLabel =
    outcome === "win" ? "Win" : outcome === "lose" ? "Defeat" : "Draw";

  const oppTotalScore = useMemo(() => {
    let total = 0;
    if (opponentPlayerId) {
      for (const round of roundResults) {
        if (round.players[opponentPlayerId]) {
          total += round.players[opponentPlayerId].score;
        }
      }
    }
    return total;
  }, [roundResults, opponentPlayerId]);

  async function handlePlayAgain() {
    if (!onPlayAgain || playAgainBusy) return;
    setPlayAgainBusy(true);
    try {
      await onPlayAgain();
    } finally {
      setPlayAgainBusy(false);
    }
  }

  function renderScoreCell(round: RoundResult, playerId: string | undefined, highlight = false) {
    if (!playerId) return <span className="text-white/35">-</span>;
    const player = round.players[playerId];
    if (!player) return <span className="text-white/35">-</span>;
    return (
      <div className={highlight ? "font-black text-white" : "font-bold text-[#dbe7ff]"}>
        <p>{player.score.toLocaleString()}</p>
        {formatGuessTime(player.guessMs) ? (
          <p className="mt-0.5 text-[11px] font-bold uppercase tracking-[0.1em] text-[#8caab0]">
            {formatGuessTime(player.guessMs)}
          </p>
        ) : null}
      </div>
    );
  }

  function renderPlayerCard({
    name,
    avatarUrl,
    fallback,
    elo,
    eloDelta,
    isAdmin,
    selectedBadge,
    reportUserId,
    avatarColor,
    opponent = false
  }: {
    name: string;
    avatarUrl?: string;
    fallback: string;
    elo?: number;
    eloDelta?: number;
    isAdmin?: boolean;
    selectedBadge?: PlayerBadgeInfo | null;
    reportUserId?: string;
    avatarColor?: string;
    opponent?: boolean;
  }) {
    const canReport = !!reportUserId && !!onReportPlayer && !reportedUserIds[reportUserId];
    const participant: ParticipantIdentityView = (!opponent ? selfParticipant : opponentParticipant) || (avatarColor
      ? {
        kind: "team",
        id: name,
        name,
        avatarFallback: fallback,
        avatarColor,
      }
      : {
        kind: "player",
        id: reportUserId || name,
        name,
        avatarUrl,
        avatarFallback: fallback,
        isAdmin,
        selectedBadge,
      });
    const reportButton = mode === 'duel' && reportUserId && onReportPlayer ? (
      <button
        type="button"
        title={reportedUserIds[reportUserId] ? "Report sent" : "Report player"}
        disabled={!canReport || reportBusyUserId === reportUserId}
        onClick={() => {
          setReportError("");
          setReportCategory("cheating");
          setReportReason("");
          setPendingReport({ userId: reportUserId, name });
        }}
        className="flex h-7 w-7 items-center justify-center rounded-full border border-red-300/25 bg-red-500/12 text-red-100 transition hover:bg-red-500/20 disabled:cursor-not-allowed disabled:opacity-55"
      >
        <Flag size={13} />
      </button>
    ) : null;
    return (
      <div className={`glass-panel flex flex-col items-center gap-3 rounded-[24px] p-6 text-center ${opponent ? 'bg-red-500/5 border-red-500/10' : 'bg-blue-500/5 border-blue-500/10'}`}>
        <div className="flex flex-col items-center">
          <PlayerIdentityCard
            participant={participant}
            opponent={opponent}
            rating={mode === "duel" ? elo : undefined}
            ratingDelta={mode === "duel" ? eloDelta : undefined}
            ratingAction={reportButton}
          />
        </div>
      </div>
    );
  }

  const breakdownTable = (
    <div className="mt-2">
      <div className="glass-panel rounded-[20px] p-1">
        {isDuelsMode ? (
          <div className="w-full overflow-x-auto">
            <table className="w-full text-left text-sm text-white">
              <thead>
                <tr className="border-b border-white/10 text-[#8caab0]">
                  <th className="py-3 px-4 font-bold uppercase tracking-wider">Round</th>
                  <th className="py-3 px-4 font-bold uppercase tracking-wider">{selfName}</th>
                  <th className="py-3 px-4 font-bold uppercase tracking-wider">{opponentName || "Opponent"}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {roundResults.map((round) => (
                  <tr key={round.roundId} className="hover:bg-white/[0.02]">
                    <td className="py-3 px-4 font-bold text-[#dce6ff]">R{round.roundNumber}</td>
                    <td className="py-3 px-4">{renderScoreCell(round, selfPlayerId, true)}</td>
                    <td className="py-3 px-4">{renderScoreCell(round, opponentPlayerId)}</td>
                  </tr>
                ))}
                <tr className="bg-white/5">
                  <td className="py-3 px-4 font-black">Total</td>
                  <td className="py-3 px-4 font-black text-[#2ad18f]">{totalScore.toLocaleString()}</td>
                  <td className="py-3 px-4 font-black text-white">{oppTotalScore.toLocaleString()}</td>
                </tr>
              </tbody>
            </table>
          </div>
        ) : (
          <div className="w-full overflow-x-auto">
            <table className="w-full text-left text-sm text-white whitespace-nowrap">
              <thead>
                <tr className="border-b border-white/10 text-[#8caab0]">
                  <th className="py-3 px-4 font-bold uppercase tracking-wider">Player</th>
                  {roundResults.map(r => (
                    <th key={r.roundId} className="py-3 px-4 font-bold uppercase tracking-wider">R{r.roundNumber}</th>
                  ))}
                  <th className="py-3 px-4 font-bold uppercase tracking-wider text-right">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {playerScores.map(player => (
                  <tr key={player.id} className={player.id === selfPlayerId ? 'bg-white/[0.04]' : 'hover:bg-white/[0.02]'}>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-3">
                        <PlayerIdentityRow
                          participant={player.participant}
                          nameClassName={player.id === selfPlayerId ? 'font-black text-[#7dc3ff]' : 'font-bold'}
                        />
                      </div>
                    </td>
                    {roundResults.map(r => (
                      <td key={r.roundId} className="py-3 px-4">
                        {renderScoreCell(r, player.id, player.id === selfPlayerId)}
                      </td>
                    ))}
                    <td className="py-3 px-4 text-right font-black text-[#2ad18f]">{player.score.toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className={`flex flex-col ${asPage ? "app-layer-match-end fixed inset-0 bg-[#0d1216]" : "absolute inset-0 z-50 bg-[#0d1216]/90 backdrop-blur-md"}`}
    >
      <div className="flex-1 min-h-0 w-full relative">
        {hasRoundResults ? (
          <GuessMap
            mode="result"
            results={roundResults}
            interactiveInResult
            resultPlayerAvatars={resultPlayerAvatars}
            resultPlayerFallbacks={resultPlayerFallbacks}
            resultPlayerBorderColors={resultPlayerBorderColors}
          />
        ) : (
          <div className="flex h-full items-center justify-center bg-black/40 text-center text-sm font-semibold text-[#a9bfd4]">
            The match ended before any round results were recorded.
          </div>
        )}
      </div>

      <div className="shrink-0 bg-[#070b0e] border-t border-white/10 p-6 md:p-8 shadow-[0_-10px_50px_rgba(0,0,0,0.6)] relative z-10 max-h-[50vh] overflow-y-auto">
        <div className="mx-auto max-w-6xl">
          {reportError && <p className="mb-4 text-center text-sm font-bold text-red-400">{reportError}</p>}

          {showBreakdown ? (
            <div className="w-full flex flex-col">
              <div className="mb-6 flex items-center justify-between">
                <h2 className="text-xl font-black uppercase tracking-[0.1em] text-white">Match Breakdown</h2>
                <button
                  type="button"
                  onClick={() => setShowBreakdown(false)}
                  className="inline-flex items-center rounded-full bg-white/10 px-6 py-2.5 text-sm font-bold uppercase tracking-[0.1em] text-white transition hover:bg-white/20 hover:scale-105 active:scale-95"
                >
                  Back
                </button>
              </div>
              {breakdownTable}
            </div>
          ) : (
            <div>
              <h2 className="sr-only">Match Complete</h2>
              {isDuelsMode ? (
                <div className="flex flex-col md:flex-row items-center justify-center gap-6 md:gap-12">
                  <div className="flex-1 max-w-[320px] w-full">
                    {renderPlayerCard({
                      name: selfName,
                      avatarUrl: selfAvatarUrl,
                      fallback: selfFallback,
                      elo: selfElo,
                      eloDelta: selfEloDelta,
                      isAdmin: selfIsAdmin,
                      selectedBadge: selfSelectedBadge,
                      avatarColor: selfAvatarColor
                    })}
                  </div>
                  <div className="flex flex-col items-center gap-4">
                    <p className={`text-[12px] font-black uppercase tracking-[0.2em] ${
                      outcome === "lose" ? "text-red-300" : outcome === "draw" ? "text-[#dbe7ff]" : "text-[#2ad18f]"
                    }`}>
                      {outcomeLabel}
                    </p>
                    <button
                      type="button"
                      className="group relative overflow-hidden rounded-full px-10 py-4 font-black uppercase tracking-[0.15em] text-white bg-[linear-gradient(135deg,#2ad18f_0%,#12a86f_100%)] shadow-[0_0_20px_rgba(42,209,143,0.3)] hover:shadow-[0_0_30px_rgba(42,209,143,0.5)] transition-all hover:scale-105 active:scale-95"
                      onClick={onLeaveGame}
                    >
                      <span className="relative z-10">{backLabel}</span>
                      <div className="absolute inset-0 bg-white/20 opacity-0 transition-opacity group-hover:opacity-100" />
                    </button>
                    <button
                      type="button"
                      className="inline-flex h-12 items-center justify-center gap-2 rounded-full border border-white/10 bg-white/5 px-6 text-white transition hover:bg-white/10 hover:scale-105 active:scale-95"
                      onClick={() => setShowBreakdown(true)}
                      title="Toggle Breakdown"
                    >
                      <List size={20} />
                      <span className="text-sm font-bold uppercase tracking-[0.1em]">Breakdown</span>
                    </button>
                  </div>
                  <div className="flex-1 max-w-[320px] w-full">
                    {renderPlayerCard({
                      name: opponentName || "Opponent",
                      avatarUrl: oppAvatarUrl,
                      fallback: oppFallback || "O",
                      elo: opponentElo,
                      eloDelta: opponentEloDelta,
                      isAdmin: opponentIsAdmin,
                      selectedBadge: opponentSelectedBadge,
                      opponent: true,
                      reportUserId: mode === "team_duel" ? undefined : (opponentUserId || opponentPlayerId),
                      avatarColor: oppAvatarColor
                    })}
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center text-center">
                  {isFFAMode && ffaWinner && (
                    <p className="mb-2 text-sm font-bold uppercase tracking-[0.2em] text-[#2ad18f]">
                      {ffaWinner.id === selfPlayerId ? 'You won!' : `${ffaWinner.name} won!`}
                    </p>
                  )}
                  {isFFAMode && (
                    <p className="mb-4 text-xs font-semibold uppercase tracking-[0.15em] text-[#8caab0]">
                      You placed <span className="text-white">{placementText}</span>
                    </p>
                  )}
                  <h2 className="text-5xl font-black text-white drop-shadow-[0_0_30px_rgba(125,195,255,0.4)] md:text-6xl">
                    {totalScore.toLocaleString()} <span className="text-xl text-[#7dc3ff] md:text-2xl">pts</span>
                  </h2>

                  <div className="mt-8 flex items-center justify-center gap-4">
                    <button
                      type="button"
                      className="inline-flex h-12 items-center justify-center gap-2 rounded-full border border-white/10 bg-white/5 px-6 text-white transition hover:bg-white/10 hover:scale-105 active:scale-95"
                      onClick={() => setShowBreakdown(true)}
                      title="Toggle Breakdown"
                    >
                      <List size={20} />
                      <span className="text-sm font-bold uppercase tracking-[0.1em]">Breakdown</span>
                    </button>
                    <button
                      type="button"
                      className="group relative overflow-hidden rounded-full px-10 py-4 font-black uppercase tracking-[0.15em] text-white bg-[linear-gradient(135deg,#2ad18f_0%,#12a86f_100%)] shadow-[0_0_20px_rgba(42,209,143,0.3)] hover:shadow-[0_0_30px_rgba(42,209,143,0.5)] transition-all hover:scale-105 active:scale-95"
                      onClick={onLeaveGame}
                    >
                      <span className="relative z-10">{backLabel}</span>
                      <div className="absolute inset-0 bg-white/20 opacity-0 transition-opacity group-hover:opacity-100" />
                    </button>
                    {showPlayAgain && (
                      <button
                        type="button"
                        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-white/10 text-white transition hover:bg-white/20 hover:scale-105 active:scale-95"
                        onClick={() => void handlePlayAgain()}
                        disabled={playAgainBusy}
                        title="Play Again"
                      >
                        <RotateCcw size={20} className={playAgainBusy ? "animate-spin" : ""} />
                      </button>
                    )}
                    <button
                      type="button"
                      className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full border border-red-500/20 bg-red-500/10 text-red-200 transition hover:bg-red-500/20 hover:scale-105 active:scale-95"
                      onClick={onLeaveGame}
                      title="Exit"
                    >
                      <LogOut size={20} />
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {pendingReport ? (
        <AppModalShell
          title="Report Player"
          onClose={() => {
            setPendingReport(null);
            setReportReason("");
          }}
          placement="center"
          showHeader={false}
          zIndexClassName="z-[1200]"
          maxWidthClassName="max-w-md"
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full border border-red-300/25 bg-red-500/15 text-red-100">
              <Flag size={18} />
            </div>
            <div>
              <p className="text-[11px] font-black uppercase tracking-[0.18em] text-red-200">
                Report Player
              </p>
              <h3 className="mt-1 text-xl font-black">
                Report {pendingReport.name}?
              </h3>
            </div>
          </div>
          <p className="mt-4 text-sm leading-6 text-[#c5d4e2]">
            Reports are for suspected cheating or abuse, including bad
            profiles, toxicity, or harassment.
          </p>
          <div className="mt-5 grid gap-2">
            {[
              ["cheating", "Cheating"],
              ["boosting", "Boosting / throwing"],
              ["harassment", "Harassment"],
              ["profile", "Offensive profile"],
              ["other", "Other"],
            ].map(([value, label]) => (
              <button
                key={value}
                type="button"
                onClick={() => setReportCategory(value)}
                className={`min-h-10 rounded-xl border px-3 text-left text-sm font-bold transition ${reportCategory === value
                    ? "border-red-200/55 bg-red-500/25 text-red-50"
                    : "border-white/10 bg-white/5 text-[#c5d4e2] hover:bg-white/10"
                  }`}
              >
                {label}
              </button>
            ))}
          </div>
          <textarea
            value={reportReason}
            onChange={(event) => setReportReason(event.target.value)}
            maxLength={1000}
            placeholder="Optional details"
            className="mt-4 min-h-24 w-full resize-none rounded-xl border border-white/10 bg-[#0d141c] px-3 py-2 text-sm text-white outline-none placeholder:text-[#6f8aa5] focus:border-red-200/50"
          />
          <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:justify-end">
            <button
              type="button"
              disabled={reportBusyUserId === pendingReport.userId}
              onClick={() => {
                setPendingReport(null);
                setReportReason("");
              }}
              className="min-h-11 rounded-xl border border-white/10 bg-white/10 px-4 text-sm font-bold text-white transition hover:bg-white/15 disabled:opacity-60"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={reportBusyUserId === pendingReport.userId}
              onClick={async () => {
                if (!onReportPlayer) return;
                setReportError("");
                setReportBusyUserId(pendingReport.userId);
                try {
                  await onReportPlayer(
                    pendingReport.userId,
                    reportCategory,
                    reportReason,
                  );
                  setReportedUserIds((current) => ({
                    ...current,
                    [pendingReport.userId]: true,
                  }));
                  setPendingReport(null);
                  setReportReason("");
                } catch (error) {
                  setReportError(
                    error instanceof Error
                      ? error.message
                      : "Failed to send report",
                  );
                } finally {
                  setReportBusyUserId("");
                }
              }}
              className="min-h-11 rounded-xl border border-red-300/35 bg-red-500/20 px-4 text-sm font-black text-red-50 transition hover:bg-red-500/30 disabled:opacity-60"
            >
              {reportBusyUserId === pendingReport.userId
                ? "Sending..."
                : "Send report"}
            </button>
          </div>
        </AppModalShell>
      ) : null}
    </motion.div>
  );
}
