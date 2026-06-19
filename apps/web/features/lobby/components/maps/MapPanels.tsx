import Link from "next/link";
import type React from "react";
import { ArrowLeft, ChartNoAxesColumnIncreasing, ChevronDown, Heart, Loader2, Map as MapIcon, MessageCircle, MoreVertical, Play, Search, ShieldCheck, Star, Trophy, Trash2, Upload, X } from "lucide-react";
import AppModalShell from "../../../../components/ui/AppModalShell";
import AvatarBadge from "../../../../components/ui/AvatarBadge";
import { cn } from "../../../../lib/cn";
import { toPublicEntityId } from "../../../../lib/entity-id";
import type { MatchConfig } from "../../../matchmaking/lib/queue-client";
import type { CustomMap, GameplayMapRole, MapDetails, MapScope, MapSort } from "../../../maps/lib/maps-client";
import {
  commentAvatarFallback,
  commentDeletedLabel,
  formatCommentAge,
} from "../../lib/lobby-ui";
import {
  LobbyActionButton,
  LobbyInput,
  LobbyMutedBox,
  LobbyPanel,
  LobbySectionHeader,
} from "../lobby-primitives";

type MapScopeLabel = { scope: MapScope; label: string };

type MapSearchProps = {
  id: string;
  value: string;
  onChange: (value: string) => void;
};

function MapSearchControl({ id, value, onChange }: MapSearchProps) {
  return (
    <div className="relative w-full sm:w-[260px]">
      <label htmlFor={id} className="sr-only">
        Search maps
      </label>
      <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[#77f0be]" size={16} />
      <LobbyInput
        id={id}
        type="search"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="Search maps"
        className="h-11 w-full rounded-xl py-2 pl-9 pr-10 font-semibold"
      />
      {value ? (
        <button
          type="button"
          aria-label="Clear map search"
          onClick={() => onChange("")}
          className="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-[#a9bfd4] transition hover:bg-white/[0.08] hover:text-white"
        >
          <X size={15} />
        </button>
      ) : null}
    </div>
  );
}

function MapScopeNav({
  labels,
  value,
  onChange,
}: {
  labels: MapScopeLabel[];
  value: MapScope;
  onChange: (scope: MapScope) => void;
}) {
  return (
    <div className="grid gap-2">
      {labels.map((item) => (
        <button
          key={item.scope}
          type="button"
          onClick={() => onChange(item.scope)}
          className={`rounded-[14px] px-4 py-3 text-left text-sm font-extrabold transition ${
            value === item.scope ? "bg-accentPrimary text-white" : "bg-white/[0.05] text-[#a9bfd4] hover:bg-white/[0.09]"
          }`}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

function MapSortControl({
  value,
  onChange,
}: {
  value: MapSort;
  onChange: (sort: MapSort) => void;
}) {
  return (
    <div className="flex rounded-[14px] border border-white/10 bg-black/20 p-1">
      {(["trending", "popular", "new"] as MapSort[]).map((sort) => (
        <button
          key={sort}
          type="button"
          onClick={() => onChange(sort)}
          className={`rounded-[10px] px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] ${
            value === sort ? "bg-white text-[#10201a]" : "text-[#a9bfd4] hover:bg-white/[0.08]"
          }`}
        >
          {sort === "popular" ? "Most Popular" : sort}
        </button>
      ))}
    </div>
  );
}

function formatMapMetric(value: number) {
  if (value < 1000) return value.toLocaleString();
  return `${Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: value < 10000 ? 1 : 0 }).format(value)}+`;
}

const difficultyTone = { easy: "text-[#4ade80]", normal: "text-[#facc15]", hard: "text-[#fb6a4a]" } satisfies Record<CustomMap["difficulty"], string>;

function MapCard({
  item,
  selected,
  mode,
  thumbnailURL,
  onSelect,
}: {
  item: CustomMap;
  selected?: boolean;
  mode: "link" | "select";
  thumbnailURL: (item: Pick<CustomMap, "thumbnailVariant" | "thumbnailKey">) => string;
  onSelect?: (item: CustomMap) => void;
}) {
  const content = (
    <div
      className={cn(
        "group relative aspect-[16/9] overflow-hidden rounded-2xl bg-[#062722] text-left",
        "transition duration-200 hover:-translate-y-0.5",
        selected && "ring-2 ring-accentPrimary",
      )}
    >
      <img
        src={thumbnailURL(item)}
        alt=""
        className="absolute inset-0 h-full w-full object-cover opacity-85 transition duration-500 group-hover:scale-[1.035]"
      />
      <div className="absolute inset-0 bg-gradient-to-b from-[#053f3d]/10 via-[#062722]/25 to-[#031817]/88" />
      <div className="absolute inset-0 bg-gradient-to-r from-black/20 via-transparent to-black/25" />
      <div className="absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t from-black/70 via-black/28 to-transparent" />

      <div className="absolute right-3 top-3 flex flex-wrap justify-end gap-2">
        {item.system || item.official ? (
          <span className="rounded-full bg-white/90 px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-[#10201a]">
            Official
          </span>
        ) : null}
        {item.rankedMoving || item.rankedNmpz ? (
          <span className="rounded-full bg-[#ffd166] px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-[#1b1400]">
            Ranked
          </span>
        ) : null}
        {selected ? (
          <span className="rounded-full bg-accentPrimary px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white shadow-[0_8px_18px_rgba(42,209,143,0.28)]">
            Selected
          </span>
        ) : null}
      </div>

      <div className="absolute inset-x-0 bottom-0 grid grid-cols-[minmax(0,1fr)_auto] items-end gap-3 p-4">
        <div className="min-w-0">
          <h3 className="truncate text-[18px] font-black leading-tight text-white sm:text-[20px]">{item.displayName}</h3>
          <p className="mt-1.5 truncate text-xs font-bold text-[#c7dde1]">{item.authorName || "GeoDuels"}</p>
        </div>
        <div className="grid gap-1.5 pb-0.5 text-[12px] font-extrabold text-[#c7dde1]">
          <span className="inline-flex items-center justify-end gap-1.5" title="Locations">
            <MapIcon className="text-[#c7dde1]" size={15} />
            {formatMapMetric(item.locationCount)}
          </span>
          <span className="inline-flex items-center justify-end gap-1.5" title="Plays">
            <Play className="text-[#c7dde1]" size={15} />
            {formatMapMetric(item.playCount)}
          </span>
          <span className="inline-flex items-center justify-end gap-1.5 uppercase" title="Difficulty">
            <ChartNoAxesColumnIncreasing className={difficultyTone[item.difficulty]} size={15} />
            {item.difficulty}
          </span>
        </div>
      </div>
    </div>
  );

  if (mode === "select") {
    return (
      <button type="button" onClick={() => onSelect?.(item)} className="block w-full text-left">
        {content}
      </button>
    );
  }
  return (
    <Link
      href={`/maps/${encodeURIComponent(toPublicEntityId(item.id))}`}
      className="block w-full text-left"
    >
      {content}
    </Link>
  );
}

export function MapsPanel({
  canUploadCustomMaps,
  hasMapSearch,
  mapScope,
  mapScopeLabels,
  mapSearchInput,
  mapSort,
  mapsLoading,
  partyActive,
  readyMaps,
  setMapScope,
  setMapSearchInput,
  setMapSort,
  selectMapForParty,
  thumbnailURL,
}: {
  canUploadCustomMaps: boolean;
  hasMapSearch: boolean;
  mapScope: MapScope;
  mapScopeLabels: MapScopeLabel[];
  mapSearchInput: string;
  mapSort: MapSort;
  mapsLoading: boolean;
  partyActive: boolean;
  readyMaps: CustomMap[];
  setMapScope: (scope: MapScope) => void;
  setMapSearchInput: (value: string) => void;
  setMapSort: (sort: MapSort) => void;
  selectMapForParty: (item: CustomMap) => void;
  thumbnailURL: (item: Pick<CustomMap, "thumbnailVariant" | "thumbnailKey">) => string;
}) {
  return (
    <LobbyPanel className="overflow-hidden rounded-3xl">
      <div className="grid min-h-[640px] lg:grid-cols-[220px_minmax(0,1fr)]">
        <aside className="border-b border-white/10 bg-black/20 p-4 lg:border-b-0 lg:border-r">
          <div className="mb-4 flex items-center gap-2 text-white">
            <MapIcon className="text-[#77f0be]" size={22} />
            <span className="text-lg font-black">Maps</span>
          </div>
          <MapScopeNav labels={mapScopeLabels} value={mapScope} onChange={setMapScope} />
        </aside>
        <section className="p-5 sm:p-7">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <LobbySectionHeader
              eyebrow={partyActive ? "Party Map Select" : "Map Browser"}
              title={mapScopeLabels.find((item) => item.scope === mapScope)?.label}
            />
            <div className="flex w-full flex-col gap-3 sm:w-auto sm:items-end">
              <MapSearchControl id="map-browser-search" value={mapSearchInput} onChange={setMapSearchInput} />
              {mapScope === "community" ? <MapSortControl value={mapSort} onChange={setMapSort} /> : null}
            </div>
          </div>

          {mapScope === "mine" && !canUploadCustomMaps ? (
            <LobbyMutedBox className="mt-6">Sign in with a permanent account to create custom maps.</LobbyMutedBox>
          ) : mapsLoading ? (
            <div className="mt-8 flex items-center gap-3 text-sm text-[#a9bfd4]">
              <Loader2 className="animate-spin" size={18} /> Loading maps...
            </div>
          ) : readyMaps.length === 0 ? (
            <LobbyMutedBox className="mt-8 border-dashed p-8 text-center">
              {hasMapSearch ? "No maps match your search." : "No maps in this section yet."}
            </LobbyMutedBox>
          ) : (
            <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {readyMaps.map((item) => (
                <div key={item.id} className="overflow-hidden rounded-2xl">
                  <MapCard
                    item={item}
                    mode={partyActive ? "select" : "link"}
                    thumbnailURL={thumbnailURL}
                    onSelect={selectMapForParty}
                  />
                </div>
              ))}
            </div>
          )}

          {mapScope === "mine" && canUploadCustomMaps ? (
            <LobbyPanel variant="subtle" className="mt-7 p-5">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <LobbySectionHeader
                  eyebrow="Upload Map"
                  title={<span className="text-[18px]">Create a custom map</span>}
                  description="Create a custom map from a JSON file."
                />
                <Link href="/maps/upload" className="inline-flex min-h-[42px] items-center justify-center rounded-[12px] bg-accentPrimary px-4 text-sm font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-accentPrimaryDeep">
                  <Upload className="mr-2" size={17} />
                  Upload
                </Link>
              </div>
            </LobbyPanel>
          ) : null}
        </section>
      </div>
    </LobbyPanel>
  );
}

export function MapUploadPanel({
  canUploadCustomMaps,
  mapUploadForm,
}: {
  canUploadCustomMaps: boolean;
  mapUploadForm: React.ReactNode;
}) {
  return (
    <LobbyPanel className="p-4 sm:p-6">
      <div className="space-y-5">
        <BackToMapsLink />
        <LobbyPanel variant="subtle" className="p-4 sm:p-5">
          <div className="mb-5">
            <LobbySectionHeader
              eyebrow="Upload Map"
              title="Create a Custom Map"
              description="Choose a JSON file, thumbnail, difficulty, and public details for your GeoDuels map."
            />
          </div>
          {canUploadCustomMaps ? mapUploadForm : <LobbyMutedBox>Sign in with a permanent account to create custom maps.</LobbyMutedBox>}
        </LobbyPanel>
      </div>
    </LobbyPanel>
  );
}

function BackToMapsLink() {
  return (
    <Link href="/maps" className="inline-flex min-h-[38px] items-center gap-2 rounded-full border border-white/10 bg-white/[0.08] px-4 text-[12px] font-extrabold uppercase tracking-[0.08em] text-[#d6e4ed] transition hover:bg-white/[0.12] hover:text-white">
      <ArrowLeft size={16} />
      Back
    </Link>
  );
}

function AdminMapOperations({
  map,
  onDeleteMap,
  onSetOfficial,
  onSetRole,
}: {
  map: CustomMap;
  onDeleteMap: (map: CustomMap) => void;
  onSetOfficial: (mapId: string, official: boolean) => void;
  onSetRole: (mapId: string, role: GameplayMapRole) => void;
}) {
  const ready = map.status === "ready" && !!map.activeRevisionId;
  const roleButtons: Array<{ role: GameplayMapRole; label: string; active?: boolean; icon: React.ReactNode }> = [
    { role: "ranked_moving", label: "Ranked Moving", active: map.rankedMoving, icon: <Trophy size={16} /> },
    { role: "ranked_nmpz", label: "Ranked NMPZ", active: map.rankedNmpz, icon: <Trophy size={16} /> },
    { role: "singleplayer_moving", label: "Default Moving", active: map.defaultMoving, icon: <Play size={16} fill="currentColor" /> },
    { role: "singleplayer_nmpz", label: "Default NMPZ", active: map.defaultNmpz, icon: <Play size={16} fill="currentColor" /> },
  ];
  return (
    <LobbyPanel variant="subtle" className="p-4 sm:p-5">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h3 className="flex items-center gap-2 text-[18px] font-extrabold tracking-tight text-white">
            <ShieldCheck className="text-[#77f0be]" size={19} />
            Admin Map Operations
          </h3>
          <p className="mt-1 text-sm font-medium text-[#a9bfd4]">
            Promote this ready map or assign it to ranked and default queues.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={!ready}
            onClick={() => onSetOfficial(map.id, !map.official)}
            className={`inline-flex min-h-[40px] items-center rounded-[12px] border px-3 text-xs font-black uppercase tracking-[0.08em] transition disabled:cursor-not-allowed disabled:opacity-50 ${
              map.official
                ? "border-white/10 bg-white text-[#10201a] hover:bg-white/90"
                : "border-[#77f0be]/20 bg-[#77f0be]/10 text-white hover:bg-[#77f0be]/15"
            }`}
          >
            <ShieldCheck className="mr-1.5" size={15} />
            {map.official ? "Remove Official" : "Mark Official"}
          </button>
          {roleButtons.map((item) => (
            <button
              key={item.role}
              type="button"
              disabled={!ready || !!item.active}
              onClick={() => onSetRole(map.id, item.role)}
              className={`inline-flex min-h-[40px] items-center rounded-[12px] border px-3 text-xs font-black uppercase tracking-[0.08em] transition disabled:cursor-not-allowed disabled:opacity-55 ${
                item.active
                  ? "border-[#ffd166]/40 bg-[#ffd166] text-[#1b1400]"
                  : "border-white/10 bg-white/[0.06] text-white hover:bg-white/[0.1]"
              }`}
            >
              <span className="mr-1.5">{item.icon}</span>
              {item.active ? `${item.label} Active` : item.label}
            </button>
          ))}
          {map.ownerUserId ? (
            <button
              type="button"
              onClick={() => onDeleteMap(map)}
              className="inline-flex min-h-[40px] items-center rounded-[12px] border border-red-400/20 bg-red-400/[0.08] px-3 text-xs font-black uppercase tracking-[0.08em] text-red-200 transition hover:bg-red-400/15"
            >
              <Trash2 className="mr-1.5" size={15} />
              Delete Map
            </button>
          ) : null}
        </div>
      </div>
      {!ready ? <p className="mt-3 text-xs font-semibold text-amber-200">Map must be ready with an active revision before it can be promoted.</p> : null}
    </LobbyPanel>
  );
}

export function MapDetailsPanel({
  accessToken,
  canInteractWithMaps,
  canUploadCustomMaps,
  commentBody,
  commentComposerFocused,
  createCommentPending,
  displayName,
  expandedCommentIds,
  favoriteMap,
  isAdmin,
  isModerator,
  mapPickerFlow,
  onCancelComment,
  onDeleteComment,
  onDeleteMap,
  onPostComment,
  onPostReply,
  onPublishMap,
  onRevisionFile,
  onSetMapOfficial,
  onSetMapRole,
  onSetCommentBody,
  onSetCommentComposerFocused,
  onSetOpenCommentMenuId,
  onSetReplyBody,
  onSetReplyToCommentId,
  onToggleCommentLike,
  onToggleCommentReplies,
  openCommentMenuId,
  playMapSingleplayer,
  replyBody,
  replyToCommentId,
  selectMapForParty,
  selectedMapDetails,
  selectedMapLoading,
  singleplayerDisabled,
  thumbnailURL,
  userAvatar,
  userAvatarFallback,
  userEmail,
  userId,
}: {
  accessToken: string;
  canInteractWithMaps: boolean;
  canUploadCustomMaps: boolean;
  commentBody: string;
  commentComposerFocused: boolean;
  createCommentPending: boolean;
  displayName: string;
  expandedCommentIds: Record<string, boolean>;
  favoriteMap: (input: { mapId: string; favorite: boolean }) => void;
  isAdmin: boolean;
  isModerator: boolean;
  mapPickerFlow: boolean;
  onCancelComment: () => void;
  onDeleteComment: (commentId: string) => void;
  onDeleteMap: (map: CustomMap) => void;
  onPostComment: () => void;
  onPostReply: (commentId: string) => void;
  onPublishMap: (mapId: string) => void;
  onRevisionFile: (mapId: string, file: File) => void;
  onSetMapOfficial: (mapId: string, official: boolean) => void;
  onSetMapRole: (mapId: string, role: GameplayMapRole) => void;
  onSetCommentBody: (body: string) => void;
  onSetCommentComposerFocused: (focused: boolean) => void;
  onSetOpenCommentMenuId: (commentId: string) => void;
  onSetReplyBody: (body: string) => void;
  onSetReplyToCommentId: (commentId: string) => void;
  onToggleCommentLike: (commentId: string, liked: boolean) => void;
  onToggleCommentReplies: (commentId: string) => void;
  openCommentMenuId: string;
  playMapSingleplayer: (item: CustomMap) => void;
  replyBody: string;
  replyToCommentId: string;
  selectMapForParty: (item: CustomMap) => void;
  selectedMapDetails: MapDetails | undefined;
  selectedMapLoading: boolean;
  singleplayerDisabled: boolean;
  thumbnailURL: (item: Pick<CustomMap, "thumbnailVariant" | "thumbnailKey">) => string;
  userAvatar?: string;
  userAvatarFallback: string;
  userEmail: string;
  userId: string;
}) {
  if (selectedMapLoading || !selectedMapDetails) {
    return (
      <LobbyPanel className="p-4 sm:p-6">
        <div className="flex items-center gap-3 text-sm text-[#a9bfd4]">
          <Loader2 className="animate-spin" size={18} /> Loading map details...
        </div>
      </LobbyPanel>
    );
  }

  const map = selectedMapDetails.map;

  return (
    <LobbyPanel className="p-4 sm:p-6">
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <BackToMapsLink />
          {canInteractWithMaps ? (
            <LobbyActionButton
              type="button"
              variant="secondary"
              size="icon"
              onClick={() => favoriteMap({ mapId: map.id, favorite: !map.favorited })}
              aria-label={map.favorited ? "Remove saved map" : "Save map"}
              title={map.favorited ? "Saved" : "Save"}
              className={cn(
                "h-11 min-h-11 w-11 rounded-full",
                map.favorited && "border-accentPrimary/40 bg-accentPrimary/15 text-accentPrimary hover:bg-accentPrimary/20",
              )}
            >
              <Star size={18} fill={map.favorited ? "currentColor" : "none"} />
            </LobbyActionButton>
          ) : null}
        </div>
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]">
          <section
            className="relative min-h-[280px] overflow-hidden rounded-[18px] bg-cover bg-center"
            style={{ backgroundImage: `url(${thumbnailURL(map)})` }}
          >
            <div className="absolute inset-0 bg-black/40" />
            <div className="relative flex min-h-[280px] flex-col justify-end p-5 sm:p-6">
              <div className="max-w-[720px]">
                <h2 className="text-[28px] font-extrabold leading-tight tracking-tight text-white sm:text-[36px]">{map.displayName}</h2>
                <div className="mt-3 flex flex-wrap items-center gap-3 text-sm font-bold text-[#d7e5ee]">
                  <span>By {map.authorName || "GeoDuels"}</span>
                  <span className="rounded-full bg-black/25 px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.12em] text-[#ffd166]">
                    {map.difficulty}
                  </span>
                </div>
              </div>
            </div>
          </section>

          <LobbyPanel variant="subtle" className="p-4 sm:p-5">
            <div className="grid gap-3">
              {[
                { label: "plays", value: map.playCount, icon: <Play size={20} fill="currentColor" /> },
                { label: "locations", value: map.locationCount, icon: <MapIcon size={20} /> },
                { label: "favorites", value: map.favoriteCount, icon: <Heart size={20} /> },
              ].map((metric) => (
                <div key={metric.label} className="grid grid-cols-[64px_minmax(0,1fr)] overflow-hidden rounded-[12px] border border-white/10 bg-black/25">
                  <div className="flex items-center justify-center bg-white/[0.06] text-[#77f0be]">{metric.icon}</div>
                  <div className="px-4 py-3">
                    <div className="text-[21px] font-extrabold leading-none text-white">{metric.value.toLocaleString()}</div>
                    <div className="mt-1 text-[12px] font-bold lowercase text-[#a9bfd4]">{metric.label}</div>
                  </div>
                </div>
              ))}
            </div>
            <p className="mt-4 text-[14px] font-medium leading-6 text-[#a9bfd4]">
              {map.description || "No description has been added for this map yet."}
            </p>
          </LobbyPanel>
        </div>

        <LobbyPanel variant="subtle" className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap gap-2">
            <span className="rounded-[12px] bg-white/[0.06] px-3 py-2 text-xs font-black uppercase tracking-[0.08em] text-[#a9bfd4]">Moving</span>
            <span className="rounded-[12px] bg-white/[0.06] px-3 py-2 text-xs font-black uppercase tracking-[0.08em] text-[#a9bfd4]">Infinite Clock</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {mapPickerFlow ? (
              <LobbyActionButton type="button" onClick={() => selectMapForParty(map)} size="lg" className="min-h-[46px] rounded-xl px-6">
                <MapIcon className="mr-2" size={18} />
                Use This Map
              </LobbyActionButton>
            ) : (
              <LobbyActionButton type="button" onClick={() => playMapSingleplayer(map)} disabled={singleplayerDisabled} size="lg" className="min-h-[46px] rounded-xl px-6">
                <Play className="mr-2" size={18} fill="currentColor" />
                Play
              </LobbyActionButton>
            )}
          </div>
        </LobbyPanel>

        {map.ownerUserId === userId && canUploadCustomMaps ? (
          <LobbyPanel variant="subtle" className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 className="text-[16px] font-extrabold tracking-tight text-white">Map Options</h3>
            </div>
            <div className="flex flex-wrap gap-2">
              {!map.publishedAt ? (
                <button type="button" onClick={() => onPublishMap(map.id)} className="min-h-[42px] rounded-[14px] border border-[#77f0be]/20 bg-[#77f0be]/10 px-4 text-xs font-black uppercase tracking-[0.08em] text-white">
                  Publish
                </button>
              ) : null}
              <label className="inline-flex min-h-[42px] cursor-pointer items-center rounded-[14px] border border-white/10 bg-white/[0.06] px-4 text-xs font-black uppercase tracking-[0.08em] text-white hover:bg-white/[0.1]">
                <Upload className="mr-1.5" size={14} /> New Version
                <input type="file" accept=".json,application/json" className="hidden" onChange={(event) => { const file = event.target.files?.[0]; if (file) onRevisionFile(map.id, file); event.currentTarget.value = ""; }} />
              </label>
              <button type="button" onClick={() => onDeleteMap(map)} className="min-h-[42px] rounded-[14px] border border-red-400/15 bg-red-400/[0.06] px-4 text-xs font-black uppercase tracking-[0.08em] text-red-200 hover:bg-red-400/10">
                Delete
              </button>
            </div>
          </LobbyPanel>
        ) : null}

        {isAdmin ? (
          <AdminMapOperations
            map={map}
            onDeleteMap={onDeleteMap}
            onSetOfficial={onSetMapOfficial}
            onSetRole={onSetMapRole}
          />
        ) : null}

        <MapComments
          accessToken={accessToken}
          canInteractWithMaps={canInteractWithMaps}
          commentBody={commentBody}
          commentComposerFocused={commentComposerFocused}
          comments={selectedMapDetails.comments}
          createCommentPending={createCommentPending}
          displayName={displayName}
          expandedCommentIds={expandedCommentIds}
          isAdmin={isAdmin}
          isModerator={isModerator}
          onCancelComment={onCancelComment}
          onDeleteComment={onDeleteComment}
          onPostComment={onPostComment}
          onPostReply={onPostReply}
          onSetCommentBody={onSetCommentBody}
          onSetCommentComposerFocused={onSetCommentComposerFocused}
          onSetOpenCommentMenuId={onSetOpenCommentMenuId}
          onSetReplyBody={onSetReplyBody}
          onSetReplyToCommentId={onSetReplyToCommentId}
          onToggleCommentLike={onToggleCommentLike}
          onToggleCommentReplies={onToggleCommentReplies}
          openCommentMenuId={openCommentMenuId}
          replyBody={replyBody}
          replyToCommentId={replyToCommentId}
          userAvatar={userAvatar}
          userAvatarFallback={userAvatarFallback}
          userEmail={userEmail}
        />
      </div>
    </LobbyPanel>
  );
}

function MapComments(props: {
  accessToken: string;
  canInteractWithMaps: boolean;
  commentBody: string;
  commentComposerFocused: boolean;
  comments: MapDetails["comments"];
  createCommentPending: boolean;
  displayName: string;
  expandedCommentIds: Record<string, boolean>;
  isAdmin: boolean;
  isModerator: boolean;
  onCancelComment: () => void;
  onDeleteComment: (commentId: string) => void;
  onPostComment: () => void;
  onPostReply: (commentId: string) => void;
  onSetCommentBody: (body: string) => void;
  onSetCommentComposerFocused: (focused: boolean) => void;
  onSetOpenCommentMenuId: (commentId: string) => void;
  onSetReplyBody: (body: string) => void;
  onSetReplyToCommentId: (commentId: string) => void;
  onToggleCommentLike: (commentId: string, liked: boolean) => void;
  onToggleCommentReplies: (commentId: string) => void;
  openCommentMenuId: string;
  replyBody: string;
  replyToCommentId: string;
  userAvatar?: string;
  userAvatarFallback: string;
  userEmail: string;
}) {
  return (
    <LobbyPanel variant="subtle" className="p-4">
      <h4 className="flex items-center gap-2 text-[18px] font-extrabold tracking-tight text-white">
        <MessageCircle size={18} /> Comments
      </h4>
      {props.canInteractWithMaps ? (
        <div className="mt-5 flex gap-3">
          <AvatarBadge avatarUrl={props.userAvatar} fallback={props.userAvatarFallback} alt={props.displayName || props.userEmail || "You"} size="sm" className="mt-1 h-10 w-10 shrink-0 border-white/15 bg-[#162130]" />
          <div className="min-w-0 flex-1">
            <textarea
              value={props.commentBody}
              onFocus={() => props.onSetCommentComposerFocused(true)}
              onChange={(event) => props.onSetCommentBody(event.target.value)}
              maxLength={1000}
              placeholder="Add a comment"
              rows={props.commentComposerFocused || props.commentBody ? 2 : 1}
              className="min-h-[36px] w-full resize-none border-0 border-b border-white/25 bg-transparent px-0 py-1.5 text-[15px] font-medium text-white outline-none placeholder:text-[#8da6b5] focus:border-[#2ad18f]"
            />
            {props.commentComposerFocused || props.commentBody ? (
              <div className="mt-3 flex justify-end gap-2">
                <button type="button" onClick={props.onCancelComment} className="rounded-full px-4 py-2 text-xs font-extrabold uppercase tracking-[0.08em] text-[#a9bfd4] hover:bg-white/[0.08]">Cancel</button>
                <LobbyActionButton type="button" disabled={!props.commentBody.trim() || props.createCommentPending} onClick={props.onPostComment} size="sm" className="rounded-full px-4 py-2">Comment</LobbyActionButton>
              </div>
            ) : null}
          </div>
        </div>
      ) : (
        <p className="mt-3 text-xs text-[#8da6b5]">{props.accessToken ? "Upgrade your guest profile to comment." : "Sign in to comment."}</p>
      )}
      <div className="mt-7 grid gap-6">
        {props.comments.map((comment) => (
          <CommentThread key={comment.id} comment={comment} depth="root" {...props} />
        ))}
      </div>
    </LobbyPanel>
  );
}

function CommentThread({ comment, depth, ...props }: Parameters<typeof MapComments>[0] & { comment: MapDetails["comments"][number]; depth: "root" | "reply" }) {
  const textSize = depth === "root" ? "text-[15px] leading-6" : "text-[14px] leading-5";
  const avatarSize = depth === "root" ? "h-10 w-10" : "h-8 w-8";
  return (
    <div className="flex gap-3">
      <AvatarBadge avatarUrl={comment.avatarUrl} fallback={commentAvatarFallback(comment.userDisplayName)} alt={comment.userDisplayName} size="sm" className={`${avatarSize} shrink-0 border-white/15 bg-[#162130] text-xs`} />
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-baseline gap-2">
              <span className="truncate text-[14px] font-extrabold text-white">{depth === "reply" ? `@${comment.userDisplayName}` : comment.userDisplayName}</span>
              <time dateTime={comment.createdAt} className="text-[13px] font-bold text-[#a9bfd4]/80">{formatCommentAge(comment.createdAt)}</time>
              {commentDeletedLabel(comment.status) ? <span className="text-[13px] font-black text-red-300">{commentDeletedLabel(comment.status)}</span> : null}
            </div>
            <p className={`mt-1 font-medium text-[#eef6fb] ${textSize}`}>{comment.body}</p>
          </div>
          {comment.status === "visible" && (comment.canDelete || props.isAdmin || props.isModerator) && props.accessToken ? (
            depth === "root" ? (
              <div className="relative shrink-0">
                <button type="button" onClick={() => props.onSetOpenCommentMenuId(props.openCommentMenuId === comment.id ? "" : comment.id)} className="flex h-8 w-8 items-center justify-center rounded-full text-[#a9bfd4] hover:bg-white/[0.08] hover:text-white" aria-label="Comment actions">
                  <MoreVertical size={17} />
                </button>
                {props.openCommentMenuId === comment.id ? (
                  <div className="absolute right-0 top-9 z-10 w-32 overflow-hidden rounded-[12px] border border-white/10 bg-[#101a20] py-1 shadow-[0_12px_30px_rgba(0,0,0,0.35)]">
                    <button type="button" onClick={() => { props.onSetOpenCommentMenuId(""); props.onDeleteComment(comment.id); }} className="flex w-full items-center gap-2 px-3 py-2 text-left text-xs font-bold text-red-200 hover:bg-red-400/10">
                      <Trash2 size={14} />
                      Delete
                    </button>
                  </div>
                ) : null}
              </div>
            ) : (
              <button type="button" onClick={() => props.onDeleteComment(comment.id)} className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-[#a9bfd4] hover:bg-red-400/10 hover:text-red-200" aria-label="Delete reply">
                <Trash2 size={14} />
              </button>
            )
          ) : null}
        </div>
        <div className="mt-3 flex items-center gap-4">
          {props.canInteractWithMaps && comment.status === "visible" ? (
            <button
              type="button"
              onClick={() => props.onToggleCommentLike(comment.id, !comment.liked)}
              aria-label={comment.liked ? "Unlike comment" : "Like comment"}
              aria-pressed={comment.liked}
              className={`inline-flex items-center gap-2 rounded-full text-[13px] font-bold transition ${comment.liked ? "text-[#77f0be]" : "text-[#d6e4ed] hover:text-white"}`}
            >
              <Heart size={depth === "root" ? 18 : 16} fill={comment.liked ? "currentColor" : "none"} />
              {comment.likeCount.toLocaleString()}
            </button>
          ) : null}
          {depth === "root" && props.canInteractWithMaps && comment.status === "visible" ? (
            <button type="button" onClick={() => { props.onSetReplyToCommentId(comment.id); props.onSetReplyBody(""); }} className="rounded-full px-2 py-1 text-[13px] font-extrabold text-white hover:bg-white/[0.08]">Reply</button>
          ) : null}
        </div>
        {depth === "root" && props.replyToCommentId === comment.id ? (
          <div className="mt-4 flex gap-3">
            <AvatarBadge avatarUrl={props.userAvatar} fallback={props.userAvatarFallback} alt={props.displayName || props.userEmail || "You"} size="sm" className="mt-1 h-8 w-8 shrink-0 border-white/15 bg-[#162130]" />
            <div className="min-w-0 flex-1">
              <textarea value={props.replyBody} onChange={(event) => props.onSetReplyBody(event.target.value)} maxLength={1000} autoFocus rows={2} placeholder={`Reply to @${comment.userDisplayName}`} className="min-h-[44px] w-full resize-none border-0 border-b border-white/25 bg-transparent px-0 py-1.5 text-[14px] font-medium text-white outline-none placeholder:text-[#8da6b5] focus:border-[#2ad18f]" />
              <div className="mt-3 flex justify-end gap-2">
                <button type="button" onClick={() => { props.onSetReplyToCommentId(""); props.onSetReplyBody(""); }} className="rounded-full px-4 py-2 text-xs font-extrabold uppercase tracking-[0.08em] text-[#a9bfd4] hover:bg-white/[0.08]">Cancel</button>
                <LobbyActionButton type="button" disabled={!props.replyBody.trim()} onClick={() => props.onPostReply(comment.id)} size="sm" className="rounded-full px-4 py-2">Reply</LobbyActionButton>
              </div>
            </div>
          </div>
        ) : null}
        {depth === "root" && comment.replies?.length ? (
          <button type="button" onClick={() => props.onToggleCommentReplies(comment.id)} className="mt-3 inline-flex items-center gap-2 rounded-full px-2 py-1 text-[14px] font-extrabold text-[#77f0be] hover:bg-[#2ad18f]/10">
            <ChevronDown size={18} className={`transition ${props.expandedCommentIds[comment.id] ? "rotate-180" : ""}`} />
            {comment.replies.length} {comment.replies.length === 1 ? "reply" : "replies"}
          </button>
        ) : null}
        {depth === "root" && props.expandedCommentIds[comment.id] ? (
          <div className="mt-4 grid gap-4 border-l border-white/[0.12] pl-4">
            {comment.replies?.map((reply) => (
              <CommentThread key={reply.id} comment={reply} depth="reply" {...props} />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}

export function MapPickerModal({
  canUploadCustomMaps,
  hasMapSearch,
  partyConfig,
  mapScope,
  mapScopeLabels,
  mapSearchInput,
  mapSort,
  mapsLoading,
  onClose,
  readyMaps,
  selectMapForParty,
  setMapScope,
  setMapSearchInput,
  setMapSort,
  thumbnailURL,
}: {
  canUploadCustomMaps: boolean;
  hasMapSearch: boolean;
  partyConfig: MatchConfig;
  mapScope: MapScope;
  mapScopeLabels: MapScopeLabel[];
  mapSearchInput: string;
  mapSort: MapSort;
  mapsLoading: boolean;
  onClose: () => void;
  readyMaps: CustomMap[];
  selectMapForParty: (item: CustomMap) => void;
  setMapScope: (scope: MapScope) => void;
  setMapSearchInput: (value: string) => void;
  setMapSort: (sort: MapSort) => void;
  thumbnailURL: (item: Pick<CustomMap, "thumbnailVariant" | "thumbnailKey">) => string;
}) {
  return (
    <AppModalShell title="Select Map" onClose={onClose} placement="center" maxWidthClassName="max-w-[1040px]" panelClassName="p-4 sm:p-5" contentClassName="space-y-4">
      <div className="grid gap-4 lg:grid-cols-[190px_minmax(0,1fr)]">
        <aside className="grid gap-2 rounded-[18px] border border-white/10 bg-black/20 p-3 lg:content-start">
          <MapScopeNav labels={mapScopeLabels} value={mapScope} onChange={setMapScope} />
        </aside>
        <section className="min-w-0">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <LobbySectionHeader eyebrow="Party Map" title={mapScopeLabels.find((item) => item.scope === mapScope)?.label} />
            <div className="flex w-full flex-col gap-3 sm:w-auto sm:items-end">
              <MapSearchControl id="party-map-search" value={mapSearchInput} onChange={setMapSearchInput} />
              {mapScope === "community" ? <MapSortControl value={mapSort} onChange={setMapSort} /> : null}
            </div>
          </div>

          {mapScope === "mine" && !canUploadCustomMaps ? (
            <LobbyMutedBox className="mt-5">Sign in with a permanent account to use your custom maps.</LobbyMutedBox>
          ) : mapsLoading ? (
            <div className="mt-6 flex items-center gap-3 text-sm text-[#a9bfd4]"><Loader2 className="animate-spin" size={18} /> Loading maps...</div>
          ) : readyMaps.length === 0 ? (
            <LobbyMutedBox className="mt-6 border-dashed p-8 text-center">{hasMapSearch ? "No ready maps match your search." : "No ready maps in this section yet."}</LobbyMutedBox>
          ) : (
            <div className="mt-5 grid max-h-[56vh] gap-3 overflow-y-auto pr-1 sm:grid-cols-2 lg:grid-cols-3">
              {readyMaps.map((item) => (
                <div key={item.id} className="overflow-hidden rounded-2xl">
                  <MapCard item={item} mode="select" selected={item.id === partyConfig.mapId} thumbnailURL={thumbnailURL} onSelect={selectMapForParty} />
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </AppModalShell>
  );
}
