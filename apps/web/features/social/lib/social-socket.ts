import type { RuntimeConfig } from "../../../lib/runtime-config";
import { socialWSBase } from "../../../lib/runtime-config";

export type SocialSocketEvent =
  | { type: "presence"; payload: Record<string, string> }
  | { type: "friend_request"; payload: Record<string, unknown> }
  | { type: "friend_accepted"; payload: Record<string, unknown> }
  | { type: "party_invite"; payload: Record<string, unknown> }
  | { type: "party_invite_dismissed"; payload: Record<string, unknown> }
  | { type: "connected" }
  | { type: "closed" };

type Listener = (event: SocialSocketEvent) => void;

export class SocialSocket {
  private ws: WebSocket | null = null;
  private readonly listeners = new Set<Listener>();
  private closedByUser = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private presenceTimer: ReturnType<typeof setInterval> | null = null;

  connect(config: RuntimeConfig, accessToken: string, signal: AbortSignal) {
    if (typeof window === "undefined") return;
    this.closedByUser = false;

    const open = () => {
      if (this.closedByUser || signal.aborted) return;
      const target = `${socialWSBase(config)}/v1/social/ws?accessToken=${encodeURIComponent(accessToken)}`;
      const ws = new WebSocket(target);
      this.ws = ws;

      ws.onopen = () => this.emit({ type: "connected" });
      ws.onmessage = (evt) => {
        let msg: any;
        try {
          msg = JSON.parse(String(evt.data));
        } catch {
          return;
        }
        if (!msg || typeof msg.type !== "string") return;
        this.emit({
          type: msg.type,
          payload: (msg.payload ?? {}) as Record<string, unknown>,
        } as SocialSocketEvent);
      };
      ws.onclose = () => {
        this.emit({ type: "closed" });
        if (!this.closedByUser && !signal.aborted) {
          this.reconnectTimer = setTimeout(open, 3000);
        }
      };
      ws.onerror = () => {
        try {
          ws.close();
        } catch {
          // ignore
        }
      };
    };

    open();

    signal.addEventListener("abort", () => this.close(), { once: true });
  }

  onMessage(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private emit(event: SocialSocketEvent) {
    for (const listener of this.listeners) listener(event);
  }

  close() {
    this.closedByUser = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.presenceTimer) {
      clearInterval(this.presenceTimer);
      this.presenceTimer = null;
    }
    if (this.ws) {
      try {
        this.ws.close();
      } catch {
        // ignore
      }
      this.ws = null;
    }
    this.listeners.clear();
  }
}

export function diffPresence(
  previous: Record<string, string>,
  next: Record<string, string>,
): Record<string, string> {
  const changed: Record<string, string> = {};
  const keys = new Set([...Object.keys(previous), ...Object.keys(next)]);
  for (const key of keys) {
    const before = key in previous ? previous[key] : "unknown";
    const after = key in next ? next[key] : "offline";
    if (before !== after) changed[key] = after;
  }
  return changed;
}
