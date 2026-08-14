import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor, fireEvent, cleanup } from '@testing-library/react';
import GuessMap, { buildGoogleMapsLocationUrl, openLocationInMaps } from './GuessMap';

type RoundPlayerResult = {
  userId: string;
  lat: number;
  lng: number;
  score: number;
  distanceKm: number;
};

type RoundPlayerSeed = {
  id?: string;
  result: RoundPlayerResult;
};

type RoundResult = {
  roundId?: string;
  actualLocation: { lat: number; lng: number };
  players: Record<string, RoundPlayerResult>;
};

function makeRound(seed: number, players: RoundPlayerSeed[] = []): RoundResult {
  return {
    roundId: `round-${seed}`,
    actualLocation: { lat: 10 + seed, lng: -20 - seed },
    players: Object.fromEntries(players.map((p, i) => [p.id ?? `p${i}`, p.result]))
  };
}

describe('openLocationInMaps', () => {
  let fakeWindow: { opener: Window | null };
  beforeEach(() => {
    fakeWindow = { opener: null };
    vi.spyOn(window, 'open').mockImplementation(() => fakeWindow as unknown as Window);
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('opens a Google Maps url built from the coordinates', () => {
    const url = openLocationInMaps(40.7128, -74.006);
    expect(url).toBe(buildGoogleMapsLocationUrl(40.7128, -74.006));
    expect(window.open).toHaveBeenCalledTimes(1);
    expect(window.open).toHaveBeenCalledWith(url, '_blank');
    expect(fakeWindow.opener).toBeNull();
  });
});

describe('GuessMap interactive result markers', () => {
  beforeEach(() => {
    vi.spyOn(window, 'open').mockImplementation(() => null);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  it('opens the location when the single-result marker is clicked', async () => {
    const round = makeRound(1);
    render(<GuessMap mode="result" result={round} interactiveInResult />);

    await waitFor(() => {
      const link = document.querySelector('.actualLocationLink');
      expect(link).not.toBeNull();
      expect(link?.getAttribute('data-lat')).toBe(String(round.actualLocation.lat));
      expect(link?.getAttribute('data-lng')).toBe(String(round.actualLocation.lng));
      fireEvent.click(link as HTMLElement);
      expect(window.open).toHaveBeenCalledTimes(1);
    });

    expect(window.open).toHaveBeenCalledWith(
      buildGoogleMapsLocationUrl(round.actualLocation.lat, round.actualLocation.lng),
      '_blank'
    );
  });

  it('opens each round url exactly once in the multi-round overlay', async () => {
    const rounds = [makeRound(1), makeRound(2), makeRound(3)];
    render(<GuessMap mode="result" results={rounds} interactiveInResult />);

    await waitFor(() => {
      const links = Array.from(document.querySelectorAll('.actualLocationLink'));
      expect(links.length).toBe(3);
      links.forEach((link, index) => {
        expect(link.getAttribute('data-lat')).toBe(String(rounds[index].actualLocation.lat));
        expect(link.getAttribute('data-lng')).toBe(String(rounds[index].actualLocation.lng));
        fireEvent.click(link);
      });
      expect(window.open).toHaveBeenCalledTimes(3);
    });

    rounds.forEach((round) => {
      expect(window.open).toHaveBeenCalledWith(
        buildGoogleMapsLocationUrl(round.actualLocation.lat, round.actualLocation.lng),
        '_blank'
      );
    });
  });
});
