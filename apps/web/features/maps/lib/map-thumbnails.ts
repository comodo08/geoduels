export type MapThumbnailCategory = "generic" | "continents" | "countries";

export type MapThumbnailOption = {
  key: string;
  label: string;
  category: MapThumbnailCategory;
  search: string;
};

const generic = Array.from({ length: 5 }, (_, index) => {
  const variant = index + 1;
  return {
    key: `generic/variant-${variant}`,
    label: `Generic ${variant}`,
    category: "generic" as const,
    search: `generic default stock variant ${variant}`,
  };
});

const continents: Array<[string, string]> = [
  ["africa", "Africa"],
  ["antarctica", "Antarctica"],
  ["asia", "Asia"],
  ["europe", "Europe"],
  ["north-america", "North America"],
  ["oceania", "Oceania"],
  ["south-america", "South America"],
];

const countries: Array<[string, string]> = [
  ["argentina", "Argentina"],
  ["australia", "Australia"],
  ["brazil", "Brazil"],
  ["canada", "Canada"],
  ["china", "China"],
  ["france", "France"],
  ["germany", "Germany"],
  ["india", "India"],
  ["indonesia", "Indonesia"],
  ["italy", "Italy"],
  ["japan", "Japan"],
  ["mexico", "Mexico"],
  ["netherlands", "Netherlands"],
  ["norway", "Norway"],
  ["south-africa", "South Africa"],
  ["south-korea", "South Korea"],
  ["spain", "Spain"],
  ["sweden", "Sweden"],
  ["turkey", "Turkey"],
  ["united-kingdom", "United Kingdom"],
  ["united-states", "United States"],
];

export const mapThumbnailOptions: MapThumbnailOption[] = [
  ...generic,
  ...continents.map(([slug, label]) => ({
    key: `continents/${slug}`,
    label,
    category: "continents" as const,
    search: `${label} continent`,
  })),
  ...countries.map(([slug, label]) => ({
    key: `countries/${slug}`,
    label,
    category: "countries" as const,
    search: `${label} country nation`,
  })),
];

export function mapThumbnailURL(key?: string, variant?: number) {
  const fallback = `generic/variant-${Math.max(1, Math.min(5, variant || 1))}`;
  const selected = mapThumbnailOptions.some((item) => item.key === key) ? key : fallback;
  return `/map-thumbnails/${selected}.webp`;
}

export function validMapThumbnailKey(key: string) {
  return mapThumbnailOptions.some((item) => item.key === key);
}
