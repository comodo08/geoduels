import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useExtensionAvailability } from "./use-extension-availability";

function Harness() {
  const available = useExtensionAvailability();
  return <div>{available === null ? "checking" : available ? "available" : "missing"}</div>;
}

describe("useExtensionAvailability", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("detects the official extension and expires a stale handshake", () => {
    render(<Harness />);
    expect(screen.getByText("checking")).toBeInTheDocument();

    act(() => {
      window.dispatchEvent(
        new MessageEvent("message", {
          source: window,
          origin: window.location.origin,
          data: {
            source: "geoduels-extension",
            version: 1,
            type: "extension_ready",
          },
        }),
      );
    });
    expect(screen.getByText("available")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(6_100);
    });
    expect(screen.getByText("missing")).toBeInTheDocument();
  });
});
