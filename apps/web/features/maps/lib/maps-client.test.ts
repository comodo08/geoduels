import { afterEach, describe, expect, it, vi } from "vitest";
import { listMaps, validateMapFile } from "./maps-client";
import type { RuntimeConfig } from "../../../lib/runtime-config";

function jsonFile(value: unknown) {
  const text = JSON.stringify(value);
  return {
    size: text.length,
    text: async () => text,
  } as File;
}

const config = { apiURL: "https://api.test" } as RuntimeConfig;

afterEach(() => {
  vi.restoreAllMocks();
});

describe("validateMapFile", () => {
  it("accepts map-making.app exports with nested pano ids", async () => {
    const locations = [
      ["3P5a5OtPyfh9ByBzuHANrg", -0.15796175210211638, 37.7503211982208],
      ["Jd2sr079nrj3lhQ9Ab_kEw", 6.428595321124186, -1.4132091930831705],
      ["b184OQ0GzootvmVPP6CuAg", 6.5462559963431595, 0.25167556026892035],
      ["cQ5UcYcmtkqA3oVIvOR8jA", 6.2225031210101065, -1.383655971662562],
      ["exGCRe5MBhjvC1PBBUzWLg", 8.555462932718319, -2.2134015863200154],
    ].map(([panoId, lat, lng]) => ({
      lat,
      lng,
      heading: 88,
      pitch: 0,
      zoom: 0,
      panoId: null,
      countryCode: null,
      stateCode: null,
      extra: { panoId, panoDate: "2025-02" },
    }));

    await expect(validateMapFile(jsonFile({ name: "test", customCoordinates: locations }))).resolves.toBe(5);
  });
});

describe("listMaps", () => {
  it("sends trimmed search and trending sort", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      json: async () => [],
    } as Response);

    await expect(listMaps(config, undefined, { scope: "community", sort: "trending", search: "  source world  " })).resolves.toEqual([]);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = new URL(String(fetchMock.mock.calls[0][0]));
    expect(url.origin + url.pathname).toBe("https://api.test/v1/maps");
    expect(url.searchParams.get("scope")).toBe("community");
    expect(url.searchParams.get("sort")).toBe("trending");
    expect(url.searchParams.get("search")).toBe("source world");
  });
});
