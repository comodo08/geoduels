import { useEffect, useState } from "react";
import LobbyScreen from "../../../components/ui/LobbyScreen";
import type {
  HomeActions,
  HomeAuthView,
  HomeLobbyView,
  HomeViewModel,
} from "../model/types";

type HomePageLobbyProps = {
  auth: HomeAuthView;
  lobby: HomeLobbyView;
  meta: HomeViewModel["meta"];
  actions: Pick<
    HomeActions,
    | "joinQueue"
    | "startSingleplayer"
    | "cancelQueue"
    | "createInviteLobby"
    | "joinInviteLobby"
    | "leavePrivateLobby"
    | "kickLobbyMember"
    | "transferLobbyOwner"
    | "startPrivateLobby"
    | "updatePrivateLobbySettings"
    | "switchPrivateLobbyTeam"
    | "devLogin"
    | "triggerGoogleSignIn"
    | "triggerDiscordSignIn"
    | "linkAuthProvider"
    | "upgradeGuestWithProvider"
    | "unlinkAuthProvider"
    | "loadLeaderboard"
    | "submitProfileNickname"
    | "selectBadge"
    | "startSupportDonation"
    | "setNicknameInput"
    | "clearAuthSession"
    | "deleteAccount"
  >;
};

export default function HomePageLobby({
  auth,
  lobby,
  meta,
  actions,
}: HomePageLobbyProps) {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (lobby.inGame) {
    return null;
  }

  return (
    <LobbyScreen
      userId={auth.userId}
      userEmail={auth.userEmail}
      displayName={auth.displayName}
      userAvatar={auth.userAvatar}
      isGuest={auth.isGuest}
      authMigrationRequired={!!auth.authMigrationRequired}
      linkedProviders={auth.linkedProviders || []}
      badges={auth.badges || []}
      selectedBadge={auth.selectedBadge}
      connected={lobby.connected}
      mmr={lobby.mmr}
      gamesPlayed={lobby.gamesPlayed}
      winsPct={lobby.winsPct}
      leaderboard={lobby.leaderboard}
      leaderboardLoading={lobby.leaderboardLoading}
      status={lobby.status}
      queueStartedAt={lobby.queueStartedAt}
      joinQueue={actions.joinQueue}
      startSingleplayer={actions.startSingleplayer}
      cancelQueue={actions.cancelQueue}
      privateLobby={lobby.privateLobby}
      createInviteLobby={actions.createInviteLobby}
      joinInviteLobby={actions.joinInviteLobby}
      leavePrivateLobby={actions.leavePrivateLobby}
      kickLobbyMember={actions.kickLobbyMember}
      transferLobbyOwner={actions.transferLobbyOwner}
      startPrivateLobby={actions.startPrivateLobby}
      updatePrivateLobbySettings={actions.updatePrivateLobbySettings}
      switchPrivateLobbyTeam={actions.switchPrivateLobbyTeam}
      queueError={lobby.queueError}
      onlinePlayers={lobby.onlinePlayers}
      maintenance={lobby.maintenance}
      googleClientId={
        mounted && auth.googleSignInEnabled ? auth.googleClientId : ""
      }
      discordClientId={
        mounted && auth.discordSignInEnabled ? auth.discordClientId : ""
      }
      appVersion={meta.appVersion}
      isAdmin={auth.isAdmin}
      changelogEyebrow={lobby.changelogEyebrow}
      changelogTitle={lobby.changelogTitle}
      changelogMarkdown={lobby.changelogMarkdown}
      changelogSlug={lobby.changelogSlug}
      changelogUpdatedAt={lobby.changelogUpdatedAt}
      devLogin={actions.devLogin}
      onGoogleSignIn={actions.triggerGoogleSignIn}
      onDiscordSignIn={actions.triggerDiscordSignIn || actions.triggerGoogleSignIn}
      onLinkAuthProvider={actions.linkAuthProvider}
      onUpgradeGuestWithProvider={actions.upgradeGuestWithProvider}
      onUnlinkAuthProvider={actions.unlinkAuthProvider}
      onBrowseLeaderboard={actions.loadLeaderboard}
      authLoading={auth.authLoading}
      authError={auth.authError}
      nicknameInput={auth.nicknameInput}
      nicknameError={auth.nicknameError}
      nicknameSaving={auth.nicknameSaving}
      onChangeNickname={actions.setNicknameInput}
      onSaveNickname={actions.submitProfileNickname}
      onSelectBadge={actions.selectBadge}
      onSupportDonation={actions.startSupportDonation}
      onLogout={() => actions.clearAuthSession()}
      onDeleteAccount={actions.deleteAccount}
    />
  );
}
