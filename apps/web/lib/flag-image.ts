import type { CSSProperties } from "react";
import { isAllowedCountryCode } from "./country-flag";

export type CountryFlagSize = "sm" | "md";

const baseFlagStyle: CSSProperties = {
  display: "inline-block",
  flexShrink: 0,
  objectFit: "contain",
  borderRadius: "2px",
};

export const COUNTRY_FLAG_STYLES: Record<CountryFlagSize, CSSProperties> = {
  sm: { ...baseFlagStyle, height: "1rem", width: "1.5rem" },
  md: { ...baseFlagStyle, height: "1.25rem", width: "1.875rem" },
};

export function countryFlagImageURL(code: string | null | undefined): string {
  if (!code || !isAllowedCountryCode(code)) {
    return "";
  }
  return `https://flagcdn.com/${code.toLowerCase()}.svg`;
}
