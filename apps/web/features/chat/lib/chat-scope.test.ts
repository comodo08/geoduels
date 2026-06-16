import { describe, expect, it } from "vitest";
import type { Snapshot } from "../../../components/ui/types";
import type { LobbyRuntimeState } from "../../lobby/controllers/lobby-controller";
import type { MatchState } from "../../matchmaking/controllers/match-controller";
import { selectActiveChatConversationId } from "./chat-scope";

function lobbyState(patch: Partial<LobbyRuntimeState> = {}): LobbyRuntimeState {
  return {
    status: "idle",
    lobbyId: "",
    inviteCode: "",
    snapshot: null,
    error: "",
    ...patch,
  };
}

function matchState(patch: Partial<MatchState> = {}): MatchState {
  return {
    matchmaking: {
      status: "ready",
      intentVersion: 0,
      activeRecoverRequestID: null,
      queueStartedAt: null,
    },
    connected: false,
    snapshot: null,
    activeMatchId: "",
    sourcePartyId: "",
    sourcePartyInviteCode: "",
    queueError: "",
    connectionIssue: "",
    onlinePlayers: 0,
    ...patch,
  };
}

function snapshot(matchId: string, mode: Snapshot["mode"] = "duel"): Snapshot {
  return {
    matchId,
    mode,
    state: "live",
    phase: "live",
    roundPhase: "round_live",
    phaseStartedAt: 0,
    phaseEndsAt: 0,
    roundMsLeft: 0,
    players: {},
    eventSequence: 1,
  };
}

describe("selectActiveChatConversationId", () => {
  it("prefers the joined lobby conversation before a match starts", () => {
    expect(
      selectActiveChatConversationId({
        userId: "u1",
        lobby: lobbyState({
          snapshot: {
            id: "lobby-1",
            inviteCode: "ABCD",
            ownerUserId: "u1",
            state: "open",
            mode: "duel",
            mapScope: "official",
            config: {},
            activeMatchId: "",
            startedMatchId: "",
            members: [
              {
                userId: "u1",
                displayName: "One",
                role: "owner",
                connected: true,
              },
            ],
          },
        }),
        match: matchState(),
      }),
    ).toBe("party:lobby-1");
  });

  it("keeps private lobby chat during the sourced match", () => {
    expect(
      selectActiveChatConversationId({
        userId: "u1",
        lobby: lobbyState(),
        match: matchState({
          sourcePartyId: "lobby-1",
          snapshot: snapshot("match-1"),
        }),
      }),
    ).toBe("party:lobby-1");
  });

  it("uses match chat for regular multiplayer and disables singleplayer", () => {
    expect(
      selectActiveChatConversationId({
        userId: "u1",
        lobby: lobbyState(),
        match: matchState({ snapshot: snapshot("ranked-1") }),
      }),
    ).toBe("match:ranked-1");

    expect(
      selectActiveChatConversationId({
        userId: "u1",
        lobby: lobbyState(),
        match: matchState({ snapshot: snapshot("solo-1", "singleplayer") }),
      }),
    ).toBe("");
  });
});
