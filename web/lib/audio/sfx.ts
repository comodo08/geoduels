export type SfxName =
  | 'chat'
  | 'select'
  | 'duel-game-start'
  | 'duel-round-countdown'
  | 'duel-round-guess'
  | 'duel-round-result-countdown'
  | 'duel-round-result-enter'
  | 'duel-round-result-exit'
  | 'duel-round-result-score-reveal'
  | 'duel-round-result-hp-hit';

export type SfxSource = {
  src: string;
  type: string;
};

export type SfxDefinition = {
  sources: readonly SfxSource[];
  volume?: number;
};

export type SfxRegistry = Record<SfxName, SfxDefinition>;

export interface SfxController {
  start(): void;
  destroy(): void;
  play(name: SfxName): void;
  playManaged(name: SfxName): void;
  playLoop(name: SfxName): void;
  stop(name: SfxName): void;
}

export const sfxRegistry: SfxRegistry = {
  chat: {
    sources: [{ src: '/sfx/chat.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.75
  },
  select: {
    sources: [{ src: '/sfx/select.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.75
  },
  'duel-game-start': {
    sources: [{ src: '/sfx/duel-game-start.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.85
  },
  'duel-round-guess': {
    sources: [{ src: '/sfx/duel-round-guess.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.8
  },
  'duel-round-countdown': {
    sources: [{ src: '/sfx/duel-round-countdown.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.75
  },
  'duel-round-result-countdown': {
    sources: [{ src: '/sfx/duel-round-result-countdown.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.75
  },
  'duel-round-result-enter': {
    sources: [{ src: '/sfx/duel-round-result-enter.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.85
  },
  'duel-round-result-score-reveal': {
    sources: [{ src: '/sfx/duel-round-result-score-reveal.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.9
  },
  'duel-round-result-hp-hit': {
    sources: [{ src: '/sfx/duel-round-result-hp-hit.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.55
  },
  'duel-round-result-exit': {
    sources: [{ src: '/sfx/duel-round-result-exit.v1.ogg', type: 'audio/ogg; codecs=vorbis' }],
    volume: 0.8
  }
};

export class NullSfxController implements SfxController {
  start() {}

  destroy() {}

  play(_name: SfxName) {}

  playManaged(_name: SfxName) {}

  playLoop(_name: SfxName) {}

  stop(_name: SfxName) {}
}
