# Frontend Architecture Rules

## Ownership

- New files under `components/ui` should be generic primitives.
- Existing gameplay-specific components under `components/ui` are transitional legacy ownership and are explicitly tracked by the architecture checker. New gameplay UI belongs under its feature directory.
- Feature UI belongs under `features/<feature>/components`.
- Feature state used by one panel belongs in that panel or a nearby hook.
- `LobbyScreen` should remain a route/composition shell, not a feature owner.

## Styling

Inline `className` is for layout and spacing:

- `flex`, `grid`, `gap-*`, `w-*`, `max-w-*`
- responsive layout classes
- small spacing overrides when a component API would be awkward

Inline `className` must not create visual recipes:

- no `bg-[#...]`
- no `rounded-[...]`
- no `shadow-[...]`
- no `backdrop-blur-*`
- no direct `glass-panel` usage outside approved primitive/legacy files
- no repeated `border border-white/10 bg-black/20` style clusters

Add or reuse a primitive/variant instead:

- `LobbyPanel`
- `LobbyInset`
- `LobbyMutedBox`
- `LobbyDangerNotice`
- `LobbyActionButton`
- `LobbyActionLink`
- `LobbyCardButton`
- `LobbyPill`
- `LobbyNotice`
- `LobbySectionHeader`
- `LobbySegmentedControl`
- `LobbyFieldLabel`
- `LobbyIconButton`
- `LobbyLoadingState`
- `LobbyInput`, `LobbyTextarea`, `LobbySelect`

## Operational UI

Admin, legal, and document surfaces should use operational primitives:

- `Surface variant="operational"`
- `Button`
- `Input`
- `Select`
- `Textarea`
- `Table`
- `Metric`
- `StatusPill`

Admin should not use decorative glass, hero-style sections, or ad hoc gradients.

## Budgets

Run:

```bash
npm --prefix apps/web run lint:architecture
```

The checker prints an advisory report by default. Existing legacy debt is tracked with per-file line and visual-recipe budgets so future changes must hold or reduce the count. A budget is temporary compatibility accounting, not approval to copy the recipe into another file.

Use strict mode when you want the report to fail the command:

```bash
npm --prefix apps/web run lint:architecture:strict
```

The checker is currently a manual/local guardrail and is not invoked by `.github/workflows/ci.yml`. CI runs the frontend unit tests and production build. Run strict architecture checks before submitting frontend standardization changes.
