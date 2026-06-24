import { useEffect, useRef, useState } from "react";

const PING_INTERVAL_MS = 2_000;
const EXPIRY_MS = 5_000;
const INITIAL_DISCOVERY_MS = 750;

export function useExtensionAvailability() {
  const [available, setAvailable] = useState<boolean | null>(null);
  const lastSeenRef = useRef(0);

  useEffect(() => {
    const ping = () => {
      window.postMessage(
        { source: "geoduels-app", version: 1, type: "extension_ping" },
        window.location.origin,
      );
      if (
        lastSeenRef.current > 0 &&
        Date.now() - lastSeenRef.current > EXPIRY_MS
      ) {
        setAvailable(false);
      }
    };
    const onMessage = (event: MessageEvent<unknown>) => {
      if (
        event.source !== window ||
        event.origin !== window.location.origin ||
        !event.data ||
        typeof event.data !== "object"
      ) {
        return;
      }
      const message = event.data as Record<string, unknown>;
      if (
        message.source === "geoduels-extension" &&
        message.version === 1 &&
        message.type === "extension_ready"
      ) {
        lastSeenRef.current = Date.now();
        setAvailable(true);
      }
    };

    window.addEventListener("message", onMessage);
    ping();
    const timer = window.setInterval(ping, PING_INTERVAL_MS);
    const discoveryTimer = window.setTimeout(() => {
      if (lastSeenRef.current === 0) setAvailable(false);
    }, INITIAL_DISCOVERY_MS);
    return () => {
      window.removeEventListener("message", onMessage);
      window.clearInterval(timer);
      window.clearTimeout(discoveryTimer);
    };
  }, []);

  return available;
}
