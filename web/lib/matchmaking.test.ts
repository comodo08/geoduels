import { describe, expect, it } from 'vitest';
import { initialMatchmakingState, matchmakingReducer } from './matchmaking';

describe('matchmakingReducer', () => {
  it('moves to queueing on join request', () => {
    const next = matchmakingReducer(initialMatchmakingState, { type: 'join_requested' });
    expect(next.status).toBe('queueing');
    expect(next.intentVersion).toBe(1);
  });

  it('ignores stale recover ready after a new user intent', () => {
    const started = matchmakingReducer(initialMatchmakingState, { type: 'recover_started', requestID: 1 });
    const joined = matchmakingReducer(started, { type: 'join_requested' });
    const resolved = matchmakingReducer(joined, {
      type: 'recover_resolved',
      requestID: 1,
      intentVersionAtStart: 0,
      outcome: 'ready',
      hasSnapshot: false
    });
    expect(resolved.status).toBe('queueing');
  });

  it('applies recover matched even when intent version changed', () => {
    const started = matchmakingReducer(initialMatchmakingState, { type: 'recover_started', requestID: 1 });
    const joined = matchmakingReducer(started, { type: 'join_requested' });
    const resolved = matchmakingReducer(joined, {
      type: 'recover_resolved',
      requestID: 1,
      intentVersionAtStart: 0,
      outcome: 'matched',
      hasSnapshot: false
    });
    expect(resolved.status).toBe('matched_connecting');
  });

  it('moves active game sockets into recovering when closed', () => {
    const next = matchmakingReducer(
      { ...initialMatchmakingState, status: 'in_match' },
      { type: 'ws_closed' }
    );
    expect(next.status).toBe('recovering');
  });
});
