import { beforeEach, describe, expect, it, vi } from 'vitest';
import { RESULT_ANIMATION_CONFIG } from '../../../components/ui/round-result-animation-config';
import type { Snapshot } from '../../../components/ui/types';
import { createRuntimeConfigFixture } from '../../../test/runtime-config.fixture';
import { GameController } from './game-controller';
import type { SfxController } from '../../../lib/audio/sfx';

function createSnapshot(overrides: Partial<Snapshot> = {}): Snapshot {
  return {
    matchId: 'match-1',
    state: 'active',
    eventSequence: 1,
    phase: 'round_result',
    roundPhase: 'round_result',
    phaseStartedAt: Date.now(),
    phaseEndsAt: Date.now() + 1000,
    roundMsLeft: 0,
    currentRound: {
      roundId: 'round-1',
      roundNumber: 1,
      location: { panoId: 'pano-123' }
    },
    players: {
      self: {
        userId: 'self',
        displayName: 'Self',
        hp: 5000,
        mmr: 1000,
        finalized: true,
        isGuest: false,
        disconnected: false
      },
      opp: {
        userId: 'opp',
        displayName: 'Opp',
        hp: 4500,
        mmr: 1000,
        finalized: true,
        isGuest: false,
        disconnected: false
      }
    },
    lastRoundResult: {
      roundId: 'round-1',
      roundNumber: 1,
      players: {
        self: {
          userId: 'self',
          lat: 0,
          lng: 0,
          score: 4200,
          distanceKm: 10,
          damageTaken: 0,
          hpAfterRound: 5000
        },
        opp: {
          userId: 'opp',
          lat: 0,
          lng: 0,
          score: 1200,
          distanceKm: 500,
          damageTaken: 500,
          hpAfterRound: 4500
        }
      },
      actualLocation: { lat: 0, lng: 0 }
    },
    ...overrides
  };
}

describe('GameController', () => {
  const runtimeConfig = createRuntimeConfigFixture();
  let sfxController: SfxController;

  beforeEach(() => {
    vi.useFakeTimers();
    sfxController = {
      start: vi.fn(),
      destroy: vi.fn(),
      play: vi.fn(),
      playManaged: vi.fn(),
      playLoop: vi.fn(),
      stop: vi.fn()
    };
  });

  it('keeps round-result timers alive across repeated snapshots for the same round', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot();
    listener();
    expect(controller.getState().resultPhase).toBe('base');

    matchState.snapshot = createSnapshot({ eventSequence: 2 });
    listener();

    vi.advanceTimersByTime(1400);
    expect(controller.getState().resultPhase).toBe('scores');
    expect(sfxController.play).toHaveBeenCalledWith('duel-round-result-enter');
    expect(sfxController.play).toHaveBeenCalledWith('duel-round-result-score-reveal');
    vi.advanceTimersByTime(RESULT_ANIMATION_CONFIG.timeline.hpApplyAtMs);
    expect(sfxController.play).toHaveBeenCalledWith('duel-round-result-hp-hit');

    controller.destroy();
    vi.useRealTimers();
  });

  it('plays round countdown sfx once the live round reaches 15 seconds and stops it on result', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now() + 16000,
      roundMsLeft: 16000,
      lastRoundResult: undefined
    });
    listener();
    vi.advanceTimersByTime(50);
    expect(sfxController.playManaged).not.toHaveBeenCalledWith('duel-round-countdown');

    vi.advanceTimersByTime(1000);
    expect(sfxController.playManaged).toHaveBeenCalledWith('duel-round-countdown');
    expect(sfxController.playManaged).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(3100);
    expect(sfxController.playManaged).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(15000);
    expect(sfxController.stop).not.toHaveBeenCalledWith('duel-round-countdown');

    matchState.snapshot = createSnapshot({
      phase: 'round_result',
      roundPhase: 'round_result',
      phaseEndsAt: Date.now() + 6000,
      roundMsLeft: 0
    });
    listener();
    expect(sfxController.stop).toHaveBeenCalledWith('duel-round-countdown');

    controller.destroy();
    vi.useRealTimers();
  });

  it('does not play game start sfx from duel gameplay snapshots', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now() + 30000,
      roundMsLeft: 30000,
      lastRoundResult: undefined
    });
    listener();
    expect(sfxController.play).not.toHaveBeenCalledWith('duel-game-start');

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_intro',
      phaseEndsAt: Date.now() + 3000,
      roundMsLeft: 3000,
      lastRoundResult: undefined
    });
    listener();
    vi.advanceTimersByTime(3100);

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now() + 30000,
      roundMsLeft: 30000,
      lastRoundResult: undefined
    });
    listener();
    listener();

    const startCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-game-start');
    expect(startCalls).toHaveLength(0);
    const beginHideExitCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-exit');
    expect(beginHideExitCalls).toHaveLength(1);

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_intro',
      phaseEndsAt: Date.now() + 3000,
      roundMsLeft: 3000,
      currentRound: {
        roundId: 'round-2',
        roundNumber: 2,
        location: { panoId: 'pano-123' }
      },
      lastRoundResult: undefined
    });
    listener();
    vi.advanceTimersByTime(3100);

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now() + 30000,
      roundMsLeft: 30000,
      currentRound: {
        roundId: 'round-2',
        roundNumber: 2,
        location: { panoId: 'pano-123' }
      },
      lastRoundResult: undefined
    });
    listener();

    const startCallsAfterRoundTwo = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-game-start');
    expect(startCallsAfterRoundTwo).toHaveLength(0);
    const exitCallsAfterRoundTwo = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-exit');
    expect(exitCallsAfterRoundTwo).toHaveLength(2);

    controller.destroy();
    vi.useRealTimers();
  });

  it('plays result exit sfx when the post-result countdown sheet hides', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'round_result',
      roundPhase: 'round_result',
      lastRoundResult: {
        roundId: 'round-1',
        roundNumber: 1,
        players: {
          self: {
            userId: 'self',
            lat: 0,
            lng: 0,
            score: 4200,
            distanceKm: 10,
            damageTaken: 0,
            hpAfterRound: 5000
          },
          opp: {
            userId: 'opp',
            lat: 0,
            lng: 0,
            score: 1200,
            distanceKm: 500,
            damageTaken: 500,
            hpAfterRound: 4500
          }
        },
        actualLocation: { lat: 0, lng: 0 }
      }
    });
    listener();

    const exitCallsBeforeCountdownHide = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-exit');
    expect(exitCallsBeforeCountdownHide).toHaveLength(0);

    matchState.snapshot = createSnapshot({
      eventSequence: 2,
      phase: 'live',
      roundPhase: 'round_intro',
      currentRound: {
        roundId: 'round-2',
        roundNumber: 2,
        location: { panoId: 'pano-123' }
      },
      lastRoundResult: undefined
    });
    listener();

    const exitCallsAfterResult = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-exit');
    expect(exitCallsAfterResult).toHaveLength(0);

    vi.advanceTimersByTime(3000);

    matchState.snapshot = createSnapshot({
      eventSequence: 3,
      phase: 'live',
      roundPhase: 'round_live',
      currentRound: {
        roundId: 'round-2',
        roundNumber: 2,
        location: { panoId: 'pano-123' }
      },
      lastRoundResult: undefined
    });
    listener();

    const exitCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-exit');
    expect(exitCalls).toHaveLength(1);

    controller.destroy();
    vi.useRealTimers();
  });

  it('plays duel-win sfx and no result exit sfx when the match ends on the end match page', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      state: 'ended',
      phase: 'round_result',
      roundPhase: 'round_result'
    });
    listener();

    controller.setShowMatchEndPage(true);
    controller.setShowMatchEndPage(true);

    const exitCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-exit');
    expect(exitCalls).toHaveLength(0);

    const winCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-win');
    expect(winCalls).toHaveLength(1);

    controller.destroy();
    vi.useRealTimers();
  });

  it('plays result countdown sfx once per round intro', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_intro',
      phaseEndsAt: Date.now() + 3000,
      roundMsLeft: 3000,
      lastRoundResult: undefined
    });
    listener();

    vi.advanceTimersByTime(3000);

    const resultCountdownCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-countdown');
    expect(resultCountdownCalls).toHaveLength(1);
    expect(sfxController.playManaged).not.toHaveBeenCalledWith('duel-round-countdown');

    controller.destroy();
    vi.useRealTimers();
  });

  it('plays guess sfx for each player finalized transition, including the guess that ends the round', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now() + 30000,
      roundMsLeft: 30000,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 4500,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now() + 15000,
      roundMsLeft: 15000,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 4500,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: undefined
    });
    listener();
    listener();

    matchState.snapshot = createSnapshot({
      phase: 'round_result',
      roundPhase: 'round_result',
      phaseEndsAt: Date.now() + 6000,
      roundMsLeft: 0,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 4500,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        }
      }
    });
    listener();

    const guessCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-guess');
    expect(guessCalls).toHaveLength(2);

    controller.destroy();
    vi.useRealTimers();
  });

  it('plays applicable sfx in singleplayer', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      mode: 'singleplayer',
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now(),
      roundMsLeft: 0,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 0,
          totalScore: 0,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: undefined
    });
    listener();
    listener();

    expect(sfxController.play).toHaveBeenCalledWith('duel-game-start');
    const startCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-game-start');
    expect(startCalls).toHaveLength(1);

    matchState.snapshot = createSnapshot({
      mode: 'singleplayer',
      phase: 'round_result',
      roundPhase: 'round_result',
      phaseEndsAt: Date.now(),
      roundMsLeft: 0,
      currentRound: undefined,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 0,
          totalScore: 4200,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: {
        roundId: 'round-1',
        roundNumber: 1,
        players: {
          self: {
            userId: 'self',
            lat: 0,
            lng: 0,
            score: 4200,
            distanceKm: 10
          }
        },
        actualLocation: { lat: 0, lng: 0 }
      }
    });
    listener();
    listener();

    const guessCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-guess');
    const resultEnterCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-enter');
    const scoreRevealCalls = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-round-result-score-reveal');
    expect(guessCalls).toHaveLength(1);
    expect(resultEnterCalls).toHaveLength(1);
    expect(scoreRevealCalls).toHaveLength(1);
    expect(sfxController.play).not.toHaveBeenCalledWith('duel-round-result-hp-hit');
    expect(sfxController.playManaged).not.toHaveBeenCalledWith('duel-round-countdown');

    matchState.snapshot = createSnapshot({
      mode: 'singleplayer',
      phase: 'live',
      roundPhase: 'round_live',
      phaseEndsAt: Date.now(),
      roundMsLeft: 0,
      currentRound: {
        roundId: 'round-2',
        roundNumber: 2,
        location: { panoId: 'pano-123' }
      },
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 0,
          totalScore: 4200,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: undefined
    });
    listener();

    const startCallsAfterRoundTwo = vi.mocked(sfxController.play).mock.calls.filter(([name]) => name === 'duel-game-start');
    expect(startCallsAfterRoundTwo).toHaveLength(1);

    controller.destroy();
    vi.useRealTimers();
  });

  it('keeps the victim hp bar at the pre-hit value during overkill result animation', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      state: 'active',
      eventSequence: 1,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 1200,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      state: 'ended',
      phase: 'round_result',
      roundPhase: 'round_result',
      eventSequence: 2,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 0,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: {
        roundId: 'round-1',
        roundNumber: 1,
        players: {
          self: {
            userId: 'self',
            lat: 0,
            lng: 0,
            score: 8700,
            distanceKm: 1,
            damageTaken: 0,
            hpAfterRound: 5000
          },
          opp: {
            userId: 'opp',
            lat: 0,
            lng: 0,
            score: 1200,
            distanceKm: 500,
            damageTaken: 7500,
            hpAfterRound: 0
          }
        },
        actualLocation: { lat: 0, lng: 0 }
      }
    });
    listener();

    expect(controller.getState().displayHP.opp).toBe(1200);
    expect(controller.getState().resultShownHP.opp).toBe(1200);

    vi.advanceTimersByTime(90);
    expect(controller.getState().displayHP.opp).toBe(0);

    controller.destroy();
    vi.useRealTimers();
  });

  it('stops duel marker changes during the final hidden half-second', () => {
    const matchState = {
      snapshot: createSnapshot({
        phase: 'live',
        roundPhase: 'round_live',
        phaseEndsAt: Date.now() + 499,
        roundMsLeft: 499,
        lastRoundResult: undefined,
        players: {
          self: {
            userId: 'self',
            displayName: 'Self',
            hp: 5000,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          },
          opp: {
            userId: 'opp',
            displayName: 'Opp',
            hp: 4500,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          }
        }
      })
    };
    const matchController = {
      subscribe: vi.fn(() => () => {}),
      getState: vi.fn(() => matchState),
      sendGameCommand: vi.fn(() => true),
      setConnectionIssue: vi.fn()
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();
    controller.placeGuess(1, 2);

    expect(matchController.sendGameCommand).not.toHaveBeenCalled();
    expect(controller.getState().guess).toBeUndefined();

    controller.destroy();
    vi.useRealTimers();
  });

  it('allows duel marker changes before the first finalized guess starts the timer', () => {
    const matchState = {
      snapshot: createSnapshot({
        phase: 'live',
        roundPhase: 'round_live',
        phaseEndsAt: 0,
        roundMsLeft: 0,
        lastRoundResult: undefined,
        currentRound: {
          roundId: 'round-1',
          roundNumber: 1,
          timerStarted: false,
          location: { panoId: 'pano-123' }
        },
        players: {
          self: {
            userId: 'self',
            displayName: 'Self',
            hp: 5000,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          },
          opp: {
            userId: 'opp',
            displayName: 'Opp',
            hp: 4500,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          }
        }
      })
    };
    const matchController = {
      subscribe: vi.fn(() => () => {}),
      getState: vi.fn(() => matchState),
      sendGameCommand: vi.fn(() => true),
      setConnectionIssue: vi.fn()
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();
    controller.placeGuess(1, 2);

    expect(matchController.sendGameCommand).toHaveBeenCalledWith(
      'guess.place',
      expect.objectContaining({ lat: 1, lng: 2 }),
      expect.any(Object)
    );
    expect(controller.getState().guess).toEqual({ lat: 1, lng: 2 });

    controller.destroy();
    vi.useRealTimers();
  });

  it('restores the viewer live guess from private snapshot state', () => {
    const matchState = {
      snapshot: createSnapshot({
        phase: 'live',
        roundPhase: 'round_live',
        lastRoundResult: undefined,
        self: {
          userId: 'self',
          currentGuess: { lat: 12.25, lng: 34.5 }
        },
        players: {
          self: {
            userId: 'self',
            displayName: 'Self',
            hp: 5000,
            mmr: 1000,
            finalized: true,
            isGuest: false,
            disconnected: false
          },
          opp: {
            userId: 'opp',
            displayName: 'Opp',
            hp: 4500,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          }
        }
      })
    };
    const matchController = {
      subscribe: vi.fn(() => () => {}),
      getState: vi.fn(() => matchState),
      sendGameCommand: vi.fn(() => true),
      setConnectionIssue: vi.fn()
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    expect(controller.getState().guess).toEqual({ lat: 12.25, lng: 34.5 });
    expect(controller.getState().guessSubmitted).toBe(true);

    controller.destroy();
    vi.useRealTimers();
  });

  it('keeps singleplayer marker changes available without the duel cutoff', () => {
    const matchState = {
      snapshot: createSnapshot({
        mode: 'singleplayer',
        phase: 'live',
        roundPhase: 'round_live',
        phaseEndsAt: Date.now() + 100,
        roundMsLeft: 100,
        lastRoundResult: undefined,
        players: {
          self: {
            userId: 'self',
            displayName: 'Self',
            hp: 0,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          }
        }
      })
    };
    const matchController = {
      subscribe: vi.fn(() => () => {}),
      getState: vi.fn(() => matchState),
      sendGameCommand: vi.fn(() => true),
      setConnectionIssue: vi.fn()
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();
    controller.placeGuess(1, 2);

    expect(matchController.sendGameCommand).toHaveBeenCalledWith(
      'guess.place',
      expect.objectContaining({ lat: 1, lng: 2 }),
      expect.any(Object)
    );
    expect(controller.getState().guess).toEqual({ lat: 1, lng: 2 });

    controller.destroy();
    vi.useRealTimers();
  });

  it('flags opponentLeftNotice when the opponent disconnects mid-match', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined,
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: true
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Opp');

    controller.destroy();
    vi.useRealTimers();
  });

  it('flags opponentLeftNotice when the opponent forfeits before any round result', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      state: 'ended',
      phase: 'ended',
      roundPhase: 'ended',
      phaseEndsAt: Date.now(),
      roundMsLeft: 0,
      currentRound: undefined,
      lastRoundResult: undefined,
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 0,
          mmr: 1000,
          finalized: false,
          isGuest: false,
          disconnected: false
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Opp');

    controller.destroy();
    vi.useRealTimers();
  });

  it('does not flag opponentLeftNotice when the match ends through a round result', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      state: 'ended',
      phase: 'round_result',
      roundPhase: 'round_result',
      players: {
        self: {
          userId: 'self',
          displayName: 'Self',
          hp: 5000,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        },
        opp: {
          userId: 'opp',
          displayName: 'Opp',
          hp: 0,
          mmr: 1000,
          finalized: true,
          isGuest: false,
          disconnected: false
        }
      },
      lastRoundResult: {
        roundId: 'round-1',
        roundNumber: 1,
        players: {
          self: {
            userId: 'self',
            lat: 0,
            lng: 0,
            score: 8700,
            distanceKm: 1,
            damageTaken: 0,
            hpAfterRound: 5000
          },
          opp: {
            userId: 'opp',
            lat: 0,
            lng: 0,
            score: 1200,
            distanceKm: 500,
            damageTaken: 7500,
            hpAfterRound: 0
          }
        },
        actualLocation: { lat: 0, lng: 0 }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);

    controller.destroy();
    vi.useRealTimers();
  });

  it('clears opponentLeftNotice when the opponent reconnects after dropping', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined,
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: true
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Opp');

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined,
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: false
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);
    expect(controller.getState().opponentLeftNoticeName).toBe('');

    controller.destroy();
    vi.useRealTimers();
  });

  it('ignores an opponent disconnect inside the match-start window', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    const startedAt = Date.now() - 2000;
    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseStartedAt: startedAt,
      serverUnixMs: Date.now(),
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseStartedAt: startedAt,
      serverUnixMs: Date.now(),
      lastRoundResult: undefined,
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: true
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);

    controller.destroy();
    vi.useRealTimers();
  });

  it('flags an opponent disconnect once the match-start window has passed', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    const startedAt = Date.now() - 15000;
    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseStartedAt: startedAt,
      serverUnixMs: Date.now(),
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      phaseStartedAt: startedAt,
      serverUnixMs: Date.now(),
      lastRoundResult: undefined,
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: true
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Opp');

    controller.destroy();
    vi.useRealTimers();
  });

  it('flags an opponent disconnect after the match ended, even for a round-1 ending', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    const endedSnap = (overrides: Partial<Snapshot> = {}) =>
      createSnapshot({
        state: 'ended',
        phase: 'ended',
        roundPhase: 'ended',
        phaseEndsAt: Date.now(),
        roundMsLeft: 0,
        currentRound: undefined,
        lastRoundResult: {
          ...createSnapshot().lastRoundResult!,
          roundNumber: 1,
          players: {
            self: { userId: 'self', lat: 0, lng: 0, score: 5000, distanceKm: 0, damageTaken: 0, hpAfterRound: 5000 },
            opp: { userId: 'opp', lat: 0, lng: 0, score: 0, distanceKm: 900, damageTaken: 5000, hpAfterRound: 0 }
          }
        },
        phaseStartedAt: Date.now(),
        serverUnixMs: Date.now(),
        ...overrides
      });

    matchState.snapshot = endedSnap();
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);

    matchState.snapshot = endedSnap({
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: true
        }
      }
    } as Partial<Snapshot>);
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Opp');

    controller.destroy();
    vi.useRealTimers();
  });

  it('keeps opponentLeftNotice while another teammate is still gone in a team duel', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState)
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    const teamPlayers = {
      self: { userId: 'self', displayName: 'Self', teamId: 't1', hp: 5000, mmr: 1000, finalized: false, isGuest: false, disconnected: false },
      a: { userId: 'a', displayName: 'Ace', teamId: 't2', hp: 5000, mmr: 1000, finalized: false, isGuest: false, disconnected: false },
      b: { userId: 'b', displayName: 'Bee', teamId: 't2', hp: 5000, mmr: 1000, finalized: false, isGuest: false, disconnected: false }
    };
    const teamSnap = (overrides: Partial<Snapshot> = {}) =>
      createSnapshot({
        mode: 'team_duel',
        phase: 'live',
        roundPhase: 'round_live',
        lastRoundResult: undefined,
        players: teamPlayers,
        ...overrides
      });

    matchState.snapshot = teamSnap();
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);

    matchState.snapshot = teamSnap({
      players: { ...teamPlayers, a: { ...teamPlayers.a, disconnected: true } }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Ace');

    matchState.snapshot = teamSnap({
      players: {
        ...teamPlayers,
        a: { ...teamPlayers.a, disconnected: true },
        b: { ...teamPlayers.b, disconnected: true }
      }
    });
    listener();
    matchState.snapshot = teamSnap({
      players: { ...teamPlayers, a: { ...teamPlayers.a, disconnected: true } }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Ace');

    matchState.snapshot = teamSnap();
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(false);
    expect(controller.getState().opponentLeftNoticeName).toBe('');

    controller.destroy();
    vi.useRealTimers();
  });

  it('clears opponentLeftNotice when leaving the match', () => {
    let listener: () => void = () => {};
    const matchState = { snapshot: null as Snapshot | null };
    const matchController = {
      subscribe: vi.fn((next: () => void) => {
        listener = next;
        return () => {
          listener = () => {};
        };
      }),
      getState: vi.fn(() => matchState),
      sendGameCommand: vi.fn(() => true),
      resetConnectionState: vi.fn(),
      setStatus: vi.fn()
    } as any;
    const sessionController = {
      getState: vi.fn(() => ({ userId: 'self' }))
    } as any;

    const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
    controller.start();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined
    });
    listener();

    matchState.snapshot = createSnapshot({
      phase: 'live',
      roundPhase: 'round_live',
      lastRoundResult: undefined,
      players: {
        ...createSnapshot().players,
        opp: {
          ...createSnapshot().players.opp,
          disconnected: true
        }
      }
    });
    listener();
    expect(controller.getState().opponentLeftNotice).toBe(true);
    expect(controller.getState().opponentLeftNoticeName).toBe('Opp');

    controller.leaveGame();
    expect(controller.getState().opponentLeftNotice).toBe(false);
    expect(controller.getState().opponentLeftNoticeName).toBe('');

    controller.destroy();
    vi.useRealTimers();
  });
  describe('singleplayer music', () => {
    function createHarness() {
      let listener: () => void = () => {};
      const matchState = { snapshot: null as Snapshot | null };
      const matchController = {
        subscribe: vi.fn((next: () => void) => {
          listener = next;
          return () => {
            listener = () => {};
          };
        }),
        getState: vi.fn(() => matchState)
      } as any;
      const sessionController = {
        getState: vi.fn(() => ({ userId: 'self' }))
      } as any;
      const controller = new GameController({ config: runtimeConfig, matchController, sessionController, sfxController });
      controller.start();
      return {
        controller,
        matchState,
        emit: () => listener()
      };
    }

    function liveSingleplayerSnapshot(overrides: Partial<Snapshot> = {}): Snapshot {
      return createSnapshot({
        mode: 'singleplayer',
        phase: 'live',
        roundPhase: 'round_live',
        state: 'active',
        players: {
          self: {
            userId: 'self',
            displayName: 'Self',
            hp: 0,
            totalScore: 0,
            mmr: 1000,
            finalized: false,
            isGuest: false,
            disconnected: false
          }
        },
        lastRoundResult: undefined,
        ...overrides
      });
    }

    it('plays background music once per active singleplayer match', () => {
      const { controller, matchState, emit } = createHarness();
      matchState.snapshot = liveSingleplayerSnapshot();
      emit();
      matchState.snapshot = liveSingleplayerSnapshot({ eventSequence: 2 });
      emit();
      expect(sfxController.playLoop).toHaveBeenCalledTimes(1);
      expect(sfxController.playLoop).toHaveBeenCalledWith('singleplayer-music');
      controller.destroy();
      vi.useRealTimers();
    });

    it('stops background music when the singleplayer match ends', () => {
      const { controller, matchState, emit } = createHarness();
      matchState.snapshot = liveSingleplayerSnapshot();
      emit();
      matchState.snapshot = liveSingleplayerSnapshot({ state: 'ended' });
      emit();
      expect(sfxController.stop).toHaveBeenCalledWith('singleplayer-music');
      controller.destroy();
      vi.useRealTimers();
    });

    it('muting stops the music and unmuting restarts it for the active match', () => {
      const { controller, matchState, emit } = createHarness();
      matchState.snapshot = liveSingleplayerSnapshot();
      emit();
      expect(controller.toggleMusicMuted()).toBe(true);
      expect(sfxController.stop).toHaveBeenCalledWith('singleplayer-music');
      matchState.snapshot = liveSingleplayerSnapshot({ eventSequence: 2 });
      emit();
      expect(sfxController.playLoop).toHaveBeenCalledTimes(1);
      expect(controller.toggleMusicMuted()).toBe(false);
      expect(sfxController.playLoop).toHaveBeenCalledTimes(2);
      controller.destroy();
      vi.useRealTimers();
    });

    it('picks up mute preference changes from other tabs', () => {
      const { controller, matchState, emit } = createHarness();
      matchState.snapshot = liveSingleplayerSnapshot();
      emit();
      window.localStorage.setItem('geoduels.musicMuted', 'true');
      window.dispatchEvent(new StorageEvent('storage', { key: 'geoduels.musicMuted' }));
      expect(controller.getState().musicMuted).toBe(true);
      matchState.snapshot = liveSingleplayerSnapshot({ eventSequence: 2 });
      emit();
      expect(sfxController.playLoop).toHaveBeenCalledTimes(1);
      controller.destroy();
      vi.useRealTimers();
    });
  });
});
