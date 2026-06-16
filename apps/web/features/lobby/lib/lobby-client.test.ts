import { afterEach, describe, expect, it, vi } from "vitest";
import { createRuntimeConfigFixture } from "../../../test/runtime-config.fixture";
import { createLobby, fetchLobby, streamLobby, updateLobbySettings } from "./lobby-client";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  closed = false;

  constructor(public url: string) {
    MockWebSocket.instances.push(this);
  }

  close() {
    this.closed = true;
  }

  emitMessage(payload: unknown) {
    this.onmessage?.({ data: JSON.stringify(payload) });
  }
}

describe("lobby-client", () => {
  const originalFetch = global.fetch;
  const originalWebSocket = global.WebSocket;
  const runtimeConfig = createRuntimeConfigFixture({
    apiURL: "http://api.example.test",
    queueURL: "http://coordinator.example.test/",
  });

  afterEach(() => {
    global.fetch = originalFetch;
    global.WebSocket = originalWebSocket;
    MockWebSocket.instances = [];
    vi.restoreAllMocks();
  });

  it("sends lobby commands to the match coordinator", async () => {
    global.fetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        id: "lob-1",
        inviteCode: "ABC123",
        ownerUserId: "u1",
        state: "open",
        mode: "duel",
        mapScope: "world",
        members: [],
      }),
    }) as Response) as typeof fetch;

    await createLobby(runtimeConfig, "access-token");
    await updateLobbySettings(
      runtimeConfig,
      "lob-1",
      "access-token",
      { ruleset: "moving", roundTimerMode: "none" },
      "team_duel",
    );

    expect(global.fetch).toHaveBeenNthCalledWith(
      1,
      "http://coordinator.example.test/parties",
      expect.objectContaining({ method: "POST" }),
    );
    expect(global.fetch).toHaveBeenNthCalledWith(
      2,
      "http://coordinator.example.test/parties/lob-1/settings",
      expect.objectContaining({ method: "PATCH" }),
    );
  });

  it("fetches invite snapshots from the match coordinator", async () => {
    global.fetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        id: "lob-1",
        inviteCode: "ABC123",
        ownerUserId: "u1",
        state: "open",
        mode: "duel",
        mapScope: "world",
        members: [],
      }),
    }) as Response) as typeof fetch;

    await fetchLobby(runtimeConfig, "ABC123");

    expect(global.fetch).toHaveBeenCalledWith(
      "http://coordinator.example.test/parties/ABC123",
    );
  });

  it("streams lobby websocket snapshots", async () => {
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket;
    const controller = new AbortController();
    const onEvent = vi.fn();
    const ready = streamLobby(
      runtimeConfig,
      {
        userId: "u1",
        accessToken: "access-token",
        onboardingRequired: false,
        nicknameInput: "",
      },
      "lob-1",
      controller.signal,
      onEvent,
    );

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0]?.url).toBe(
      "ws://coordinator.example.test/parties/lob-1/ws?accessToken=access-token",
    );
    MockWebSocket.instances[0]?.emitMessage({
      type: "party_snapshot",
      payload: {
        id: "lob-1",
        inviteCode: "ABC123",
        ownerUserId: "u1",
        state: "open",
        mode: "duel",
        mapScope: "world",
        members: [{ userId: "u1", displayName: "Player", role: "owner", connected: true }],
      },
    });

    expect(onEvent).toHaveBeenCalledWith({
      type: "party_snapshot",
      lobby: expect.objectContaining({
        id: "lob-1",
        inviteCode: "ABC123",
        members: [expect.objectContaining({ connected: true })],
      }),
    });
    controller.abort();
    await expect(ready).rejects.toMatchObject({ name: "AbortError" });
    expect(MockWebSocket.instances[0]?.closed).toBe(true);
  });
});
