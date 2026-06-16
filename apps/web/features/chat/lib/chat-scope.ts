import type { LobbyRuntimeState } from "../../lobby/controllers/lobby-controller";
import type { MatchState } from "../../matchmaking/controllers/match-controller";

export function selectActiveChatConversationId(params: {
  userId: string;
  lobby: LobbyRuntimeState;
  match: MatchState;
}) {
  const lobbySnapshot = params.lobby.snapshot;
  if (
    lobbySnapshot?.id &&
    lobbySnapshot.members.some((member) => member.userId === params.userId)
  ) {
    return `party:${lobbySnapshot.id}`;
  }
  if (params.match.sourcePartyId) {
    return `party:${params.match.sourcePartyId}`;
  }
  if (params.match.snapshot?.matchId && params.match.snapshot.mode !== "singleplayer") {
    return `match:${params.match.snapshot.matchId}`;
  }
  return "";
}
