import { afterEach, describe, expect, it, vi } from "vitest";
import { createRuntimeConfig } from "../../../lib/runtime-config";
import { requestPlayerMatches } from "./player-client";

describe("player client", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("passes cursor pagination parameters", async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({
      matches: [],
      nextCursor: "",
    }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await requestPlayerMatches(createRuntimeConfig(), "player-1", 20, "cursor-value");

    expect(fetchMock).toHaveBeenCalledWith(
      "/v1/players/player-1/matches?limit=20&cursor=cursor-value",
      undefined,
    );
  });
});
