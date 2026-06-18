import type { ComponentProps } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import EndMatchOverlay from "../EndMatchOverlay";

vi.mock("next/dynamic", () => ({
  default: () => () => <div data-testid="guess-map" />,
}));

type Props = ComponentProps<typeof EndMatchOverlay>;

function createProps(overrides: Partial<Props> = {}): Props {
  return {
    onLeaveGame: vi.fn(),
    mode: "duel",
    outcome: "win",
    selfUserId: "self",
    sides: {
      self: {
        id: "self",
        participant: {
          kind: "player",
          id: "self",
          name: "Self",
          avatarFallback: "S",
        },
        hp: 5000,
        connection: "connected",
      },
      opponent: {
        id: "opponent",
        participant: {
          kind: "player",
          id: "opponent",
          name: "Opponent",
          avatarFallback: "O",
        },
        hp: 0,
        connection: "connected",
      },
    },
    totalScore: 9000,
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
            distanceKm: 10,
            score: 4500,
            hpAfterRound: 5000,
          },
          opponent: {
            userId: "opponent",
            lat: 2,
            lng: 2,
            distanceKm: 500,
            score: 1000,
            hpAfterRound: 0,
          },
        },
      },
    ],
    resultPlayerNames: { self: "Self", opponent: "Opponent" },
    resultPlayerAvatars: {},
    resultPlayerFallbacks: { self: "S", opponent: "O" },
    ...overrides,
  };
}

function openBreakdown() {
  fireEvent.click(screen.getByRole("button", { name: "Breakdown" }));
}

describe("EndMatchOverlay breakdown", () => {
  afterEach(cleanup);

  it("shows each player's post-round health for a duel, including zero health", () => {
    render(<EndMatchOverlay {...createProps()} />);
    openBreakdown();

    expect(screen.getByText("5,000 HP")).toBeInTheDocument();
    expect(screen.getByText("0 HP")).toBeInTheDocument();
    expect(screen.queryByText("4,500")).not.toBeInTheDocument();
    expect(screen.queryByText("Total")).not.toBeInTheDocument();
  });

  it("shows team health rather than individual player health for a team duel", () => {
    const props = createProps();
    render(
      <EndMatchOverlay
        {...props}
        mode="team_duel"
        sides={{
          self: {
            id: "a",
            participant: {
              kind: "team",
              id: "a",
              name: "Team Red",
              avatarFallback: "R",
              avatarColor: "#dc2626",
              members: [],
            },
            hp: 6000,
            connection: "connected",
          },
          opponent: {
            id: "b",
            participant: {
              kind: "team",
              id: "b",
              name: "Team Blue",
              avatarFallback: "B",
              avatarColor: "#2563eb",
              members: [],
            },
            hp: 3200,
            connection: "connected",
          },
        }}
        roundResults={[
          {
            ...props.roundResults[0],
            players: {
              self: { ...props.roundResults[0].players.self, hpAfterRound: 1111 },
              opponent: {
                ...props.roundResults[0].players.opponent,
                hpAfterRound: 2222,
              },
            },
            teams: {
              a: {
                teamId: "a",
                lat: 1,
                lng: 1,
                distanceKm: 10,
                score: 4500,
                hpAfterRound: 6000,
              },
              b: {
                teamId: "b",
                lat: 2,
                lng: 2,
                distanceKm: 500,
                score: 1000,
                hpAfterRound: 3200,
              },
            },
          },
        ]}
      />,
    );
    openBreakdown();

    expect(screen.getByText("6,000 HP")).toBeInTheDocument();
    expect(screen.getByText("3,200 HP")).toBeInTheDocument();
    expect(screen.queryByText("1,111 HP")).not.toBeInTheDocument();
    expect(screen.queryByText("2,222 HP")).not.toBeInTheDocument();
  });

  it("shows a placeholder when an older round result has no health value", () => {
    const props = createProps();
    render(
      <EndMatchOverlay
        {...props}
        roundResults={[
          {
            ...props.roundResults[0],
            players: {
              self: { ...props.roundResults[0].players.self, hpAfterRound: undefined },
              opponent: {
                ...props.roundResults[0].players.opponent,
                hpAfterRound: undefined,
              },
            },
          },
        ]}
      />,
    );
    openBreakdown();

    expect(screen.getAllByText("-")).toHaveLength(2);
  });

  it("keeps score breakdowns for points-based modes", () => {
    render(
      <EndMatchOverlay
        {...createProps({
          mode: "singleplayer",
        })}
      />,
    );
    openBreakdown();

    expect(screen.getAllByText("4,500").length).toBeGreaterThan(0);
    expect(screen.getByText("Total")).toBeInTheDocument();
    expect(screen.queryByText("5,000 HP")).not.toBeInTheDocument();
  });
});
