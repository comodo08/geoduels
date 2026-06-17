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
npm run maps:thumbnails
```

The script crops/resizes each source image to `1280x720`, writes optimized WebP files to `public/map-thumbnails/`, and regenerates the thumbnail picker catalog.

Use `npm run maps:thumbnails:check` in CI or before commits to verify required country thumbnails exist without rewriting files.
