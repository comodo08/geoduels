import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { COUNTRY_CODE_ALLOWLIST } from "./country-flag";
import { COUNTRY_NAMES } from "./country-names";

const source = JSON.parse(
  readFileSync(resolve(process.cwd(), "../../datasets-config/iso3166-alpha2.json"), "utf8"),
) as { countries: Array<{ code: string; name: string }> };

const sourceCodes = source.countries.map((entry) => entry.code);
const sourceNames = Object.fromEntries(
  source.countries.map((entry) => [entry.code, entry.name]),
);

describe("country code source of truth", () => {
  it("generated allowlist matches the source JSON", () => {
    expect([...COUNTRY_CODE_ALLOWLIST].sort()).toEqual(sourceCodes);
  });

  it("generated names match the source JSON", () => {
    expect(COUNTRY_NAMES).toEqual(sourceNames);
  });

  it("allowlist and names cover exactly the same codes", () => {
    expect(Object.keys(COUNTRY_NAMES).sort()).toEqual([...COUNTRY_CODE_ALLOWLIST].sort());
  });
});
