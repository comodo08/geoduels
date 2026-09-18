# GeoDuels Enhancer

The extension operates inside Google Maps Embed frames because the web app cannot cross that iframe boundary. The app owns visible controls; enhancements remain inactive until a trusted GeoDuels embedding origin/configuration is established. Preserve that origin gate when changing the bridge.

For localhost development, load this directory unpacked in Chrome/Brave, or select its `manifest.json` through Firefox's `about:debugging`. The checked-in manifest includes localhost permissions.

From the repository root:

```sh
npm --prefix web test -- features/browser-extension/lib/google-street-view-script.test.ts
node extension/scripts/package.mjs
```

Packaging requires `zip` and writes Chrome/Firefox archives under `extension/dist/geoduels-enhancer/`. It removes localhost permissions; use the source directory, not a production package, for local development. Chrome loads the generated `chrome` directory unpacked; Firefox loads the generated manifest as a temporary add-on. In-game controls appear only after the Google frame reports bridge availability.
