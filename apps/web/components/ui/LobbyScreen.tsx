import React, { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import type { PlayerBadgeInfo } from "./PlayerBadge";
import type { LeaderboardSummary } from "../../features/auth/controllers/session-controller";
import type { LobbySnapshot as PartySnapshot, LobbyTeamId as PartyTeamId, PartyMode } from "../../features/lobby/lib/lobby-client";
import type { LobbyRuntimeStatus as PartyRuntimeStatus } from "../../features/lobby/controllers/lobby-controller";
import type { GameRuleset, MaintenanceStatus, MatchConfig } from "../../features/matchmaking/lib/queue-client";
import {
  NAV_ITEMS,
  formatApproximateTime,
  formatQueueElapsed,
  formatRelativeDuration,
  isLobbyNavRoute,
  lobbyRouteStorageKey,
  parseTime,
  type LobbyContentRoute,
} from "../../features/lobby/lib/lobby-ui";
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
import { MapPickerController, MapRouteSurface } from "../../features/lobby/components/maps/MapRouteSurfaces";
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

const lobbyBackgroundImage = "/bg2.v1.jpg";
const lobbyBackgroundPlaceholder = "/bg2.placeholder.v1.jpg";
const lobbyBackgroundOverlay = "linear-gradient(rgba(18, 56, 41, 0.4), rgba(0, 0, 0, 0.9))";

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
  const [nowMs, setNowMs] = useState(() => Date.now());
  const [dismissedMaintenanceAlertKey, setDismissedMaintenanceAlertKey] = useState("");
  const [highQualityBackgroundReady, setHighQualityBackgroundReady] = useState(false);
  const currentNavRoute: LobbyContentRoute = contentRoute === "map-details" || contentRoute === "map-upload" ? "maps" : contentRoute;
  const [visualNavRoute, setVisualNavRoute] = useState<LobbyContentRoute>(() => {
    if (typeof window === "undefined") return currentNavRoute;
    const stored = window.sessionStorage.getItem(lobbyRouteStorageKey) || "";
    return isLobbyNavRoute(stored) ? stored : currentNavRoute;
  });
  const canInteractWithMaps = !!accessToken && !isGuest;
  const canUploadCustomMaps = canInteractWithMaps;

  useEffect(() => {
    if (contentRoute === "top") {
      onBrowseLeaderboard();
    }
  }, [contentRoute, onBrowseLeaderboard]);

  useEffect(() => {
    let cancelled = false;
    const loadBackground = () => {
      const image = new Image();
      image.onload = async () => {
        try {
          await image.decode();
        } catch {
          // Some browsers resolve onload before decode support is available.
        }
        if (!cancelled) {
          setHighQualityBackgroundReady(true);
        }
      };
      image.src = lobbyBackgroundImage;
    };
    const timer = window.setTimeout(loadBackground, 350);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const frame = window.requestAnimationFrame(() => {
      setVisualNavRoute(currentNavRoute);
      window.sessionStorage.setItem(lobbyRouteStorageKey, currentNavRoute);
    });
    return () => window.cancelAnimationFrame(frame);
  }, [currentNavRoute]);

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
  const maintenanceAlertKey = maintenance
    ? [maintenance.phase, maintenance.startsAt, maintenance.endsAt, maintenance.message].join("|")
    : "";
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

  const mapPickerFlow = privateLobbyActive && privateLobby.isOwner && privateLobby.snapshot?.state === "open";
  const mapRouteSurface =
    contentRoute === "maps" || contentRoute === "map-details" || contentRoute === "map-upload" ? (
      <MapRouteSurface
        accessToken={accessToken}
        canUploadCustomMaps={canUploadCustomMaps}
        contentRoute={contentRoute}
        createInviteLobby={createInviteLobby}
        displayName={displayName}
        isAdmin={isAdmin}
        isModerator={isModerator}
        mapId={mapId}
        mapPickerFlow={!!mapPickerFlow}
        privateLobbyActive={privateLobbyActive}
        saveLobbyConfig={saveLobbyConfig}
        singleplayerDisabled={singleplayerDisabled}
        startSingleplayer={startSingleplayer}
        userAvatar={userAvatar}
        userAvatarFallback={userAvatarFallback}
        userEmail={userEmail}
        userId={userId}
      />
    ) : null;

  const privateLobbyPanel = privateLobbyActive ? (
    <PartyPanel
      authError={authError}
      authLoading={authLoading}
      inviteCopied={inviteCopied}
      joinInviteLobby={joinInviteLobby}
      kickLobbyMember={kickLobbyMember}
      leavePrivateLobby={leavePrivateLobby}
      privateLobby={privateLobby}
      setMapPickerOpen={setMapPickerOpen}
      startPrivateLobby={startPrivateLobby}
      state={partyPanelState}
      switchPrivateLobbyTeam={switchPrivateLobbyTeam}
      transferLobbyOwner={transferLobbyOwner}
      userId={userId}
    />
  ) : null;

  const maintenanceAlertDismissed = isAdmin && dismissedMaintenanceAlertKey === maintenanceAlertKey;
  const dismissMaintenanceAlert = isAdmin
    ? () => setDismissedMaintenanceAlertKey(maintenanceAlertKey)
    : undefined;
  const showMaintenanceBanner = maintenanceIsWarning && !maintenanceAlertDismissed;
  const maintenanceBanner = showMaintenanceBanner ? (
    <MaintenanceBanner
      message={maintenanceMessage}
      countdown={warningCountdown}
      onDismiss={dismissMaintenanceAlert}
    />
  ) : null;

  const maintenanceOverlay = maintenanceIsActive && !maintenanceAlertDismissed ? (
    <MaintenanceOverlay
      message={maintenanceMessage}
      eta={activeEta}
      onDismiss={dismissMaintenanceAlert}
    />
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
    <MapPickerController
      accessToken={accessToken}
      canUploadCustomMaps={canUploadCustomMaps}
      lobbyConfig={lobbyConfig}
      onClose={() => setMapPickerOpen(false)}
      saveLobbyConfig={saveLobbyConfig}
      userId={userId}
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
            backgroundImage: `${lobbyBackgroundOverlay}, url('${lobbyBackgroundPlaceholder}')`,
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
            backgroundSize: "cover",
            filter: "blur(14px)",
            transform: "scale(1.06)",
          }}
        />
        <div
          className={`absolute inset-0 transition-opacity duration-700 ease-out ${
            highQualityBackgroundReady ? "opacity-100" : "opacity-0"
          }`}
          style={{
            backgroundImage: `${lobbyBackgroundOverlay}, url('${lobbyBackgroundImage}')`,
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

          {!showPartyPanel && mapRouteSurface}

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
