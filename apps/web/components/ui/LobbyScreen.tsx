import React, { useState, useEffect } from "react";
import {
  CheckCircle2,
  Github,
  HelpCircle,
  Heart,
  Play,
  Loader2,
  Pencil,
  Check,
  ChevronDown,
  ChevronUp,
  ArrowLeft,
  ArrowUpRight,
  Shield,
  Twitter,
  UserPlus,
  Copy,
  Crown,
  LogOut,
  UserMinus,
  Trash2,
  Youtube,
  Map as MapIcon,
  Upload,
  MessageCircle,
  Star,
  MoreVertical,
  Search,
  X,
} from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { motion, AnimatePresence } from "framer-motion";
import Link from "next/link";
import Router from "next/router";
import AppModalShell from "./AppModalShell";
import MarkdownContent from "./MarkdownContent";
import PlayerBadge, { type PlayerBadgeInfo } from "./PlayerBadge";
import AvatarBadge from "./AvatarBadge";
import PlayerNameWithBadge from "./PlayerNameWithBadge";
import { RatingTrophyIcon } from "./PlayerIdentity";
import type { LeaderboardSummary } from "../../features/auth/controllers/session-controller";
import type { LobbySnapshot as PartySnapshot, LobbyTeamId as PartyTeamId, PartyMode } from "../../features/lobby/lib/lobby-client";
import type { LobbyRuntimeStatus as PartyRuntimeStatus } from "../../features/lobby/controllers/lobby-controller";
import type { GameRuleset, MaintenanceStatus, MatchConfig } from "../../features/matchmaking/lib/queue-client";
import { getRuntimeConfig } from "../../lib/runtime-config";
import {
  createMap,
  validateMapFile,
  type CustomMap,
  type MapScope,
  type MapSort,
} from "../../features/maps/lib/maps-client";
import { useFavoriteMap, useMapComments, useMapDetails, useMapList, useMapManagement } from "../../features/maps/lib/map-hooks";
import { mapThumbnailOptions, mapThumbnailURL } from "../../features/maps/lib/map-thumbnails";

type PartyModal = "help" | "profile" | "invite" | "signin" | null;

const CLOCK_OPTIONS = [
  { value: "infinite", label: "Infinite" },
  { value: "30", label: "30s" },
  { value: "45", label: "45s" },
  { value: "60", label: "60s" },
  { value: "90", label: "90s" },
  { value: "120", label: "120s" },
] as const;

const PRESSURE_OPTIONS = [
  { value: "none", label: "None" },
  { value: "15", label: "15s" },
] as const;

function lobbyTeamLabel(teamId?: string) {
  return teamId === "b" ? "Team Blue" : "Team Red";
}

function lobbyTeamTextClass(teamId?: string) {
  return teamId === "b" ? "text-[#93c5fd]" : "text-[#fca5a5]";
}

function lobbyTeamPillClass(teamId?: string, active = false) {
  if (teamId === "b") {
    return active
      ? "bg-[#2563eb] text-white"
      : "border border-[#60a5fa]/25 bg-[#2563eb]/15 text-[#bfdbfe] hover:bg-[#2563eb]/25";
  }
  return active
    ? "bg-[#dc2626] text-white"
    : "border border-[#f87171]/25 bg-[#dc2626]/15 text-[#fecaca] hover:bg-[#dc2626]/25";
}

type PrivateLobbyView = {
  status: PartyRuntimeStatus;
  snapshot: PartySnapshot | null;
  inviteCode: string;
  isMember: boolean;
  isOwner: boolean;
  busy: boolean;
  error: string;
};

function isMapScope(value: unknown): value is MapScope {
  return value === "official" || value === "community" || value === "favorites" || value === "mine";
}

export type LobbyContentRoute = "play" | "friends" | "maps" | "map-details" | "map-upload" | "top" | "party";

type Props = {
  contentRoute?: LobbyContentRoute;
  mapId?: string;
  userId: string;
  accessToken?: string;
  userEmail: string;
  displayName: string;
  userAvatar?: string;
  isGuest: boolean;
  authMigrationRequired?: boolean;
  linkedProviders?: string[];
  badges?: PlayerBadgeInfo[];
  selectedBadge?: PlayerBadgeInfo | null;
  connected: boolean;
  mmr: number;
  gamesPlayed: number;
  winsPct: number;
  leaderboard: LeaderboardSummary | null;
  leaderboardLoading: boolean;
  status: string;
  queueStartedAt: number | null;
  joinQueue: (rulesets?: GameRuleset[]) => void;
  startSingleplayer: (config?: MatchConfig) => void | Promise<string>;
  cancelQueue: () => void;
  privateLobby?: PrivateLobbyView;
  createInviteLobby?: (mode?: PartyMode, config?: MatchConfig) => Promise<boolean>;
  joinInviteLobby?: (inviteCode?: string) => Promise<boolean>;
  leavePrivateLobby?: () => Promise<void>;
  kickLobbyMember?: (userId: string) => Promise<void>;
  transferLobbyOwner?: (userId: string) => Promise<void>;
  startPrivateLobby?: () => Promise<void>;
  updatePrivateLobbySettings?: (config: MatchConfig, mode?: PartyMode) => Promise<void>;
  switchPrivateLobbyTeam?: (teamId: PartyTeamId) => Promise<void>;
  queueError: string;
  onlinePlayers: number;
  maintenance: MaintenanceStatus | null;
  googleClientId: string;
  discordClientId?: string;
  appVersion: string;
  isAdmin: boolean;
  isModerator?: boolean;
  changelogEyebrow: string;
  changelogTitle: string;
  changelogMarkdown: string;
  changelogSlug: string;
  changelogUpdatedAt: string;
  devLogin: () => void;
  onGoogleSignIn: () => void;
  onDiscordSignIn?: () => void;
  onLinkAuthProvider?: (provider: "google" | "discord") => void;
  onUpgradeGuestWithProvider?: (provider: "google" | "discord") => void;
  onUnlinkAuthProvider?: (provider: "google" | "discord") => void;
  onBrowseLeaderboard: () => void;
  authLoading: boolean;
  authError: string;
  nicknameInput: string;
  nicknameError: string;
  nicknameSaving: boolean;
  onChangeNickname: (value: string) => void;
  onSaveNickname: () => Promise<boolean>;
  onSelectBadge?: (badgeId: string) => Promise<void>;
  onSupportDonation?: () => Promise<void>;
  onLogout: () => void;
  onDeleteAccount?: () => Promise<void>;
};

const defaultPrivateLobby: PrivateLobbyView = {
  status: "idle",
  snapshot: null,
  inviteCode: "",
  isMember: false,
  isOwner: false,
  busy: false,
  error: "",
};

const NAV_ITEMS: Array<{ label: string; route: LobbyContentRoute; href: string }> = [
  { label: "FRIENDS", route: "friends", href: "/friends" },
  { label: "PLAY", route: "play", href: "/" },
  { label: "MAPS", route: "maps", href: "/maps" },
  { label: "TOP", route: "top", href: "/top" },
];

const lobbyRouteStorageKey = "geoduels.lobbyRoute";

function isLobbyNavRoute(value: string): value is LobbyContentRoute {
  return NAV_ITEMS.some((item) => item.route === value);
}

const tabPanelMotion = {
  initial: {
    opacity: 0,
    y: 16,
    scale: 0.97,
  },
  animate: {
    opacity: 1,
    y: 0,
    scale: 1,
  },
  exit: {
    opacity: 0,
    y: 10,
    scale: 0.97,
  },
  transition: {
    duration: 0.22,
    ease: [0.16, 1, 0.3, 1] as const,
  },
};

function parseTime(value?: string) {
  if (!value) return null;
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? ms : null;
}

function formatRelativeDuration(ms: number) {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0)
    return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
  return `${seconds}s`;
}

function formatApproximateTime(ms: number) {
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

function formatQueueElapsed(ms: number) {
  const totalSeconds = ms > 0 ? Math.max(1, Math.ceil(ms / 1000)) : 0;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

function formatChangelogDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  }).format(date);
}

function formatCommentAge(value: string) {
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

function commentAvatarFallback(name: string) {
  return (name.trim() || "?").slice(0, 1).toUpperCase();
}

function commentDeletedLabel(status: string) {
  return status === "visible" ? "" : "(deleted)";
}

export default function LobbyScreen({
  contentRoute = "play",
  mapId = "",
  userId,
  accessToken = "",
  userEmail,
  displayName,
  userAvatar,
  isGuest,
  authMigrationRequired = false,
  linkedProviders = [],
  badges = [],
  selectedBadge = null,
  connected,
  mmr,
  gamesPlayed,
  winsPct,
  leaderboard,
  leaderboardLoading,
  status,
  queueStartedAt,
  joinQueue,
  startSingleplayer,
  cancelQueue,
  privateLobby = defaultPrivateLobby,
  createInviteLobby = async () => false,
  joinInviteLobby = async () => false,
  leavePrivateLobby = async () => {},
  kickLobbyMember = async () => { },
  transferLobbyOwner = async () => { },
  startPrivateLobby = async () => {},
  updatePrivateLobbySettings = async () => {},
  switchPrivateLobbyTeam = async () => {},
  queueError,
  googleClientId,
  discordClientId,
  devLogin,
  onGoogleSignIn,
  onDiscordSignIn = devLogin,
  onLinkAuthProvider = async () => { },
  onUpgradeGuestWithProvider = async () => { },
  onUnlinkAuthProvider = async () => { },
  onBrowseLeaderboard,
  authLoading,
  authError,
  nicknameInput,
  nicknameError,
  nicknameSaving,
  onChangeNickname,
  onSaveNickname,
  onSelectBadge = async () => { },
  onSupportDonation = async () => { },
  maintenance,
  onlinePlayers,
  appVersion,
  isAdmin,
  isModerator = false,
  changelogEyebrow,
  changelogTitle,
  changelogMarkdown,
  changelogSlug,
  changelogUpdatedAt,
  onLogout,
  onDeleteAccount = async () => { },
}: Props) {
  const [openModal, setOpenModal] = useState<PartyModal>(null);
  const [profileTab, setProfileTab] = useState<"account" | "stats" | "badges">("stats");
  const [inspectedBadgeId, setInspectedBadgeId] = useState("");
  const [hoveredBadgeId, setHoveredBadgeId] = useState("");
  const [isEditingProfileName, setIsEditingProfileName] = useState(false);
  const [isBlogExpanded, setIsBlogExpanded] = useState(false);
  const [queueRulesets, setQueueRulesets] = useState<GameRuleset[]>(["moving"]);
  const [inviteCopied, setInviteCopied] = useState(false);
  const [inviteCodeInput, setInviteCodeInput] = useState("");
  const [mapPickerOpen, setMapPickerOpen] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [mapName, setMapName] = useState("");
  const [mapDescription, setMapDescription] = useState("");
  const [mapDifficulty, setMapDifficulty] = useState<"easy" | "normal" | "hard">("normal");
  const [mapThumbnailKey, setMapThumbnailKey] = useState("generic/variant-1");
  const [mapThumbnailCategory, setMapThumbnailCategory] = useState<"generic" | "continents" | "countries">("generic");
  const [mapThumbnailSearch, setMapThumbnailSearch] = useState("");
  const [mapScope, setMapScope] = useState<MapScope>("community");
  const [mapSort, setMapSort] = useState<MapSort>("trending");
  const [mapSearchInput, setMapSearchInput] = useState("");
  const [debouncedMapSearch, setDebouncedMapSearch] = useState("");
  const [commentBody, setCommentBody] = useState("");
  const [replyBody, setReplyBody] = useState("");
  const [replyToCommentId, setReplyToCommentId] = useState("");
  const [commentComposerFocused, setCommentComposerFocused] = useState(false);
  const [expandedCommentIds, setExpandedCommentIds] = useState<Record<string, boolean>>({});
  const [likedCommentIds, setLikedCommentIds] = useState<Record<string, boolean>>({});
  const [openCommentMenuId, setOpenCommentMenuId] = useState("");
  const [mapFile, setMapFile] = useState<File | null>(null);
  const [mapUploadError, setMapUploadError] = useState("");
  const [nowMs, setNowMs] = useState(() => Date.now());
  const currentNavRoute: LobbyContentRoute = contentRoute === "map-details" || contentRoute === "map-upload" ? "maps" : contentRoute;
  const [visualNavRoute, setVisualNavRoute] = useState<LobbyContentRoute>(() => {
    if (typeof window === "undefined") return currentNavRoute;
    const stored = window.sessionStorage.getItem(lobbyRouteStorageKey) || "";
    return isLobbyNavRoute(stored) ? stored : currentNavRoute;
  });
  const runtimeConfig = getRuntimeConfig();
  const queryClient = useQueryClient();
  const canInteractWithMaps = !!accessToken && !isGuest;
  const canUploadCustomMaps = canInteractWithMaps;
  const selectedMapId = contentRoute === "map-details" ? mapId : "";
  const mapsQuery = useMapList(runtimeConfig, accessToken, userId, mapScope, mapSort, debouncedMapSearch);
  const selectedMapQuery = useMapDetails(runtimeConfig, accessToken, selectedMapId);
  const favoriteMapMutation = useFavoriteMap(runtimeConfig, accessToken);
  const mapComments = useMapComments(runtimeConfig, accessToken, selectedMapId);
  const mapManagement = useMapManagement(runtimeConfig, accessToken, setMapUploadError);
  const createMapMutation = useMutation({
    mutationFn: async () => {
      if (!accessToken) throw new Error("Sign in to upload maps");
      if (isGuest) throw new Error("Upgrade your guest profile to upload custom maps");
      if (!mapFile) throw new Error("Choose a JSON map file");
      await validateMapFile(mapFile);
      return createMap(runtimeConfig, accessToken, { file: mapFile, displayName: mapName, description: mapDescription, difficulty: mapDifficulty, thumbnailKey: mapThumbnailKey });
    },
    onSuccess: () => {
      setMapName(""); setMapDescription(""); setMapFile(null); setMapUploadError(""); setMapDifficulty("normal"); setMapThumbnailKey("generic/variant-1"); setMapThumbnailCategory("generic"); setMapThumbnailSearch("");
      setMapScope("mine");
      void queryClient.invalidateQueries({ queryKey: ["maps"] });
      void Router.push({ pathname: "/maps", query: { scope: "mine" } });
    },
    onError: (error) => setMapUploadError(error instanceof Error ? error.message : "Map upload failed"),
  });
  const archiveMapMutation = mapManagement.archiveMap;
  const revisionMutation = mapManagement.uploadRevision;
  const publishMapMutation = mapManagement.publishMap;
  const createCommentMutation = mapComments.createComment;
  const deleteCommentMutation = mapComments.deleteComment;
  const postMapComment = () => {
    if (!canInteractWithMaps) return;
    createCommentMutation.mutate(
      { body: commentBody },
      { onSuccess: () => {
        setCommentBody("");
        setCommentComposerFocused(false);
      } },
    );
  };
  const postMapReply = (parentId: string) => {
    if (!canInteractWithMaps) return;
    createCommentMutation.mutate(
      { body: replyBody, parentId },
      { onSuccess: () => {
        setReplyBody("");
        setReplyToCommentId("");
        setExpandedCommentIds((current) => ({ ...current, [parentId]: true }));
      } },
    );
  };
  const deleteMap = (map: CustomMap) => {
    if (!window.confirm(`Delete ${map.displayName}?`)) return;
    archiveMapMutation.mutate(map.id, {
      onSuccess: () => {
        setMapScope("mine");
        void Router.push({ pathname: "/maps", query: { scope: "mine" } });
      },
    });
  };

  useEffect(() => {
    if (contentRoute !== "maps" || typeof window === "undefined") return;
    const value = new URLSearchParams(window.location.search).get("scope");
    if (isMapScope(value)) setMapScope(value);
  }, [contentRoute]);
  useEffect(() => {
    const handle = window.setTimeout(() => setDebouncedMapSearch(mapSearchInput.trim()), 500);
    return () => window.clearTimeout(handle);
  }, [mapSearchInput]);
  const toggleCommentLike = (commentId: string) => {
    if (!canInteractWithMaps) return;
    setLikedCommentIds((current) => ({ ...current, [commentId]: !current[commentId] }));
  };
  const toggleCommentReplies = (commentId: string) => {
    setExpandedCommentIds((current) => ({ ...current, [commentId]: !current[commentId] }));
  };

  useEffect(() => {
    if (contentRoute === "top") {
      onBrowseLeaderboard();
    }
  }, [contentRoute, onBrowseLeaderboard]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const frame = window.requestAnimationFrame(() => {
      setVisualNavRoute(currentNavRoute);
      window.sessionStorage.setItem(lobbyRouteStorageKey, currentNavRoute);
    });
    return () => window.cancelAnimationFrame(frame);
  }, [currentNavRoute]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      NAV_ITEMS.forEach((item) => {
        if (item.href !== window.location.pathname && Router.router) {
          void Router.router.prefetch(item.href);
        }
      });
    }, 250);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    if (!maintenance && status !== "queueing") {
      return;
    }
    setNowMs(Date.now());
    const timer = window.setInterval(() => setNowMs(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [maintenance, status]);

  useEffect(() => {
    try {
      const raw = window.localStorage.getItem("geoduels.queueRulesets");
      const parsed = raw ? JSON.parse(raw) : null;
      if (Array.isArray(parsed)) {
        const next = parsed.filter((item): item is GameRuleset => item === "moving" || item === "nmpz");
        setQueueRulesets(Array.from(new Set(next)));
      }
    } catch {
      setQueueRulesets(["moving"]);
    }
  }, []);

  useEffect(() => {
    try {
      window.localStorage.setItem("geoduels.queueRulesets", JSON.stringify(queueRulesets));
    } catch {
      // Ignore storage failures; defaults still work.
    }
  }, [queueRulesets]);

  const isQueueing = status === "queueing";
  const isSingleplayerLoading = status === "matched_connecting";
  const canUseRankedQueue = !!userId && !isGuest;
  const toggleQueueRuleset = (ruleset: GameRuleset) => {
    setQueueRulesets((current) => {
      if (current.includes(ruleset)) {
        return current.filter((item) => item !== ruleset);
      }
      return [...current, ruleset];
    });
  };
  const queueElapsedLabel = formatQueueElapsed(
    queueStartedAt ? nowMs - queueStartedAt : 0,
  );
  const showConnectionError =
    !connected && queueError.toLowerCase() === "connection error";
  const primaryButtonLabel = showConnectionError ? "Connection Error" : "Play";
  const userAvatarFallback = !userEmail
    ? "?"
    : (displayName || userEmail || "P").slice(0, 1).toUpperCase();
  const duelModeLabel = isQueueing ? "Searching..." : "Ranked";
  const showGoogleButton = !!googleClientId;
  const showDiscordButton = !!discordClientId;
  const hasGoogleProvider = linkedProviders.includes("google");
  const hasDiscordProvider = linkedProviders.includes("discord");
  const linkedProviderCount = linkedProviders.filter((provider) =>
    provider === "google" || provider === "discord"
  ).length;
  const profileTabs: Array<{ id: "account" | "stats" | "badges"; label: string }> = [
    { id: "stats", label: "Stats" },
    { id: "badges", label: "Badges" },
    { id: "account", label: "Account" },
  ];
  const focusedBadge =
    badges.find((badge) => badge.id === hoveredBadgeId) ||
    badges.find((badge) => badge.id === inspectedBadgeId) ||
    badges.find((badge) => badge.id === selectedBadge?.id) ||
    badges[0] ||
    null;
  const maintenanceStartMs = parseTime(maintenance?.startsAt);
  const maintenanceEndMs = parseTime(maintenance?.endsAt);
  const maintenanceIsWarning = maintenance?.phase === "warning";
  const maintenanceIsActive = maintenance?.phase === "active";
  const queuePaused = !!maintenance?.queuePaused;
  const playPaused = !!maintenance?.playPaused;
  const maintenanceMessage = maintenance?.message?.trim() || "";
  const warningCountdown =
    maintenanceIsWarning && maintenanceStartMs && maintenanceStartMs > nowMs
      ? formatRelativeDuration(maintenanceStartMs - nowMs)
      : "";
  const activeEta =
    maintenanceIsActive && maintenanceEndMs && maintenanceEndMs > nowMs
      ? formatApproximateTime(maintenanceEndMs - nowMs)
      : "";
  const duelDisabled =
    authLoading ||
    authMigrationRequired ||
    nicknameSaving ||
    queuePaused ||
    playPaused ||
    maintenanceIsActive ||
    queueRulesets.length === 0;
  const singleplayerDisabled =
    isQueueing ||
    isSingleplayerLoading ||
    authLoading ||
    authMigrationRequired ||
    nicknameSaving ||
    playPaused ||
    maintenanceIsActive;
  const onRankedPlay = () => {
    if (!canUseRankedQueue) {
      setOpenModal("signin");
      return;
    }
    joinQueue(queueRulesets);
  };

  const discordProviderButton = showDiscordButton ? (
    <button
      type="button"
      onClick={onDiscordSignIn}
      disabled={authLoading}
      className="glass-panel glass-panel-interactive group inline-flex items-center justify-center gap-3 rounded-[20px] px-3 py-2.5 text-[12px] font-extrabold uppercase tracking-[0.1em] text-white disabled:cursor-not-allowed disabled:opacity-60 sm:px-4"
    >
      <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[#5865f2] text-white shadow-sm">
        <svg viewBox="0 0 127.14 96.36" className="h-3.5 w-4" aria-hidden="true">
          <path
            fill="currentColor"
            d="M107.7 8.07A105.15 105.15 0 0 0 81.47 0a72.06 72.06 0 0 0-3.36 6.83 97.68 97.68 0 0 0-29.11 0A72.37 72.37 0 0 0 45.64 0 105.89 105.89 0 0 0 19.39 8.09C2.79 32.65-1.71 56.6.54 80.21a105.73 105.73 0 0 0 32.17 16.15 77.7 77.7 0 0 0 6.89-11.11 68.42 68.42 0 0 1-10.85-5.18c.91-.66 1.8-1.34 2.66-2.04a75.57 75.57 0 0 0 64.32 0c.87.71 1.76 1.39 2.66 2.04a68.68 68.68 0 0 1-10.87 5.19 77 77 0 0 0 6.89 11.1 105.25 105.25 0 0 0 32.19-16.14c2.64-27.38-4.52-51.11-18.9-72.15ZM42.45 65.69c-6.27 0-11.43-5.73-11.43-12.78s5.05-12.79 11.43-12.79 11.54 5.78 11.43 12.79-5.06 12.78-11.43 12.78Zm42.24 0c-6.27 0-11.43-5.73-11.43-12.78s5.05-12.79 11.43-12.79 11.54 5.78 11.43 12.79-5.05 12.78-11.43 12.78Z"
          />
        </svg>
      </span>
      {authLoading ? "Signing In..." : "Continue With Discord"}
    </button>
  ) : null;

  const signInButton =
    showGoogleButton || showDiscordButton ? (
      <button
        type="button"
        onClick={() => setOpenModal("signin")}
        disabled={authLoading}
        className="glass-panel glass-panel-interactive group inline-flex items-center justify-center gap-3 rounded-[20px] px-3 py-2.5 text-[12px] font-extrabold uppercase tracking-[0.1em] text-white disabled:cursor-not-allowed disabled:opacity-60 sm:px-4"
      >
        {authLoading ? "Signing In..." : "Sign In"}
      </button>
    ) : (
      <button
        type="button"
        onClick={devLogin}
        className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-[12px] font-extrabold uppercase tracking-[0.1em] text-white transition hover:bg-white/10"
      >
        {authLoading ? "Signing In..." : "Dev Login"}
      </button>
    );

  const googleProviderButton = showGoogleButton ? (
    <button
      type="button"
      onClick={onGoogleSignIn}
      disabled={authLoading}
      className="glass-panel glass-panel-interactive group inline-flex items-center justify-center gap-3 rounded-[20px] px-3 py-2.5 text-[12px] font-extrabold uppercase tracking-[0.1em] text-white disabled:cursor-not-allowed disabled:opacity-60 sm:px-4"
    >
      <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white text-[#111827] shadow-sm">
        <svg viewBox="0 0 24 24" className="h-3.5 w-3.5" aria-hidden="true">
          <path
            fill="#4285F4"
            d="M21.805 10.023h-9.81v3.955h5.627c-.242 1.272-.967 2.35-2.06 3.073v2.55h3.332c1.95-1.796 3.073-4.44 3.073-7.578 0-.662-.06-1.298-.162-1.999Z"
          />
          <path
            fill="#34A853"
            d="M11.995 22c2.79 0 5.132-.924 6.842-2.5l-3.332-2.55c-.924.62-2.102.987-3.51.987-2.699 0-4.985-1.822-5.805-4.272H2.758v2.63A10.329 10.329 0 0 0 11.995 22Z"
          />
          <path
            fill="#FBBC05"
            d="M6.19 13.665a6.214 6.214 0 0 1-.324-1.967c0-.684.118-1.347.324-1.967v-2.63H2.758A10.329 10.329 0 0 0 1.663 11.7c0 1.66.398 3.232 1.095 4.598l3.432-2.633Z"
          />
          <path
            fill="#EA4335"
            d="M11.995 5.463c1.518 0 2.88.523 3.95 1.55l2.962-2.962C17.122 2.397 14.782 1.4 11.995 1.4 7.958 1.4 4.47 3.707 2.758 7.101l3.432 2.63c.82-2.45 3.106-4.268 5.805-4.268Z"
          />
        </svg>
      </span>
      {authLoading ? "Opening Google..." : "Continue With Google"}
    </button>
  ) : null;

  const newsPanel = (
    <div
      className="glass-panel glass-panel-interactive lobby-feature-card group w-full rounded-[20px] p-5"
      style={{ animationDelay: "-3s" }}
    >
      <button
        type="button"
        onClick={() => setIsBlogExpanded((prev) => !prev)}
        className="block w-full text-left"
      >
        <div className="flex items-center justify-between">
          <div>
            <span className="mb-1 block text-[12px] font-bold uppercase tracking-[0.16em] text-[#2ad18f] drop-shadow-sm">
              {changelogEyebrow}
            </span>
            <h2 className="text-[20px] font-extrabold leading-tight tracking-tight text-white drop-shadow-md">
              {changelogTitle}
            </h2>
          </div>
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-white/5 text-white/70 transition-colors group-hover:bg-white/10 group-hover:text-white">
            {isBlogExpanded ? (
              <ChevronUp size={20} />
            ) : (
              <ChevronDown size={20} />
            )}
          </div>
        </div>
        <AnimatePresence>
          {isBlogExpanded && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              className="overflow-hidden"
            >
              <div className="mt-5 space-y-4 border-t border-white/[0.06] pt-5">
                <MarkdownContent
                  markdown={changelogMarkdown || "No changelog content yet."}
                  compact
                />
                <div className="flex items-center justify-between gap-3 pt-1">
                  {changelogUpdatedAt ? (
                    <time
                      dateTime={changelogUpdatedAt}
                      className="text-[12px] font-semibold text-[#a9bfd4]/70"
                    >
                      Updated {formatChangelogDate(changelogUpdatedAt)}
                    </time>
                  ) : <span />}
                  <Link
                    href={changelogSlug ? `/changelog/${encodeURIComponent(changelogSlug)}` : "/changelog"}
                    className="inline-flex items-center gap-1 text-[12px] font-extrabold uppercase tracking-[0.12em] text-[#77f0be] transition hover:text-white"
                  >
                    Read More
                    <ArrowUpRight size={14} />
                  </Link>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </button>
    </div>
  );

  const donateCard = (
    <button
      type="button"
      onClick={() => void onSupportDonation()}
      className="glass-panel glass-panel-interactive lobby-feature-card group flex w-full items-center gap-4 rounded-[20px] p-5 text-left"
      style={{ animationDelay: "-0.75s" }}
    >
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-[#ef476f]/14 text-[#f7a1b5]">
        <Heart size={22} />
      </div>
      <div className="min-w-0 flex-1">
        <span className="mb-1 block text-[12px] font-bold uppercase tracking-[0.16em] text-[#ee7f98]">
          Donate
        </span>
        <div className="flex items-center justify-between gap-3">
          <div>
            <h3 className="text-[18px] font-extrabold tracking-tight text-white">
              Support GeoDuels
            </h3>
            <p className="mt-1 text-[13px] leading-relaxed text-[#a9bfd4]">
              Help GeoDuels stay ad-free and in active development by donating :D
            </p>
          </div>
          <ArrowUpRight
            size={18}
            className="shrink-0 text-white/50 transition-colors group-hover:text-white"
          />
        </div>
      </div>
    </button>
  );

  const socialLinksCard = (
    <div
      className="glass-panel lobby-feature-card flex w-full flex-col gap-4 rounded-[20px] p-5"
      style={{ animationDelay: "-1s" }}
    >
      <span className="block text-[12px] font-bold uppercase tracking-[0.16em] text-[#6b8b80]">
        Community
      </span>
      <div className="flex flex-wrap gap-3">
        {[
          {
            href: "https://discord.gg/xxz8V9UU7Z",
            label: "Discord",
            icon: <svg viewBox="0 0 127.14 96.36" className="h-5 w-5" aria-hidden="true"><path fill="currentColor" d="M107.7 8.07A105.15 105.15 0 0 0 81.47 0a72.06 72.06 0 0 0-3.36 6.83 97.68 97.68 0 0 0-29.11 0A72.37 72.37 0 0 0 45.64 0 105.89 105.89 0 0 0 19.39 8.09C2.79 32.65-1.71 56.6.54 80.21a105.73 105.73 0 0 0 32.17 16.15 77.7 77.7 0 0 0 6.89-11.11 68.42 68.42 0 0 1-10.85-5.18c.91-.66 1.8-1.34 2.66-2.04a75.57 75.57 0 0 0 64.32 0c.87.71 1.76 1.39 2.66 2.04a68.68 68.68 0 0 1-10.87 5.19 77 77 0 0 0 6.89 11.1 105.25 105.25 0 0 0 32.19-16.14c2.64-27.38-4.52-51.11-18.9-72.15ZM42.45 65.69c-6.27 0-11.43-5.73-11.43-12.78s5.05-12.79 11.43-12.79 11.54 5.78 11.43 12.79-5.06 12.78-11.43 12.78Zm42.24 0c-6.27 0-11.43-5.73-11.43-12.78s5.05-12.79 11.43-12.79 11.54 5.78 11.43 12.79-5.05 12.78-11.43 12.78Z" /></svg>,
          },
          {
            href: "https://github.com/sourcelocation/geoduels",
            label: "GitHub",
            icon: <Github size={20} />,
          },
          {
            href: "http://twitter.com/sourceloc",
            label: "Twitter",
            icon: <Twitter size={20} />,
          },
          {
            href: "https://youtube.com/@sourcelocation",
            label: "YouTube",
            icon: <Youtube size={20} />,
          },
        ].map((social) => (
          <a
            key={social.label}
            href={social.href}
            target="_blank"
            rel="noreferrer"
            aria-label={social.label}
            className="glass-panel glass-panel-interactive flex h-12 w-12 items-center justify-center rounded-full text-white"
          >
            {social.icon}
          </a>
        ))}
      </div>
    </div>
  );

  const onlineStatusCard = (
    <div
      className="glass-panel lobby-feature-card flex w-full items-center gap-3 rounded-[20px] px-[20px] py-3"
      style={{ animationDelay: "-0.5s" }}
    >
      <div className="status-dot-wrap relative flex h-4 w-4 shrink-0 items-center justify-center">
        <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-[#2ad18f]" />
      </div>
      <div className="min-w-0">
        <p className="text-[12px] font-semibold text-[#2ad18f] transition-colors">
          {onlinePlayers} Playing
        </p>
      </div>
    </div>
  );

  const lobbyInviteURL =
    typeof window !== "undefined" && privateLobby.inviteCode
      ? `${window.location.origin}/party/${privateLobby.inviteCode}`
      : "";
  const privateLobbyLoading =
    !privateLobby.snapshot &&
    ["creating", "joining", "connecting", "reconnecting"].includes(
      privateLobby.status,
    );
  const privateLobbyActive =
    !!privateLobby.snapshot ||
    privateLobby.status !== "idle";
  const lobbyMembers = privateLobby.snapshot?.members || [];
  const activeLobbyMatchId = privateLobby.snapshot?.activeMatchId || privateLobby.snapshot?.startedMatchId || "";
  const lobbyMatchInProgress = privateLobby.snapshot?.state === "in_match" || privateLobby.snapshot?.state === "started";
  const currentLobbyMember = lobbyMembers.find((member) => member.userId === userId);
  const lobbyConfig = privateLobby.snapshot?.config || { ruleset: "moving", roundTimerMode: "none", pressureTimeLimitMs: 15000 };
  const lobbyMode = privateLobby.snapshot?.mode || "duel";
  const lobbyClockOn = lobbyConfig.roundTimerMode === "fixed";
  const lobbyPressureOn =
    (typeof lobbyConfig.pressureTimeLimitMs === "number" && lobbyConfig.pressureTimeLimitMs > 0) ||
    lobbyConfig.roundTimerMode === "pressure";
  const lobbyRoundSeconds = Math.round((lobbyConfig.roundTimeLimitMs || 45000) / 1000);
  const lobbyPressureSeconds = lobbyPressureOn ? Math.round((lobbyConfig.pressureTimeLimitMs || 15000) / 1000) : 0;
  const saveLobbyConfig = (patch: MatchConfig) => {
    const next: MatchConfig = {
      ...lobbyConfig,
      ...patch,
    };
    if (next.roundTimerMode !== "fixed") {
      next.roundTimerMode = "none";
      next.roundTimeLimitMs = undefined;
    } else {
      next.roundTimeLimitMs = Math.max(10000, Math.min(120000, next.roundTimeLimitMs || 45000));
    }
    if ((next.pressureTimeLimitMs || 0) !== 15000) {
      next.pressureTimeLimitMs = undefined;
    }
    void updatePrivateLobbySettings(next);
  };
  const saveLobbyMode = (mode: PartyMode) => {
    void updatePrivateLobbySettings(lobbyConfig, mode);
  };
  const missingLobbyMembers = lobbyMembers.filter((member) => (member.presenceStatus || (member.connected ? "online" : "offline")) !== "online");
  const teamACount = lobbyMembers.filter((member) => (member.teamId || "a") === "a").length;
  const teamBCount = lobbyMembers.filter((member) => member.teamId === "b").length;
  const canStartPrivateParty =
    privateLobby.isOwner &&
    privateLobby.snapshot?.state === "open" &&
    (
      (lobbyMode === "duel" && lobbyMembers.length === 2) ||
      (lobbyMode === "team_duel" && lobbyMembers.length >= 2 && lobbyMembers.length <= 8 && teamACount > 0 && teamBCount > 0) ||
      (lobbyMode === "free_for_all" && lobbyMembers.length >= 2 && lobbyMembers.length <= 8)
    ) &&
    missingLobbyMembers.length === 0;
  const copyInvite = () => {
    if (!lobbyInviteURL) return;
    void navigator.clipboard?.writeText(lobbyInviteURL);
    setInviteCopied(true);
    window.setTimeout(() => setInviteCopied(false), 1600);
  };

  const availableMaps = mapsQuery.data || [];
  const readyMaps = availableMaps.filter((item) => item.status === "ready");
  const hasMapSearch = debouncedMapSearch.length > 0;
  const selectedMapDetails = selectedMapQuery.data;
  const mapScopeLabels: Array<{ scope: MapScope; label: string }> = [
    { scope: "official", label: "Official" },
    { scope: "community", label: "Community" },
    { scope: "favorites", label: "Favorites" },
    { scope: "mine", label: "My Maps" },
  ];
  const selectedThumbnail = mapThumbnailOptions.find((item) => item.key === mapThumbnailKey) || mapThumbnailOptions[0];
  const thumbnailURL = (item: Pick<CustomMap, "thumbnailVariant" | "thumbnailKey">) => mapThumbnailURL(item.thumbnailKey, item.thumbnailVariant);
  const renderMapSearchControl = (id: string) => (
    <div className="relative w-full sm:w-[260px]">
      <label htmlFor={id} className="sr-only">Search maps</label>
      <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[#77f0be]" size={16} />
      <input
        id={id}
        type="search"
        value={mapSearchInput}
        onChange={(event) => setMapSearchInput(event.target.value)}
        placeholder="Search maps"
        className="h-11 w-full rounded-[14px] border border-white/10 bg-black/25 py-2 pl-9 pr-10 text-sm font-semibold text-white outline-none transition placeholder:text-[#6f8998] focus:border-[#77f0be]/60 focus:bg-black/35"
      />
      {mapSearchInput ? (
        <button
          type="button"
          aria-label="Clear map search"
          onClick={() => setMapSearchInput("")}
          className="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-[#a9bfd4] transition hover:bg-white/[0.08] hover:text-white"
        >
          <X size={15} />
        </button>
      ) : null}
    </div>
  );
  const filteredThumbnailOptions = mapThumbnailOptions.filter((item) => {
    const q = mapThumbnailSearch.trim().toLowerCase();
    return item.category === mapThumbnailCategory && (!q || item.label.toLowerCase().includes(q) || item.search.toLowerCase().includes(q) || item.key.includes(q));
  });
  const mapPickerFlow = privateLobbyActive && privateLobby.isOwner && privateLobby.snapshot?.state === "open";
  const selectMapForParty = (item: CustomMap) => {
    if (privateLobbyActive && privateLobby.isOwner) {
      saveLobbyConfig({ mapId: item.id, mapName: item.displayName });
      setMapPickerOpen(false);
      return;
    }
    void createInviteLobby("duel", { mapId: item.id, mapName: item.displayName, ruleset: "moving", roundTimerMode: "none", pressureTimeLimitMs: 15000 });
  };
  const playMapSingleplayer = (item: CustomMap) => {
    void startSingleplayer({
      mapId: item.id,
      mapName: item.displayName,
      ruleset: "moving",
      roundTimerMode: "none",
      pressureTimeLimitMs: 15000,
    });
  };
  const mapUploadForm = (
    <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_280px]">
      <div className="grid gap-3">
        <input value={mapName} onChange={(event)=>setMapName(event.target.value)} maxLength={80} placeholder="Map name" disabled={isGuest} className="h-11 rounded-[12px] border border-white/10 bg-black/25 px-3 text-sm font-semibold text-white outline-none focus:border-[#2ad18f]/60 disabled:opacity-50" />
        <textarea value={mapDescription} onChange={(event)=>setMapDescription(event.target.value)} maxLength={500} placeholder="Description (optional)" disabled={isGuest} className="min-h-20 resize-none rounded-[12px] border border-white/10 bg-black/25 p-3 text-sm text-white outline-none focus:border-[#2ad18f]/60 disabled:opacity-50" />
        <div className="grid gap-3 sm:grid-cols-2">
          <select value={mapDifficulty} onChange={(event)=>setMapDifficulty(event.target.value as "easy"|"normal"|"hard")} disabled={isGuest} className="h-11 rounded-[12px] border border-white/10 bg-[#101a20] px-3 text-sm font-bold text-white outline-none disabled:opacity-50"><option value="easy">Easy</option><option value="normal">Normal</option><option value="hard">Hard</option></select>
          <input value={mapThumbnailSearch} onChange={(event)=>setMapThumbnailSearch(event.target.value)} placeholder="Search thumbnails" disabled={isGuest} className="h-11 rounded-[12px] border border-white/10 bg-black/25 px-3 text-sm font-semibold text-white outline-none focus:border-[#2ad18f]/60 disabled:opacity-50" />
        </div>
        <div className="rounded-[14px] border border-white/10 bg-black/20 p-3">
          <div className="mb-3 flex flex-wrap gap-2">
            {(["generic", "continents", "countries"] as const).map((category) => (
              <button key={category} type="button" onClick={()=>setMapThumbnailCategory(category)} className={`rounded-[10px] px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] ${mapThumbnailCategory === category ? "bg-white text-[#10201a]" : "bg-white/[0.06] text-[#a9bfd4] hover:bg-white/[0.1]"}`}>
                {category}
              </button>
            ))}
          </div>
          <div className="grid max-h-[320px] gap-2 overflow-y-auto pr-1 sm:grid-cols-2">
            {filteredThumbnailOptions.map((option) => (
              <button key={option.key} type="button" onClick={()=>setMapThumbnailKey(option.key)} className={`overflow-hidden rounded-[12px] border text-left transition ${mapThumbnailKey === option.key ? "border-[#77f0be] bg-[#77f0be]/10" : "border-white/10 bg-white/[0.04] hover:bg-white/[0.08]"}`}>
                <img src={mapThumbnailURL(option.key)} alt="" className="aspect-[16/9] w-full object-cover" />
                <div className="p-2 text-[11px] font-black text-white">{option.label}</div>
              </button>
            ))}
          </div>
        </div>
        <label className="flex min-h-20 cursor-pointer flex-col items-center justify-center rounded-[14px] border border-dashed border-white/20 bg-black/20 px-4 text-center text-sm font-semibold text-[#a9bfd4] hover:border-[#2ad18f]/50">
          <Upload className="mb-2 text-[#2ad18f]" size={22} />{mapFile ? mapFile.name : "Choose JSON file"}
          <input type="file" accept=".json,application/json" className="hidden" disabled={isGuest} onChange={(event)=>{setMapFile(event.target.files?.[0]||null);setMapUploadError("");}} />
        </label>
      </div>
      <div className="grid content-start gap-3">
        <img src={mapThumbnailURL(mapThumbnailKey)} alt="" className="aspect-[16/9] w-full rounded-[14px] object-cover" />
        <p className="text-xs font-bold text-[#a9bfd4]">Selected: <span className="text-white">{selectedThumbnail.label}</span></p>
        {mapUploadError ? <p className="text-xs font-semibold text-red-300">{mapUploadError}</p> : null}
        <button type="button" disabled={isGuest || !mapName.trim() || !mapFile || createMapMutation.isPending} onClick={()=>createMapMutation.mutate()} className="inline-flex h-11 items-center justify-center rounded-[12px] bg-[#22d385] text-sm font-extrabold uppercase tracking-[0.08em] text-white disabled:cursor-not-allowed disabled:opacity-50">{createMapMutation.isPending ? <Loader2 className="mr-2 animate-spin" size={17}/> : <Upload className="mr-2" size={17}/>}Upload</button>
        <p className="text-[11px] leading-5 text-[#6f8998]">Limits: 10 maps, 100,000 locations per map, 250,000 active locations per account, 3 uploads/hour.</p>
      </div>
    </div>
  );
  const mapsPanel = (
    <motion.div key="maps" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <div className="glass-panel overflow-hidden rounded-[24px]">
        <div className="grid min-h-[640px] lg:grid-cols-[220px_minmax(0,1fr)]">
          <aside className="border-b border-white/10 bg-black/20 p-4 lg:border-b-0 lg:border-r">
            <div className="mb-4 flex items-center gap-2 text-white">
              <MapIcon className="text-[#77f0be]" size={22} />
              <span className="text-lg font-black">Maps</span>
            </div>
            <div className="grid gap-2">
              {mapScopeLabels.map((item) => (
                <button key={item.scope} type="button" onClick={() => setMapScope(item.scope)} className={`rounded-[14px] px-4 py-3 text-left text-sm font-extrabold transition ${mapScope === item.scope ? "bg-[#22d385] text-white" : "bg-white/[0.05] text-[#a9bfd4] hover:bg-white/[0.09]"}`}>
                  {item.label}
                </button>
              ))}
            </div>
          </aside>
          <section className="p-5 sm:p-7">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <span className="text-[11px] font-black uppercase tracking-[0.16em] text-[#77f0be]">{privateLobbyActive ? "Party Map Select" : "Map Browser"}</span>
                <h2 className="mt-1 text-[30px] font-black text-white">{mapScopeLabels.find((item) => item.scope === mapScope)?.label}</h2>
                <p className="mt-2 text-sm text-[#a9bfd4]">{privateLobbyActive ? "Click a ready map to select it for the upcoming party game." : "Browse official and community maps, inspect details, favorite the good stuff."}</p>
              </div>
              <div className="flex w-full flex-col gap-3 sm:w-auto sm:items-end">
                {renderMapSearchControl("map-browser-search")}
                {mapScope === "community" ? (
                  <div className="flex rounded-[14px] border border-white/10 bg-black/20 p-1">
                    {(["trending", "popular", "new"] as MapSort[]).map((sort) => (
                      <button key={sort} type="button" onClick={() => setMapSort(sort)} className={`rounded-[10px] px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] ${mapSort === sort ? "bg-white text-[#10201a]" : "text-[#a9bfd4] hover:bg-white/[0.08]"}`}>
                        {sort === "popular" ? "Most Popular" : sort}
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>
            </div>

            {mapScope === "mine" && !canUploadCustomMaps ? (
              <div className="mt-6 rounded-[16px] border border-white/10 bg-black/20 p-5 text-sm font-semibold text-[#a9bfd4]">Sign in with a permanent account to create custom maps.</div>
            ) : mapsQuery.isLoading ? (
              <div className="mt-8 flex items-center gap-3 text-sm text-[#a9bfd4]"><Loader2 className="animate-spin" size={18} /> Loading maps...</div>
            ) : readyMaps.length === 0 ? (
              <div className="mt-8 rounded-[18px] border border-dashed border-white/15 bg-black/15 p-8 text-center text-sm text-[#a9bfd4]">{hasMapSearch ? "No maps match your search." : "No maps in this section yet."}</div>
            ) : (
              <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {readyMaps.map((item) => (
                  <article key={item.id} className="overflow-hidden rounded-[18px] border border-white/10 bg-black/25">
                    {privateLobbyActive ? (
                    <button type="button" onClick={() => selectMapForParty(item)} className="block w-full text-left">
                      <div className="relative aspect-[16/9] overflow-hidden bg-[#10201a]">
                        <img src={thumbnailURL(item)} alt="" className="h-full w-full object-cover opacity-90 transition hover:scale-[1.03]" />
                        <div className="absolute left-3 top-3 rounded-full bg-black/55 px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white">{item.difficulty}</div>
                        {item.system ? <div className="absolute right-3 top-3 rounded-full bg-[#22d385] px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white">Official</div> : null}
                      </div>
                      <div className="p-4">
                        <h3 className="truncate text-base font-black text-white">{item.displayName}</h3>
                        <p className="mt-1 truncate text-xs font-semibold text-[#8da6b5]">by {item.authorName || "GeoDuels"}</p>
                        <div className="mt-3 flex flex-wrap gap-2 text-[11px] font-bold text-[#a9bfd4]">
                          <span>{item.locationCount.toLocaleString()} locations</span>
                          <span>{item.playCount.toLocaleString()} plays</span>
                          <span>{item.favoriteCount.toLocaleString()} favorites</span>
                        </div>
                      </div>
                    </button>
                    ) : (
                    <Link href={`/maps/${encodeURIComponent(item.id)}`} className="block w-full text-left">
                      <div className="relative aspect-[16/9] overflow-hidden bg-[#10201a]">
                        <img src={thumbnailURL(item)} alt="" className="h-full w-full object-cover opacity-90 transition hover:scale-[1.03]" />
                        <div className="absolute left-3 top-3 rounded-full bg-black/55 px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white">{item.difficulty}</div>
                        {item.system ? <div className="absolute right-3 top-3 rounded-full bg-[#22d385] px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white">Official</div> : null}
                      </div>
                      <div className="p-4">
                        <h3 className="truncate text-base font-black text-white">{item.displayName}</h3>
                        <p className="mt-1 truncate text-xs font-semibold text-[#8da6b5]">by {item.authorName || "GeoDuels"}</p>
                        <div className="mt-3 grid grid-cols-3 gap-2 text-[11px] font-bold text-[#a9bfd4]">
                          <span className="inline-flex items-center gap-1" title="Locations"><MapIcon size={13} />{item.locationCount.toLocaleString()}</span>
                          <span className="inline-flex items-center gap-1" title="Plays"><Play size={13} />{item.playCount.toLocaleString()}</span>
                          <span className="inline-flex items-center gap-1" title="Favorites"><Star size={13} />{item.favoriteCount.toLocaleString()}</span>
                        </div>
                      </div>
                    </Link>
                    )}
                  </article>
                ))}
              </div>
            )}

            {mapScope === "mine" && canUploadCustomMaps ? (
              <div className="mt-7 rounded-[20px] border border-white/10 bg-black/20 p-5">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <span className="text-[11px] font-black uppercase tracking-[0.16em] text-[#77f0be]">Upload Map</span>
                    <p className="mt-2 text-sm font-semibold text-[#a9bfd4]">Create a custom map from a GeoDuels JSON file.</p>
                  </div>
                  <Link href="/maps/upload" className="inline-flex min-h-[42px] items-center justify-center rounded-[12px] bg-[#22d385] px-4 text-sm font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-[#2ae091]">
                    <Upload className="mr-2" size={17} />
                    Upload
                  </Link>
                </div>
              </div>
            ) : null}
          </section>
        </div>
      </div>
      {contentRoute === "map-details" ? (
        <div className="mt-5 rounded-[22px] border border-white/10 bg-[#071114]/95 p-5 sm:p-6">
          {selectedMapQuery.isLoading || !selectedMapDetails ? (
            <div className="flex items-center gap-3 text-sm text-[#a9bfd4]"><Loader2 className="animate-spin" size={18} /> Loading map details...</div>
          ) : (
            <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
              <div>
                <Link href="/maps" className="mb-3 inline-flex text-xs font-black uppercase tracking-[0.12em] text-[#77f0be]">Back to maps</Link>
                <img src={thumbnailURL(selectedMapDetails.map)} alt="" className="aspect-[16/9] w-full rounded-[18px] object-cover" />
                <h3 className="mt-4 text-2xl font-black text-white">{selectedMapDetails.map.displayName}</h3>
                <p className="mt-1 text-sm text-[#a9bfd4]">by {selectedMapDetails.map.authorName || "GeoDuels"} · {selectedMapDetails.map.difficulty} · {selectedMapDetails.map.locationCount.toLocaleString()} locations</p>
                {selectedMapDetails.map.description ? <p className="mt-3 text-sm leading-6 text-[#8da6b5]">{selectedMapDetails.map.description}</p> : null}
                <div className="mt-4 grid gap-2 sm:grid-cols-3">
                  {selectedMapDetails.countryStats.slice(0, 12).map((stat) => (
                    <div key={stat.country} className="rounded-[12px] border border-white/10 bg-white/[0.04] p-3">
                      <div className="truncate text-xs font-bold text-[#a9bfd4]">{stat.country}</div>
                      <div className="mt-1 h-2 overflow-hidden rounded-full bg-white/10"><div className="h-full rounded-full bg-[#22d385]" style={{ width: `${Math.max(6, Math.round((stat.locationCount / Math.max(1, selectedMapDetails.map.locationCount)) * 100))}%` }} /></div>
                      <div className="mt-1 text-[11px] font-black text-white">{stat.locationCount.toLocaleString()}</div>
                    </div>
                  ))}
                </div>
              </div>
              <aside>
                <div className="flex flex-wrap gap-2">
                  <button type="button" onClick={() => selectMapForParty(selectedMapDetails.map)} className="rounded-[12px] bg-[#22d385] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-white">Use in Party</button>
                  {canInteractWithMaps ? <button type="button" onClick={() => favoriteMapMutation.mutate({ mapId: selectedMapDetails.map.id, favorite: !selectedMapDetails.map.favorited })} className="rounded-[12px] border border-white/10 bg-white/[0.06] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-white">{selectedMapDetails.map.favorited ? "Unfavorite" : "Favorite"}</button> : null}
                  {selectedMapDetails.map.ownerUserId === userId && canUploadCustomMaps ? (
                    <>
                      {!selectedMapDetails.map.publishedAt ? <button type="button" onClick={() => publishMapMutation.mutate(selectedMapDetails.map.id)} className="rounded-[12px] border border-[#77f0be]/20 bg-[#77f0be]/10 px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-[#baf7dc]">Publish</button> : null}
                      <label className="inline-flex cursor-pointer items-center rounded-[12px] border border-white/10 bg-white/[0.06] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-white hover:bg-white/[0.1]">
                        <Upload className="mr-1.5" size={14} /> New Version
                        <input type="file" accept=".json,application/json" className="hidden" onChange={(event) => { const file=event.target.files?.[0]; if(file) revisionMutation.mutate({mapId:selectedMapDetails.map.id,file}); event.currentTarget.value=""; }} />
                      </label>
                      <button type="button" onClick={() => deleteMap(selectedMapDetails.map)} className="rounded-[12px] border border-red-400/15 bg-red-400/[0.06] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-red-200 hover:bg-red-400/10">Delete</button>
                    </>
                  ) : null}
                </div>
                <div className="mt-5 rounded-[18px] border border-white/10 bg-black/20 p-4">
                  <h4 className="flex items-center gap-2 text-sm font-black text-white"><MessageCircle size={16} /> Comments</h4>
                  {canInteractWithMaps ? (
                    <div className="mt-3">
                      <textarea value={commentBody} onChange={(event)=>setCommentBody(event.target.value)} maxLength={1000} placeholder="Add a comment" className="min-h-20 w-full resize-none rounded-[12px] border border-white/10 bg-black/25 p-3 text-sm text-white outline-none focus:border-[#2ad18f]/60" />
                      <button type="button" disabled={!commentBody.trim() || createCommentMutation.isPending} onClick={postMapComment} className="mt-2 rounded-[10px] bg-[#22d385] px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] text-white disabled:opacity-50">Post</button>
                    </div>
                  ) : <p className="mt-3 text-xs text-[#8da6b5]">{accessToken ? "Upgrade your guest profile to comment." : "Sign in to comment."}</p>}
                  <div className="mt-4 grid gap-3">
                    {selectedMapDetails.comments.map((comment) => (
                      <div key={comment.id} className="rounded-[14px] border border-white/10 bg-white/[0.04] p-3">
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <p className="text-sm font-black text-white">
                              {comment.userDisplayName}
                              {commentDeletedLabel(comment.status) ? <span className="ml-2 text-xs font-black text-red-300">{commentDeletedLabel(comment.status)}</span> : null}
                            </p>
                            <p className="mt-1 text-sm leading-5 text-[#a9bfd4]">{comment.body}</p>
                          </div>
                          {comment.status === "visible" && (comment.canDelete || isAdmin || isModerator) && accessToken ? <button type="button" onClick={() => deleteCommentMutation.mutate({ commentId: comment.id })} className="text-red-200 hover:text-red-100"><Trash2 size={15} /></button> : null}
                        </div>
                        {canInteractWithMaps && comment.status === "visible" ? <button type="button" onClick={() => { setReplyToCommentId(comment.id); setReplyBody(""); }} className="mt-2 text-[11px] font-black uppercase tracking-[0.08em] text-[#77f0be]">Reply</button> : null}
                        {comment.replies?.map((reply) => (
                          <div key={reply.id} className="mt-3 border-l border-white/10 pl-3">
                            <div className="flex items-start justify-between gap-3">
                              <p className="text-sm leading-5 text-[#a9bfd4]">
                                <span className="font-black text-white">{reply.userDisplayName}</span>
                                {commentDeletedLabel(reply.status) ? <span className="ml-2 text-xs font-black text-red-300">{commentDeletedLabel(reply.status)}</span> : null}
                                {" "}{reply.body}
                              </p>
                              {reply.status === "visible" && (reply.canDelete || isAdmin || isModerator) && accessToken ? <button type="button" onClick={() => deleteCommentMutation.mutate({ commentId: reply.id })} className="text-red-200 hover:text-red-100"><Trash2 size={14} /></button> : null}
                            </div>
                          </div>
                        ))}
                      </div>
                    ))}
                  </div>
                </div>
              </aside>
            </div>
          )}
        </div>
      ) : null}
    </motion.div>
  );

  const mapUploadPanel = (
    <motion.div key="map-upload" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <div className="rounded-[20px] border border-white/10 bg-white/[0.045] p-4 shadow-[0_22px_70px_rgba(0,0,0,0.28)] backdrop-blur-xl sm:p-6">
        <div className="space-y-5">
          <Link href="/maps" className="inline-flex min-h-[38px] items-center gap-2 rounded-full border border-white/10 bg-white/[0.08] px-4 text-[12px] font-extrabold uppercase tracking-[0.08em] text-[#d6e4ed] transition hover:bg-white/[0.12] hover:text-white">
            <ArrowLeft size={16} />
            Back
          </Link>

          <section className="rounded-[18px] border border-white/10 bg-white/[0.055] p-4 backdrop-blur-md sm:p-5">
            <div className="mb-5">
              <span className="text-[11px] font-black uppercase tracking-[0.16em] text-[#77f0be]">Upload Map</span>
              <h2 className="mt-1 text-[28px] font-extrabold leading-tight tracking-tight text-white sm:text-[36px]">Create a Custom Map</h2>
              <p className="mt-2 max-w-2xl text-sm font-semibold leading-6 text-[#a9bfd4]">Choose a JSON file, thumbnail, difficulty, and public details for your GeoDuels map.</p>
            </div>
            {canUploadCustomMaps ? (
              mapUploadForm
            ) : (
              <div className="rounded-[16px] border border-white/10 bg-black/20 p-5 text-sm font-semibold text-[#a9bfd4]">Sign in with a permanent account to create custom maps.</div>
            )}
          </section>
        </div>
      </div>
    </motion.div>
  );

  const mapDetailsPanel = (
    <motion.div key="map-details" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <div className="rounded-[22px] border border-white/10 bg-[#071114]/95 p-5 sm:p-6">
        {selectedMapQuery.isLoading || !selectedMapDetails ? (
          <div className="flex items-center gap-3 text-sm text-[#a9bfd4]">
            <Loader2 className="animate-spin" size={18} /> Loading map details...
          </div>
        ) : (
          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div>
              <Link href="/maps" className="mb-3 inline-flex text-xs font-black uppercase tracking-[0.12em] text-[#77f0be]">Back to maps</Link>
              <img src={thumbnailURL(selectedMapDetails.map)} alt="" className="aspect-[16/9] w-full rounded-[18px] object-cover" />
              <h3 className="mt-4 text-2xl font-black text-white">{selectedMapDetails.map.displayName}</h3>
              <p className="mt-1 text-sm text-[#a9bfd4]">by {selectedMapDetails.map.authorName || "GeoDuels"} · {selectedMapDetails.map.difficulty} · {selectedMapDetails.map.locationCount.toLocaleString()} locations</p>
              {selectedMapDetails.map.description ? <p className="mt-3 text-sm leading-6 text-[#8da6b5]">{selectedMapDetails.map.description}</p> : null}
              <div className="mt-4 grid gap-2 sm:grid-cols-3">
                {selectedMapDetails.countryStats.slice(0, 12).map((stat) => (
                  <div key={stat.country} className="rounded-[12px] border border-white/10 bg-white/[0.04] p-3">
                    <div className="truncate text-xs font-bold text-[#a9bfd4]">{stat.country}</div>
                    <div className="mt-1 h-2 overflow-hidden rounded-full bg-white/10">
                      <div className="h-full rounded-full bg-[#22d385]" style={{ width: `${Math.max(6, Math.round((stat.locationCount / Math.max(1, selectedMapDetails.map.locationCount)) * 100))}%` }} />
                    </div>
                    <div className="mt-1 text-[11px] font-black text-white">{stat.locationCount.toLocaleString()}</div>
                  </div>
                ))}
              </div>
            </div>
            <aside>
              <div className="flex flex-wrap gap-2">
                <button type="button" onClick={() => selectMapForParty(selectedMapDetails.map)} className="rounded-[12px] bg-[#22d385] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-white">Use in Party</button>
                {canInteractWithMaps ? <button type="button" onClick={() => favoriteMapMutation.mutate({ mapId: selectedMapDetails.map.id, favorite: !selectedMapDetails.map.favorited })} className="rounded-[12px] border border-white/10 bg-white/[0.06] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-white">{selectedMapDetails.map.favorited ? "Unfavorite" : "Favorite"}</button> : null}
                {selectedMapDetails.map.ownerUserId === userId && canUploadCustomMaps ? (
                  <>
                    {!selectedMapDetails.map.publishedAt ? <button type="button" onClick={() => publishMapMutation.mutate(selectedMapDetails.map.id)} className="rounded-[12px] border border-[#77f0be]/20 bg-[#77f0be]/10 px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-[#baf7dc]">Publish</button> : null}
                    <label className="inline-flex cursor-pointer items-center rounded-[12px] border border-white/10 bg-white/[0.06] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-white hover:bg-white/[0.1]">
                      <Upload className="mr-1.5" size={14} /> New Version
                      <input type="file" accept=".json,application/json" className="hidden" onChange={(event) => { const file=event.target.files?.[0]; if(file) revisionMutation.mutate({mapId:selectedMapDetails.map.id,file}); event.currentTarget.value=""; }} />
                    </label>
                    <button type="button" onClick={() => deleteMap(selectedMapDetails.map)} className="rounded-[12px] border border-red-400/15 bg-red-400/[0.06] px-4 py-2 text-xs font-black uppercase tracking-[0.08em] text-red-200 hover:bg-red-400/10">Delete</button>
                  </>
                ) : null}
              </div>
              <div className="mt-5 rounded-[18px] border border-white/10 bg-black/20 p-4">
                <h4 className="flex items-center gap-2 text-sm font-black text-white"><MessageCircle size={16} /> Comments</h4>
                {canInteractWithMaps ? (
                  <div className="mt-3">
                    <textarea value={commentBody} onChange={(event)=>setCommentBody(event.target.value)} maxLength={1000} placeholder="Add a comment" className="min-h-20 w-full resize-none rounded-[12px] border border-white/10 bg-black/25 p-3 text-sm text-white outline-none focus:border-[#2ad18f]/60" />
                    <button type="button" disabled={!commentBody.trim() || createCommentMutation.isPending} onClick={postMapComment} className="mt-2 rounded-[10px] bg-[#22d385] px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] text-white disabled:opacity-50">Post</button>
                  </div>
                ) : <p className="mt-3 text-xs text-[#8da6b5]">{accessToken ? "Upgrade your guest profile to comment." : "Sign in to comment."}</p>}
                <div className="mt-4 grid gap-3">
                  {selectedMapDetails.comments.map((comment) => (
                    <div key={comment.id} className="rounded-[14px] border border-white/10 bg-white/[0.04] p-3">
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-sm font-black text-white">
                            {comment.userDisplayName}
                            {commentDeletedLabel(comment.status) ? <span className="ml-2 text-xs font-black text-red-300">{commentDeletedLabel(comment.status)}</span> : null}
                          </p>
                          <p className="mt-1 text-sm leading-5 text-[#a9bfd4]">{comment.body}</p>
                        </div>
                        {comment.status === "visible" && (comment.canDelete || isAdmin || isModerator) && accessToken ? <button type="button" onClick={() => deleteCommentMutation.mutate({ commentId: comment.id })} className="text-red-200 hover:text-red-100"><Trash2 size={15} /></button> : null}
                      </div>
                      {canInteractWithMaps && comment.status === "visible" ? <button type="button" onClick={() => { setReplyToCommentId(comment.id); setReplyBody(""); }} className="mt-2 text-[11px] font-black uppercase tracking-[0.08em] text-[#77f0be]">Reply</button> : null}
                      {comment.replies?.map((reply) => (
                        <div key={reply.id} className="mt-3 border-l border-white/10 pl-3">
                          <div className="flex items-start justify-between gap-3">
                            <p className="text-sm leading-5 text-[#a9bfd4]">
                              <span className="font-black text-white">{reply.userDisplayName}</span>
                              {commentDeletedLabel(reply.status) ? <span className="ml-2 text-xs font-black text-red-300">{commentDeletedLabel(reply.status)}</span> : null}
                              {" "}{reply.body}
                            </p>
                            {reply.status === "visible" && (reply.canDelete || isAdmin || isModerator) && accessToken ? <button type="button" onClick={() => deleteCommentMutation.mutate({ commentId: reply.id })} className="text-red-200 hover:text-red-100"><Trash2 size={14} /></button> : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            </aside>
          </div>
        )}
      </div>
    </motion.div>
  );

  const mapDetailsPanelV2 = (
    <motion.div key="map-details" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <div className="glass-panel rounded-[20px] p-4 sm:p-6">
        {selectedMapQuery.isLoading || !selectedMapDetails ? (
          <div className="flex items-center gap-3 text-sm text-[#a9bfd4]">
            <Loader2 className="animate-spin" size={18} /> Loading map details...
          </div>
        ) : (
          <div className="space-y-5">
            <Link href="/maps" className="inline-flex min-h-[38px] items-center gap-2 rounded-full border border-white/10 bg-white/[0.08] px-4 text-[12px] font-extrabold uppercase tracking-[0.08em] text-[#d6e4ed] transition hover:bg-white/[0.12] hover:text-white">
              <ArrowLeft size={16} />
              Back
            </Link>

            <div className="grid gap-5 lg:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]">
              <section
                className="relative min-h-[280px] overflow-hidden rounded-[18px] border border-white/10 bg-cover bg-center"
                style={{ backgroundImage: `url(${thumbnailURL(selectedMapDetails.map)})` }}
              >
                <div className="absolute inset-0 bg-black/40" />
                <div className="relative flex min-h-[280px] flex-col justify-end p-5 sm:p-6">
                  <div className="max-w-[720px]">
                    <h2 className="text-[28px] font-extrabold leading-tight tracking-tight text-white sm:text-[36px]">
                      {selectedMapDetails.map.displayName}
                    </h2>
                    <div className="mt-3 flex flex-wrap items-center gap-3 text-sm font-bold text-[#d7e5ee]">
                      <span>By {selectedMapDetails.map.authorName || "GeoDuels"}</span>
                      <span className="rounded-full bg-black/25 px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.12em] text-[#ffd166]">
                        {selectedMapDetails.map.difficulty}
                      </span>
                    </div>
                  </div>
                </div>
              </section>

              <section className="rounded-[18px] border border-white/10 bg-white/[0.055] p-4 backdrop-blur-md sm:p-5">
                <div className="grid gap-3">
                  {[
                    { label: "plays", value: selectedMapDetails.map.playCount, icon: <Play size={20} fill="currentColor" /> },
                    { label: "locations", value: selectedMapDetails.map.locationCount, icon: <MapIcon size={20} /> },
                    { label: "favorites", value: selectedMapDetails.map.favoriteCount, icon: <Heart size={20} /> },
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
                  {selectedMapDetails.map.description || "No description has been added for this map yet."}
                </p>
              </section>
            </div>

            <section className="flex flex-col gap-3 rounded-[18px] border border-white/10 bg-white/[0.055] p-4 backdrop-blur-md sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-wrap gap-2">
                <span className="rounded-[12px] bg-white/[0.06] px-3 py-2 text-xs font-black uppercase tracking-[0.08em] text-[#a9bfd4]">Moving</span>
                <span className="rounded-[12px] bg-white/[0.06] px-3 py-2 text-xs font-black uppercase tracking-[0.08em] text-[#a9bfd4]">Infinite Clock</span>
              </div>
              <div className="flex flex-wrap gap-2">
                {mapPickerFlow ? (
                  <button type="button" onClick={() => selectMapForParty(selectedMapDetails.map)} className="inline-flex min-h-[46px] items-center justify-center rounded-[14px] bg-[#22d385] px-6 text-sm font-black uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(34,211,133,0.28)] transition hover:bg-[#2ae091]">
                    <MapIcon className="mr-2" size={18} />
                    Use This Map
                  </button>
                ) : (
                  <button type="button" onClick={() => playMapSingleplayer(selectedMapDetails.map)} disabled={singleplayerDisabled} className="inline-flex min-h-[46px] items-center justify-center rounded-[14px] bg-[#22d385] px-6 text-sm font-black uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(34,211,133,0.3)] transition hover:bg-[#2ae091] disabled:cursor-not-allowed disabled:opacity-60">
                    <Play className="mr-2" size={18} fill="currentColor" />
                    Play
                  </button>
                )}
                {canInteractWithMaps ? (
                  <button type="button" onClick={() => favoriteMapMutation.mutate({ mapId: selectedMapDetails.map.id, favorite: !selectedMapDetails.map.favorited })} className="inline-flex min-h-[46px] items-center justify-center rounded-[14px] border border-white/10 bg-white/[0.06] px-4 text-sm font-black uppercase tracking-[0.08em] text-white transition hover:bg-white/[0.1]">
                    <Star className="mr-2" size={17} fill={selectedMapDetails.map.favorited ? "currentColor" : "none"} />
                    {selectedMapDetails.map.favorited ? "Saved" : "Save"}
                  </button>
                ) : null}
                {selectedMapDetails.map.ownerUserId === userId && canUploadCustomMaps ? (
                  <>
                    {!selectedMapDetails.map.publishedAt ? <button type="button" onClick={() => publishMapMutation.mutate(selectedMapDetails.map.id)} className="rounded-[14px] border border-[#77f0be]/20 bg-[#77f0be]/10 px-4 text-xs font-black uppercase tracking-[0.08em] text-[#baf7dc]">Publish</button> : null}
                    <label className="inline-flex min-h-[46px] cursor-pointer items-center rounded-[14px] border border-white/10 bg-white/[0.06] px-4 text-xs font-black uppercase tracking-[0.08em] text-white hover:bg-white/[0.1]">
                      <Upload className="mr-1.5" size={14} /> New Version
                      <input type="file" accept=".json,application/json" className="hidden" onChange={(event) => { const file=event.target.files?.[0]; if(file) revisionMutation.mutate({mapId:selectedMapDetails.map.id,file}); event.currentTarget.value=""; }} />
                    </label>
                    <button type="button" onClick={() => deleteMap(selectedMapDetails.map)} className="rounded-[14px] border border-red-400/15 bg-red-400/[0.06] px-4 text-xs font-black uppercase tracking-[0.08em] text-red-200 hover:bg-red-400/10">Delete</button>
                  </>
                ) : null}
              </div>
            </section>

            <section className="rounded-[18px] border border-white/10 bg-white/[0.045] p-4 backdrop-blur-md">
              <h4 className="flex items-center gap-2 text-[18px] font-extrabold tracking-tight text-white"><MessageCircle size={18} /> Comments</h4>
              {canInteractWithMaps ? (
                <div className="mt-5 flex gap-3">
                  <AvatarBadge
                    avatarUrl={userAvatar}
                    fallback={userAvatarFallback}
                    alt={displayName || userEmail || "You"}
                    size="sm"
                    className="mt-1 h-10 w-10 shrink-0 border-white/15 bg-[#162130]"
                  />
                  <div className="min-w-0 flex-1">
                    <textarea
                      value={commentBody}
                      onFocus={() => setCommentComposerFocused(true)}
                      onChange={(event)=>setCommentBody(event.target.value)}
                      maxLength={1000}
                      placeholder="Add a comment"
                      rows={commentComposerFocused || commentBody ? 2 : 1}
                      className="min-h-[36px] w-full resize-none border-0 border-b border-white/25 bg-transparent px-0 py-1.5 text-[15px] font-medium text-white outline-none placeholder:text-[#8da6b5] focus:border-[#2ad18f]"
                    />
                    {(commentComposerFocused || commentBody) ? (
                    <div className="mt-3 flex justify-end gap-2">
                      <button type="button" onClick={() => { setCommentBody(""); setCommentComposerFocused(false); }} className="rounded-full px-4 py-2 text-xs font-extrabold uppercase tracking-[0.08em] text-[#a9bfd4] hover:bg-white/[0.08]">Cancel</button>
                      <button type="button" disabled={!commentBody.trim() || createCommentMutation.isPending} onClick={postMapComment} className="rounded-full bg-[#22d385] px-4 py-2 text-xs font-extrabold uppercase tracking-[0.08em] text-white disabled:opacity-50">Comment</button>
                    </div>
                    ) : null}
                  </div>
                </div>
              ) : <p className="mt-3 text-xs text-[#8da6b5]">{accessToken ? "Upgrade your guest profile to comment." : "Sign in to comment."}</p>}
              <div className="mt-7 grid gap-6">
                {selectedMapDetails.comments.map((comment) => (
                  <div key={comment.id} className="flex gap-3">
                    <AvatarBadge
                      avatarUrl={comment.avatarUrl}
                      fallback={commentAvatarFallback(comment.userDisplayName)}
                      alt={comment.userDisplayName}
                      size="sm"
                      className="h-10 w-10 shrink-0 border-white/15 bg-[#162130]"
                    />
	                    <div className="min-w-0 flex-1">
	                      <div className="flex items-start justify-between gap-3">
	                        <div className="min-w-0">
	                          <div className="flex flex-wrap items-baseline gap-2">
	                            <span className="truncate text-[14px] font-extrabold text-white">{comment.userDisplayName}</span>
	                            <time dateTime={comment.createdAt} className="text-[13px] font-bold text-[#a9bfd4]/80">{formatCommentAge(comment.createdAt)}</time>
	                            {commentDeletedLabel(comment.status) ? <span className="text-[13px] font-black text-red-300">{commentDeletedLabel(comment.status)}</span> : null}
	                          </div>
	                          <p className="mt-1 text-[15px] font-medium leading-6 text-[#eef6fb]">{comment.body}</p>
	                        </div>
	                        {comment.status === "visible" && (comment.canDelete || isAdmin || isModerator) && accessToken ? (
	                          <div className="relative shrink-0">
	                            <button type="button" onClick={() => setOpenCommentMenuId(openCommentMenuId === comment.id ? "" : comment.id)} className="flex h-8 w-8 items-center justify-center rounded-full text-[#a9bfd4] hover:bg-white/[0.08] hover:text-white" aria-label="Comment actions">
	                              <MoreVertical size={17} />
	                            </button>
                            {openCommentMenuId === comment.id ? (
                              <div className="absolute right-0 top-9 z-10 w-32 overflow-hidden rounded-[12px] border border-white/10 bg-[#101a20] py-1 shadow-[0_12px_30px_rgba(0,0,0,0.35)]">
                                <button type="button" onClick={() => { setOpenCommentMenuId(""); deleteCommentMutation.mutate({ commentId: comment.id }); }} className="flex w-full items-center gap-2 px-3 py-2 text-left text-xs font-bold text-red-200 hover:bg-red-400/10">
                                  <Trash2 size={14} />
                                  Delete
                                </button>
                              </div>
                            ) : null}
                          </div>
                        ) : null}
                      </div>
                      <div className="mt-3 flex items-center gap-4">
                        {canInteractWithMaps ? (
                          <button type="button" onClick={() => toggleCommentLike(comment.id)} className={`inline-flex items-center gap-2 rounded-full text-[13px] font-bold transition ${likedCommentIds[comment.id] ? "text-[#77f0be]" : "text-[#d6e4ed] hover:text-white"}`}>
                            <Heart size={18} fill={likedCommentIds[comment.id] ? "currentColor" : "none"} />
                            {likedCommentIds[comment.id] ? 1 : 0}
                          </button>
                        ) : null}
                        {canInteractWithMaps && comment.status === "visible" ? <button type="button" onClick={() => { setReplyToCommentId(comment.id); setReplyBody(""); }} className="rounded-full px-2 py-1 text-[13px] font-extrabold text-white hover:bg-white/[0.08]">Reply</button> : null}
                      </div>
                      {replyToCommentId === comment.id ? (
                        <div className="mt-4 flex gap-3">
                          <AvatarBadge
                            avatarUrl={userAvatar}
                            fallback={userAvatarFallback}
                            alt={displayName || userEmail || "You"}
                            size="sm"
                            className="mt-1 h-8 w-8 shrink-0 border-white/15 bg-[#162130]"
                          />
                          <div className="min-w-0 flex-1">
                            <textarea
                              value={replyBody}
                              onChange={(event) => setReplyBody(event.target.value)}
                              maxLength={1000}
                              autoFocus
                              rows={2}
                              placeholder={`Reply to @${comment.userDisplayName}`}
                              className="min-h-[44px] w-full resize-none border-0 border-b border-white/25 bg-transparent px-0 py-1.5 text-[14px] font-medium text-white outline-none placeholder:text-[#8da6b5] focus:border-[#2ad18f]"
                            />
                            <div className="mt-3 flex justify-end gap-2">
                              <button type="button" onClick={() => { setReplyToCommentId(""); setReplyBody(""); }} className="rounded-full px-4 py-2 text-xs font-extrabold uppercase tracking-[0.08em] text-[#a9bfd4] hover:bg-white/[0.08]">Cancel</button>
                              <button type="button" disabled={!replyBody.trim() || createCommentMutation.isPending} onClick={() => postMapReply(comment.id)} className="rounded-full bg-[#22d385] px-4 py-2 text-xs font-extrabold uppercase tracking-[0.08em] text-white disabled:opacity-50">Reply</button>
                            </div>
                          </div>
                        </div>
                      ) : null}
                      {comment.replies?.length ? (
                        <button type="button" onClick={() => toggleCommentReplies(comment.id)} className="mt-3 inline-flex items-center gap-2 rounded-full px-2 py-1 text-[14px] font-extrabold text-[#77f0be] hover:bg-[#2ad18f]/10">
                          <ChevronDown size={18} className={`transition ${expandedCommentIds[comment.id] ? "rotate-180" : ""}`} />
                          {comment.replies.length} {comment.replies.length === 1 ? "reply" : "replies"}
                        </button>
                      ) : null}
                      {expandedCommentIds[comment.id] ? (
                        <div className="mt-4 grid gap-4 border-l border-white/[0.12] pl-4">
                          {comment.replies?.map((reply) => (
                            <div key={reply.id} className="flex gap-3">
                              <AvatarBadge
                                avatarUrl={reply.avatarUrl}
                                fallback={commentAvatarFallback(reply.userDisplayName)}
                                alt={reply.userDisplayName}
                                size="sm"
                                className="h-8 w-8 shrink-0 border-white/15 bg-white/[0.08] text-xs"
                              />
	                              <div className="min-w-0 flex-1">
	                                <div className="flex items-start justify-between gap-3">
	                                  <div className="min-w-0">
	                                    <div className="flex flex-wrap items-baseline gap-2">
	                                      <span className="truncate text-[13px] font-extrabold text-white">@{reply.userDisplayName}</span>
	                                      <time dateTime={reply.createdAt} className="text-[12px] font-bold text-[#a9bfd4]/80">{formatCommentAge(reply.createdAt)}</time>
	                                      {commentDeletedLabel(reply.status) ? <span className="text-[12px] font-black text-red-300">{commentDeletedLabel(reply.status)}</span> : null}
	                                    </div>
	                                    <p className="mt-1 text-[14px] font-medium leading-5 text-[#eef6fb]">{reply.body}</p>
	                                  </div>
	                                  {reply.status === "visible" && (reply.canDelete || isAdmin || isModerator) && accessToken ? (
	                                    <button type="button" onClick={() => deleteCommentMutation.mutate({ commentId: reply.id })} className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-[#a9bfd4] hover:bg-red-400/10 hover:text-red-200" aria-label="Delete reply">
	                                      <Trash2 size={14} />
	                                    </button>
	                                  ) : null}
                                </div>
                                <div className="mt-2 flex items-center gap-4">
                                  {canInteractWithMaps ? (
                                    <button type="button" onClick={() => toggleCommentLike(reply.id)} className={`inline-flex items-center gap-2 rounded-full text-[12px] font-bold transition ${likedCommentIds[reply.id] ? "text-[#77f0be]" : "text-[#d6e4ed] hover:text-white"}`}>
                                      <Heart size={16} fill={likedCommentIds[reply.id] ? "currentColor" : "none"} />
                                      {likedCommentIds[reply.id] ? 1 : 0}
                                    </button>
                                  ) : null}
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </section>
          </div>
        )}
      </div>
    </motion.div>
  );

  const privateLobbyPanel = privateLobbyActive ? (
    <motion.div
      key="private-lobby"
      {...tabPanelMotion}
      className="w-full max-w-[980px] pointer-events-auto"
    >
      <div className="glass-panel overflow-hidden rounded-[24px]">
        <div className="relative min-h-[220px] p-5 sm:p-7">
          <div className="absolute inset-0 pointer-events-none bg-[linear-gradient(180deg,rgba(42,209,143,0.16)_0%,rgba(10,23,26,0.74)_100%)]" />
          <div className="relative z-10 flex flex-col gap-5">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <span className="mb-2 block text-[12px] font-black uppercase tracking-[0.16em] text-[#77f0be]">
                  CUSTOM
                </span>
                <h2 className="text-[34px] font-black leading-tight tracking-tight text-white sm:text-[42px]">
                  Private Party
                </h2>
                <p className="mt-2 max-w-[48ch] text-[14px] leading-6 text-[#a9bfd4]">
                  {lobbyMatchInProgress
                    ? "A game is in progress. Friends can still join the lobby and wait for the next one."
                    : "Share the invite, wait for one opponent, then the leader starts the match."}
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                {privateLobby.inviteCode ? (
                  <button
                    type="button"
                    onClick={copyInvite}
                    className="inline-flex min-h-[42px] items-center justify-center rounded-[12px] border border-white/10 bg-white/[0.08] px-4 text-[12px] font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-white/[0.12]"
                  >
                    <Copy className="mr-2 text-[#77f0be]" size={16} />
                    {inviteCopied ? "Copied" : "Copy Invite"}
                  </button>
                ) : null}
                {privateLobby.isMember && !(privateLobby.isOwner && lobbyMatchInProgress) ? (
                  <button
                    type="button"
                    onClick={() => void leavePrivateLobby()}
                    disabled={privateLobby.busy}
                    className="inline-flex min-h-[42px] items-center justify-center rounded-[12px] border border-white/10 bg-white/[0.08] px-4 text-[12px] font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-white/[0.12] disabled:opacity-50"
                  >
                    <LogOut className="mr-2" size={16} />
                    Leave
                  </button>
                ) : null}
              </div>
            </div>

            {privateLobby.inviteCode ? (
              <div className="rounded-[16px] border border-white/10 bg-black/20 px-4 py-3">
                <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
                  Invite Code
                </p>
                <p className="mt-1 font-mono text-[26px] font-black tracking-[0.18em] text-white">
                  {privateLobby.inviteCode}
                </p>
              </div>
            ) : null}

            {lobbyMatchInProgress ? (
              <div className="rounded-[16px] border border-[#77f0be]/20 bg-[#22d385]/10 px-4 py-4">
                <p className="text-[11px] font-black uppercase tracking-[0.14em] text-[#77f0be]">
                  Game In Progress
                </p>
                <p className="mt-1 text-sm font-semibold text-[#d8f7e9]">
                  {!privateLobby.isMember
                    ? "Join the lobby now and you will be ready for the next game."
                    : currentLobbyMember?.inActiveMatch
                    ? "You are part of this game and can reconnect whenever you are ready."
                    : "You joined after this game started and will be able to play in the next one."}
                </p>
                {currentLobbyMember?.inActiveMatch && activeLobbyMatchId ? (
                  <Link
                    href={`/match/${encodeURIComponent(activeLobbyMatchId)}`}
                    className="mt-3 inline-flex min-h-[42px] items-center justify-center rounded-[12px] bg-[#22d385] px-4 text-[12px] font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-[#2ae091]"
                  >
                    <Play className="mr-2" size={16} fill="currentColor" />
                    Reconnect to Game
                  </Link>
                ) : null}
              </div>
            ) : null}

            {privateLobby.snapshot ? (
              <div className="rounded-[16px] border border-white/10 bg-black/20 p-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  {!(privateLobby.isOwner && privateLobby.snapshot.state === "open") ? (
                    <div>
                      <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
                        Game Settings
                      </p>
                      <p className="mt-1 text-sm font-semibold text-white">
                        {privateLobby.snapshot.mapName || lobbyConfig.mapId || "Official Map"} · {(lobbyConfig.ruleset === "nmpz" ? "NMPZ" : "Moving")} · {lobbyClockOn ? `${lobbyRoundSeconds}s clock` : "Infinite clock"} · {lobbyPressureOn ? `${lobbyPressureSeconds}s pressure` : "No pressure"}
                      </p>
                    </div>
                  ) : null}
                  {privateLobby.isOwner && privateLobby.snapshot.state === "open" ? (
                    <div className="grid w-full gap-3 sm:max-w-[820px] sm:grid-cols-5">
                      <label className="grid gap-1.5">
                        <span className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">
                          Mode
                        </span>
                        <select
                          value={lobbyMode}
                          onChange={(event) => saveLobbyMode(event.target.value as PartyMode)}
                          disabled={privateLobby.busy}
                          className="h-[40px] rounded-[10px] border border-white/10 bg-[#101a20]/90 px-3 text-[13px] font-bold text-white outline-none transition focus:border-[#2ad18f]/60 disabled:opacity-50"
                        >
                          <option value="duel">Duel</option>
                          <option value="team_duel">Team Duel</option>
                          <option value="free_for_all">Free For All</option>
                        </select>
                      </label>
                      <label className="grid gap-1.5">
                        <span className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">
                          Map
                        </span>
                        <button
                          type="button"
                          onClick={() => setMapPickerOpen(true)}
                          disabled={privateLobby.busy || mapsQuery.isLoading}
                          className="inline-flex h-[40px] min-w-0 items-center justify-center rounded-[10px] border border-white/10 bg-[#101a20]/90 px-3 text-[13px] font-bold text-white outline-none transition hover:border-[#2ad18f]/50 hover:bg-white/[0.08] disabled:opacity-50"
                        >
                          <MapIcon className="mr-1.5 shrink-0 text-[#77f0be]" size={14} />
                          <span className="truncate">{lobbyConfig.mapName || readyMaps.find((item) => item.id === lobbyConfig.mapId)?.displayName || "Select map"}</span>
                        </button>
                      </label>
                      <label className="grid gap-1.5">
                        <span className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">
                          Rules
                        </span>
                        <select
                          value={lobbyConfig.ruleset || "moving"}
                          onChange={(event) => saveLobbyConfig({ ruleset: event.target.value as GameRuleset })}
                          disabled={privateLobby.busy}
                          className="h-[40px] rounded-[10px] border border-white/10 bg-[#101a20]/90 px-3 text-[13px] font-bold text-white outline-none transition focus:border-[#2ad18f]/60 disabled:opacity-50"
                        >
                          <option value="moving">Moving</option>
                          <option value="nmpz">NMPZ</option>
                        </select>
                      </label>
                      <label className="grid gap-1.5">
                        <span className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">
                          Clock
                        </span>
                        <select
                          value={lobbyClockOn ? String(lobbyRoundSeconds) : "infinite"}
                          onChange={(event) => {
                            const value = event.target.value;
                            saveLobbyConfig(
                              value === "infinite"
                                ? { roundTimerMode: "none", roundTimeLimitMs: undefined }
                                : { roundTimerMode: "fixed", roundTimeLimitMs: Number(value) * 1000 },
                            );
                          }}
                          disabled={privateLobby.busy}
                          className="h-[40px] rounded-[10px] border border-white/10 bg-[#101a20]/90 px-3 text-[13px] font-bold text-white outline-none transition focus:border-[#2ad18f]/60 disabled:opacity-50"
                        >
                          {CLOCK_OPTIONS.map((option) => (
                            <option key={option.value} value={option.value}>{option.label}</option>
                          ))}
                        </select>
                      </label>
                      <label className="grid gap-1.5">
                        <span className="text-[10px] font-black uppercase tracking-[0.14em] text-[#6b8b80]">
                          Pressure
                        </span>
                        <select
                          value={lobbyPressureOn ? "15" : "none"}
                          onChange={(event) => {
                            const value = event.target.value;
                            saveLobbyConfig({
                              pressureTimeLimitMs: value === "15" ? 15000 : undefined,
                            });
                          }}
                          disabled={privateLobby.busy}
                          className="h-[40px] rounded-[10px] border border-white/10 bg-[#101a20]/90 px-3 text-[13px] font-bold text-white outline-none transition focus:border-[#2ad18f]/60 disabled:opacity-50"
                        >
                          {PRESSURE_OPTIONS.map((option) => (
                            <option key={option.value} value={option.value}>{option.label}</option>
                          ))}
                        </select>
                      </label>
                    </div>
                  ) : null}
                </div>
              </div>
            ) : null}

            {privateLobbyLoading ? (
              <div className="rounded-[18px] border border-white/10 bg-black/20 p-4">
                <div className="flex min-h-[64px] items-center justify-center text-sm font-semibold text-[#a9bfd4]">
                  <Loader2 className="mr-2 animate-spin text-[#77f0be]" size={18} />
                  Connecting to lobby
                </div>
              </div>
            ) : !privateLobby.isMember ? (
              <div className="rounded-[18px] border border-white/10 bg-black/20 p-4">
                <button
                  type="button"
                  onClick={() => void joinInviteLobby()}
                  disabled={privateLobby.busy || authLoading}
                  className="inline-flex min-h-[46px] w-full items-center justify-center rounded-[14px] bg-[#22d385] px-5 text-[14px] font-extrabold uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(34,211,133,0.3)] transition hover:bg-[#2ae091] disabled:opacity-60"
                >
                  {privateLobby.busy ? (
                    <Loader2 className="mr-2 animate-spin" size={18} />
                  ) : (
                    <UserPlus className="mr-2" size={18} />
                  )}
                  Join Party
                </button>
                {authError ? (
                  <p className="mt-3 text-center text-xs font-semibold text-red-300">
                    {authError}
                  </p>
                ) : null}
              </div>
            ) : null}

            {privateLobby.snapshot ? (
              <div className="grid gap-3">
                {lobbyMembers.map((member) => {
                  const isLeader = member.userId === privateLobby.snapshot?.ownerUserId;
                  const isSelf = member.userId === userId;
                  const presenceStatus = member.presenceStatus || (member.connected ? "online" : "offline");
                  const lobbyStatus =
                    presenceStatus === "online"
                      ? "Online"
                      : presenceStatus === "away"
                        ? "Reconnecting"
                        : "Offline";
                  return (
                    <div
                      key={member.userId}
                      className={`flex min-h-[72px] flex-col gap-3 rounded-[16px] border border-white/10 bg-white/[0.06] px-4 py-3 sm:flex-row sm:items-center sm:justify-between ${presenceStatus === "offline" ? 'opacity-50' : ''}`}
                    >
	                      <div className="min-w-0">
	                        <div className="flex items-center gap-2">
	                          <p className="truncate text-[16px] font-extrabold text-white">
	                            {member.displayName || member.userId}
	                          </p>
                          {isLeader ? (
                            <span className="inline-flex items-center rounded-full bg-[#22d385]/15 px-2 py-0.5 text-[10px] font-black uppercase tracking-[0.1em] text-[#77f0be]">
                              <Crown className="mr-1" size={12} />
                              Leader
                            </span>
	                          ) : null}
	                        </div>
	                        <p className="mt-1 text-[12px] font-semibold text-[#a9bfd4]">
		                          {isSelf ? `You · ${lobbyStatus}` : lobbyStatus}
	                        </p>
	                        {lobbyMode === "team_duel" ? (
	                          <p className="mt-1 text-[12px] font-semibold uppercase tracking-[0.12em]">
                            <span className={lobbyTeamTextClass(member.teamId)}>
                              {lobbyTeamLabel(member.teamId)}
                            </span>
                          </p>
                        ) : null}
                      </div>
                      {lobbyMode === "team_duel" && isSelf && privateLobby.snapshot?.state === "open" ? (
                        <div className="flex gap-2">
                          {(["a", "b"] as const).map((teamId) => (
                            <button
                              key={teamId}
                              type="button"
                              onClick={() => void switchPrivateLobbyTeam(teamId)}
                              disabled={privateLobby.busy || (member.teamId || "a") === teamId}
                              className={`inline-flex min-h-[36px] items-center rounded-[10px] px-3 text-[11px] font-extrabold uppercase tracking-[0.08em] transition disabled:opacity-50 ${lobbyTeamPillClass(teamId, (member.teamId || "a") === teamId)}`}
                            >
                              {lobbyTeamLabel(teamId)}
                            </button>
                          ))}
                        </div>
                      ) : null}
                      {privateLobby.isOwner && privateLobby.snapshot?.state === "open" && !isSelf ? (
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => void transferLobbyOwner(member.userId)}
                            disabled={privateLobby.busy}
                            className="inline-flex min-h-[36px] items-center rounded-[10px] border border-white/10 bg-white/[0.08] px-3 text-[11px] font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-white/[0.12] disabled:opacity-50"
                          >
                            <Crown className="mr-1.5" size={14} />
                            Make Leader
                          </button>
                          <button
                            type="button"
                            onClick={() => void kickLobbyMember(member.userId)}
                            disabled={privateLobby.busy}
                            className="inline-flex min-h-[36px] items-center rounded-[10px] border border-red-300/20 bg-red-400/10 px-3 text-[11px] font-extrabold uppercase tracking-[0.08em] text-red-100 transition hover:bg-red-400/15 disabled:opacity-50"
                          >
                            <UserMinus className="mr-1.5" size={14} />
                            Kick
                          </button>
                        </div>
                      ) : null}
                    </div>
                  );
                })}
              </div>
            ) : null}

            {privateLobby.isOwner && privateLobby.snapshot?.state === "open" ? (
              <button
                type="button"
                onClick={() => void startPrivateLobby()}
                disabled={!canStartPrivateParty || privateLobby.busy}
                className="inline-flex min-h-[52px] w-full items-center justify-center rounded-[16px] bg-[#22d385] px-5 text-[15px] font-extrabold uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(34,211,133,0.3)] transition hover:bg-[#2ae091] disabled:cursor-not-allowed disabled:opacity-50"
              >
                {privateLobby.busy ? (
                  <Loader2 className="mr-2 animate-spin" size={18} />
                ) : (
                  <Play className="mr-2" size={18} fill="currentColor" />
                )}
                {lobbyMode === "team_duel" ? "Start Team Duel" : lobbyMode === "free_for_all" ? "Start FFA" : "Start Duel"}
              </button>
            ) : privateLobby.isMember && privateLobby.snapshot?.state === "open" ? (
              <div className="rounded-[16px] border border-white/10 bg-white/[0.06] px-4 py-3 text-center text-sm font-semibold text-[#a9bfd4]">
                Waiting for the leader to start.
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </motion.div>
  ) : null;

  const maintenanceBanner = maintenanceIsWarning ? (
    <motion.div
      initial={{ opacity: 0, y: -14 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      transition={{ duration: 0.22, ease: "easeOut" }}
      className="mb-4 rounded-[24px] border border-[#f3cf68]/40 bg-[linear-gradient(135deg,rgba(242,197,67,0.22),rgba(115,75,0,0.28))] px-5 py-4 text-[#fff6d8] shadow-[0_12px_40px_rgba(91,63,7,0.24)] backdrop-blur-sm"
    >
      <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-[11px] font-black uppercase tracking-[0.18em] text-[#ffe69a]">
            Maintenance
          </p>
          <p className="mt-1 text-[15px] font-semibold text-white">
            {maintenanceMessage || "Queueing has been paused."}
          </p>
        </div>
        <p className="text-[15px] font-semibold text-[#ffefb5]">
          {warningCountdown ? `${warningCountdown}` : "Soon"}
        </p>
      </div>
    </motion.div>
  ) : null;

  const maintenanceOverlay = maintenanceIsActive ? (
    <AppModalShell
      title="Maintenance Break"
      placement="center"
      showHeader={false}
      zIndexClassName="z-[2100]"
      maxWidthClassName="max-w-[560px]"
      panelClassName="p-7 sm:p-10"
    >
      <div className="flex flex-col items-center text-center">
        <div className="mb-5 flex h-16 w-16 items-center justify-center rounded-full border border-[#f4c84c]/30 bg-[#f4c84c]/10">
          <Loader2 size={30} className="animate-spin text-[#f4c84c]" />
        </div>
        <p className="text-[11px] font-black uppercase tracking-[0.22em] text-[#f4d98a]">
          Maintenance Break
        </p>
        <h2 className="mt-3 text-[30px] font-black tracking-tight text-white sm:text-[38px]">
          We&apos;ll Be Back Shortly
        </h2>
        <p className="mt-3 max-w-[42ch] text-[15px] leading-relaxed text-[#d9e7f5]">
          {maintenanceMessage ||
            "GeoDuels is temporarily offline while we finish a scheduled upgrade."}
        </p>
        <div className="mt-6 rounded-[20px] border border-white/10 bg-white/5 px-5 py-4">
          <p className="text-[11px] font-bold uppercase tracking-[0.16em] text-[#a9bfd4]">
            Approximate Time
          </p>
          <p className="mt-2 text-[18px] font-extrabold text-white">
            {activeEta || "A few minutes"}
          </p>
        </div>
      </div>
    </AppModalShell>
  ) : null;

  const legalCard = (
    <div className="pointer-events-auto flex w-full items-center justify-center px-1 py-1">
      <div className="flex items-center gap-6">
        <Link
          href="/changelog"
          className="text-[12px] font-semibold text-[#6b8b80] transition-colors hover:text-white"
        >
          Changelog
        </Link>
        <div className="h-1 w-1 rounded-full bg-[#6b8b80]/40" />
        <Link
          href="/privacy"
          className="text-[12px] font-semibold text-[#6b8b80] transition-colors hover:text-white"
        >
          Privacy Policy
        </Link>
        <div className="h-1 w-1 rounded-full bg-[#6b8b80]/40" />
        <Link
          href="/terms"
          className="text-[12px] font-semibold text-[#6b8b80] transition-colors hover:text-white"
        >
          Terms of Service
        </Link>
        <div className="h-1 w-1 rounded-full bg-[#6b8b80]/40" />
        <span className="text-[12px] font-semibold text-[#6b8b80]">
          {appVersion}
        </span>
      </div>
    </div>
  );

  const leaderboardPanel = (
    <div className="glass-panel flex w-full max-w-[980px] flex-col gap-5 rounded-[24px] p-5 sm:p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <span className="mb-1 block text-[12px] font-bold uppercase tracking-[0.16em] text-[#2ad18f]">
            Season Ladder
          </span>
          <h2 className="text-[28px] font-extrabold tracking-tight text-white">
            Leaderboard
          </h2>
          <p className="mt-2 text-[14px] text-[#a9bfd4]">
            {leaderboardLoading
              ? "Loading ranked players..."
              : leaderboard?.totalPlayers
                ? `${leaderboard.totalPlayers} ranked players`
                : "No ranked players yet."}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-3 sm:min-w-[240px]">
          <div className="rounded-2xl bg-black/30 p-4">
            <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
              Your Rank
            </p>
            <p className="mt-2 text-3xl font-black text-white">
              {leaderboard?.selfRank ? `#${leaderboard.selfRank}` : "--"}
            </p>
          </div>
          <div className="rounded-2xl bg-black/30 p-4">
            <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
              Rating
            </p>
            <p className="mt-2 text-3xl font-black text-white">{mmr}</p>
          </div>
        </div>
      </div>

      <div className="overflow-hidden rounded-[20px]">
        <div className="grid grid-cols-[72px_minmax(0,1fr)_90px] gap-3 border-b border-white/[0.06] px-4 py-3 text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80] sm:grid-cols-[72px_minmax(0,1fr)_110px_110px]">
          <span>Rank</span>
          <span>Player</span>
          <span className="text-right">MMR</span>
          <span className="hidden text-right sm:block">Win Rate</span>
        </div>
        <div className="divide-y divide-white/[0.06]">
          {(leaderboard?.entries || []).map((entry) => {
            const isSelf = entry.userId === userId;
            const winsValue =
              entry.gamesPlayed > 0
                ? Math.round((entry.wins / entry.gamesPlayed) * 100)
                : 0;
            return (
              <div
                key={`${entry.rank}-${entry.userId}`}
                className={`grid grid-cols-[72px_minmax(0,1fr)_90px] gap-3 px-4 py-3 text-sm sm:grid-cols-[72px_minmax(0,1fr)_110px_110px] ${isSelf ? "bg-[#18382e]/70" : "bg-transparent"}`}
              >
                <div className="flex items-center">
                  <span
                    className={`inline-flex min-w-[48px] items-center justify-center rounded-full px-3 py-1 text-[12px] font-black ${entry.rank <= 3 ? "bg-[#2ad18f]/16 text-[#77f0be]" : "bg-white/[0.05] text-white"}`}
                  >
                    #{entry.rank}
                  </span>
                </div>
                <div className="min-w-0">
                  <p className="truncate font-bold text-white">
                    {entry.displayName || entry.userId}
                  </p>
                  <p className="truncate text-[12px] text-[#8caab0]">
                    {isSelf ? "You" : `${entry.gamesPlayed} games`}
                  </p>
                </div>
                <div className="flex items-center justify-end font-black text-white">
                  {entry.mmr}
                </div>
                <div className="hidden items-center justify-end text-[#a9bfd4] sm:flex">
                  {winsValue}%
                </div>
              </div>
            );
          })}
          {leaderboardLoading ? (
            <div className="px-4 py-10 text-center text-[14px] text-[#a9bfd4]">
              Loading leaderboard...
            </div>
          ) : !leaderboard || leaderboard.entries.length === 0 ? (
            <div className="px-4 py-10 text-center text-[14px] text-[#a9bfd4]">
              No ranked players yet.
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );

  // Modal renderers inside LobbyScreen
  const renderHelpModal = () => (
    <AppModalShell title="Help" onClose={() => setOpenModal(null)}>
      <div className="space-y-5 text-[15px] leading-relaxed text-[#a9bfd4]">
        <div className="glass-panel rounded-xl p-4">
          <h3 className="mb-2 font-bold text-white tracking-wide">
            1. Rules of the Game
          </h3>
          <p>
            You and your opponent will be dropped into the same random street
            view location somewhere in the world. Your goal is to figure out
            where you are and place your guess on the map.
          </p>
        </div>
        <div className="glass-panel rounded-xl p-4">
          <h3 className="mb-2 font-bold text-white tracking-wide">
            2. How to Join
          </h3>
          <p>
            Click "PLAY" on the main menu to enter the matchmaking queue. We'll
            automatically find you an opponent with a similar skill rating
            (MMR).
          </p>
        </div>
        <div className="glass-panel rounded-xl p-4">
          <h3 className="mb-2 font-bold text-white tracking-wide">
            3. How Duels Work
          </h3>
          <p>
            Both players start with 6,000 HP. The first person to guess starts a
            countdown timer. When the round ends, whoever is closer to the
            actual location deals damage to the other player based on the
            distance difference. The game ends when a player's HP hits 0!
          </p>
        </div>
      </div>
    </AppModalShell>
  );

  const renderInviteLobbyModal = () => {
    const normalizedInviteCode = inviteCodeInput.trim().toUpperCase();
    const inviteActionsDisabled =
      privateLobby.busy || authLoading || maintenanceIsActive;

    return (
      <AppModalShell title="Private Party" onClose={() => setOpenModal(null)}>
        <div className="space-y-4">
          <button
            type="button"
            onClick={() => {
              void (async () => {
                if (await createInviteLobby()) {
                  setOpenModal(null);
                }
              })();
            }}
            disabled={inviteActionsDisabled || playPaused}
            className="inline-flex min-h-[48px] w-full items-center justify-center rounded-[14px] bg-[#22d385] px-5 text-[14px] font-extrabold uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(34,211,133,0.3)] transition hover:bg-[#2ae091] disabled:cursor-not-allowed disabled:opacity-60"
          >
            {privateLobby.busy ? (
              <Loader2 className="mr-2 animate-spin" size={18} />
            ) : (
              <UserPlus className="mr-2" size={18} />
            )}
            Create Party
          </button>
          {authError ? (
            <p className="text-center text-xs font-semibold text-red-300">
              {authError}
            </p>
          ) : null}

          <div className="rounded-[18px] border border-white/10 bg-black/20 p-4">
            <label
              htmlFor="invite-code-input"
              className="mb-2 block text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]"
            >
              Join With Code
            </label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                id="invite-code-input"
                value={inviteCodeInput}
                onChange={(event) =>
                  setInviteCodeInput(event.target.value.toUpperCase())
                }
                onKeyDown={(event) => {
                  if (event.key !== "Enter" || !normalizedInviteCode) return;
                  void (async () => {
                    if (await joinInviteLobby(normalizedInviteCode)) {
                      setOpenModal(null);
                    }
                  })();
                }}
                disabled={inviteActionsDisabled}
                className="min-h-[46px] min-w-0 flex-1 rounded-[14px] border border-white/10 bg-[#101a20]/80 px-4 font-mono text-[15px] font-black uppercase tracking-[0.16em] text-white outline-none transition focus:border-[#2ad18f]/60 disabled:opacity-60"
                placeholder="CODE"
                maxLength={16}
                autoComplete="off"
              />
              <button
                type="button"
                onClick={() => {
                  void (async () => {
                    if (await joinInviteLobby(normalizedInviteCode)) {
                      setOpenModal(null);
                    }
                  })();
                }}
                disabled={inviteActionsDisabled || !normalizedInviteCode}
                className="inline-flex min-h-[46px] items-center justify-center rounded-[14px] border border-white/10 bg-white/[0.08] px-5 text-[12px] font-extrabold uppercase tracking-[0.08em] text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
              >
                Join
              </button>
            </div>
          </div>

        </div>
      </AppModalShell>
    );
  };

  const renderSignInModal = () => (
    <AppModalShell
      title="Sign In"
      onClose={() => setOpenModal(null)}
      placement="center"
    >
      <div className="space-y-3">
        {googleProviderButton ? (
          <div
            onClick={() => setOpenModal(null)}
            className="flex justify-center"
          >
            {googleProviderButton}
          </div>
        ) : null}
        {discordProviderButton ? (
          <div
            onClick={() => setOpenModal(null)}
            className="flex justify-center"
          >
            {discordProviderButton}
          </div>
        ) : null}
        {!googleProviderButton && !discordProviderButton ? (
          <div className="flex justify-center">{signInButton}</div>
        ) : null}
        {authError ? (
          <p className="text-center text-xs font-semibold text-red-300">
            {authError}
          </p>
        ) : null}
      </div>
    </AppModalShell>
  );

  const renderProfileModal = () => (
    <AppModalShell
      title="Profile"
      onClose={() => {
        setOpenModal(null);
        setIsEditingProfileName(false);
      }}
    >
      <div className="glass-panel flex items-center gap-4 rounded-2xl p-5">
        <AvatarBadge
          avatarUrl={userAvatar}
          fallback={userAvatarFallback}
          alt={displayName || userEmail || "Guest"}
          size="lg"
          className="border-white/20 bg-[#162130]"
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {isEditingProfileName && !isGuest ? (
              <input
                value={nicknameInput}
                onChange={(e) => onChangeNickname(e.target.value)}
                disabled={nicknameSaving}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    void (async () => {
                      const saved = await onSaveNickname();
                      if (saved) {
                        setIsEditingProfileName(false);
                      }
                    })();
                  }
                  if (e.key === "Escape") {
                    setIsEditingProfileName(false);
                    onChangeNickname(displayName || userEmail || "");
                  }
                }}
                className="min-w-0 flex-1 rounded-lg border border-white/10 bg-[#101a20] px-3 py-2 text-base font-bold text-white outline-none transition focus:border-[#2ad18f]/60"
                placeholder="Enter nickname"
                maxLength={14}
                autoFocus
              />
            ) : (
              <PlayerNameWithBadge
                name={displayName || userEmail || "Guest"}
                isAdmin={isAdmin}
                selectedBadge={null}
                nameClassName="text-xl font-bold text-white"
                wrapperClassName="min-w-0"
              />
            )}
            {userId && !isGuest ? (
              <button
                type="button"
                onClick={() => {
                  if (isEditingProfileName) {
                    void (async () => {
                      const saved = await onSaveNickname();
                      if (saved) {
                        setIsEditingProfileName(false);
                      }
                    })();
                    return;
                  }
                  onChangeNickname(displayName || userEmail || "");
                  setIsEditingProfileName(true);
                }}
                disabled={nicknameSaving}
                className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white/70 transition hover:bg-white/10 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                aria-label={
                  isEditingProfileName ? "Save nickname" : "Edit nickname"
                }
              >
                {nicknameSaving ? (
                  <Loader2 size={16} className="animate-spin" />
                ) : isEditingProfileName ? (
                  <Check size={16} />
                ) : (
                  <Pencil size={16} />
                )}
              </button>
            ) : null}
          </div>
          {isGuest || selectedBadge ? (
            <div className="mt-1 flex items-center gap-2 text-sm text-[#a9bfd4]">
              {isGuest ? <span>Guest profile</span> : null}
              <PlayerBadge badge={selectedBadge} size="sm" />
            </div>
          ) : null}
          {nicknameError ? (
            <p className="mt-2 text-xs font-semibold text-red-400">
              {nicknameError}
            </p>
          ) : null}
        </div>
      </div>

      <div className="mt-4 grid grid-cols-3 gap-2 rounded-2xl border border-white/10 bg-black/20 p-1">
        {profileTabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setProfileTab(tab.id)}
            className={`min-h-[38px] rounded-xl text-[11px] font-black uppercase tracking-[0.12em] transition ${profileTab === tab.id ? "bg-[#2ad18f] text-[#06130d]" : "text-[#a9bfd4] hover:bg-white/10 hover:text-white"}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {profileTab === "stats" ? (
        <div className="mt-4 grid grid-cols-1 gap-3 text-center uppercase tracking-wider text-[#a9bfd4] sm:grid-cols-3">
          <div className="glass-panel rounded-xl p-3 py-4">
            <p className="text-[11px] font-bold">MMR</p>
            <p className="mt-1.5 text-2xl font-black text-white">{mmr}</p>
          </div>
          <div className="glass-panel rounded-xl p-3 py-4">
            <p className="text-[11px] font-bold">Games</p>
            <p className="mt-1.5 text-2xl font-black text-white">{gamesPlayed}</p>
          </div>
          <div className="glass-panel rounded-xl p-3 py-4">
            <p className="text-[11px] font-bold">Winrate</p>
            <p className="mt-1.5 text-2xl font-black text-white">{winsPct}%</p>
          </div>
        </div>
      ) : null}

      {profileTab === "badges" ? (
        <div className="mt-4">
          <div className="grid grid-cols-4 gap-3 rounded-2xl border border-white/10 bg-black/20 p-4 sm:grid-cols-6">
            {badges.map((badge) => {
              const owned = !!badge.owned;
              const selected = selectedBadge?.id === badge.id;
              const focused = focusedBadge?.id === badge.id;
              return (
                <button
                  key={badge.id}
                  type="button"
                  onMouseEnter={() => setHoveredBadgeId(badge.id)}
                  onMouseLeave={() => setHoveredBadgeId("")}
                  onFocus={() => setHoveredBadgeId(badge.id)}
                  onBlur={() => setHoveredBadgeId("")}
                  onClick={() => {
                    setInspectedBadgeId(badge.id);
                    if (owned) void onSelectBadge(selected ? "" : badge.id);
                  }}
                  className={`relative flex aspect-square items-center justify-center rounded-2xl border transition ${focused ? "border-[#2ad18f]/70 bg-[#123f2d]/45" : "border-white/10 bg-white/[0.04] hover:bg-white/[0.08]"} ${selected ? "shadow-[0_0_24px_rgba(42,209,143,0.24)]" : ""}`}
                  aria-label={badge.label}
                >
                  <PlayerBadge badge={badge} size="lg" muted={!owned} />
                  {selected ? (
                    <span className="absolute right-1.5 top-1.5 h-2.5 w-2.5 rounded-full bg-[#2ad18f] shadow-[0_0_10px_rgba(42,209,143,0.8)]" />
                  ) : null}
                  {!owned ? (
                    <span className="absolute inset-x-2 bottom-1.5 rounded-full bg-black/40 py-0.5 text-[8px] font-black uppercase tracking-[0.12em] text-white/45">
                      Locked
                    </span>
                  ) : null}
                </button>
              );
            })}
          </div>
          {focusedBadge ? (
            <div className="mt-3 h-[112px] rounded-2xl border border-white/10 bg-black/25 p-4">
              <div className="flex h-full items-start justify-between gap-3 overflow-hidden">
                <div className="min-w-0">
                  <p className="text-sm font-black text-white">{focusedBadge.label}</p>
                  <p className="mt-1 max-h-[48px] overflow-hidden text-xs leading-relaxed text-[#a9bfd4]">
                    {focusedBadge.description}
                  </p>
                </div>
                <span className={`shrink-0 rounded-full px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.12em] ${focusedBadge.owned ? "bg-[#2ad18f]/18 text-[#8ff0c2]" : "bg-white/[0.06] text-white/45"}`}>
                  {selectedBadge?.id === focusedBadge.id
                    ? "Shown"
                    : focusedBadge.owned
                      ? "Available"
                      : "Locked"}
                </span>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}

      {profileTab === "account" && userId && !isGuest ? (
        <div className="glass-panel mt-6 rounded-xl p-4">
          <div className="mb-4 rounded-xl border border-white/10 bg-black/15 p-3">
            <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">
              Email
            </p>
            <p className="mt-1 truncate text-sm font-semibold text-white">
              {userEmail || "No email on this account"}
            </p>
          </div>
          <p className="mb-3 text-center text-[12px] font-semibold uppercase tracking-[0.14em] text-[#8cb0a1]">
            Sign-in Methods
          </p>
          <div className="space-y-3">
            {showGoogleButton ? (
              <div className="flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-black/15 p-3">
                <div>
                  <p className="text-sm font-bold text-white">Google</p>
                  <p className="text-xs text-[#a9bfd4]">
                    {hasGoogleProvider ? "Linked" : "Not linked"}
                  </p>
                </div>
                {hasGoogleProvider ? (
                  <button
                    type="button"
                    onClick={() => onUnlinkAuthProvider("google")}
                    disabled={authLoading || linkedProviderCount <= 1}
                    className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Unlink
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => onLinkAuthProvider("google")}
                    disabled={authLoading}
                    className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Link
                  </button>
                )}
              </div>
            ) : null}
            {showDiscordButton ? (
              <div className="flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-black/15 p-3">
                <div>
                  <p className="text-sm font-bold text-white">Discord</p>
                  <p className="text-xs text-[#a9bfd4]">
                    {hasDiscordProvider ? "Linked" : "Not linked"}
                  </p>
                </div>
                {hasDiscordProvider ? (
                  <button
                    type="button"
                    onClick={() => onUnlinkAuthProvider("discord")}
                    disabled={authLoading || linkedProviderCount <= 1}
                    className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Unlink
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => onLinkAuthProvider("discord")}
                    disabled={authLoading}
                    className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Link
                  </button>
                )}
              </div>
            ) : null}
          </div>
          {authError ? (
            <p className="mt-3 text-center text-xs font-semibold text-red-300">
              {authError}
            </p>
          ) : null}
        </div>
      ) : null}
      {profileTab === "account" && userId && isGuest ? (
        <div className="glass-panel mt-6 rounded-xl p-4">
          <p className="mb-3 text-center text-[12px] font-semibold uppercase tracking-[0.14em] text-[#8cb0a1]">
            Save Progress
          </p>
          <div className="flex flex-wrap justify-center gap-3">
            {showGoogleButton ? (
              <button
                type="button"
                onClick={() => onUpgradeGuestWithProvider("google")}
                disabled={authLoading}
                className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
              >
                Google
              </button>
            ) : null}
            {showDiscordButton ? (
              <button
                type="button"
                onClick={() => onUpgradeGuestWithProvider("discord")}
                disabled={authLoading}
                className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
              >
                Discord
              </button>
            ) : null}
          </div>
        </div>
      ) : null}

      {profileTab === "account" && userId ? (
        <div className="mt-6 rounded-xl border border-red-500/25 bg-red-950/20 p-4">
          <p className="text-center text-[12px] font-semibold uppercase tracking-[0.14em] text-red-200">
            Delete Account
          </p>
          <p className="mt-2 text-center text-xs leading-relaxed text-red-100/70">
            This signs you out, removes sign-in links, and clears your profile.
            Match and moderation records are retained.
          </p>
          <input
            value={deleteConfirmation}
            onChange={(event) => setDeleteConfirmation(event.target.value)}
            placeholder="Type DELETE"
            className="mt-4 w-full rounded-xl border border-red-300/20 bg-black/25 px-3 py-2 text-center text-sm font-bold tracking-[0.18em] text-white outline-none transition placeholder:text-red-100/35 focus:border-red-300/50"
          />
          <button
            type="button"
            onClick={async () => {
              if (deleteConfirmation !== "DELETE") return;
              try {
                await onDeleteAccount();
                setOpenModal(null);
              } catch {
                // The model surfaces the failure in the profile modal.
              }
            }}
            disabled={authLoading || deleteConfirmation !== "DELETE"}
            className="mt-3 flex w-full items-center justify-center gap-2 rounded-xl border border-red-500/40 bg-red-500/15 py-3 text-[13px] font-bold uppercase tracking-wider text-red-100 transition hover:bg-red-500/25 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {authLoading ? (
              <Loader2 size={16} className="animate-spin" />
            ) : (
              <Trash2 size={16} />
            )}
            Delete Account
          </button>
        </div>
      ) : null}

      {profileTab === "account" && userId ? (
        <button
          type="button"
          onClick={() => {
            setOpenModal(null);
            onLogout();
          }}
          className="mt-6 w-full rounded-xl border border-red-500/30 bg-red-500/10 py-3 text-[14px] font-bold uppercase tracking-wider text-red-400 transition hover:bg-red-500/20"
        >
          Sign Out
        </button>
      ) : null}
    </AppModalShell>
  );

  const inviteLobbyCard = (
    <button
      type="button"
      onClick={() => setOpenModal("invite")}
      disabled={authLoading || nicknameSaving || playPaused || maintenanceIsActive}
      className="glass-panel glass-panel-interactive lobby-feature-card group flex w-full items-center gap-4 rounded-[20px] p-5 text-left transition disabled:cursor-not-allowed disabled:opacity-60"
    >
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-[#2ad18f]/14 text-[#77f0be]">
        <UserPlus size={22} />
      </div>
      <div className="min-w-0 flex-1">
        <span className="mb-1 block text-[12px] font-bold uppercase tracking-[0.16em] text-[#6b8b80]">
          CUSTOM
        </span>
        <h3 className="text-[18px] font-extrabold tracking-tight text-white">
          Private Party
        </h3>
        <p className="mt-1 text-[13px] leading-relaxed text-[#a9bfd4]">
          Create a lobby or join your friend
        </p>
      </div>
      <ArrowUpRight
        size={18}
        className="shrink-0 text-white/50 transition-colors group-hover:text-white"
      />
    </button>
  );

  const privateLobbyErrorNotice = privateLobby.error ? (
    <div
      role="alert"
      className="mb-4 flex w-full max-w-[1160px] items-start gap-3 rounded-[18px] border border-red-300/20 bg-red-500/10 px-4 py-3 text-left text-sm font-semibold leading-6 text-red-100 shadow-[0_14px_40px_rgba(0,0,0,0.22)] pointer-events-auto sm:px-5"
    >
      <Shield className="mt-0.5 shrink-0 text-red-200" size={18} />
      <span>{privateLobby.error}</span>
    </div>
  ) : null;
  const mapPickerModal = mapPickerOpen ? (
    <AppModalShell
      title="Select Map"
      onClose={() => setMapPickerOpen(false)}
      placement="center"
      maxWidthClassName="max-w-[1040px]"
      panelClassName="p-4 sm:p-5"
      contentClassName="space-y-4"
    >
      <div className="grid gap-4 lg:grid-cols-[190px_minmax(0,1fr)]">
        <aside className="grid gap-2 rounded-[18px] border border-white/10 bg-black/20 p-3 lg:content-start">
          {mapScopeLabels.map((item) => (
            <button
              key={item.scope}
              type="button"
              onClick={() => setMapScope(item.scope)}
              className={`rounded-[12px] px-3 py-2.5 text-left text-xs font-black uppercase tracking-[0.08em] transition ${mapScope === item.scope ? "bg-[#22d385] text-white" : "bg-white/[0.05] text-[#a9bfd4] hover:bg-white/[0.09]"}`}
            >
              {item.label}
            </button>
          ))}
        </aside>
        <section className="min-w-0">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-[11px] font-black uppercase tracking-[0.16em] text-[#77f0be]">Party Map</p>
              <h3 className="mt-1 text-2xl font-black text-white">{mapScopeLabels.find((item) => item.scope === mapScope)?.label}</h3>
            </div>
            <div className="flex w-full flex-col gap-3 sm:w-auto sm:items-end">
              {renderMapSearchControl("party-map-search")}
              {mapScope === "community" ? (
                <div className="flex rounded-[14px] border border-white/10 bg-black/20 p-1">
                  {(["trending", "popular", "new"] as MapSort[]).map((sort) => (
                    <button key={sort} type="button" onClick={() => setMapSort(sort)} className={`rounded-[10px] px-3 py-2 text-[11px] font-black uppercase tracking-[0.08em] ${mapSort === sort ? "bg-white text-[#10201a]" : "text-[#a9bfd4] hover:bg-white/[0.08]"}`}>
                      {sort === "popular" ? "Most Popular" : sort}
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          </div>

          {mapScope === "mine" && !canUploadCustomMaps ? (
            <div className="mt-5 rounded-[16px] border border-white/10 bg-black/20 p-5 text-sm font-semibold text-[#a9bfd4]">Sign in with a permanent account to use your custom maps.</div>
          ) : mapsQuery.isLoading ? (
            <div className="mt-6 flex items-center gap-3 text-sm text-[#a9bfd4]"><Loader2 className="animate-spin" size={18} /> Loading maps...</div>
          ) : readyMaps.length === 0 ? (
            <div className="mt-6 rounded-[18px] border border-dashed border-white/15 bg-black/15 p-8 text-center text-sm text-[#a9bfd4]">{hasMapSearch ? "No ready maps match your search." : "No ready maps in this section yet."}</div>
          ) : (
            <div className="mt-5 grid max-h-[56vh] gap-3 overflow-y-auto pr-1 sm:grid-cols-2 lg:grid-cols-3">
              {readyMaps.map((item) => (
                <button key={item.id} type="button" onClick={() => selectMapForParty(item)} className="overflow-hidden rounded-[16px] border border-white/10 bg-black/25 text-left transition hover:border-[#77f0be]/50 hover:bg-white/[0.06]">
                  <div className="relative aspect-[16/9] overflow-hidden bg-[#10201a]">
                    <img src={thumbnailURL(item)} alt="" className="h-full w-full object-cover opacity-90 transition hover:scale-[1.03]" />
                    <div className="absolute left-3 top-3 rounded-full bg-black/55 px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white">{item.difficulty}</div>
                    {item.id === lobbyConfig.mapId ? <div className="absolute right-3 top-3 rounded-full bg-[#22d385] px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.08em] text-white">Selected</div> : null}
                  </div>
                  <div className="p-3">
                    <h4 className="truncate text-sm font-black text-white">{item.displayName}</h4>
                    <p className="mt-1 truncate text-xs font-semibold text-[#8da6b5]">by {item.authorName || "GeoDuels"}</p>
                    <div className="mt-3 grid grid-cols-3 gap-2 text-[11px] font-bold text-[#a9bfd4]">
                      <span className="inline-flex items-center gap-1" title="Locations"><MapIcon size={13} />{item.locationCount.toLocaleString()}</span>
                      <span className="inline-flex items-center gap-1" title="Plays"><Play size={13} />{item.playCount.toLocaleString()}</span>
                      <span className="inline-flex items-center gap-1" title="Favorites"><Star size={13} />{item.favoriteCount.toLocaleString()}</span>
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </section>
      </div>
    </AppModalShell>
  ) : null;
  const showPartyPanel = privateLobbyActive && contentRoute !== "maps" && contentRoute !== "map-details" && contentRoute !== "map-upload";
  const visualNavIndex = Math.max(0, NAV_ITEMS.findIndex((item) => item.route === visualNavRoute));

  return (
    <div className="relative flex min-h-screen flex-col overflow-hidden font-sans text-[#f4f9ff] selection:bg-accentPrimary/30">
      <AnimatePresence>{maintenanceOverlay}</AnimatePresence>
      <div className="pointer-events-none fixed inset-0 z-0 overflow-hidden">
        <div
          className="absolute inset-0"
          style={{
            backgroundImage:
              "linear-gradient(rgba(18, 56, 41, 0.4), rgba(0, 0, 0, 0.9)), url('/bg2.v1.jpg')",
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
            backgroundSize: "cover",
            transform: "scale(1.06)",
          }}
        />
      </div>
      <AnimatePresence>
        {openModal === "help" && renderHelpModal()}
        {openModal === "profile" && renderProfileModal()}
        {openModal === "invite" && renderInviteLobbyModal()}
        {openModal === "signin" && renderSignInModal()}
        {mapPickerModal}
      </AnimatePresence>

      {/* Header */}
      <header className="sticky top-0 z-20 px-4 pb-4 pt-4 sm:px-6 sm:pb-5 sm:pt-5 lg:px-8 lg:pb-6 lg:pt-6">
        <AnimatePresence>{maintenanceBanner}</AnimatePresence>
        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 sm:gap-4 lg:gap-6">
          <div className="flex items-center gap-3 sm:gap-5">
            <button
              onClick={() => setOpenModal("help")}
              className="text-[#a9bfd4] transition-colors hover:text-white"
              aria-label="Help"
            >
              <HelpCircle
                size={20}
                strokeWidth={2}
                className="sm:h-[22px] sm:w-[22px]"
              />
            </button>
            {isAdmin ? (
              <Link
                href="/admin"
                prefetch={false}
                className="inline-flex items-center gap-2 rounded-full border border-[#2ad18f]/35 bg-[#2ad18f]/10 px-3 py-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-[#b9f5da] transition hover:bg-[#2ad18f]/18 sm:text-[12px]"
              >
                <Shield size={14} />
                Admin
              </Link>
            ) : null}
          </div>

          <div className="flex min-w-0 items-center justify-center">
            <Link href="/" aria-label="GeoDuels home" className="inline-flex">
              <img
                src="/logo.v2.png"
                alt="GeoDuels"
                width={140}
                height={38}
                className="h-auto w-[112px] sm:w-[140px]"
              />
            </Link>
          </div>

          {userId && userEmail ? (
            <div
              className="group flex min-w-0 items-center justify-self-end gap-2.5 cursor-pointer sm:gap-3"
              onClick={() => {
                setIsEditingProfileName(false);
                setOpenModal("profile");
              }}
            >
              <div className="flex min-w-0 max-w-[7.5rem] flex-col items-end justify-center sm:max-w-none">
            <PlayerNameWithBadge
              name={displayName || userEmail || "Player"}
              isAdmin={isAdmin}
              selectedBadge={null}
              nameClassName="text-[12px] font-bold leading-tight text-white transition-colors group-hover:text-emerald-100 sm:text-[15px]"
            />
                <div className="mt-0.5 flex items-center text-[10px] font-bold text-[#2ad18f] sm:text-[12px]">
                  <RatingTrophyIcon className="mr-1 h-3 w-3" />
                  {mmr}
                  <PlayerBadge badge={selectedBadge} size="sm" className="ml-1" />
                </div>
              </div>
              <AvatarBadge
                avatarUrl={userAvatar}
                fallback={userAvatarFallback}
                alt={displayName || userEmail || "Player"}
                size="sm"
                className="h-9 w-9 border-[1.5px] border-white/20 bg-[#162130] transition-colors group-hover:border-white/40 sm:h-[42px] sm:w-[42px]"
              />
            </div>
          ) : (
            <div className="pointer-events-auto justify-self-end">
              {signInButton}
            </div>
          )}
        </div>

        {!showPartyPanel ? (
          <div className="flex justify-center pt-5 sm:pt-6">
            <div className="relative flex h-9 w-full max-w-[340px] items-center justify-center pointer-events-auto sm:h-10 sm:max-w-[400px] lg:max-w-[440px]">
              {NAV_ITEMS.map((item, idx) => {
                const isActive = item.route === visualNavRoute;
                const offset = idx - visualNavIndex;

                return (
                  <motion.div
                    key={item.route}
                    initial={false}
                    animate={{
                      x: offset * 104,
                      scale: isActive ? 1.05 : 0.95,
                      opacity: isActive ? 1 : 0.4,
                    }}
                    transition={{ type: "spring", stiffness: 350, damping: 35 }}
                    className={`absolute font-bold text-[15px] tracking-[0.18em] transition-colors duration-200 sm:text-[16px] lg:text-[17px] ${isQueueing ? "cursor-not-allowed text-[#a9bfd4]/50" : "cursor-pointer"}`}
                    style={{
                      color: isActive
                        ? isQueueing
                          ? "#8cb0a1"
                          : "#ffffff"
                        : "#a9bfd4",
                      transformOrigin: "center",
                    }}
                  >
                    {isQueueing ? (
                      <span className="cursor-not-allowed">{item.label}</span>
                    ) : (
                      <Link
                        href={item.href}
                        prefetch
                        onClick={() => {
                          try {
                            window.sessionStorage.setItem(lobbyRouteStorageKey, currentNavRoute);
                          } catch {
                            // Navigation still works if session storage is unavailable.
                          }
                        }}
                      >
                        {item.label}
                      </Link>
                    )}
                  </motion.div>
                );
              })}
            </div>
          </div>
        ) : null}
      </header>

      {/* Main Content Area */}
      <main className="relative z-10 flex flex-1 flex-col items-center justify-start px-4 pb-10 pt-4 pointer-events-none sm:px-6 sm:pb-12 sm:pt-8">
        {privateLobbyErrorNotice}

        <AnimatePresence mode="popLayout">
          {showPartyPanel ? privateLobbyPanel : null}

          {!showPartyPanel && contentRoute === "play" && (
            <motion.div
              key="play"
              {...tabPanelMotion}
              className="flex w-full max-w-[1160px] flex-col items-center gap-5 pointer-events-auto lg:grid lg:grid-cols-[minmax(0,480px)_minmax(280px,360px)] lg:items-start lg:justify-center lg:gap-6"
            >
              <div className="flex w-full max-w-[480px] flex-col gap-5 lg:max-w-none">
                <div className="glass-panel lobby-feature-card relative flex w-full flex-col gap-4 rounded-[20px] p-5 transition-colors duration-500 sm:p-8">
                  <div
                    className={`absolute inset-0 pointer-events-none transition-opacity duration-500 ${isQueueing ? "opacity-95" : "opacity-80"} bg-[linear-gradient(180deg,rgba(72,128,106,0.28)_0%,rgba(22,42,34,0.78)_100%)]`}
                  />

                  {/* Decorative background mountains */}
                  <div
                    className={`absolute inset-x-0 bottom-0 pointer-events-none h-full transition-opacity duration-500 ${isQueueing ? "opacity-[0.24]" : "opacity-[0.32]"}`}
                  >
                    <img
                      src="/mountains.v1.svg"
                      alt=""
                      aria-hidden="true"
                      className="absolute inset-0 h-full w-full object-cover object-center"
                      style={{ objectPosition: "center bottom" }}
                    />
                  </div>

                  {/* Content over background */}
                  <div className="relative z-10 mt-1 flex flex-col sm:mt-2">
                    <span className="mb-1 text-[12px] font-bold uppercase tracking-[0.16em] text-[#8cb0a1] drop-shadow-sm">
                      {duelModeLabel}
                    </span>
                    <h2 className="mb-2 text-[36px] font-extrabold leading-tight tracking-tight text-white drop-shadow-md sm:text-[44px]">
                      Duel
                    </h2>
                  </div>

                  <div className="relative z-10 mx-auto mt-1 flex w-full flex-col px-0 pb-1 sm:mt-2 sm:px-2">
                    {queueError && (
                      <p className="mb-3 text-center text-xs font-semibold text-red-300">
                        {queueError}
                      </p>
                    )}
                    {!isQueueing ? (
                      <div className="mb-3 overflow-hidden rounded-[14px] border border-white/10 bg-black/25">
                        {([
                          ["moving", "Moving"],
                          ["nmpz", "NMPZ"],
                        ] as const).map(([ruleset, label]) => (
                          <button
                            key={ruleset}
                            type="button"
                            aria-pressed={queueRulesets.includes(ruleset)}
                            onClick={() => toggleQueueRuleset(ruleset)}
                            className={`flex min-h-[44px] w-full items-center justify-between px-4 text-left text-[13px] font-extrabold uppercase tracking-[0.08em] transition ${
                              queueRulesets.includes(ruleset)
                                ? "bg-[#22d385]/12 text-[#d7ffec]"
                                : "text-white/70 hover:bg-white/[0.07] hover:text-white"
                            }`}
                          >
                            <span>{label}</span>
                            <span className="flex h-[18px] w-[18px] items-center justify-center">
                              {queueRulesets.includes(ruleset) ? (
                                <CheckCircle2
                                  size={18}
                                  strokeWidth={2.5}
                                  className="text-[#22d385]"
                                />
                              ) : null}
                            </span>
                          </button>
                        ))}
                      </div>
                    ) : null}

                    {!isQueueing ? (
                      <button
                        onClick={onRankedPlay}
                        disabled={duelDisabled}
                        className="w-full flex items-center justify-center rounded-[16px] bg-[#22d385] py-[14px] text-[16px] font-extrabold uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(34,211,133,0.3)] transition-all duration-200 hover:scale-[1.01] hover:bg-[#2ae091] hover:shadow-[0_6px_24px_rgba(34,211,133,0.4)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:scale-100 disabled:hover:bg-[#22d385] disabled:hover:shadow-[0_4px_16px_rgba(34,211,133,0.3)]"
                      >
                        <Play
                          fill="currentColor"
                          size={20}
                          className="mr-2.5"
                        />
                        {queuePaused || playPaused || maintenanceIsActive
                          ? "Paused"
                          : primaryButtonLabel}
                      </button>
                    ) : (
                      <button
                        onClick={cancelQueue}
                        className="group w-full flex items-center justify-center rounded-[16px] border border-white/[0.1] bg-white/[0.08] py-[14px] text-[14px] font-bold uppercase tracking-[0.08em] text-white transition-colors hover:bg-white/[0.12]"
                      >
                        <Loader2
                          size={18}
                          className="mr-3 animate-spin text-[#2ad18f] transition-colors group-hover:text-[#3deb9e]"
                        />
                        <span className="text-accentPrimary">{queueElapsedLabel}</span>
                      </button>
                    )}
                  </div>
                </div>

                <div
                  className="glass-panel lobby-feature-card relative flex min-h-[240px] w-full flex-col justify-between rounded-[20px] p-5 transition-colors duration-500 sm:min-h-[260px] sm:p-8"
                  style={{ animationDelay: "-2s" }}
                >
                  <div className="absolute inset-0 pointer-events-none bg-[linear-gradient(180deg,rgba(72,106,128,0.28)_0%,rgba(22,34,42,0.85)_100%)] opacity-80 transition-opacity duration-500" />

                  {/* Decorative background mountains */}
                  <div className="absolute inset-x-0 bottom-0 h-full pointer-events-none opacity-[0.25] transition-opacity duration-500">
                    <img
                      src="/mountains.v1.svg"
                      alt=""
                      aria-hidden="true"
                      className="absolute inset-0 h-full w-full object-cover object-center opacity-50"
                      style={{
                        objectPosition: "center bottom",
                        filter: "hue-rotate(190deg)",
                      }}
                    />
                  </div>

                  {/* Content over background */}
                  <div className="relative z-10 mt-1 flex flex-col sm:mt-2">
                    <span className="mb-1 text-[12px] font-bold uppercase tracking-[0.16em] text-[#8caab0] drop-shadow-sm">
                      Casual
                    </span>
                    <h2 className="mb-2 text-[36px] font-extrabold leading-tight tracking-tight text-white drop-shadow-md sm:text-[44px]">
                      Singleplayer
                    </h2>
                    <span className="text-[15px] font-medium text-white/90 drop-shadow-sm sm:text-[16px]">
                      Practice indefinitely
                    </span>
                  </div>

                  <div className="relative z-10 mx-auto mt-5 flex h-full w-full flex-col justify-end px-0 pb-1 sm:mt-6 sm:px-2">
                    <button
                      onClick={() => startSingleplayer()}
                      disabled={singleplayerDisabled}
                      className="w-full flex items-center justify-center rounded-[16px] bg-[#3b82f6] py-[14px] text-[16px] font-extrabold uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(59,130,246,0.3)] transition-all duration-200 hover:scale-[1.01] hover:bg-[#4b8df8] hover:shadow-[0_6px_24px_rgba(59,130,246,0.4)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:scale-100"
                    >
                      {isSingleplayerLoading ? (
                        <Loader2 size={20} className="mr-2.5 animate-spin" />
                      ) : (
                        <Play fill="currentColor" size={20} className="mr-2.5" />
                      )}
                      {isSingleplayerLoading ? "Loading..." : playPaused || maintenanceIsActive ? "Paused" : "Play"}
                    </button>
                  </div>
                </div>

              </div>

              <div className="flex w-full max-w-[480px] flex-col gap-5 lg:sticky lg:top-8 lg:max-w-none">
                {onlineStatusCard}
                {newsPanel}
                {donateCard}
                {socialLinksCard}
              </div>
            </motion.div>
          )}

          {!showPartyPanel && contentRoute === "top" && (
            <motion.div
              key="top"
              {...tabPanelMotion}
              className="flex w-full justify-center pointer-events-auto"
            >
              {leaderboardPanel}
            </motion.div>
          )}

          {!showPartyPanel && contentRoute === "maps" && mapsPanel}
          {!showPartyPanel && contentRoute === "map-details" && mapDetailsPanelV2}
          {!showPartyPanel && contentRoute === "map-upload" && mapUploadPanel}

          {!showPartyPanel && contentRoute === "friends" && (
            <motion.div
              key="friends"
              {...tabPanelMotion}
              className="flex w-full max-w-[480px] flex-col gap-5 pointer-events-auto"
            >
              {inviteLobbyCard}
            </motion.div>
          )}
        </AnimatePresence>

        {!showPartyPanel && contentRoute === "play" ? (
          <>
            <section
              aria-labelledby="geoduels-seo-heading"
              className="glass-panel mt-8 w-full max-w-[1220px] rounded-[24px] p-6 pointer-events-auto sm:mt-[156px] sm:p-8"
            >
              <div className="space-y-6 text-left">
                <div className="max-w-3xl space-y-3">
                  <span className="inline-flex rounded-full border border-[#2ad18f]/30 bg-[#2ad18f]/10 px-3 py-1 text-[11px] font-bold uppercase tracking-[0.16em] text-[#7de3b7]">
                    Tutorial
                  </span>
                  <h1
                    id="geoduels-seo-heading"
                    className="text-[30px] font-extrabold leading-tight tracking-tight text-white sm:text-[40px]"
                  >
                    GeoDuels
                  </h1>
                  <p className="text-[15px] leading-7 text-[#a9bfd4] sm:text-[16px]">
                    A free GeoGuessr-inspired Street View game. Queue for ranked
                    matches against other players, with friends, or jump into singleplayer.
                  </p>
                </div>

                <div className="grid gap-5 lg:grid-cols-3">
                  <section className="glass-panel rounded-[18px] p-5">
                    <h2 className="text-[18px] font-extrabold tracking-tight text-white">
                      How to Play?
                    </h2>
                    <p className="mt-3 text-[14px] leading-7 text-[#a9bfd4]">
                      Find the location, place your guess. The closer you are, the
                      more points you get.
                    </p>
                  </section>
                  
                  <section className="glass-panel rounded-[18px] p-5">
                    <h2 className="text-[18px] font-extrabold tracking-tight text-white">
                      100% Free (seriously)
                    </h2>
                    <p className="mt-3 text-[14px] leading-7 text-[#a9bfd4]">
                      No subscriptions to play, no pay-to-win. Considered one of the best GeoGuessr free alternatives.
                    </p>
                  </section>

                  <section className="glass-panel rounded-[18px] p-5">
                    <h2 className="text-[18px] font-extrabold tracking-tight text-white">
                      Ranked & Casual
                    </h2>
                    <p className="mt-3 text-[14px] leading-7 text-[#a9bfd4]">
                      Climb the ladder or practice in casual mode, which not many GeoGuessr alternatives offer.
                    </p>
                  </section>
                </div>
              </div>
            </section>
            <div className="mt-4 w-full max-w-[1220px] px-6 sm:px-8">
              {legalCard}
            </div>
          </>
        ) : null}
      </main>
    </div>
  );
}
