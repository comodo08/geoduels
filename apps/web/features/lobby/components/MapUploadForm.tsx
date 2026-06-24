import { Loader2, Pencil, Upload } from "lucide-react";
import { useState, type Dispatch, type SetStateAction } from "react";
import type { MapUploadQuota, MapVisibility } from "../../maps/lib/maps-client";
import { mapThumbnailOptions, mapThumbnailURL } from "../../maps/lib/map-thumbnails";
import { MapUploadLimitsModal } from "./MapUploadLimitsModal";
import { MapThumbnailPickerModal, type ThumbnailCategory } from "./MapThumbnailPickerModal";
import {
  LobbyActionButton,
  LobbyInput,
  LobbySegmentedControl,
  LobbyTextarea,
} from "./lobby-primitives";

type MapDifficulty = "easy" | "normal" | "hard";

type MapUploadFormProps = {
  isGuest: boolean;
  mapName: string;
  setMapName: Dispatch<SetStateAction<string>>;
  mapDescription: string;
  setMapDescription: Dispatch<SetStateAction<string>>;
  mapDifficulty: MapDifficulty;
  setMapDifficulty: Dispatch<SetStateAction<MapDifficulty>>;
  mapVisibility: MapVisibility;
  setMapVisibility: Dispatch<SetStateAction<MapVisibility>>;
  mapThumbnailKey: string;
  setMapThumbnailKey: Dispatch<SetStateAction<string>>;
  mapThumbnailCategory: ThumbnailCategory;
  setMapThumbnailCategory: Dispatch<SetStateAction<ThumbnailCategory>>;
  mapThumbnailSearch: string;
  setMapThumbnailSearch: Dispatch<SetStateAction<string>>;
  mapFile: File | null;
  setMapFile: Dispatch<SetStateAction<File | null>>;
  mapUploadError: string;
  quota?: MapUploadQuota;
  setMapUploadError: Dispatch<SetStateAction<string>>;
  uploadPending: boolean;
  onUpload: () => void;
};

export function MapUploadForm({
  isGuest,
  mapName,
  setMapName,
  mapDescription,
  setMapDescription,
  mapDifficulty,
  setMapDifficulty,
  mapVisibility,
  setMapVisibility,
  mapThumbnailKey,
  setMapThumbnailKey,
  mapThumbnailCategory,
  setMapThumbnailCategory,
  mapThumbnailSearch,
  setMapThumbnailSearch,
  mapFile,
  setMapFile,
  mapUploadError,
  quota,
  setMapUploadError,
  uploadPending,
  onUpload,
}: MapUploadFormProps) {
  const [limitsOpen, setLimitsOpen] = useState(false);
  const [thumbnailPickerOpen, setThumbnailPickerOpen] = useState(false);
  const selectedThumbnail =
    mapThumbnailOptions.find((item) => item.key === mapThumbnailKey) ||
    mapThumbnailOptions[0];
  const moderationNote = quota?.restrictedByModeration ? " An active moderation restriction currently forces Base limits." : "";
  const quotaBlockedReason = quota && quota.currentMaps >= quota.maxMaps
    ? `You have reached the ${quota.maxMaps.toLocaleString()} map limit for the ${quota.tier} tier.${moderationNote}`
    : quota && quota.currentActiveLocations >= quota.maxActiveLocations
      ? `You have reached the ${quota.maxActiveLocations.toLocaleString()} active-location limit for the ${quota.tier} tier.${moderationNote}`
      : "";
  const limitError = /limit|rate|throughput|too many/i.test(mapUploadError) ? mapUploadError : "";
  const blockedReason = quotaBlockedReason || limitError;
  const uploadDisabled = isGuest || !!quotaBlockedReason || !mapName.trim() || !mapFile || uploadPending;
  const difficultyOptions: Array<{ value: MapDifficulty; label: string }> = [
    { value: "easy", label: "Easy" },
    { value: "normal", label: "Normal" },
    { value: "hard", label: "Hard" },
  ];
  const visibilityOptions: Array<{ value: MapVisibility; label: string }> = [
    { value: "private", label: "Private" },
    { value: "unlisted", label: "Unlisted" },
    { value: "public", label: "Public" },
  ];

  return (
    <>
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_300px]">
        <div className="grid gap-4">
          <LobbyInput value={mapName} onChange={(event) => setMapName(event.target.value)} maxLength={80} placeholder="Map name" disabled={isGuest} className="h-11 rounded-xl font-semibold" aria-label="Map name" />
          <LobbyTextarea value={mapDescription} onChange={(event) => setMapDescription(event.target.value)} maxLength={500} placeholder="Description (optional)" disabled={isGuest} className="min-h-24 resize-none rounded-xl" aria-label="Description" />

          <div className="grid gap-2">
            <p className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">Difficulty</p>
            <LobbySegmentedControl
              value={mapDifficulty}
              items={difficultyOptions}
              onChange={(value) => {
                if (!isGuest) setMapDifficulty(value);
              }}
              className={isGuest ? "pointer-events-none opacity-50" : undefined}
            />
          </div>

          <div className="grid gap-2">
            <p className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">Visibility</p>
            <LobbySegmentedControl
              value={mapVisibility}
              items={visibilityOptions}
              onChange={(value) => {
                if (!isGuest) setMapVisibility(value);
              }}
              className={isGuest ? "pointer-events-none opacity-50" : undefined}
            />
          </div>

          <div className="grid gap-2">
            <p className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">Upload JSON</p>
            <label className="flex min-h-20 cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed border-white/20 bg-black/20 px-4 text-center text-sm font-semibold text-[#a9bfd4] hover:border-[#2ad18f]/50">
              <Upload className="mb-2 text-[#2ad18f]" size={22} />
              {mapFile ? mapFile.name : "Choose JSON file"}
              <input type="file" accept=".json,application/json" className="hidden" disabled={isGuest} onChange={(event) => { setMapFile(event.target.files?.[0] || null); setMapUploadError(""); }} />
            </label>
          </div>

          {mapUploadError ? <p className="text-xs font-semibold text-red-300">{mapUploadError}</p> : null}
          <LobbyActionButton type="button" disabled={uploadDisabled} onClick={onUpload} className="h-11 rounded-xl">
            {uploadPending ? <Loader2 className="mr-2 animate-spin" size={17} /> : <Upload className="mr-2" size={17} />}
            Upload
          </LobbyActionButton>
          {!isGuest ? (
            <button type="button" onClick={() => setLimitsOpen(true)} className={`text-xs font-bold transition hover:text-white ${blockedReason ? "text-amber-200" : "text-[#6f8998]"}`}>
              {blockedReason ? "Why?" : "Limits & tiers"}
            </button>
          ) : null}
        </div>

        <div className="grid content-start gap-2">
          <p className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">Thumbnail</p>
          <button
            type="button"
            disabled={isGuest}
            onClick={() => setThumbnailPickerOpen(true)}
            className="group relative overflow-hidden rounded-xl text-left transition disabled:cursor-not-allowed disabled:opacity-50"
            aria-label={`Choose thumbnail. Current thumbnail: ${selectedThumbnail.label}`}
          >
            <img src={mapThumbnailURL(mapThumbnailKey)} alt="" className="aspect-[16/9] w-full object-cover" />
            <span className="absolute bottom-3 left-3 rounded-full bg-black/70 px-3 py-1 text-xs font-extrabold text-white">
              {selectedThumbnail.label}
            </span>
            <span className="absolute right-3 top-3 inline-flex h-10 w-10 items-center justify-center rounded-full bg-black/70 text-white transition group-hover:bg-accentPrimary">
              <Pencil size={17} aria-hidden="true" />
            </span>
          </button>
        </div>
      </div>
      {thumbnailPickerOpen ? (
        <MapThumbnailPickerModal
          mapThumbnailCategory={mapThumbnailCategory}
          mapThumbnailKey={mapThumbnailKey}
          mapThumbnailSearch={mapThumbnailSearch}
          onClose={() => setThumbnailPickerOpen(false)}
          setMapThumbnailCategory={setMapThumbnailCategory}
          setMapThumbnailKey={setMapThumbnailKey}
          setMapThumbnailSearch={setMapThumbnailSearch}
        />
      ) : null}
      {limitsOpen ? <MapUploadLimitsModal quota={quota} blockedReason={blockedReason} onClose={() => setLimitsOpen(false)} /> : null}
    </>
  );
}
