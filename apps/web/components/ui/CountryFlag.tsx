import { useState } from "react";
import {
  COUNTRY_FLAG_STYLES,
  countryFlagImageURL,
  type CountryFlagSize,
} from "../../lib/flag-image";
import { countryCodeToEmoji } from "../../lib/country-flag";

export type CountryFlagProps = {
  code?: string | null;
  size?: CountryFlagSize;
  className?: string;
  title?: string;
};

export function CountryFlag({
  code,
  size = "sm",
  className,
  title,
}: CountryFlagProps) {
  const [imageFailed, setImageFailed] = useState(false);
  const src = code ? countryFlagImageURL(code) : "";
  if (!src) return null;
  const label = title ?? code ?? "";
  if (imageFailed) {
    return (
      <span
        style={COUNTRY_FLAG_STYLES[size]}
        className={className}
        title={label}
        aria-hidden="true"
      >
        {countryCodeToEmoji(code ?? "")}
      </span>
    );
  }
  return (
    <img
      src={src}
      alt={code ?? ""}
      title={label}
      style={COUNTRY_FLAG_STYLES[size]}
      className={className}
      onError={() => setImageFailed(true)}
      aria-hidden="true"
    />
  );
}
