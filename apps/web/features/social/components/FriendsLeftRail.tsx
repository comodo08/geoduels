import { Check, UserCheck, Users } from "lucide-react";
import PlayerProfileLink from "../../../components/ui/PlayerProfileLink";
import AvatarBadge from "../../../components/ui/AvatarBadge";
import PlayerBadge, { type PlayerBadgeInfo } from "../../../components/ui/PlayerBadge";
import { LobbyIconButton } from "../../lobby/components/lobby-primitives";
import { useSocial } from "../SocialProvider";
import type { FriendRow, SocialPresenceStatus } from "../lib/friends-client";

function statusDotClass(status: SocialPresenceStatus): string {
  switch (status) {
    case "online":
      return "bg-emerald-400";
    case "away":
      return "bg-yellow-400";
    default:
      return "bg-slate-500";
  }
}

function FriendRailItem({
  friend,
  status,
  canInvite,
  invited,
  onInvite,
  onRemove,
}: {
  friend: FriendRow;
  status: SocialPresenceStatus;
  canInvite: boolean;
  invited: boolean;
  onInvite: (userId: string) => void;
  onRemove: (userId: string) => void;
}) {
  const fallback = (friend.displayName || friend.userId || "?").slice(0, 2).toUpperCase();
  return (
    <li className="group flex items-center gap-3 rounded-xl px-2.5 py-2.5 hover:bg-white/[0.06]">
      <PlayerProfileLink
        userId={friend.userId}
        nickname={friend.displayName}
        className="flex min-w-0 flex-1 items-center gap-2.5"
        title={`View ${friend.displayName || "player"} profile`}
      >
        <span className="relative inline-flex h-11 w-11 shrink-0 items-center justify-center">
          <AvatarBadge
          avatarUrl={friend.avatarUrl}
          fallback={fallback}
          alt={friend.displayName || "Player"}
          size="md"
          className="bg-slate-800"
        />
        <span
          className={`absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-slate-900 ${statusDotClass(status)}`}
        />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[14px] font-bold text-white">
          {friend.displayName || "Player"}
        </span>
        {friend.selectedBadge ? (
          <PlayerBadge badge={friend.selectedBadge as PlayerBadgeInfo} size="sm" className="mt-0.5" />
        ) : null}
      </span>
      </PlayerProfileLink>
      {canInvite ? (
        <LobbyIconButton
          aria-label={`Invite ${friend.displayName || "friend"} to party`}
          title={invited ? "Invite sent" : "Invite to party"}
          disabled={invited}
          onClick={() => onInvite(friend.userId)}
          className={`h-9 w-9 ${invited ? "border-emerald-400/30 bg-emerald-500/20 text-emerald-200 hover:bg-emerald-500/20" : ""}`}
        >
          {invited ? <Check size={15} aria-hidden="true" /> : <Users size={15} aria-hidden="true" />}
        </LobbyIconButton>
      ) : null}
    </li>
  );
}

export function FriendsLeftRail() {
  const social = useSocial();

  if (!social.enabled || !social.activeParty?.isOwner || social.friends.length === 0) return null;

  const presenceOf = (id: string): SocialPresenceStatus => social.presence[id] ?? "offline";
  const canInvite = !!social.activeParty && social.activeParty.isOwner;
  const partyMemberIds = new Set(social.activeParty?.memberIds ?? []);

  return (
    <aside className="fixed left-6 top-1/4 z-40 hidden max-h-[70vh] w-64 flex-col overflow-hidden rounded-2xl xl:flex">
      <div className="flex items-center gap-2 px-4 py-3">
        <UserCheck size={16} aria-hidden="true" />
        <span className="text-[11px] font-black uppercase tracking-[0.14em] text-white">
          Friends
        </span>
      </div>
          <div className="flex-1 overflow-y-auto px-2 py-2">
            {social.friends.length === 0 ? (
                <p className="px-2 py-2 text-[12px] font-semibold text-[#7f9a8f]">
                  Add friends to see them here
                </p>
            ) : (
              <ul className="space-y-1">
                {social.friends.length === 0 ? (
                  <li className="px-3 py-6 text-center text-[12px] font-semibold text-[#7f9a8f]">
                    Add friends to see them here
                  </li>
                ) : (
                  [...social.friends]
                    .sort((a, b) => {
                      const rank: Record<string, number> = { online: 0, away: 1, offline: 2 };
                      const diff = rank[presenceOf(a.userId)] - rank[presenceOf(b.userId)];
                      return diff !== 0 ? diff : a.displayName.localeCompare(b.displayName);
                    })
                    .map((friend) => (
                      <FriendRailItem
                        key={friend.userId}
                        friend={friend}
                        status={presenceOf(friend.userId)}
                        canInvite={canInvite && !partyMemberIds.has(friend.userId) && presenceOf(friend.userId) === "online"}
                        invited={social.invitedFriends.has(friend.userId)}
                        onInvite={(userId) => {
                          void social.sendPartyInvite(userId);
                          social.markInvited(userId);
                        }}
                        onRemove={social.removeFriend}
                      />
                    ))
                )}
              </ul>
            )}
          </div>
        </aside>
  );
}
