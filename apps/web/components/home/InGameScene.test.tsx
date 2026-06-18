import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import InGameScene from './InGameScene';
import type { InGameSceneProps } from './InGameScene';

function createProps(overrides: Partial<InGameSceneProps> = {}): InGameSceneProps {
  return {
    uiPhase: 'live_round',
    streetViewSrc: 'https://www.google.com/maps/embed/v1/streetview?key=test&pano=pano-1',
    streetViewInteractive: true,
    showResultStage: false,
    isSingleplayer: false,
    isPointsMode: false,
    resultOverlay: undefined,
    sides: {
      self: {
        id: 'self',
        participant: {
          kind: 'player',
          id: 'self',
          name: 'Self',
          avatarFallback: 'S',
          rating: 1200,
        },
        hp: 5000,
        connection: 'connected',
      },
      opponent: {
        id: 'opp',
        participant: {
          kind: 'player',
          id: 'opp',
          name: 'Opponent',
          avatarFallback: 'O',
          rating: 1200,
        },
        hp: 5000,
        connection: 'connected',
      },
    },
    hpPct: (hp) => `${hp}%`,
    mm: '01',
    ss: '00',
    isRoundTimerRunning: true,
    timerProgressPct: 50,
    isTimerCritical: false,
    isTimerPulseActive: false,
    resultMode: false,
    selfHP: 5000,
    oppHP: 5000,
    totalScore: 0,
    currentRoundScore: 0,
    currentRoundDistanceKm: 0,
    onForfeit: vi.fn(() => true),
    onAdvanceRound: vi.fn(() => true),
    onLeaveGame: vi.fn(),
    canFinalizeGuess: false,
    canAdvanceRound: false,
    onFinalizeGuess: vi.fn(),
    guessMapNode: null,
    selfUserId: 'self',
    damageMultiplier: 1,
    guessSubmitted: false,
    opponentGuessAlert: false,
    connectionIssue: '',
    ...overrides,
  };
}

describe('InGameScene', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
  });

  it('keeps interactive Street View iframes in keyboard tab navigation', () => {
    render(<InGameScene {...createProps()} />);

    const streetViewFrame = screen.getByTitle('Street View');

    expect(streetViewFrame).not.toHaveAttribute('tabindex');
  });

  it('keeps NMPZ Street View iframes out of keyboard tab navigation', () => {
    render(<InGameScene {...createProps({ streetViewInteractive: false })} />);

    const streetViewFrame = screen.getByTitle('Street View');

    expect(streetViewFrame).toHaveAttribute('tabindex', '-1');
  });

  it('allows interactive Street View iframes to keep focus', () => {
    render(<InGameScene {...createProps()} />);

    const streetViewFrame = screen.getByTitle('Street View');

    streetViewFrame.focus();

    expect(document.activeElement).toBe(streetViewFrame);
  });

  it('releases focus if the NMPZ Street View iframe captures it', () => {
    render(<InGameScene {...createProps({ streetViewInteractive: false })} />);

    const streetViewFrame = screen.getByTitle('Street View');

    streetViewFrame.focus();

    expect(document.activeElement).not.toBe(streetViewFrame);
    expect(document.activeElement?.tagName).toBe('SECTION');
  });

  it('renders team identity without promoting member avatars, badges, or ratings', () => {
    const memberBadge = {
      id: 'badge-1',
      kind: 'rank',
      label: 'Member Badge',
      description: 'Player-only badge',
      imageUrl: '/badge.png',
    };
    render(
      <InGameScene
        {...createProps({
          partyMode: 'team_duel',
          sides: {
            self: {
              id: 'a',
              participant: {
                kind: 'team',
                id: 'a',
                name: 'Team Red',
                avatarFallback: 'R',
                avatarColor: '#dc2626',
                members: [
                  {
                    kind: 'player',
                    id: 'self',
                    name: 'Red Member',
                    avatarFallback: 'M',
                    rating: 2100,
                    selectedBadge: memberBadge,
                  },
                ],
              },
              hp: 6000,
              connection: 'connected',
            },
            opponent: {
              id: 'b',
              participant: {
                kind: 'team',
                id: 'b',
                name: 'Team Blue',
                avatarFallback: 'B',
                avatarColor: '#2563eb',
                members: [],
              },
              hp: 3200,
              connection: 'connected',
            },
          },
          selfHP: 6000,
          oppHP: 3200,
        })}
      />,
    );

    expect(screen.getByText('Team Red')).toBeInTheDocument();
    expect(screen.getByText('Team Blue')).toBeInTheDocument();
    expect(screen.queryByText('Red Member')).not.toBeInTheDocument();
    expect(screen.queryByText('(2100)')).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/Member Badge/)).not.toBeInTheDocument();
  });
});
