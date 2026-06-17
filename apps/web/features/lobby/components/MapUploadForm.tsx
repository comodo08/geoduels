import { Loader2, Upload } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { mapThumbnailOptions, mapThumbnailURL } from "../../maps/lib/map-thumbnails";
import {
  LobbyActionButton,
  LobbyInput,
  LobbyPanel,
  LobbySelect,
  LobbyTextarea,
} from "./lobby-primitives";

type ThumbnailCategory = "generic" | "continents" | "countries";
type MapDifficulty = "easy" | "normal" | "hard";

type MapUploadFormProps = {
  isGuest: boolean;
  mapName: string;
  setMapName: Dispatch<SetStateAction<string>>;
  mapDescription: string;
  setMapDescription: Dispatch<SetStateAction<string>>;
  mapDifficulty: MapDifficulty;
  setMapDifficulty: Dispatch<SetStateAction<MapDifficulty>>;
  mapThumbnailKey: string;
  setMapThumbnailKey: Dispatch<SetStateAction<string>>;
  mapThumbnailCategory: ThumbnailCategory;
  setMapThumbnailCategory: Dispatch<SetStateAction<ThumbnailCategory>>;
  mapThumbnailSearch: string;
  setMapThumbnailSearch: Dispatch<SetStateAction<string>>;
  mapFile: File | null;
  setMapFile: Dispatch<SetStateAction<File | null>>;
  mapUploadError: string;
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
  mapThumbnailKey,
  setMapThumbnailKey,
  mapThumbnailCategory,
  setMapThumbnailCategory,
  mapThumbnailSearch,
  setMapThumbnailSearch,
  mapFile,
  setMapFile,
  mapUploadError,
  setMapUploadError,
  uploadPending,
  onUpload,
}: MapUploadFormProps) {
  const selectedThumbnail =
    mapThumbnailOptions.find((item) => item.key === mapThumbnailKey) ||
    mapThumbnailOptions[0];
  const filteredThumbnailOptions = mapThumbnailOptions.filter((item) => {
    const q = mapThumbnailSearch.trim().toLowerCase();
    return (
      item.category === mapThumbnailCategory &&
      (!q ||
        item.label.toLowerCase().includes(q) ||
        item.search.toLowerCase().includes(q) ||
        item.key.includes(q))
    );
  });

  return (
    <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_280px]">
      <div className="grid gap-3">
        <LobbyInput value={mapName} onChange={(event) => setMapName(event.target.value)} maxLength={80} placeholder="Map name" disabled={isGuest} className="h-11 rounded-xl font-semibold" />
        <LobbyTextarea value={mapDescription} onChange={(event) => setMapDescription(event.target.value)} maxLength={500} placeholder="Description (optional)" disabled={isGuest} className="min-h-20 resize-none rounded-xl" />
        <div className="grid gap-3 sm:grid-cols-2">
          <LobbySelect value={mapDifficulty} onChange={(event) => setMapDifficulty(event.target.value as MapDifficulty)} disabled={isGuest} className="h-11 rounded-xl font-bold">
            <option value="easy">Easy</option>
            <option value="normal">Normal</option>
            <option value="hard">Hard</option>
          </LobbySelect>
          <LobbyInput value={mapThumbnailSearch} onChange={(event) => setMapThumbnailSearch(event.target.value)} placeholder="Search thumbnails" disabled={isGuest} className="h-11 rounded-xl font-semibold" />
        </div>
        <LobbyPanel variant="subtle" className="rounded-xl p-3">
          <div className="mb-3 flex flex-wrap gap-2">
            {(["generic", "continents", "countries"] as const).map((category) => (
              <button key={category} type="button" onClick={() => setMapThumbnailCategory(category)} className={`rounded-lg px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] ${mapThumbnailCategory === category ? "bg-white text-[#10201a]" : "bg-white/[0.06] text-[#a9bfd4] hover:bg-white/[0.1]"}`}>
                {category}
              </button>
            ))}
          </div>
          <div className="grid max-h-[320px] gap-2 overflow-y-auto pr-1 sm:grid-cols-2">
            {filteredThumbnailOptions.map((option) => (
              <button key={option.key} type="button" onClick={() => setMapThumbnailKey(option.key)} className={`overflow-hidden rounded-xl border text-left transition ${mapThumbnailKey === option.key ? "border-[#77f0be] bg-[#77f0be]/10" : "border-white/10 bg-white/[0.04] hover:bg-white/[0.08]"}`}>
                <img src={mapThumbnailURL(option.key)} alt="" className="aspect-[16/9] w-full object-cover" />
                <div className="p-2 text-[11px] font-black text-white">{option.label}</div>
              </button>
            ))}
          </div>
        </LobbyPanel>
        <label className="flex min-h-20 cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed border-white/20 bg-black/20 px-4 text-center text-sm font-semibold text-[#a9bfd4] hover:border-[#2ad18f]/50">
          <Upload className="mb-2 text-[#2ad18f]" size={22} />
          {mapFile ? mapFile.name : "Choose JSON file"}
          <input type="file" accept=".json,application/json" className="hidden" disabled={isGuest} onChange={(event) => { setMapFile(event.target.files?.[0] || null); setMapUploadError(""); }} />
        </label>
      </div>
      <div className="grid content-start gap-3">
        <img src={mapThumbnailURL(mapThumbnailKey)} alt="" className="aspect-[16/9] w-full rounded-xl object-cover" />
        <p className="text-xs font-bold text-[#a9bfd4]">Selected: <span className="text-white">{selectedThumbnail.label}</span></p>
        {mapUploadError ? <p className="text-xs font-semibold text-red-300">{mapUploadError}</p> : null}
        <LobbyActionButton type="button" disabled={isGuest || !mapName.trim() || !mapFile || uploadPending} onClick={onUpload} className="h-11 rounded-xl">
          {uploadPending ? <Loader2 className="mr-2 animate-spin" size={17} /> : <Upload className="mr-2" size={17} />}
          Upload
        </LobbyActionButton>
        <p className="text-[11px] leading-5 text-[#6f8998]">Limits: 10 maps, 100,000 locations per map, 250,000 active locations per account, 3 uploads/hour.</p>
      </div>
    </div>
  );
}
