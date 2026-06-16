import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRuntimeConfigFixture } from "../../../test/runtime-config.fixture";
import { ChatController } from "./chat-controller";

class MockWebSocket {
  static instances: MockWebSocket[] = [];

  url: string;
  readyState: number = WebSocket.CONNECTING;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = WebSocket.CLOSED;
  });

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  emitMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }

  open() {
    this.readyState = WebSocket.OPEN;
  }
}

describe("ChatController", () => {
  const originalWebSocket = global.WebSocket;
  const runtimeConfig = createRuntimeConfigFixture();

  beforeEach(() => {
    MockWebSocket.instances = [];
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket;
    window.WebSocket = MockWebSocket as unknown as typeof WebSocket;
  });

  afterEach(() => {
    global.WebSocket = originalWebSocket;
    window.WebSocket = originalWebSocket;
    vi.restoreAllMocks();
  });

  it("connects to the selected conversation and sends through the active socket", () => {
    const controller = new ChatController({ config: runtimeConfig });

    controller.setConversation("party:lobby-1", "token-1");

    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0]?.url).toBe(
      "ws://localhost:8090/chat/ws?conversationId=party%3Alobby-1&accessToken=token-1",
    );

    MockWebSocket.instances[0]?.open();
    expect(controller.sendMessage(" hello ")).toBe(true);
    expect(MockWebSocket.instances[0]?.send).toHaveBeenCalledWith(
      JSON.stringify({ type: "chat.send", payload: { body: "hello" } }),
    );
  });

  it("caches messages by conversation across temporary conversation changes", () => {
    const controller = new ChatController({ config: runtimeConfig });
    controller.setConversation("party:lobby-1", "token-1");

    MockWebSocket.instances[0]?.emitMessage({
      type: "chat.message",
      payload: {
        id: "chat-1",
        conversationId: "party:lobby-1",
        matchId: "",
        senderUserId: "u2",
        senderDisplayName: "Two",
        kind: "text",
        body: "hi",
        createdAt: "2026-05-18T00:00:00Z",
      },
    });

    expect(controller.getState().messages.map((message) => message.id)).toEqual([
      "chat-1",
    ]);

    controller.setConversation("", "");
    expect(controller.getState().messages).toEqual([]);

    controller.setConversation("party:lobby-1", "token-1");
    expect(controller.getState().messages.map((message) => message.id)).toEqual([
      "chat-1",
    ]);
  });
});
