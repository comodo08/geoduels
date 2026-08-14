import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ParticipantName } from "../ParticipantIdentity";
import type {
  PlayerIdentityView,
  TeamIdentityView,
} from "../participant-types";

function player(overrides: Partial<PlayerIdentityView> = {}): PlayerIdentityView {
  return {
    kind: "player",
    id: "player-1",
    name: "Explorer",
    avatarFallback: "E",
    ...overrides,
  };
}

function team(overrides: Partial<TeamIdentityView> = {}): TeamIdentityView {
  return {
    kind: "team",
    id: "a",
    name: "Team Red",
    avatarFallback: "R",
    avatarColor: "#dc2626",
    members: [],
    ...overrides,
  };
}

describe("ParticipantName damage multiplier badge", () => {
  afterEach(cleanup);

  it("renders the multiplier next to a player nickname", () => {
    render(<ParticipantName participant={player()} multiplier={1.5} />);

    expect(screen.getByText("Explorer")).toBeInTheDocument();
    expect(screen.getByTestId("participant-multiplier-badge")).toHaveTextContent(
      "1.5x",
    );
  });

  it("hides the badge at the baseline multiplier", () => {
    render(<ParticipantName participant={player()} multiplier={1} />);

    expect(
      screen.queryByTestId("participant-multiplier-badge"),
    ).not.toBeInTheDocument();
  });

  it("hides the badge when no multiplier is known", () => {
    render(<ParticipantName participant={player()} />);

    expect(
      screen.queryByTestId("participant-multiplier-badge"),
    ).not.toBeInTheDocument();
  });

  it("renders the multiplier next to a team name", () => {
    render(<ParticipantName participant={team()} multiplier={2} />);

    expect(screen.getByText("Team Red")).toBeInTheDocument();
    expect(screen.getByTestId("participant-multiplier-badge")).toHaveTextContent(
      "2x",
    );
  });
});
