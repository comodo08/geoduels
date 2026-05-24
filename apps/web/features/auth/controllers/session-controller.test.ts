import { describe, expect, it, vi } from 'vitest';
import { createRuntimeConfigFixture } from '../../../test/runtime-config.fixture';
import { SessionController } from './session-controller';

function base64URL(value: string) {
  return btoa(value).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

function tokenWithExp(expiresAtMs: number) {
  return [
    base64URL(JSON.stringify({ alg: 'none', typ: 'JWT' })),
    base64URL(JSON.stringify({ exp: Math.floor(expiresAtMs / 1000) })),
    'signature'
  ].join('.');
}

describe('SessionController', () => {
  const runtimeConfig = createRuntimeConfigFixture();

  it('bootstraps a cookie-backed session without local persistence', async () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.setNetworkHandlers({
      bootstrapSession: vi.fn(async () => ({
        userId: 'user-1',
        accessToken: tokenWithExp(Date.now() + 60 * 60_000),
        onboardingRequired: false,
        nicknameInput: 'Player'
      }))
    });
    controller.start();

    const session = await controller.bootstrapSession();
    expect(session?.userId).toBe('user-1');

    controller.applySessionSnapshot(session!, {
      displayName: 'Player',
      authLoading: false
    });

    expect(controller.getState().userId).toBe('user-1');
    expect(controller.getState().displayName).toBe('Player');
    expect(controller.getState().accessToken).not.toBe('');

    controller.destroy();
  });

  it('refreshes an expired playable session before returning it', async () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.applySessionSnapshot(
      {
        userId: 'user-1',
        accessToken: tokenWithExp(Date.now() - 60_000),
        onboardingRequired: false,
        nicknameInput: 'Player'
      },
      {
        displayName: 'Player'
      }
    );
    controller.setNetworkHandlers({
      refreshSession: vi.fn(async () => ({
        userId: 'user-1',
        accessToken: tokenWithExp(Date.now() + 60 * 60_000),
        onboardingRequired: false,
        nicknameInput: 'Player'
      })),
      getPlayableSession: vi.fn(async () => null)
    });
    controller.start();

    const session = await controller.getPlayableSession();

    expect(session?.userId).toBe('user-1');
    expect(session?.accessToken).not.toBe('');
    expect(session?.expiresAt).toBeGreaterThan(Date.now());

    controller.destroy();
  });

  it('clears an expired guest session when refresh cannot restore it', async () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.applySessionSnapshot(
      {
        userId: 'guest-1',
        accessToken: tokenWithExp(Date.now() - 60_000),
        onboardingRequired: false,
        nicknameInput: 'Guest'
      },
      {
        displayName: 'Guest',
        isGuest: true
      }
    );
    controller.setNetworkHandlers({
      refreshSession: vi.fn(async () => null)
    });
    controller.start();

    const session = await controller.ensureFreshSession();

    expect(session).toBeNull();
    expect(controller.getState().userId).toBe('');
    expect(controller.getState().authError).toBe('Guest session expired. Please start again or sign in.');

    controller.destroy();
  });

  it('keeps registered session state generic when refresh cannot restore it', async () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.applySessionSnapshot(
      {
        userId: 'user-1',
        accessToken: tokenWithExp(Date.now() - 60_000),
        onboardingRequired: false,
        nicknameInput: 'Player'
      },
      {
        displayName: 'Player',
        isGuest: false
      }
    );
    controller.setNetworkHandlers({
      refreshSession: vi.fn(async () => null)
    });
    controller.start();

    const session = await controller.ensureFreshSession();

    expect(session).toBeNull();
    expect(controller.getState().userId).toBe('user-1');
    expect(controller.getState().authError).toBe('');

    controller.destroy();
  });

  it('updates leaderboard state separately from profile state', () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });

    controller.applyProfileSnapshot({
      display_name: 'Player',
      mmr: 1234,
      wins: 7
    });
    controller.applyLeaderboardSummary({
      mode: 'duel',
      season: 's2',
      selfRank: 3,
      totalPlayers: 99,
      entries: [{ rank: 1, userId: 'top', displayName: 'Top', avatarUrl: '', mmr: 1500, gamesPlayed: 20, wins: 15 }]
    });

    expect(controller.getState().displayName).toBe('Player');
    expect(controller.getState().mmr).toBe(1234);
    expect(controller.getState().leaderboard?.selfRank).toBe(3);
    expect(controller.getState().leaderboard?.entries).toHaveLength(1);
  });
});
