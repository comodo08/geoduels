import Head from "next/head";
import Link from "next/link";
import { useRouter } from "next/router";
import { useMemo } from "react";
import EndMatchOverlay from "../../components/ui/EndMatchOverlay";
import type { Snapshot } from "../../components/ui/types";
import { requestMatchReport } from "../../features/auth/lib/auth-client";
import HomePageChatDock from "../../features/home/page/HomePageChatDock";
import HomePageGame from "../../features/home/page/HomePageGame";
import HomePageOverlays from "../../features/home/page/HomePageOverlays";
import { useHomeModel } from "../../features/home/model/useHomeModel";
import { useMatchRouteSession } from "../../features/matchmaking/hooks/use-match-route-session";
import { getRuntimeConfig } from "../../lib/runtime-config";
import { getSiteURL } from "../../lib/site";
import type { MatchSessionResponse } from "../../features/matchmaking/lib/queue-client";

export function normalizeRouteMatchId(
  raw: string | string[] | undefined,
  asPath: string,
) {
  if (typeof raw === "string") {
    const value = raw.trim();
    return /^\[[^/]+\]$/.test(value) ? "" : value;
  }
  const pathMatch = asPath.match(/^\/match\/([^?#/]+)/);
  if (pathMatch?.[1]) {
    try {
      const value = decodeURIComponent(pathMatch[1]).trim();
      return /^\[[^/]+\]$/.test(value) ? "" : value;
    } catch {
      const value = pathMatch[1].trim();
      return /^\[[^/]+\]$/.test(value) ? "" : value;
    }
  }
  return "";
}

function buildHistoryOverlay(
  snapshot: Snapshot,
  userId: string,
  displayName: string,
  userAvatar: string,
) {
  const playerIds = Object.keys(snapshot.players || {});
  const selfPlayer =
    snapshot.players[userId] || snapshot.players[playerIds[0] || ""];
  const opponentId = playerIds.find((id) => id !== selfPlayer?.userId) || "";
  const opponentPlayer = opponentId ? snapshot.players[opponentId] : undefined;
  const mode = snapshot.mode || "duel";
  const roundResults =
    snapshot.roundResults && snapshot.roundResults.length > 0
      ? snapshot.roundResults
      : snapshot.lastRoundResult
        ? [snapshot.lastRoundResult]
        : [];
  const resultPlayerNames: Record<string, string | undefined> = {};
  const resultPlayerAvatars: Record<string, string | undefined> = {};
  const resultPlayerFallbacks: Record<string, string | undefined> = {};
  const resultPlayerBorderColors: Record<string, string | undefined> = {};
  Object.entries(snapshot.players || {}).forEach(([id, player]) => {
    resultPlayerNames[id] = player.displayName || player.userId;
    resultPlayerAvatars[id] = player.avatarUrl;
    resultPlayerBorderColors[id] =
      mode === "team_duel" ? (player.teamId === "b" ? "#2563eb" : "#dc2626") : undefined;
    resultPlayerFallbacks[id] = (player.displayName || player.userId || "P")
      .slice(0, 1)
      .toUpperCase();
  });
  const selfName = selfPlayer?.displayName || displayName || "You";
  const opponentName = opponentPlayer?.displayName || "Opponent";
  const selfIsAdmin = !!selfPlayer?.isAdmin;
  const opponentIsAdmin = !!opponentPlayer?.isAdmin;
  const selfHP = selfPlayer?.hp || 0;
  const oppHP = opponentPlayer?.hp || 0;
  const outcome: "win" | "lose" | "draw" | undefined =
    mode === "singleplayer"
      ? undefined
      : selfHP === oppHP
        ? "draw"
        : selfHP > oppHP
          ? "win"
          : "lose";

  return {
    mode,
    outcome,
    selfName,
    opponentName: mode === "singleplayer" || mode === "free_for_all" ? undefined : opponentName,
    opponentUserId: mode === "singleplayer" || mode === "free_for_all" ? undefined : opponentId,
    selfElo: mode === "singleplayer" || mode === "free_for_all" ? undefined : selfPlayer?.mmr,
    opponentElo: mode === "singleplayer" || mode === "free_for_all" ? undefined : opponentPlayer?.mmr,
    selfHP,
    oppHP: mode === "singleplayer" || mode === "free_for_all" ? undefined : oppHP,
    selfAvatarUrl: selfPlayer?.avatarUrl || userAvatar,
    oppAvatarUrl:
      mode === "singleplayer" || mode === "free_for_all" ? undefined : opponentPlayer?.avatarUrl,
    selfFallback: (selfName || "Y").slice(0, 1).toUpperCase(),
    oppFallback:
      mode === "singleplayer" || mode === "free_for_all"
        ? undefined
        : (opponentName || "O").slice(0, 1).toUpperCase(),
    selfIsAdmin,
    opponentIsAdmin: mode === "singleplayer" || mode === "free_for_all" ? undefined : opponentIsAdmin,
    totalScore: selfPlayer?.totalScore || 0,
    roundResults,
    resultPlayerNames,
    resultPlayerAvatars,
    resultPlayerFallbacks,
    resultPlayerBorderColors,
  };
}

function matchSourcePartyInviteCode(
  response: MatchSessionResponse | null,
): string {
  if (!response) return "";
  if (
    response.status === "live_connectable" ||
    response.status === "history" ||
    response.status === "replaced"
  ) {
    return response.sourcePartyInviteCode || "";
  }
  return "";
}

export default function MatchPage() {
  const router = useRouter();
  const routeMatchId = router.isReady
    ? normalizeRouteMatchId(router.query.id, router.asPath)
    : "";
  const model = useHomeModel({
    routeMatchId: routeMatchId || null,
    routeContext: "match",
  });
  const config = getRuntimeConfig();
  const routeSession = useMatchRouteSession(routeMatchId || null);
  const siteURL = getSiteURL();
  const canonicalURL = routeMatchId
    ? `${siteURL}/match/${encodeURIComponent(routeMatchId)}`
    : `${siteURL}/`;
  const handleLeaveToParty = () => {
    const sourcePartyInviteCode =
      model.view.meta.sourcePartyInviteCode ||
      matchSourcePartyInviteCode(routeSession.replacement) ||
      "";
    model.actions.leaveGame();
    void router.push(
      sourcePartyInviteCode
        ? `/party/${encodeURIComponent(sourcePartyInviteCode)}`
        : "/",
    );
  };
  const handlePlayAgain = async () => {
    const nextMatchId = await model.actions.startSingleplayer();
    if (nextMatchId) {
      void router.push(`/match/${encodeURIComponent(nextMatchId)}`);
    }
    return nextMatchId;
  };
  const handleHistoryReport = async (
    reportedUserId: string,
    category = "cheating",
    reason = "",
  ) => {
    if (!model.view.auth.accessToken || !routeMatchId) {
      throw new Error("Report unavailable");
    }
    await requestMatchReport(
      config,
      model.view.auth.accessToken,
      routeMatchId,
      reportedUserId,
      category,
      reason,
    );
  };

  const historyOverlay = useMemo(
    () =>
      routeSession.historySnapshot
        ? buildHistoryOverlay(
            routeSession.historySnapshot,
            model.view.auth.userId,
            model.view.auth.displayName,
            model.view.auth.userAvatar,
          )
        : null,
    [
      routeSession.historySnapshot,
      model.view.auth.displayName,
      model.view.auth.userAvatar,
      model.view.auth.userId,
    ],
  );

  const loadingLabel = useMemo(() => {
    switch (routeSession.status) {
      case "bootstrapping_auth":
        return "Restoring session...";
      case "resolving":
        return "Reconnecting to match...";
      case "awaiting_first_snapshot":
        return "Joining live match...";
      case "replaced":
        return "This match was replaced";
      case "forbidden":
        return "Sign in to view this match";
      case "missing":
        return "Match unavailable";
      default:
        return "Loading match...";
    }
  }, [routeSession.status]);
  const replacementMatchId =
    routeSession.replacement?.status === "replaced"
      ? routeSession.replacement.replacementMatchId
      : "";

  return (
    <>
      <Head>
        <title>GeoDuels | Match</title>
        <meta name="robots" content="noindex,nofollow" />
        <link rel="canonical" href={canonicalURL} />
      </Head>
      <main className="relative min-h-screen overflow-hidden bg-[#08111b] text-ink">
        <HomePageOverlays
          auth={model.view.auth}
          overlays={model.view.overlays}
          actions={{
            ...model.actions,
            leaveGame: handleLeaveToParty,
            startSingleplayer: handlePlayAgain,
          }}
        />
        <HomePageGame
          game={model.view.game}
          maxHP={model.view.meta.maxHP}
          actions={{ ...model.actions, leaveGame: handleLeaveToParty }}
        />
        <HomePageChatDock chat={model.view.chat} actions={model.actions} />
        {!model.view.game.inGame &&
          !model.view.overlays.endMatch.open &&
          historyOverlay && (
            <EndMatchOverlay
              onLeaveGame={handleLeaveToParty}
              mode={historyOverlay.mode}
              outcome={historyOverlay.outcome}
              selfName={historyOverlay.selfName}
              opponentName={historyOverlay.opponentName}
              opponentUserId={historyOverlay.opponentUserId}
              selfElo={historyOverlay.selfElo}
              opponentElo={historyOverlay.opponentElo}
              selfHP={historyOverlay.selfHP}
              oppHP={historyOverlay.oppHP}
              selfAvatarUrl={historyOverlay.selfAvatarUrl}
              oppAvatarUrl={historyOverlay.oppAvatarUrl}
              selfFallback={historyOverlay.selfFallback}
              oppFallback={historyOverlay.oppFallback}
              selfIsAdmin={historyOverlay.selfIsAdmin}
              opponentIsAdmin={historyOverlay.opponentIsAdmin}
              totalScore={historyOverlay.totalScore}
              roundResults={historyOverlay.roundResults}
              resultPlayerNames={historyOverlay.resultPlayerNames}
              resultPlayerAvatars={historyOverlay.resultPlayerAvatars}
              resultPlayerFallbacks={historyOverlay.resultPlayerFallbacks}
              resultPlayerBorderColors={historyOverlay.resultPlayerBorderColors}
              onReportPlayer={handleHistoryReport}
              onPlayAgain={
                historyOverlay.mode === "singleplayer"
                  ? handlePlayAgain
                  : undefined
              }
              asPage
            />
          )}
        {!model.view.game.inGame &&
          !model.view.overlays.endMatch.open &&
          !historyOverlay && (
            <div className="flex min-h-screen items-center justify-center p-6">
              <div className="w-full max-w-md rounded-[28px] border border-white/10 bg-white/5 p-8 text-center text-white shadow-[0_24px_60px_rgba(0,0,0,0.38)] backdrop-blur-xl">
                {routeSession.status === "bootstrapping_auth" ||
                routeSession.status === "resolving" ||
                routeSession.status === "awaiting_first_snapshot" ? (
                  <div className="flex flex-col items-center py-6">
                    <div className="h-10 w-10 animate-spin rounded-full border-2 border-white/15 border-t-[#7fb3d8]" />
                  </div>
                ) : (
                  <p className="text-[12px] font-bold uppercase tracking-[0.16em] text-[#7fb3d8]">
                    Match Session
                  </p>
                )}
                <h1 className="mt-3 text-[28px] font-black tracking-tight">
                  {loadingLabel}
                </h1>
                {routeSession.status === "replaced" &&
                routeSession.replacement?.status === "replaced" &&
                routeSession.replacement.replacement ? (
                  <button
                    type="button"
                    onClick={() =>
                      void router.replace(
                        `/match/${encodeURIComponent(replacementMatchId)}`,
                      )
                    }
                    className="mt-6 inline-flex rounded-full border border-[#2ad18f]/40 bg-[#2ad18f]/10 px-5 py-2.5 text-[12px] font-extrabold uppercase tracking-[0.1em] text-[#b6f5d8] transition hover:bg-[#2ad18f]/20"
                  >
                    Resume Current Match
                  </button>
                ) : null}
                <Link
                  href="/"
                  className="mt-4 inline-flex rounded-full border border-white/10 bg-white/10 px-5 py-2.5 text-[12px] font-extrabold uppercase tracking-[0.1em] text-white transition hover:bg-white/15"
                >
                  Back To Party
                </Link>
              </div>
            </div>
          )}
      </main>
    </>
  );
}
