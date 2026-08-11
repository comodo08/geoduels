import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import MatchSideHPCard from "../MatchSideHPCard";
import type { MatchSideView } from "../ParticipantIdentity";

function side(multiplier?: number, hp = 4000): MatchSideView {
  return {
    id: "self",
    hp,
    connection: "connected",
    participant: {
      kind: "player",
      id: "self",
      name: "You",
      avatarFallback: "Y",
      damageMultiplier: multiplier,
    },
  };
}

function hpBarWidth(): number {
  const bar = document.querySelector('[data-testid="player-name-row"]')
    ?.parentElement?.querySelector(".absolute.inset-0.flex.items-center.justify-center.text-md");
  if (!bar) throw new Error("hp bar text node not found");
  let el = bar.parentElement?.parentElement;
  while (el && !el.className.includes("bg-[linear-gradient(180deg,#595B69")) {
    el = el.parentElement;
  }
  if (!el) throw new Error("hp bar container not found");
  return el.getBoundingClientRect().width;
}

describe("MatchSideHPCard HP bar width stability", () => {
  afterEach(cleanup);

  it("keeps the same HP bar width with and without the multiplier badge", () => {
    const { unmount } = render(
      <MatchSideHPCard position="left" side={side(1)} hpPct="66%" />,
    );
    const withoutBadge = hpBarWidth();
    unmount();

    render(<MatchSideHPCard position="left" side={side(1.5)} hpPct="66%" />);
    const withBadge = hpBarWidth();

    expect(withBadge).toBeCloseTo(withoutBadge, 0);
  });
});
