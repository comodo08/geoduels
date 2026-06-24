export const GEODUELS_EXTENSION_PROTOCOL_VERSION = 1;
export const GEODUELS_EXTENSION_SOURCE = "geoduels-extension";
export const GEODUELS_APP_SOURCE = "geoduels-app";

export type GeoDuelsExtensionCapabilities = {
  heading: boolean;
  roadLabels: boolean;
};

export type GeoDuelsExtensionMessage =
  | {
      source: typeof GEODUELS_EXTENSION_SOURCE;
      version: typeof GEODUELS_EXTENSION_PROTOCOL_VERSION;
      type: "ready";
      capabilities: GeoDuelsExtensionCapabilities;
    }
  | {
      source: typeof GEODUELS_EXTENSION_SOURCE;
      version: typeof GEODUELS_EXTENSION_PROTOCOL_VERSION;
      type: "pov";
      heading: number;
    }
  | {
      source: typeof GEODUELS_EXTENSION_SOURCE;
      version: typeof GEODUELS_EXTENSION_PROTOCOL_VERSION;
      type: "configured";
      ruleset: "moving" | "no_move" | "nmpz";
      streetNames: "shown" | "hidden";
    };

export type GeoDuelsExtensionConfigMessage = {
  source: typeof GEODUELS_APP_SOURCE;
  version: typeof GEODUELS_EXTENSION_PROTOCOL_VERSION;
  type: "configure";
  ruleset: "moving" | "no_move" | "nmpz";
  streetNames: "shown" | "hidden";
};

export function isGeoDuelsExtensionMessage(
  value: unknown,
): value is GeoDuelsExtensionMessage {
  if (!value || typeof value !== "object") return false;
  const message = value as Partial<GeoDuelsExtensionMessage>;
  if (
    message.source !== GEODUELS_EXTENSION_SOURCE ||
    message.version !== GEODUELS_EXTENSION_PROTOCOL_VERSION
  ) {
    return false;
  }
  if (message.type === "ready") {
    return (
      typeof message.capabilities?.heading === "boolean" &&
      typeof message.capabilities?.roadLabels === "boolean"
    );
  }
  if (message.type === "pov") {
    return typeof message.heading === "number";
  }
  return (
    message.type === "configured" &&
    (message.ruleset === "moving" ||
      message.ruleset === "no_move" ||
      message.ruleset === "nmpz") &&
    (message.streetNames === "shown" || message.streetNames === "hidden")
  );
}

export function isTrustedGoogleMapsOrigin(origin: string) {
  try {
    const hostname = new URL(origin).hostname;
    return hostname === "google.com" || hostname.endsWith(".google.com");
  } catch {
    return false;
  }
}
