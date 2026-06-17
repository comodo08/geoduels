import React, { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { motion, AnimatePresence } from "framer-motion";
import Router from "next/router";
import type { PlayerBadgeInfo } from "./PlayerBadge";
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
import { mapThumbnailURL } from "../../features/maps/lib/map-thumbnails";
import {
  NAV_ITEMS,
  formatApproximateTime,
  formatQueueElapsed,
  formatRelativeDuration,
  isLobbyNavRoute,
  isMapScope,
  lobbyRouteStorageKey,
  parseTime,
  type LobbyContentRoute,
} from "../../features/lobby/lib/lobby-ui";
import { MapUploadForm } from "../../features/lobby/components/MapUploadForm";
import { PlayPanel } from "../../features/lobby/components/PlayPanel";
import { LobbyTutorialSection } from "../../features/lobby/components/LobbyTutorialSection";
import { LobbyHeader } from "../../features/lobby/components/LobbyHeader";
import { DiscordProviderButton, GoogleProviderButton, SignInButton } from "../../features/lobby/components/LobbyAuthButtons";
import { ProfileModal } from "../../features/lobby/components/ProfileModal";
import { PartyPanel } from "../../features/lobby/components/PartyPanel";
import { LeaderboardPanel } from "../../features/lobby/components/LeaderboardPanel";
import {
  DonateCard,
  InviteLobbyCard,
  LegalFooter,
  NewsPanel,
  OnlineStatusCard,
  PrivateLobbyErrorNotice,
  SocialLinksCard,
} from "../../features/lobby/components/LobbyShellPieces";
import { MaintenanceBanner, MaintenanceOverlay } from "../../features/lobby/components/MaintenanceNotice";
import {
  MapDetailsPanel,
  MapPickerModal,
  MapsPanel,
  MapUploadPanel,
} from "../../features/lobby/components/maps/MapPanels";
import { HelpModal } from "../../features/lobby/components/modals/HelpModal";
import { InviteModal } from "../../features/lobby/components/modals/InviteModal";
import { SignInModal } from "../../features/lobby/components/modals/SignInModal";
import { usePartyPanelState } from "../../features/lobby/hooks/usePartyPanelState";
import { useQueueRulesetSelection } from "../../features/lobby/hooks/useQueueRulesetSelection";

export type { LobbyContentRoute } from "../../features/lobby/lib/lobby-ui";

type PartyModal = "help" | "profile" | "invite" | "signin" | null;

type PrivateLobbyView = {
  status: PartyRuntimeStatus;
  snapshot: PartySnapshot | null;
  inviteCode: string;
  isMember: boolean;
  isOwner: boolean;
  busy: boolean;
  error: string;
};

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
  const [isBlogExpanded, setIsBlogExpanded] = useState(false);
  const { queueRulesets, toggleQueueRuleset } = useQueueRulesetSelection();
  const [inviteCopied, setInviteCopied] = useState(false);
  const [inviteCodeInput, setInviteCodeInput] = useState("");
  const [mapPickerOpen, setMapPickerOpen] = useState(false);
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

  const isQueueing = status === "queueing";
  const isSingleplayerLoading = status === "matched_connecting";
  const canUseRankedQueue = !!userId && !isGuest;
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
    <DiscordProviderButton authLoading={authLoading} onClick={onDiscordSignIn} />
  ) : null;

  const signInButton =
    showGoogleButton || showDiscordButton ? (
      <SignInButton authLoading={authLoading} onClick={() => setOpenModal("signin")}>
        Sign In
      </SignInButton>
    ) : (
      <SignInButton authLoading={authLoading} onClick={devLogin} rounded="full">
        Dev Login
      </SignInButton>
    );

  const googleProviderButton = showGoogleButton ? (
    <GoogleProviderButton authLoading={authLoading} onClick={onGoogleSignIn} />
  ) : null;

  const newsPanel = (
    <NewsPanel
      changelogEyebrow={changelogEyebrow}
      changelogMarkdown={changelogMarkdown}
      changelogSlug={changelogSlug}
      changelogTitle={changelogTitle}
      changelogUpdatedAt={changelogUpdatedAt}
      expanded={isBlogExpanded}
      onToggle={() => setIsBlogExpanded((prev) => !prev)}
    />
  );
  const donateCard = <DonateCard onSupportDonation={onSupportDonation} />;
  const socialLinksCard = <SocialLinksCard />;
  const onlineStatusCard = <OnlineStatusCard onlinePlayers={onlinePlayers} />;

  const partyPanelState = usePartyPanelState({
    privateLobby,
    userId,
    updateSettings: updatePrivateLobbySettings,
    setInviteCopied,
  });
  const privateLobbyActive = partyPanelState.active;
  const lobbyConfig = partyPanelState.config;
  const saveLobbyConfig = partyPanelState.saveConfig;

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
  const thumbnailURL = (item: Pick<CustomMap, "thumbnailVariant" | "thumbnailKey">) => mapThumbnailURL(item.thumbnailKey, item.thumbnailVariant);
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
    <MapUploadForm
      isGuest={isGuest}
      mapName={mapName}
      setMapName={setMapName}
      mapDescription={mapDescription}
      setMapDescription={setMapDescription}
      mapDifficulty={mapDifficulty}
      setMapDifficulty={setMapDifficulty}
      mapThumbnailKey={mapThumbnailKey}
      setMapThumbnailKey={setMapThumbnailKey}
      mapThumbnailCategory={mapThumbnailCategory}
      setMapThumbnailCategory={setMapThumbnailCategory}
      mapThumbnailSearch={mapThumbnailSearch}
      setMapThumbnailSearch={setMapThumbnailSearch}
      mapFile={mapFile}
      setMapFile={setMapFile}
      mapUploadError={mapUploadError}
      setMapUploadError={setMapUploadError}
      uploadPending={createMapMutation.isPending}
      onUpload={() => createMapMutation.mutate()}
    />
  );
  const mapsPanel = (
    <motion.div key="maps" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <MapsPanel
        canUploadCustomMaps={canUploadCustomMaps}
        hasMapSearch={hasMapSearch}
        mapScope={mapScope}
        mapScopeLabels={mapScopeLabels}
        mapSearchInput={mapSearchInput}
        mapSort={mapSort}
        mapsLoading={mapsQuery.isLoading}
        privateLobbyActive={privateLobbyActive}
        readyMaps={readyMaps}
        setMapScope={setMapScope}
        setMapSearchInput={setMapSearchInput}
        setMapSort={setMapSort}
        selectMapForParty={selectMapForParty}
        thumbnailURL={thumbnailURL}
      />
    </motion.div>
  );

  const mapUploadPanel = (
    <motion.div key="map-upload" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <MapUploadPanel canUploadCustomMaps={canUploadCustomMaps} mapUploadForm={mapUploadForm} />
    </motion.div>
  );

  const mapDetailsPanel = (
    <motion.div key="map-details" {...tabPanelMotion} className="w-full max-w-[1120px] pointer-events-auto">
      <MapDetailsPanel
        accessToken={accessToken}
        canInteractWithMaps={canInteractWithMaps}
        canUploadCustomMaps={canUploadCustomMaps}
        commentBody={commentBody}
        commentComposerFocused={commentComposerFocused}
        createCommentPending={createCommentMutation.isPending}
        displayName={displayName}
        expandedCommentIds={expandedCommentIds}
        favoriteMap={(input) => favoriteMapMutation.mutate(input)}
        isAdmin={isAdmin}
        isModerator={isModerator}
        likedCommentIds={likedCommentIds}
        mapPickerFlow={mapPickerFlow}
        onCancelComment={() => { setCommentBody(""); setCommentComposerFocused(false); }}
        onDeleteComment={(commentId) => deleteCommentMutation.mutate({ commentId })}
        onDeleteMap={deleteMap}
        onPostComment={postMapComment}
        onPostReply={postMapReply}
        onPublishMap={(mapId) => publishMapMutation.mutate(mapId)}
        onRevisionFile={(mapId, file) => revisionMutation.mutate({ mapId, file })}
        onSetCommentBody={setCommentBody}
        onSetCommentComposerFocused={setCommentComposerFocused}
        onSetOpenCommentMenuId={setOpenCommentMenuId}
        onSetReplyBody={setReplyBody}
        onSetReplyToCommentId={setReplyToCommentId}
        onToggleCommentLike={toggleCommentLike}
        onToggleCommentReplies={toggleCommentReplies}
        openCommentMenuId={openCommentMenuId}
        playMapSingleplayer={playMapSingleplayer}
        replyBody={replyBody}
        replyToCommentId={replyToCommentId}
        selectMapForParty={selectMapForParty}
        selectedMapDetails={selectedMapDetails}
        selectedMapLoading={selectedMapQuery.isLoading}
        singleplayerDisabled={singleplayerDisabled}
        thumbnailURL={thumbnailURL}
        userAvatar={userAvatar}
        userAvatarFallback={userAvatarFallback}
        userEmail={userEmail}
        userId={userId}
      />
    </motion.div>
  );

  const privateLobbyPanel = privateLobbyActive ? (
    <PartyPanel
      authError={authError}
      authLoading={authLoading}
      inviteCopied={inviteCopied}
      joinInviteLobby={joinInviteLobby}
      kickLobbyMember={kickLobbyMember}
      leavePrivateLobby={leavePrivateLobby}
      mapsLoading={mapsQuery.isLoading}
      privateLobby={privateLobby}
      readyMaps={readyMaps}
      setMapPickerOpen={setMapPickerOpen}
      startPrivateLobby={startPrivateLobby}
      state={partyPanelState}
      switchPrivateLobbyTeam={switchPrivateLobbyTeam}
      transferLobbyOwner={transferLobbyOwner}
      userId={userId}
    />
  ) : null;

  const maintenanceBanner = maintenanceIsWarning ? (
    <MaintenanceBanner message={maintenanceMessage} countdown={warningCountdown} />
  ) : null;

  const maintenanceOverlay = maintenanceIsActive ? (
    <MaintenanceOverlay message={maintenanceMessage} eta={activeEta} />
  ) : null;

  const legalCard = <LegalFooter appVersion={appVersion} />;

  const leaderboardPanel = (
    <LeaderboardPanel
      leaderboard={leaderboard}
      leaderboardLoading={leaderboardLoading}
      mmr={mmr}
      userId={userId}
    />
  );

  const renderHelpModal = () => <HelpModal onClose={() => setOpenModal(null)} />;

  const renderInviteLobbyModal = () => (
    <InviteModal
      inviteCodeInput={inviteCodeInput}
      setInviteCodeInput={setInviteCodeInput}
      busy={privateLobby.busy}
      authLoading={authLoading}
      maintenanceIsActive={maintenanceIsActive}
      playPaused={playPaused}
      authError={authError}
      createInviteLobby={createInviteLobby}
      joinInviteLobby={joinInviteLobby}
      onClose={() => setOpenModal(null)}
    />
  );

  const renderSignInModal = () => (
    <SignInModal
      googleProviderButton={googleProviderButton}
      discordProviderButton={discordProviderButton}
      fallbackButton={signInButton}
      authError={authError}
      onClose={() => setOpenModal(null)}
    />
  );

  const renderProfileModal = () => (
    <ProfileModal
      userId={userId}
      userEmail={userEmail}
      displayName={displayName}
      userAvatar={userAvatar}
      isGuest={isGuest}
      isAdmin={isAdmin}
      selectedBadge={selectedBadge}
      badges={badges}
      mmr={mmr}
      gamesPlayed={gamesPlayed}
      winsPct={winsPct}
      authLoading={authLoading}
      authError={authError}
      nicknameInput={nicknameInput}
      nicknameError={nicknameError}
      nicknameSaving={nicknameSaving}
      linkedProviderCount={linkedProviderCount}
      showGoogleButton={showGoogleButton}
      showDiscordButton={showDiscordButton}
      hasGoogleProvider={hasGoogleProvider}
      hasDiscordProvider={hasDiscordProvider}
      onChangeNickname={onChangeNickname}
      onSaveNickname={onSaveNickname}
      onSelectBadge={onSelectBadge}
      onLinkAuthProvider={onLinkAuthProvider}
      onUnlinkAuthProvider={onUnlinkAuthProvider}
      onUpgradeGuestWithProvider={onUpgradeGuestWithProvider}
      onDeleteAccount={onDeleteAccount}
      onLogout={onLogout}
      onClose={() => setOpenModal(null)}
    />
  );

  const inviteLobbyCard = (
    <InviteLobbyCard
      disabled={authLoading || nicknameSaving || playPaused || maintenanceIsActive}
      onClick={() => setOpenModal("invite")}
    />
  );

  const privateLobbyErrorNotice = <PrivateLobbyErrorNotice message={privateLobby.error} />;
  const mapPickerModal = mapPickerOpen ? (
    <MapPickerModal
      canUploadCustomMaps={canUploadCustomMaps}
      hasMapSearch={hasMapSearch}
      lobbyConfig={lobbyConfig}
      mapScope={mapScope}
      mapScopeLabels={mapScopeLabels}
      mapSearchInput={mapSearchInput}
      mapSort={mapSort}
      mapsLoading={mapsQuery.isLoading}
      onClose={() => setMapPickerOpen(false)}
      readyMaps={readyMaps}
      selectMapForParty={selectMapForParty}
      setMapScope={setMapScope}
      setMapSearchInput={setMapSearchInput}
      setMapSort={setMapSort}
      thumbnailURL={thumbnailURL}
    />
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

      <LobbyHeader
        currentNavRoute={currentNavRoute}
        displayName={displayName}
        isAdmin={isAdmin}
        isQueueing={isQueueing}
        maintenanceBanner={maintenanceBanner}
        mmr={mmr}
        selectedBadge={selectedBadge}
        setOpenHelp={() => setOpenModal("help")}
        setOpenProfile={() => setOpenModal("profile")}
        showPartyPanel={showPartyPanel}
        signInButton={signInButton}
        userAvatar={userAvatar}
        userAvatarFallback={userAvatarFallback}
        userEmail={userEmail}
        userId={userId}
        visualNavIndex={visualNavIndex}
        visualNavRoute={visualNavRoute}
      />

      {/* Main Content Area */}
      <main className="relative z-10 flex flex-1 flex-col items-center justify-start px-4 pb-10 pt-4 pointer-events-none sm:px-6 sm:pb-12 sm:pt-8">
        {privateLobbyErrorNotice}

        <AnimatePresence mode="popLayout">
          {showPartyPanel ? privateLobbyPanel : null}

          {!showPartyPanel && contentRoute === "play" && (
            <PlayPanel
              isQueueing={isQueueing}
              isSingleplayerLoading={isSingleplayerLoading}
              queueError={queueError}
              queueRulesets={queueRulesets}
              toggleQueueRuleset={toggleQueueRuleset}
              onRankedPlay={onRankedPlay}
              cancelQueue={cancelQueue}
              startSingleplayer={startSingleplayer}
              duelDisabled={duelDisabled}
              singleplayerDisabled={singleplayerDisabled}
              queuePaused={queuePaused}
              playPaused={playPaused}
              maintenanceIsActive={maintenanceIsActive}
              primaryButtonLabel={primaryButtonLabel}
              queueElapsedLabel={queueElapsedLabel}
              duelModeLabel={duelModeLabel}
              sideCards={
                <>
                  {onlineStatusCard}
                  {newsPanel}
                  {donateCard}
                  {socialLinksCard}
                </>
              }
            />
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
          {!showPartyPanel && contentRoute === "map-details" && mapDetailsPanel}
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
            <LobbyTutorialSection />
            <div className="mt-4 w-full max-w-[1220px] px-6 sm:px-8">
              {legalCard}
            </div>
          </>
        ) : null}
      </main>
    </div>
  );
}
