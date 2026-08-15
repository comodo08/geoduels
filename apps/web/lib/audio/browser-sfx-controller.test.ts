import { afterEach, describe, expect, it, vi } from 'vitest';
import { BrowserSfxController } from './browser-sfx-controller';

function installFakeWebAudio() {
  const gain = {
    gain: { value: 1 },
    connect: vi.fn(),
    disconnect: vi.fn()
  };
  const source = {
    buffer: null,
    loop: false,
    onended: null,
    connect: vi.fn(),
    disconnect: vi.fn(),
    start: vi.fn(),
    stop: vi.fn()
  };
  const context = {
    state: 'running',
    destination: {},
    createGain: vi.fn(() => gain),
    createBufferSource: vi.fn(() => source),
    decodeAudioData: vi.fn(async () => ({})),
    resume: vi.fn(async () => {}),
    close: vi.fn(async () => {})
  };
  const AudioContextCtor = function () {
    return context;
  } as unknown as typeof AudioContext;
  Object.defineProperty(window, 'AudioContext', {
    configurable: true,
    writable: true,
    value: AudioContextCtor
  });
  return { context, source, gain };
}

describe('BrowserSfxController', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('skips lazy sounds during preload and loads them on first play', async () => {
    installFakeWebAudio();
    const fetchMock = vi.fn(async () => ({
      ok: true,
      arrayBuffer: async () => new ArrayBuffer(8)
    }));
    vi.stubGlobal('fetch', fetchMock);

    const controller = new BrowserSfxController();
    controller.start();
    window.dispatchEvent(new Event('pointerdown'));

    await vi.waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/sfx/chat.v1.ogg');
    });
    expect(fetchMock).not.toHaveBeenCalledWith('/sfx/singleplayer-music.ogg');

    controller.playLoop('singleplayer-music');
    await vi.waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/sfx/singleplayer-music.ogg');
    });

    controller.destroy();
  });
});
