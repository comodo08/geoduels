#!/usr/bin/env node
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import sharp from "sharp";

const root = process.cwd();
const configPath = path.join(root, "config/map-thumbnails.json");
const sourceDir = path.join(root, "assets/source-map-thumbnails/countries");
const publicRoot = path.join(root, "public/map-thumbnails");
const countriesOutDir = path.join(publicRoot, "countries");
const catalogPath = path.join(root, "features/maps/lib/map-thumbnails.ts");
const sourceExtensions = [".jpg", ".jpeg", ".png", ".webp", ".avif", ".tif", ".tiff"];

const args = new Set(process.argv.slice(2));
const checkOnly = args.has("--check");

function slugify(value) {
  return value
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/&/g, " and ")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function countryName(code) {
  const display = new Intl.DisplayNames(["en"], { type: "region" });
  return display.of(code) || code;
}

async function exists(file) {
  try {
    await fs.access(file);
    return true;
  } catch {
    return false;
  }
}

async function findSource(code) {
  for (const ext of sourceExtensions) {
    for (const name of [code, code.toLowerCase()]) {
      const file = path.join(sourceDir, `${name}${ext}`);
      if (await exists(file)) return file;
    }
  }
  return "";
}

function tsString(value) {
  return JSON.stringify(value);
}

function optionLine(option) {
  return `  { key: ${tsString(option.key)}, label: ${tsString(option.label)}, category: ${tsString(option.category)}, search: ${tsString(option.search)} },`;
}

async function main() {
  const config = JSON.parse(await fs.readFile(configPath, "utf8"));
  await fs.mkdir(countriesOutDir, { recursive: true });

  const generic = Array.from({ length: config.genericCount || 5 }, (_, index) => {
    const variant = index + 1;
    return {
      key: `generic/variant-${variant}`,
      label: `Generic ${variant}`,
      category: "generic",
      search: `generic default stock variant ${variant}`,
    };
  });

  const continents = (config.continents || []).map((item) => ({
    key: `continents/${item.slug}`,
    label: item.name,
    category: "continents",
    search: `${item.name} continent`,
  }));

  const generatedCountries = [];
  const missingRequired = [];
  const missingOptional = [];

  for (const rawCode of config.streetViewCountries || []) {
    const code = String(rawCode).trim().toUpperCase();
    if (!code) continue;
    const label = countryName(code);
    const slug = slugify(label);
    const source = await findSource(code);
    const output = path.join(countriesOutDir, `${slug}.webp`);
    const hasExistingOutput = await exists(output);

    if (source && !checkOnly) {
      await sharp(source)
        .resize(1280, 720, { fit: "cover", position: "center" })
        .webp({ quality: 82 })
        .toFile(output);
    }

    const hasOutput = source || hasExistingOutput || (await exists(output));
    if (!hasOutput) {
      if ((config.requiredCountries || []).includes(code)) {
        missingRequired.push(`${code} ${label}`);
      } else {
        missingOptional.push(`${code} ${label}`);
      }
      continue;
    }

    const aliases = config.aliases?.[code] || [];
    generatedCountries.push({
      key: `countries/${slug}`,
      label,
      category: "countries",
      search: [label, code, ...aliases, "country", "street view"].join(" "),
    });
  }

  const content = `export type MapThumbnailCategory = "generic" | "continents" | "countries";

export type MapThumbnailOption = {
  key: string;
  label: string;
  category: MapThumbnailCategory;
  search: string;
};

export const mapThumbnailOptions: MapThumbnailOption[] = [
${[...generic, ...continents, ...generatedCountries].map(optionLine).join("\n")}
];

export function mapThumbnailURL(key?: string, variant?: number) {
  const fallback = \`generic/variant-\${Math.max(1, Math.min(5, variant || 1))}\`;
  const selected = mapThumbnailOptions.some((item) => item.key === key) ? key : fallback;
  return \`/map-thumbnails/\${selected}.webp\`;
}

export function validMapThumbnailKey(key: string) {
  return mapThumbnailOptions.some((item) => item.key === key);
}
`;

  if (missingOptional.length > 0) {
    console.warn(`Optional country thumbnails missing (${missingOptional.length}):`);
    for (const item of missingOptional) console.warn(`- ${item}`);
  }
  if (missingRequired.length > 0) {
    console.error(`Required country thumbnails missing (${missingRequired.length}):`);
    for (const item of missingRequired) console.error(`- ${item}`);
    process.exitCode = 1;
    return;
  }

  if (!checkOnly) {
    await fs.writeFile(catalogPath, content);
  }

  console.log(`${checkOnly ? "Checked" : "Generated"} ${generatedCountries.length} country thumbnails and ${generic.length + continents.length + generatedCountries.length} catalog options.`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
