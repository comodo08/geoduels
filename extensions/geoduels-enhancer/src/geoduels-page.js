(() => {
  "use strict";

  function reportReady() {
    window.postMessage(
      {
        source: "geoduels-extension",
        version: 1,
        type: "extension_ready",
      },
      window.location.origin,
    );
  }

  window.addEventListener("message", (event) => {
    if (
      event.source === window &&
      event.origin === window.location.origin &&
      event.data?.source === "geoduels-app" &&
      event.data?.version === 1 &&
      event.data?.type === "extension_ping"
    ) {
      reportReady();
    }
  });

  reportReady();
})();
