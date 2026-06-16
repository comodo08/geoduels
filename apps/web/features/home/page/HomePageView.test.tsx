import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { HomeModel } from "../model/types";
import HomePageView from "./HomePageView";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("next/dynamic", () => ({
  default: () => () => null,
}));

function createModel(overrides?: Partial<HomeModel["view"]>): HomeModel {
  return {
    view: {
      auth: {
        userId: "self",
        accessToken: "access-token",
        userEmail: "self@example.com",
        displayName: "Self",
        userAvatar: "",
        onboardingRequired: true,
        isAdmin: false,
        isGuest: false,
        nicknameInput: "Self",
        nicknameError: "",
        nicknameSaving: false,
        authLoading: false,
        authError: "",
        googleSignInEnabled: true,
        googleClientId: "google-client",
      },
      lobby: {
        inGame: false,
        connected: true,
        mmr: 1200,
        gamesPlayed: 10,
        winsPct: 60,
        leaderboard: null,
        leaderboardLoading: false,
        status: "ready",
        queueStartedAt: null,
        queueError: "",
        onlinePlayers: 42,
        canStartSingleplayer: true,
        maintenance: null,
        changelogEyebrow: "News",
        changelogTitle: "Latest",
        changelogMarkdown: "",
        changelogSlug: "",
        changelogUpdatedAt: "",
        privateLobby: {
          status: "idle",
          snapshot: null,
          inviteCode: "",
          isMember: false,
          isOwner: false,
          busy: false,
          error: "",
        },
      },
      game: {
        inGame: true,
        mode: "duel",
        isSingleplayer: false,
        isPointsMode: false,
        uiPhase: "match_end",
        showResultStage: false,
        showMatchEndPage: true,
        streetViewSrc: "",
        roundResult: undefined,
        roundResults: [],
        resultOverlay: undefined,
        resultPlayerNames: { self: "Self", opp: "Opponent" },
        resultPlayerAvatars: {},
        resultPlayerFallbacks: {},
        resultPlayerBorderColors: {},
        participantsById: {
          self: { kind: "player", id: "self", name: "Self", avatarFallback: "S" },
          opp: { kind: "player", id: "opp", name: "Opponent", avatarFallback: "O" },
        },
        selfParticipant: { kind: "player", id: "self", name: "Self", avatarFallback: "S" },
        opponentParticipant: { kind: "player", id: "opp", name: "Opponent", avatarFallback: "O" },
        selfName: "Self",
        selfAvatarUrl: "",
        selfFallback: "S",
        selfIsAdmin: false,
        opponentName: "Opponent",
        opponentIsAdmin: false,
        opponentDisconnected: false,
        oppAvatarUrl: "",
        oppFallback: "O",
        mm: "00",
        ss: "00",
        isRoundTimerRunning: false,
        timerProgressPct: 0,
        isTimerCritical: false,
        isTimerPulseActive: false,
        resultMode: true,
        selfHP: 5000,
        oppHP: 0,
        totalScore: 0,
        currentRoundScore: 0,
        currentRoundDistanceKm: 0,
        canFinalizeGuess: false,
        canAdvanceRound: false,
        guess: undefined,
        currentRoundId: "round-1",
        currentRoundNumber: 1,
        userAvatar: "",
        selfElo: 1200,
        opponentElo: 1100,
        damageMultiplier: 1,
        guessSubmitted: false,
        opponentGuessAlert: false,
        connectionIssue: "",
        modeName: "Moving",
        mapName: "A Source World",
        streetViewInteractive: true,
        selfUserId: "self",
      },
      chat: {
        conversationId: "",
        messages: [],
        selfUserId: "self",
        error: "",
      },
      overlays: {
        onboardingOpen: true,
        notifications: [],
        guestVerification: {
          open: false,
          siteKey: "",
          status: "checking",
          error: "",
          resetKey: 0,
        },
        endMatch: {
          open: true,
          mode: "duel",
          outcome: "win",
          selfName: "Self",
          opponentName: "Opponent",
          selfElo: 1200,
          opponentElo: 1100,
          selfEloDelta: 15,
          opponentEloDelta: -15,
          selfHP: 5000,
          oppHP: 0,
          selfIsAdmin: false,
          opponentIsAdmin: false,
          selfAvatarUrl: "",
          oppAvatarUrl: "",
          selfFallback: "S",
          oppFallback: "O",
          totalScore: 0,
          roundResults: [
            {
              roundId: "round-1",
              roundNumber: 1,
              actualLocation: { lat: 0, lng: 0 },
              players: {
                self: {
                  userId: "self",
                  lat: 1,
                  lng: 1,
                  score: 4000,
                  distanceKm: 20,
                },
                opp: {
                  userId: "opp",
                  lat: 5,
                  lng: 5,
                  score: 1000,
                  distanceKm: 500,
                },
              },
            },
          ],
          resultPlayerNames: { self: "Self", opp: "Opponent" },
          resultPlayerAvatars: { self: "", opp: "" },
          resultPlayerFallbacks: { self: "S", opp: "O" },
          resultPlayerBorderColors: {},
          participantsById: {
            self: { kind: "player", id: "self", name: "Self", avatarFallback: "S" },
            opp: { kind: "player", id: "opp", name: "Opponent", avatarFallback: "O" },
          },
          selfParticipant: { kind: "player", id: "self", name: "Self", avatarFallback: "S" },
          opponentParticipant: { kind: "player", id: "opp", name: "Opponent", avatarFallback: "O" },
        },
      },
      meta: {
        activeMatchId: "match-1",
        sourcePartyInviteCode: "",
        appVersion: "dev",
        maxHP: 6000,
      },
      ...overrides,
    },
    actions: {
      joinQueue: vi.fn(),
      startSingleplayer: vi.fn(),
      startSupportDonation: vi.fn(),
      cancelQueue: vi.fn(),
      placeGuess: vi.fn(),
      finalizeGuess: vi.fn(),
      advanceRound: vi.fn(() => true),
      forfeitMatch: vi.fn(() => true),
      leaveGame: vi.fn(),
      sendChatMessage: vi.fn(() => true),
      sendChatEmote: vi.fn(() => true),
      reportPlayer: vi.fn(async () => {}),
      createInviteLobby: vi.fn(async () => true),
      joinInviteLobby: vi.fn(async () => true),
      leavePrivateLobby: vi.fn(async () => {}),
      kickLobbyMember: vi.fn(async () => {}),
      transferLobbyOwner: vi.fn(async () => {}),
      startPrivateLobby: vi.fn(async () => {}),
      updatePrivateLobbySettings: vi.fn(async () => {}),
      switchPrivateLobbyTeam: vi.fn(async () => {}),
      devLogin: vi.fn(async () => null),
      triggerGoogleSignIn: vi.fn(async () => {}),
      triggerDiscordSignIn: vi.fn(async () => {}),
      linkAuthProvider: vi.fn(async () => {}),
      upgradeGuestWithProvider: vi.fn(async () => {}),
      unlinkAuthProvider: vi.fn(async () => {}),
      loadLeaderboard: vi.fn(),
      clearAuthSession: vi.fn(),
      deleteAccount: vi.fn(async () => {}),
      submitOnboardingNickname: vi.fn(async () => {}),
      submitProfileNickname: vi.fn(async () => true),
      selectBadge: vi.fn(async () => {}),
      setNicknameInput: vi.fn(),
      dismissNotification: vi.fn(async () => {}),
      submitGuestVerificationToken: vi.fn(),
      markGuestVerificationExpired: vi.fn(),
      cancelGuestVerification: vi.fn(),
    },
  };
}

describe("HomePageView", () => {
  it("renders onboarding and end match overlays while hiding the game scene", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={client}><HomePageView model={createModel()} /></QueryClientProvider>);

    expect(screen.getByText("Choose Your Nickname")).toBeInTheDocument();
    expect(screen.getByText("Match Complete")).toBeInTheDocument();
    expect(screen.queryByTitle("Street View")).not.toBeInTheDocument();
  });

  it("keeps chat on the top app layer over the match end page", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}><HomePageView
        model={createModel({
          chat: {
            conversationId: "match:match-1",
            messages: [],
            selfUserId: "self",
            error: "",
          },
        })}
      /></QueryClientProvider>,
    );

    expect(screen.getByLabelText("Open chat").parentElement).toHaveClass(
      "app-layer-chat",
    );
  });
});
