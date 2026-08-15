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
        nicknameRequired: false,
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

  it('does not repeat a completed signed-out bootstrap unless forced', async () => {
    const bootstrapSession = vi.fn(async () => null);
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.setNetworkHandlers({ bootstrapSession });

    await controller.bootstrapSession();
    await controller.bootstrapSession();

    expect(bootstrapSession).toHaveBeenCalledTimes(1);

    await controller.bootstrapSession({ force: true });

    expect(bootstrapSession).toHaveBeenCalledTimes(2);
    controller.destroy();
  });

  it('re-validates the session against the server after the local session is cleared', async () => {
    const bootstrapSession = vi.fn(async () => ({
      userId: 'user-1',
      accessToken: tokenWithExp(Date.now() + 60 * 60_000),
      nicknameRequired: false,
      nicknameInput: 'Player'
    }));
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.setNetworkHandlers({ bootstrapSession });

    await controller.bootstrapSession();
    await controller.bootstrapSession();
    expect(bootstrapSession).toHaveBeenCalledTimes(1);

    controller.clearAuthSession();
    await controller.bootstrapSession();

    expect(bootstrapSession).toHaveBeenCalledTimes(2);
    controller.destroy();
  });

  it('refreshes an expired playable session before returning it', async () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.applySessionSnapshot(
      {
        userId: 'user-1',
        accessToken: tokenWithExp(Date.now() - 60_000),
        nicknameRequired: false,
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
        nicknameRequired: false,
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
        nicknameRequired: false,
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
        nicknameRequired: false,
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
      nextResetAt: '2026-07-01T21:00:00Z',
      selfRank: 3,
      totalPlayers: 99,
      entries: [{ rank: 1, userId: 'top', displayName: 'Top', avatarUrl: '', mmr: 1500, gamesPlayed: 20, wins: 15 }]
    });

    expect(controller.getState().displayName).toBe('Player');
    expect(controller.getState().mmr).toBe(1234);
    expect(controller.getState().leaderboard?.selfRank).toBe(3);
    expect(controller.getState().leaderboard?.nextResetAt).toBe('2026-07-01T21:00:00Z');
    expect(controller.getState().leaderboard?.entries).toHaveLength(1);
  });

  it('applies a committed match rating to the active profile', () => {
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.applySessionSnapshot(
      {
        userId: 'self',
        accessToken: tokenWithExp(Date.now() + 60 * 60_000),
        nicknameRequired: false,
        nicknameInput: 'Player'
      },
      {
        mmr: 1000,
        ratingRd: 220,
        gamesPlayed: 9,
        wins: 5,
        rankedGamesPlayed: 7,
        rankedWins: 4
      }
    );

    controller.applyCommittedRating(1025, 180);

    expect(controller.getState()).toEqual(
      expect.objectContaining({
        mmr: 1025,
        ratingRd: 180,
        gamesPlayed: 9,
        wins: 5,
        rankedGamesPlayed: 7,
        rankedWins: 4
      })
    );
  });

  it('blocks session restoration during sign out and restores it after', async () => {
    const bootstrapSession = vi.fn(async () => ({
      userId: 'user-1',
      accessToken: tokenWithExp(Date.now() + 60 * 60_000),
      nicknameRequired: false,
      nicknameInput: 'Player'
    }));
    const refreshSession = vi.fn(async () => null);
    const getPlayableSession = vi.fn(async () => null);
    const controller = new SessionController({ config: runtimeConfig, onResetSession: vi.fn() });
    controller.applySessionSnapshot(
      {
        userId: 'user-1',
        accessToken: tokenWithExp(Date.now() + 60 * 60_000),
        nicknameRequired: false,
        nicknameInput: 'Player'
      },
      { displayName: 'Player' }
    );
    controller.setNetworkHandlers({ bootstrapSession, refreshSession, getPlayableSession });
    controller.start();

    let finishLogout!: () => void;
    const logoutRequest = new Promise<void>((resolve) => {
      finishLogout = resolve;
    });
    const signOutPromise = controller.signOut(() => logoutRequest);

    expect(await controller.bootstrapSession()).toBeNull();
    expect(await controller.refreshSession()).toBeNull();
    expect(await controller.ensureFreshSession()).toBeNull();
    expect(await controller.getPlayableSession()).toBeNull();
    expect(bootstrapSession).not.toHaveBeenCalled();
    expect(refreshSession).not.toHaveBeenCalled();
    expect(getPlayableSession).not.toHaveBeenCalled();

    finishLogout();
    await signOutPromise;

    expect(controller.getState().userId).toBe('');
    expect((await controller.bootstrapSession())?.userId).toBe('user-1');

    controller.destroy();
  });
});
