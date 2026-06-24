import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type RefObject,
} from "react";
import {
  GEODUELS_APP_SOURCE,
  GEODUELS_EXTENSION_PROTOCOL_VERSION,
  GEODUELS_EXTENSION_SOURCE,
  isGeoDuelsExtensionMessage,
  isExtensionVersionSupported,
  isTrustedGoogleMapsOrigin,
  type GeoDuelsExtensionCapabilities,
  type GeoDuelsExtensionConfigMessage,
} from "../lib/geoduels-extension-protocol";

export function useGeoDuelsExtension(
  streetViewFrameRef: RefObject<HTMLIFrameElement>,
  streetViewSrc: string,
  ruleset: "moving" | "no_move" | "nmpz",
  streetNames: "shown" | "hidden",
) {
  const [capabilities, setCapabilities] =
    useState<GeoDuelsExtensionCapabilities | null>(null);
  const [heading, setHeading] = useState(0);
  const headingRef = useRef(0);
  const [configured, setConfigured] = useState(false);
  const [unsupportedVersion, setUnsupportedVersion] = useState<string | null>(null);

  const sendConfiguration = useCallback(() => {
    const target = streetViewFrameRef.current?.contentWindow;
    if (!target) return;
    const message: GeoDuelsExtensionConfigMessage = {
      source: GEODUELS_APP_SOURCE,
      version: GEODUELS_EXTENSION_PROTOCOL_VERSION,
      type: "configure",
      ruleset,
      streetNames,
    };
    target.postMessage(message, "https://www.google.com");
  }, [ruleset, streetNames, streetViewFrameRef]);

  useEffect(() => {
    setCapabilities(null);
    setConfigured(false);
    setUnsupportedVersion(null);
  }, [streetViewSrc, ruleset, streetNames]);

  useEffect(() => {
    const onMessage = (event: MessageEvent<unknown>) => {
      if (
        event.source !== streetViewFrameRef.current?.contentWindow ||
        !isTrustedGoogleMapsOrigin(event.origin) ||
        !event.data ||
        typeof event.data !== "object"
      ) {
        return;
      }
      const rawMessage = event.data as Record<string, unknown>;
      if (
        rawMessage.source === GEODUELS_EXTENSION_SOURCE &&
        rawMessage.version === GEODUELS_EXTENSION_PROTOCOL_VERSION &&
        typeof rawMessage.type === "string" &&
        !isExtensionVersionSupported(rawMessage.extensionVersion)
      ) {
        setUnsupportedVersion(
          typeof rawMessage.extensionVersion === "string"
            ? rawMessage.extensionVersion
            : null,
        );
        return;
      }
      if (!isGeoDuelsExtensionMessage(event.data)) {
        return;
      }
      const message = event.data;
      setUnsupportedVersion(null);

      if (message.type === "ready") {
        setCapabilities(message.capabilities);
        window.setTimeout(sendConfiguration, 0);
        return;
      }

      if (message.type === "configured") {
        setConfigured(
          message.ruleset === ruleset &&
            message.streetNames === streetNames,
        );
        return;
      }

      {
        const current = headingRef.current;
        const normalizedCurrent = ((current % 360) + 360) % 360;
        let delta = message.heading - normalizedCurrent;
        if (delta > 180) delta -= 360;
        if (delta < -180) delta += 360;
        headingRef.current = current + delta;
        setHeading(headingRef.current);
      }
    };

    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [ruleset, sendConfiguration, streetNames, streetViewFrameRef]);

  useEffect(() => {
    if (capabilities) sendConfiguration();
  }, [capabilities, sendConfiguration]);

  return {
    available: capabilities !== null,
    capabilities,
    heading,
    configured,
    unsupportedVersion,
  };
}
