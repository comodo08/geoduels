(() => {
  "use strict";

  if (window.top === window) return;

  const PROTOCOL_VERSION = 1;
  const EXTENSION_SOURCE = "geoduels-extension";
  const APP_SOURCE = "geoduels-app";
  const instances = new Set();
  let ruleset = "moving";
  let streetNames = "shown";
  let patchedConstructor = null;
  let readyReported = false;

  function isAllowedGeoDuelsOrigin(origin) {
    try {
      const url = new URL(origin);
      return (
        url.hostname === "geoduels.io" ||
        url.hostname.endsWith(".geoduels.io") ||
        url.hostname === "localhost" ||
        url.hostname === "127.0.0.1"
      );
    } catch {
      return false;
    }
  }

  if (!isAllowedGeoDuelsOrigin(document.referrer)) return;

  function postToApp(message) {
    window.top.postMessage(
      {
        source: EXTENSION_SOURCE,
        version: PROTOCOL_VERSION,
        ...message,
      },
      "*",
    );
  }

  function reportHeading(instance) {
    const heading = instance.getPov?.().heading;
    if (Number.isFinite(heading)) {
      postToApp({ type: "pov", heading });
    }
  }

  function applyConfiguration(instance) {
    const noMove = ruleset === "no_move";
    instance.setOptions?.({
      showRoadLabels: streetNames !== "hidden",
      clickToGo: !noMove,
      linksControl: !noMove,
    });
    if (noMove) {
      const position = instance.__geoduelsSpawnPosition;
      if (position) instance.setPosition?.(position);
    }
  }

  function registerInstance(instance) {
    instances.add(instance);
    const captureSpawn = () => {
      if (!instance.__geoduelsSpawnPosition) {
        const position = instance.getPosition?.();
        if (position) instance.__geoduelsSpawnPosition = position;
      }
    };
    captureSpawn();
    applyConfiguration(instance);
    instance.addListener?.("pov_changed", () => reportHeading(instance));
    instance.addListener?.("visible_changed", () => reportHeading(instance));
    instance.addListener?.("position_changed", () => {
      captureSpawn();
      if (ruleset === "no_move" && instance.__geoduelsSpawnPosition) {
        const current = instance.getPosition?.();
        const spawn = instance.__geoduelsSpawnPosition;
        if (
          current &&
          (Math.abs(current.lat() - spawn.lat()) > 0.0000001 ||
            Math.abs(current.lng() - spawn.lng()) > 0.0000001)
        ) {
          instance.setPosition?.(spawn);
        }
      }
    });
    window.setTimeout(() => reportHeading(instance), 0);
  }

  function reportReady() {
    postToApp({
      type: "ready",
      capabilities: { heading: true, roadLabels: true },
    });
  }

  function patchStreetView() {
    const StreetViewPanorama =
      window.google?.maps?.StreetViewPanorama;
    if (
      typeof StreetViewPanorama !== "function" ||
      StreetViewPanorama === patchedConstructor
    ) {
      return;
    }

    const wrapped = new Proxy(StreetViewPanorama, {
      construct(target, args, newTarget) {
        const nextArgs = [...args];
        nextArgs[1] = {
          ...(nextArgs[1] || {}),
          showRoadLabels: streetNames !== "hidden",
          clickToGo: ruleset !== "no_move",
          linksControl: ruleset !== "no_move",
        };
        const instance = Reflect.construct(target, nextArgs, newTarget);
        registerInstance(instance);
        return instance;
      },
    });

    patchedConstructor = wrapped;
    window.google.maps.StreetViewPanorama = wrapped;
    if (!readyReported) {
      readyReported = true;
      reportReady();
      window.setTimeout(reportReady, 250);
      window.setTimeout(reportReady, 1_000);
    }
  }

  window.addEventListener("message", (event) => {
    if (
      event.source !== window.top ||
      !isAllowedGeoDuelsOrigin(event.origin) ||
      !event.data ||
      event.data.source !== APP_SOURCE ||
      event.data.version !== PROTOCOL_VERSION ||
      event.data.type !== "configure"
    ) {
      return;
    }

    ruleset =
      event.data.ruleset === "no_move" || event.data.ruleset === "nmpz"
        ? event.data.ruleset
        : "moving";
    streetNames =
      event.data.streetNames === "hidden" ? "hidden" : "shown";
    instances.forEach(applyConfiguration);
    instances.forEach(reportHeading);
    postToApp({ type: "configured", ruleset, streetNames });
  });

  window.addEventListener(
    "keydown",
    (event) => {
      if (
        ruleset === "no_move" &&
        ["ArrowUp", "ArrowDown", "w", "W", "s", "S"].includes(event.key)
      ) {
        event.preventDefault();
        event.stopImmediatePropagation();
      }
    },
    true,
  );

  const patchInterval = window.setInterval(patchStreetView, 10);
  window.setTimeout(() => window.clearInterval(patchInterval), 15_000);
})();
