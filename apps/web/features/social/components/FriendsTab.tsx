import { useEffect, useRef, useState } from "react";
import { Bell, Check, Search, UserCheck, UserPlus, X } from "lucide-react";
import PlayerProfileLink from "../../../components/ui/PlayerProfileLink";
import AvatarBadge from "../../../components/ui/AvatarBadge";
import PlayerBadge, { type PlayerBadgeInfo } from "../../../components/ui/PlayerBadge";
import {
  LobbyActionButton,
  LobbyIconButton,
  LobbyInset,
  LobbyInput,
  LobbyPill,
  LobbySectionHeader,
} from "../../lobby/components/lobby-primitives";
import { getRuntimeConfig } from "../../../lib/runtime-config";
import { searchPlayers, type FriendRow } from "../lib/friends-client";
import { useSocial } from "../SocialProvider";

function SectionHeader({
  icon,
  eyebrow,
  title,
  description,
}: {
  icon: React.ReactNode;
  eyebrow?: React.ReactNode;
  title?: React.ReactNode;
  description?: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-4">
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-[#2ad18f]/14 text-[#77f0be]">
        {icon}
      </div>
      <LobbySectionHeader
        eyebrow={eyebrow}
        title={title}
        description={description}
        titleClassName="text-[18px] font-extrabold tracking-tight text-white"
      />
    </div>
  );
}

function FriendSummary({
  friend,
  trailing,
}: {
  friend: FriendRow;
  trailing?: React.ReactNode;
}) {
  const fallback = (friend.displayName || friend.userId || "?").slice(0, 2).toUpperCase();
  return (
    <PlayerProfileLink
      userId={friend.userId}
      nickname={friend.displayName}
      className="group flex min-w-0 items-center gap-3 hover:opacity-80"
      title={`View ${friend.displayName || "player"} profile`}
    >
      <AvatarBadge
        avatarUrl={friend.avatarUrl}
        fallback={fallback}
        alt={friend.displayName || "Player"}
        size="sm"
        className="h-9 w-9 shrink-0 border border-white/10 bg-slate-800"
      />
      <span className="min-w-0">
        <span className="block truncate text-sm font-bold text-white group-hover:text-emerald-100">
          {friend.displayName || "Player"}
        </span>
        {friend.selectedBadge ? (
          <PlayerBadge badge={friend.selectedBadge as PlayerBadgeInfo} size="sm" className="mt-0.5" />
        ) : null}
      </span>
      {trailing}
    </PlayerProfileLink>
  );
}

function FriendSearch() {
  const social = useSocial();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<FriendRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [requested, setRequested] = useState<Set<string>>(new Set());

  const runSearch = async (value: string) => {
    setQuery(value);
    if (value.trim().length < 2) {
      setResults([]);
      return;
    }
    setLoading(true);
    try {
      const config = getRuntimeConfig();
      const found = await searchPlayers(config, social.accessToken, value.trim());
      setResults(found);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onSearchChange = (value: string) => {
    setQuery(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (value.trim().length < 2) {
      setResults([]);
      setLoading(false);
      return;
    }
    debounceRef.current = setTimeout(() => void runSearch(value), 300);
  };

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const alreadyFriend = (userId: string) =>
    social.friends.some((f) => f.userId === userId);
  const pendingOutgoing = (userId: string) =>
    social.outgoing.some((f) => f.userId === userId);

  const request = async (userId: string) => {
    setRequested((prev) => new Set(prev).add(userId));
    try {
      await social.sendRequest(userId);
    } finally {
      setRequested((prev) => {
        const next = new Set(prev);
        next.delete(userId);
        return next;
      });
    }
  };

  return (
    <LobbyInset density="lg">
      <SectionHeader icon={<UserPlus size={22} />} eyebrow="Add friends" />
      <div className="mt-1 space-y-3">
        <div className="relative ml-3 max-w-[360px]">
          <Search
            size={16}
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-white/40"
            aria-hidden="true"
          />
          <LobbyInput
            value={query}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search nickname..."
            aria-label="Search players by nickname"
            className="pl-9 !border-white/5 !bg-black/20"
          />
        </div>
        <div className="ml-3 space-y-2">
        {loading ? (
          <p className="text-xs font-semibold text-[#7f9a8f]">Searching...</p>
        ) : results.length === 0 && query.trim().length >= 2 ? (
          <p className="text-xs font-semibold text-[#7f9a8f]">No players found</p>
        ) : null}
        {results.map((player) => {
          const isFriend = alreadyFriend(player.userId);
          const isPending = pendingOutgoing(player.userId);
          const isRequested = requested.has(player.userId);
          return (
            <div
              key={player.userId}
              className="flex items-center justify-between gap-3 rounded-xl border border-white/5 bg-black/20 px-3 py-2"
            >
              <FriendSummary friend={player} />
              {isFriend ? (
                <LobbyPill tone="success">Friends</LobbyPill>
              ) : isPending || isRequested ? (
                <LobbyPill tone="blue">Requested</LobbyPill>
              ) : (
                <LobbyIconButton
                  aria-label={`Add ${player.displayName || "player"}`}
                  title="Send friend request"
                  onClick={() => request(player.userId)}
                  disabled={isRequested}
                >
                  <UserPlus size={16} aria-hidden="true" />
                </LobbyIconButton>
              )}
            </div>
          );
        })}
        </div>
      </div>
    </LobbyInset>
  );
}

function IncomingRequests() {
  const social = useSocial();
  if (social.incoming.length === 0) return null;
  return (
    <LobbyInset density="lg">
      <SectionHeader
        icon={<Bell size={22} />}
        eyebrow="Pending"
        title={`Friend requests (${social.incoming.length})`}
      />
      <ul className="mt-3 space-y-2">
        {social.incoming.map((requester) => (
          <li
            key={requester.userId}
            className="flex items-center justify-between gap-3 rounded-xl border border-white/5 bg-black/20 px-3 py-2"
          >
            <FriendSummary friend={requester} />
            <div className="flex items-center gap-2">
              <LobbyIconButton
                aria-label={`Accept ${requester.displayName || "request"}`}
                title="Accept"
                onClick={() => social.acceptRequest(requester.userId)}
                className="border-emerald-400/30 text-emerald-200 hover:bg-emerald-500/20"
              >
                <Check size={16} aria-hidden="true" />
              </LobbyIconButton>
              <LobbyIconButton
                aria-label={`Decline ${requester.displayName || "request"}`}
                title="Decline"
                onClick={() => social.declineRequest(requester.userId)}
                className="border-red-400/30 text-red-200 hover:bg-red-500/20"
              >
                <X size={16} aria-hidden="true" />
              </LobbyIconButton>
            </div>
          </li>
        ))}
      </ul>
    </LobbyInset>
  );
}

function FriendsList() {
  const social = useSocial();
  const canInvite = !!social.activeParty && social.activeParty.isOwner;
  const presenceOf = (id: string) => social.presence[id] ?? "offline";

  const sorted = [...social.friends].sort((a, b) => {
    const rank: Record<string, number> = { online: 0, away: 1, offline: 2 };
    const diff = rank[presenceOf(a.userId)] - rank[presenceOf(b.userId)];
    return diff !== 0 ? diff : a.displayName.localeCompare(b.displayName);
  });

  return (
    <LobbyInset density="lg">
      <SectionHeader
        icon={<UserCheck size={22} />}
        eyebrow="Your friends"
        title={`Friends (${social.friends.length})`}
        description={canInvite ? "Invite friends to your party from here." : undefined}
      />
      {social.friends.length === 0 ? (
        <p className="mt-3 pl-3 text-xs font-semibold text-[#7f9a8f]">
          You have not added any friends yet
        </p>
      ) : (
      <ul className="mt-3 space-y-2">
          {sorted.map((friend) => {
            const status = presenceOf(friend.userId);
            const statusTone =
              status === "online" ? "success" : status === "away" ? "warning" : "muted";
            return (
              <li
                key={friend.userId}
                className="flex items-center justify-between gap-3 rounded-xl border border-white/5 bg-black/20 px-3 py-2"
              >
                <FriendSummary
                  friend={friend}
                  trailing={
                    <LobbyPill tone={statusTone as "success" | "warning" | "muted"}>
                      {status}
                    </LobbyPill>
                  }
                />
                <div className="flex items-center gap-2">
                  {canInvite ? (
                    <LobbyIconButton
                      aria-label={`Invite ${friend.displayName || "friend"} to party`}
                      title="Invite to party"
                      onClick={() => social.sendPartyInvite(friend.userId)}
                    >
                      <UserPlus size={16} aria-hidden="true" />
                    </LobbyIconButton>
                  ) : null}
                  <LobbyIconButton
                    aria-label={`Remove ${friend.displayName || "friend"}`}
                    title="Remove friend"
                    onClick={() => social.removeFriend(friend.userId)}
                    className="border-red-400/30 text-red-200 hover:bg-red-500/20"
                  >
                    <X size={16} aria-hidden="true" />
                  </LobbyIconButton>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </LobbyInset>
  );
}

export function FriendsTab() {
  const social = useSocial();
  if (!social.enabled) {
    return (
      <LobbyInset density="lg">
        <p className="text-sm font-semibold text-[#7f9a8f]">
          Sign in to add friends and see who is online.
        </p>
      </LobbyInset>
    );
  }
  return (
    <div className="flex w-full max-w-[520px] flex-col gap-5">
      <FriendSearch />
      <IncomingRequests />
      <FriendsList />
    </div>
  );
}
