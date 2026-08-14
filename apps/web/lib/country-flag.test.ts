import { describe, expect, it } from "vitest";
import {
  COUNTRY_CODE_ALLOWLIST,
  countryCodeToEmoji,
  isAllowedCountryCode,
} from "./country-flag";

describe("countryCodeToEmoji", () => {
  it("maps a valid alpha-2 code to its regional indicator flag", () => {
    expect(countryCodeToEmoji("FR")).toBe("🇫🇷");
    expect(countryCodeToEmoji("US")).toBe("🇺🇸");
    expect(countryCodeToEmoji("ZW")).toBe("🇿🇼");
  });

  it("returns an empty string for empty or invalid codes", () => {
    expect(countryCodeToEmoji("")).toBe("");
    expect(countryCodeToEmoji("XX")).toBe("");
    expect(countryCodeToEmoji("fr")).toBe("");
    expect(countryCodeToEmoji("FRA")).toBe("");
    expect(countryCodeToEmoji("EU")).toBe("");
  });

  it("never emits arbitrary unicode for disallowed input", () => {
    expect(countryCodeToEmoji("ZWJ")).toBe("");
    expect(countryCodeToEmoji("1A")).toBe("");
  });
});

describe("isAllowedCountryCode", () => {
  it("matches the generated allowlist", () => {
    expect(isAllowedCountryCode("FR")).toBe(true);
    expect(isAllowedCountryCode("xx")).toBe(false);
    expect(COUNTRY_CODE_ALLOWLIST.has("US")).toBe(true);
    expect(COUNTRY_CODE_ALLOWLIST.size).toBeGreaterThan(200);
  });
});
