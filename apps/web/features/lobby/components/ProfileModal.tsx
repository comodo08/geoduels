import { Check, Loader2, Pencil, Trash2 } from "lucide-react";
import type React from "react";
import { useState } from "react";
import AppModalShell from "../../../components/ui/AppModalShell";
import AvatarBadge from "../../../components/ui/AvatarBadge";
import PlayerBadge, { type PlayerBadgeInfo } from "../../../components/ui/PlayerBadge";
import PlayerNameWithBadge from "../../../components/ui/PlayerNameWithBadge";
import { LobbyPanel } from "./lobby-primitives";

type ProfileTab = "account" | "stats" | "badges";

const profileTabs: Array<{ id: ProfileTab; label: string }> = [
  { id: "stats", label: "Stats" },
  { id: "badges", label: "Badges" },
  { id: "account", label: "Account" },
];

type ProfileModalProps = {
  userId: string;
  userEmail: string;
  displayName: string;
  userAvatar?: string;
  isGuest: boolean;
  isAdmin: boolean;
  selectedBadge: PlayerBadgeInfo | null;
  badges: PlayerBadgeInfo[];
  mmr: number;
  gamesPlayed: number;
  winsPct: number;
  authLoading: boolean;
  authError: string;
  nicknameInput: string;
  nicknameError: string;
  nicknameSaving: boolean;
  linkedProviderCount: number;
  showGoogleButton: boolean;
  showDiscordButton: boolean;
  hasGoogleProvider: boolean;
  hasDiscordProvider: boolean;
  onChangeNickname: (value: string) => void;
  onSaveNickname: () => Promise<boolean>;
  onSelectBadge: (badgeId: string) => Promise<void>;
  onLinkAuthProvider: (provider: "google" | "discord") => void | Promise<void>;
  onUnlinkAuthProvider: (provider: "google" | "discord") => void | Promise<void>;
  onUpgradeGuestWithProvider: (provider: "google" | "discord") => void | Promise<void>;
  onDeleteAccount: () => Promise<void>;
  onLogout: () => void;
  onClose: () => void;
};

export function ProfileModal(props: ProfileModalProps) {
  const [profileTab, setProfileTab] = useState<ProfileTab>("stats");
  const [inspectedBadgeId, setInspectedBadgeId] = useState("");
  const [hoveredBadgeId, setHoveredBadgeId] = useState("");
  const [isEditingProfileName, setIsEditingProfileName] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const focusedBadge =
    props.badges.find((badge) => badge.id === hoveredBadgeId) ||
    props.badges.find((badge) => badge.id === inspectedBadgeId) ||
    props.badges.find((badge) => badge.id === props.selectedBadge?.id) ||
    props.badges[0] ||
    null;
  const avatarFallback = !props.userEmail
    ? "?"
    : (props.displayName || props.userEmail || "P").slice(0, 1).toUpperCase();

  const close = () => {
    setIsEditingProfileName(false);
    props.onClose();
  };

  const saveNickname = async () => {
    const saved = await props.onSaveNickname();
    if (saved) setIsEditingProfileName(false);
  };

  return (
    <AppModalShell title="Profile" onClose={close}>
      <LobbyPanel className="flex items-center gap-4 p-5">
        <AvatarBadge
          avatarUrl={props.userAvatar}
          fallback={avatarFallback}
          alt={props.displayName || props.userEmail || "Guest"}
          size="lg"
          className="border-white/20 bg-[#162130]"
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {isEditingProfileName && !props.isGuest ? (
              <input
                value={props.nicknameInput}
                onChange={(e) => props.onChangeNickname(e.target.value)}
                disabled={props.nicknameSaving}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void saveNickname();
                  if (e.key === "Escape") {
                    setIsEditingProfileName(false);
                    props.onChangeNickname(props.displayName || props.userEmail || "");
                  }
                }}
                className="min-w-0 flex-1 rounded-lg border border-white/10 bg-[#101a20] px-3 py-2 text-base font-bold text-white outline-none transition focus:border-[#2ad18f]/60"
                placeholder="Enter nickname"
                maxLength={14}
                autoFocus
              />
            ) : (
              <PlayerNameWithBadge
                name={props.displayName || props.userEmail || "Guest"}
                isAdmin={props.isAdmin}
                selectedBadge={null}
                nameClassName="text-xl font-bold text-white"
                wrapperClassName="min-w-0"
              />
            )}
            {props.userId && !props.isGuest ? (
              <button
                type="button"
                onClick={() => {
                  if (isEditingProfileName) {
                    void saveNickname();
                    return;
                  }
                  props.onChangeNickname(props.displayName || props.userEmail || "");
                  setIsEditingProfileName(true);
                }}
                disabled={props.nicknameSaving}
                className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white/70 transition hover:bg-white/10 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                aria-label={isEditingProfileName ? "Save nickname" : "Edit nickname"}
              >
                {props.nicknameSaving ? <Loader2 size={16} className="animate-spin" /> : isEditingProfileName ? <Check size={16} /> : <Pencil size={16} />}
              </button>
            ) : null}
          </div>
          {props.isGuest || props.selectedBadge ? (
            <div className="mt-1 flex items-center gap-2 text-sm text-[#a9bfd4]">
              {props.isGuest ? <span>Guest profile</span> : null}
              <PlayerBadge badge={props.selectedBadge} size="sm" />
            </div>
          ) : null}
          {props.nicknameError ? <p className="mt-2 text-xs font-semibold text-red-400">{props.nicknameError}</p> : null}
        </div>
      </LobbyPanel>

      <div className="mt-4 grid grid-cols-3 gap-2 rounded-2xl border border-white/10 bg-black/20 p-1">
        {profileTabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setProfileTab(tab.id)}
            className={`min-h-[38px] rounded-xl text-[11px] font-black uppercase tracking-[0.12em] transition ${profileTab === tab.id ? "bg-accentPrimary text-white" : "text-[#a9bfd4] hover:bg-white/10 hover:text-white"}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {profileTab === "stats" ? (
        <div className="mt-4 grid grid-cols-1 gap-3 text-center uppercase tracking-wider text-[#a9bfd4] sm:grid-cols-3">
          <ProfileStat label="MMR" value={props.mmr} />
          <ProfileStat label="Games" value={props.gamesPlayed} />
          <ProfileStat label="Winrate" value={`${props.winsPct}%`} />
        </div>
      ) : null}

      {profileTab === "badges" ? (
        <BadgeTab
          badges={props.badges}
          selectedBadge={props.selectedBadge}
          focusedBadge={focusedBadge}
          setHoveredBadgeId={setHoveredBadgeId}
          setInspectedBadgeId={setInspectedBadgeId}
          onSelectBadge={props.onSelectBadge}
        />
      ) : null}

      {profileTab === "account" ? (
        <AccountTab
          {...props}
          deleteConfirmation={deleteConfirmation}
          setDeleteConfirmation={setDeleteConfirmation}
          close={close}
        />
      ) : null}
    </AppModalShell>
  );
}

function ProfileStat(props: { label: string; value: string | number }) {
  return (
    <LobbyPanel className="rounded-xl p-3 py-4">
      <p className="text-[11px] font-bold">{props.label}</p>
      <p className="mt-1.5 text-2xl font-black text-white">{props.value}</p>
    </LobbyPanel>
  );
}

function BadgeTab(props: {
  badges: PlayerBadgeInfo[];
  selectedBadge: PlayerBadgeInfo | null;
  focusedBadge: PlayerBadgeInfo | null;
  setHoveredBadgeId: (badgeId: string) => void;
  setInspectedBadgeId: (badgeId: string) => void;
  onSelectBadge: (badgeId: string) => Promise<void>;
}) {
  return (
    <div className="mt-4">
      <div className="grid grid-cols-4 gap-3 rounded-2xl border border-white/10 bg-black/20 p-4 sm:grid-cols-6">
        {props.badges.map((badge) => {
          const owned = !!badge.owned;
          const selected = props.selectedBadge?.id === badge.id;
          const focused = props.focusedBadge?.id === badge.id;
          return (
            <button
              key={badge.id}
              type="button"
              onMouseEnter={() => props.setHoveredBadgeId(badge.id)}
              onMouseLeave={() => props.setHoveredBadgeId("")}
              onFocus={() => props.setHoveredBadgeId(badge.id)}
              onBlur={() => props.setHoveredBadgeId("")}
              onClick={() => {
                props.setInspectedBadgeId(badge.id);
                if (owned) void props.onSelectBadge(selected ? "" : badge.id);
              }}
              className={`relative flex aspect-square items-center justify-center rounded-2xl border transition ${focused ? "border-[#2ad18f]/70 bg-[#123f2d]/45" : "border-white/10 bg-white/[0.04] hover:bg-white/[0.08]"} ${selected ? "shadow-[0_0_24px_rgba(42,209,143,0.24)]" : ""}`}
              aria-label={badge.label}
            >
              <PlayerBadge badge={badge} size="lg" muted={!owned} />
              {selected ? <span className="absolute right-1.5 top-1.5 h-2.5 w-2.5 rounded-full bg-accentPrimary shadow-[0_0_10px_rgba(42,209,143,0.8)]" /> : null}
              {!owned ? <span className="absolute inset-x-2 bottom-1.5 rounded-full bg-black/40 py-0.5 text-[8px] font-black uppercase tracking-[0.12em] text-white/45">Locked</span> : null}
            </button>
          );
        })}
      </div>
      {props.focusedBadge ? (
        <div className="mt-3 h-[112px] rounded-2xl border border-white/10 bg-black/25 p-4">
          <div className="flex h-full items-start justify-between gap-3 overflow-hidden">
            <div className="min-w-0">
              <p className="text-sm font-black text-white">{props.focusedBadge.label}</p>
              <p className="mt-1 max-h-[48px] overflow-hidden text-xs leading-relaxed text-[#a9bfd4]">{props.focusedBadge.description}</p>
            </div>
            <span className={`shrink-0 rounded-full px-2.5 py-1 text-[10px] font-black uppercase tracking-[0.12em] ${props.focusedBadge.owned ? "bg-[#2ad18f]/18 text-[#8ff0c2]" : "bg-white/[0.06] text-white/45"}`}>
              {props.selectedBadge?.id === props.focusedBadge.id ? "Shown" : props.focusedBadge.owned ? "Available" : "Locked"}
            </span>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function AccountTab(props: ProfileModalProps & {
  deleteConfirmation: string;
  setDeleteConfirmation: (value: string) => void;
  close: () => void;
}) {
  return (
    <>
      {props.userId && !props.isGuest ? (
        <LobbyPanel className="mt-6 rounded-xl p-4">
          <div className="mb-4 rounded-xl border border-white/10 bg-black/15 p-3">
            <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-[#6b8b80]">Email</p>
            <p className="mt-1 truncate text-sm font-semibold text-white">{props.userEmail || "No email on this account"}</p>
          </div>
          <p className="mb-3 text-center text-[12px] font-semibold uppercase tracking-[0.14em] text-[#8cb0a1]">Sign-in Methods</p>
          <ProviderRows {...props} />
          {props.authError ? <p className="mt-3 text-center text-xs font-semibold text-red-300">{props.authError}</p> : null}
        </LobbyPanel>
      ) : null}
      {props.userId && props.isGuest ? (
        <LobbyPanel className="mt-6 rounded-xl p-4">
          <p className="mb-3 text-center text-[12px] font-semibold uppercase tracking-[0.14em] text-[#8cb0a1]">Save Progress</p>
          <div className="flex flex-wrap justify-center gap-3">
            {props.showGoogleButton ? <ProviderButton onClick={() => props.onUpgradeGuestWithProvider("google")} disabled={props.authLoading}>Google</ProviderButton> : null}
            {props.showDiscordButton ? <ProviderButton onClick={() => props.onUpgradeGuestWithProvider("discord")} disabled={props.authLoading}>Discord</ProviderButton> : null}
          </div>
        </LobbyPanel>
      ) : null}
      {props.userId ? <DeleteAccountSection {...props} /> : null}
      {props.userId ? (
        <button
          type="button"
          onClick={() => {
            props.close();
            props.onLogout();
          }}
          className="mt-6 w-full rounded-xl border border-red-500/30 bg-red-500/10 py-3 text-sm font-bold uppercase tracking-wider text-red-400 transition hover:bg-red-500/20"
        >
          Sign Out
        </button>
      ) : null}
    </>
  );
}

function ProviderRows(props: ProfileModalProps) {
  return (
    <div className="space-y-3">
      {props.showGoogleButton ? (
        <ProviderRow
          label="Google"
          linked={props.hasGoogleProvider}
          onLink={() => props.onLinkAuthProvider("google")}
          onUnlink={() => props.onUnlinkAuthProvider("google")}
          disabled={props.authLoading}
          unlinkDisabled={props.authLoading || props.linkedProviderCount <= 1}
        />
      ) : null}
      {props.showDiscordButton ? (
        <ProviderRow
          label="Discord"
          linked={props.hasDiscordProvider}
          onLink={() => props.onLinkAuthProvider("discord")}
          onUnlink={() => props.onUnlinkAuthProvider("discord")}
          disabled={props.authLoading}
          unlinkDisabled={props.authLoading || props.linkedProviderCount <= 1}
        />
      ) : null}
    </div>
  );
}

function ProviderRow(props: {
  label: string;
  linked: boolean;
  disabled: boolean;
  unlinkDisabled: boolean;
  onLink: () => void | Promise<void>;
  onUnlink: () => void | Promise<void>;
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-black/15 p-3">
      <div>
        <p className="text-sm font-bold text-white">{props.label}</p>
        <p className="text-xs text-[#a9bfd4]">{props.linked ? "Linked" : "Not linked"}</p>
      </div>
      {props.linked ? (
        <ProviderButton onClick={props.onUnlink} disabled={props.unlinkDisabled}>Unlink</ProviderButton>
      ) : (
        <ProviderButton onClick={props.onLink} disabled={props.disabled}>Link</ProviderButton>
      )}
    </div>
  );
}

function ProviderButton(props: {
  children: React.ReactNode;
  disabled?: boolean;
  onClick: () => void | Promise<void>;
}) {
  return (
    <button
      type="button"
      onClick={() => void props.onClick()}
      disabled={props.disabled}
      className="rounded-lg border border-white/10 bg-white/[0.08] px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition hover:bg-white/[0.12] disabled:cursor-not-allowed disabled:opacity-50"
    >
      {props.children}
    </button>
  );
}

function DeleteAccountSection(props: ProfileModalProps & {
  deleteConfirmation: string;
  setDeleteConfirmation: (value: string) => void;
  close: () => void;
}) {
  return (
    <div className="mt-6 rounded-xl border border-red-500/25 bg-red-950/20 p-4">
      <p className="text-center text-[12px] font-semibold uppercase tracking-[0.14em] text-red-200">Delete Account</p>
      <p className="mt-2 text-center text-xs leading-relaxed text-red-100/70">
        This signs you out, removes sign-in links, and clears your profile.
        Match and moderation records are retained.
      </p>
      <input
        value={props.deleteConfirmation}
        onChange={(event) => props.setDeleteConfirmation(event.target.value)}
        placeholder="Type DELETE"
        className="mt-4 w-full rounded-xl border border-red-300/20 bg-black/25 px-3 py-2 text-center text-sm font-bold tracking-[0.18em] text-white outline-none transition placeholder:text-red-100/35 focus:border-red-300/50"
      />
      <button
        type="button"
        onClick={async () => {
          if (props.deleteConfirmation !== "DELETE") return;
          try {
            await props.onDeleteAccount();
            props.close();
          } catch {
            // The model surfaces the failure in the profile modal.
          }
        }}
        disabled={props.authLoading || props.deleteConfirmation !== "DELETE"}
        className="mt-3 flex w-full items-center justify-center gap-2 rounded-xl border border-red-500/40 bg-red-500/15 py-3 text-[13px] font-bold uppercase tracking-wider text-red-100 transition hover:bg-red-500/25 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {props.authLoading ? <Loader2 size={16} className="animate-spin" /> : <Trash2 size={16} />}
        Delete Account
      </button>
    </div>
  );
}
