import type { MapScope } from "../../maps/lib/maps-client";

export type LobbyContentRoute =
  | "play"
  | "friends"
  | "maps"
  | "map-details"
  | "map-upload"
  | "top"
  | "party";

export const CLOCK_OPTIONS = [
  { value: "infinite", label: "Infinite" },
  { value: "30", label: "30s" },
  { value: "45", label: "45s" },
  { value: "60", label: "60s" },
  { value: "90", label: "90s" },
  { value: "120", label: "120s" },
] as const;

export const PRESSURE_OPTIONS = [
  { value: "none", label: "None" },
  { value: "15", label: "15s" },
] as const;

export const NAV_ITEMS: Array<{ label: string; route: LobbyContentRoute; href: string }> = [
  { label: "FRIENDS", route: "friends", href: "/friends" },
  { label: "PLAY", route: "play", href: "/" },
  { label: "MAPS", route: "maps", href: "/maps" },
  { label: "TOP", route: "top", href: "/top" },
];

export const lobbyRouteStorageKey = "geoduels.lobbyRoute";

export function lobbyTeamLabel(teamId?: string) {
  return teamId === "b" ? "Team Blue" : "Team Red";
}

export function lobbyTeamTextClass(teamId?: string) {
  return teamId === "b" ? "text-[#93c5fd]" : "text-[#fca5a5]";
}

export function lobbyTeamPillClass(teamId?: string, active = false) {
  if (teamId === "b") {
    return active
      ? "bg-[#2563eb] text-white"
      : "border border-[#60a5fa]/25 bg-[#2563eb]/15 text-[#bfdbfe] hover:bg-[#2563eb]/25";
  }
  return active
    ? "bg-[#dc2626] text-white"
    : "border border-[#f87171]/25 bg-[#dc2626]/15 text-[#fecaca] hover:bg-[#dc2626]/25";
}

export function isMapScope(value: unknown): value is MapScope {
  return value === "official" || value === "community" || value === "favorites" || value === "mine";
}

export function isLobbyNavRoute(value: string): value is LobbyContentRoute {
  return NAV_ITEMS.some((item) => item.route === value);
}

export function parseTime(value?: string) {
  if (!value) return null;
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? ms : null;
}

export function formatRelativeDuration(ms: number) {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) {
    return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
  }
  return `${seconds}s`;
}

export function formatApproximateTime(ms: number) {
  const totalMinutes = Math.max(1, Math.ceil(ms / 60000));
  if (totalMinutes >= 60) {
    const hours = Math.floor(totalMinutes / 60);
    const minutes = totalMinutes % 60;
    return minutes === 0
      ? `about ${hours} hour${hours === 1 ? "" : "s"}`
      : `about ${hours}h ${minutes}m`;
  }
  return `about ${totalMinutes} minute${totalMinutes === 1 ? "" : "s"}`;
}

export function formatQueueElapsed(ms: number) {
  const totalSeconds = ms > 0 ? Math.max(1, Math.ceil(ms / 1000)) : 0;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export function formatChangelogDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  }).format(date);
}

export function formatCommentAge(value: string) {
  const ms = Date.parse(value);
  if (!Number.isFinite(ms)) return "";
  const seconds = Math.max(1, Math.floor((Date.now() - ms) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

export function commentAvatarFallback(name: string) {
  return (name.trim() || "?").slice(0, 1).toUpperCase();
}

export function commentDeletedLabel(status: string) {
  return status === "visible" ? "" : "(deleted)";
}
