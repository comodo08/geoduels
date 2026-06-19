# Map Thumbnail Sources

Generic thumbnail source images live in `generic/`:

```text
generic/variant-1.png
generic/variant-2.png
generic/variant-3.png
generic/variant-4.png
generic/variant-5.png
```

Drop source country images in `countries/` using ISO 3166-1 alpha-2 country codes:

```text
countries/US.jpg
countries/GB.png
countries/FR.webp
```

Then run:

```bash
npm --prefix apps/web run maps:thumbnails
```

The script crops/resizes each source image to `1280x720`, writes optimized WebP files to `public/map-thumbnails/`, and regenerates the thumbnail picker catalog.

Use the check command before commits:

```bash
npm --prefix apps/web run maps:thumbnails:check
```

Check mode verifies that every configured generic output and required country output exists. It reports missing optional country sources but does not rewrite images or regenerate the TypeScript catalog.

The command exits nonzero when a configured generic or required country output is missing. A checkout that contains only the generic sources will therefore fail this check until the required country images are supplied and generated.

When already inside `apps/web`, the shorter `npm run maps:thumbnails` and `npm run maps:thumbnails:check` forms are equivalent.
