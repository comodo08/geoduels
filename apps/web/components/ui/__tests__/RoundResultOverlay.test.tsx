import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import RoundResultOverlay from '../RoundResultOverlay';
import type { RoundResultOverlayProps } from '../types';

function createProps(overrides: Partial<RoundResultOverlayProps> = {}): RoundResultOverlayProps {
  return {
    roundNumber: 3,
    mapNode: <div>Map Node</div>,
    phase: 'scores',
    showScoreReveal: true,
    winner: 'self',
    damage: 123,
    sides: {
      self: {
        id: 'self',
        participant: {
          kind: 'player',
          id: 'self',
          name: 'You',
          avatarFallback: 'Y',
          damageMultiplier: 1.5,
        },
        hp: 4000,
        score: 4321,
        connection: 'connected',
      },
      opponent: {
        id: 'opp',
        participant: {
          kind: 'player',
          id: 'opp',
          name: 'Opp',
          avatarFallback: 'O',
          damageMultiplier: 1,
        },
        hp: 3200,
        score: 1111,
        connection: 'connected',
      }
    },
    hpPct: (hp) => `${Math.max(0, Math.min(100, (hp / 6000) * 100))}%`,
    ...overrides
  };
}

function withSelfMultiplier(multiplier?: number) {
  const base = createProps();
  return {
    ...base.sides,
    self: {
      ...base.sides.self,
      participant: { ...base.sides.self.participant, damageMultiplier: multiplier },
    },
  };
}

describe('RoundResultOverlay component', () => {
  afterEach(() => {
    cleanup();
  });

  it('does not render score travel token on tie', () => {
    render(
      <RoundResultOverlay
        {...createProps({
          phase: 'crush',
          winner: 'tie',
          damage: 0,
          sides: {
            ...createProps().sides,
            self: { ...createProps().sides.self, score: 2500 },
            opponent: { ...createProps().sides.opponent, score: 2500 },
          }
        })}
      />
    );

    expect(screen.queryByTestId('score-travel-token')).not.toBeInTheDocument();
  });

  it('renders score travel token in crush phase for non-tie with damage', async () => {
    render(<RoundResultOverlay {...createProps({ phase: 'crush', winner: 'self', damage: 321 })} />);

    await waitFor(() => {
      expect(screen.getByTestId('score-travel-token')).toBeInTheDocument();
    });
  });

  it('scales the damage with the winning side multiplier when the multiplier phase starts', () => {
    render(
      <RoundResultOverlay
        {...createProps({
          phase: 'damage_multiplier',
          winner: 'self',
          damage: 123,
          sides: withSelfMultiplier(1.5)
        })}
      />
    );

    expect(screen.getByTestId('damage-multiplier-label')).toHaveTextContent('1.5x');
    expect(screen.getByText('185')).toBeInTheDocument();
  });

  it('does not show a baseline damage multiplier label', () => {
    render(
      <RoundResultOverlay
        {...createProps({
          phase: 'damage_multiplier',
          winner: 'self',
          damage: 123,
          sides: withSelfMultiplier(1)
        })}
      />
    );

    expect(screen.queryByTestId('damage-multiplier-label')).not.toBeInTheDocument();
    expect(screen.getAllByText('123').length).toBeGreaterThan(0);
  });

  it('falls back to unscaled damage when the round predates per-player multipliers', () => {
    render(
      <RoundResultOverlay
        {...createProps({
          phase: 'damage_multiplier',
          winner: 'self',
          damage: 123,
          sides: withSelfMultiplier(undefined)
        })}
      />
    );

    expect(screen.queryByTestId('damage-multiplier-label')).not.toBeInTheDocument();
    expect(screen.getAllByText('123').length).toBeGreaterThan(0);
  });

  it('uses the opponent multiplier when the opponent wins the round', () => {
    const base = createProps();
    render(
      <RoundResultOverlay
        {...createProps({
          phase: 'damage_multiplier',
          winner: 'opp',
          damage: 123,
          sides: {
            self: base.sides.self,
            opponent: {
              ...base.sides.opponent,
              participant: { ...base.sides.opponent.participant, damageMultiplier: 2 },
            },
          }
        })}
      />
    );

    expect(screen.getByTestId('damage-multiplier-label')).toHaveTextContent('2x');
    expect(screen.getByText('246')).toBeInTheDocument();
  });
});
