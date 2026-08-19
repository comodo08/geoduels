import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { getRuntimeConfig } from "../../lib/runtime-config";
import { markUserNotificationRead, requestUserNotifications } from "../auth/lib/auth-client";
import {
  LobbyActionButton,
  LobbyPill,
} from "../lobby/components/lobby-primitives";
import {
  acceptFriendRequest,
  declineFriendRequest,
  listFriendRequests,
  listFriends,
  removeFriend,
  sendFriendRequest,
  sendPartyInvite,
  type FriendRow,
  type SocialPresenceStatus,
} from "./lib/friends-client";
import { SocialSocket } from "./lib/social-socket";

export type ActiveParty = {
  id: string;
  inviteCode: string;
  isOwner: boolean;
  memberIds: string[];
} | null;

export type PartyInviteEvent = {
  notificationId?: number;
  partyId: string;
  inviteCode: string;
  inviterId?: string;
  inviterName?: string;
};

type SocialContextValue = {
  enabled: boolean;
  accessToken: string;
  friends: FriendRow[];
  incoming: FriendRow[];
  outgoing: FriendRow[];
  presence: Record<string, SocialPresenceStatus>;
  activeParty: ActiveParty;
  partyInvite: PartyInviteEvent | null;
  refresh: () => Promise<void>;
  sendRequest: (targetUserId: string) => Promise<void>;
  acceptRequest: (requesterId: string) => Promise<void>;
  declineRequest: (requesterId: string) => Promise<void>;
  removeFriend: (friendUserId: string) => Promise<void>;
  sendPartyInvite: (friendUserId: string) => Promise<void>;
  dismissPartyInvite: () => void;
  acceptPartyInvite: (inviteCode: string) => void;
  invitedFriends: Set<string>;
  markInvited: (friendUserId: string) => void;
};

const defaultContextValue: SocialContextValue = {
  enabled: false,
  accessToken: "",
  friends: [],
  incoming: [],
  outgoing: [],
  presence: {},
  activeParty: null,
  partyInvite: null,
  refresh: async () => {},
  sendRequest: async () => {},
  acceptRequest: async () => {},
  declineRequest: async () => {},
  removeFriend: async () => {},
  sendPartyInvite: async () => {},
  dismissPartyInvite: () => {},
  acceptPartyInvite: () => {},
  invitedFriends: new Set<string>(),
  markInvited: () => {},
};

const SocialContext = createContext<SocialContextValue>(defaultContextValue);

export function useSocial() {
  return useContext(SocialContext);
}

const DISMISSED_INVITES_MAX = 50;

function dismissedInvitesStorageKey(userId: string): string {
  return `geoduels.dismissedPartyInvites.${userId || "anon"}`;
}

function loadDismissedInvites(userId: string): Set<string> {
  try {
    const raw = localStorage.getItem(dismissedInvitesStorageKey(userId));
    if (!raw) return new Set();
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return new Set();
    return new Set(parsed.filter((v): v is string => typeof v === "string"));
  } catch {
    return new Set();
  }
}

function saveDismissedInvites(userId: string, ids: Set<string>) {
  try {
    const trimmed = [...ids].slice(-DISMISSED_INVITES_MAX);
    localStorage.setItem(dismissedInvitesStorageKey(userId), JSON.stringify(trimmed));
  } catch {
    // Ignore persistence failures; the in-memory set still protects this session.
  }
}

function inviteIdentity(inviteCode?: string, inviterId?: string): string {
  return `${String(inviteCode ?? "").toLowerCase()}|${String(inviterId ?? "")}`;
}

type SocialProviderProps = {
  accessToken: string;
  userId: string;
  isGuest: boolean;
  activeParty: ActiveParty;
  joinParty?: (inviteCode?: string) => Promise<boolean>;
  children: ReactNode;
};

export function SocialProvider({
  accessToken,
  userId,
  isGuest,
  activeParty,
  joinParty,
  children,
}: SocialProviderProps) {
  const config = useMemo(() => getRuntimeConfig(), []);
  const enabled = !!accessToken && !isGuest && !!userId;

  const [friends, setFriends] = useState<FriendRow[]>([]);
  const [incoming, setIncoming] = useState<FriendRow[]>([]);
  const [outgoing, setOutgoing] = useState<FriendRow[]>([]);
  const [presence, setPresence] = useState<Record<string, SocialPresenceStatus>>({});
  const [partyInvite, setPartyInvite] = useState<PartyInviteEvent | null>(null);
  const [invitedFriends, setInvitedFriends] = useState<Set<string>>(new Set());
  const prevPartyMembersRef = useRef<Set<string>>(new Set());
  const shownInviteIdRef = useRef<number | undefined>(undefined);
  const dismissedInvitesRef = useRef<Set<string>>(loadDismissedInvites(userId));

  const showPartyInvite = useCallback(
    (notificationId: number | undefined, payload: Record<string, unknown>) => {
      if (notificationId !== undefined) shownInviteIdRef.current = notificationId;
      setPartyInvite({
        notificationId,
        partyId: String(payload.partyId ?? ""),
        inviteCode: String(payload.inviteCode ?? ""),
        inviterId: payload.inviterId ? String(payload.inviterId) : undefined,
        inviterName: payload.inviterName ? String(payload.inviterName) : undefined,
      });
    },
    [],
  );

  // Shows an incoming party invite only if it hasn't already been dismissed
  // (by invite code + inviter) and isn't already on screen. Keeping the
  // dismissed set in localStorage prevents a dismissed invite from reappearing
  // after a page refresh or a re-invite to the same party.
  const presentPartyInvite = useCallback(
    (notificationId: number | undefined, payload: Record<string, unknown>) => {
      const inviteCode = payload.inviteCode ? String(payload.inviteCode) : undefined;
      const inviterId = payload.inviterId ? String(payload.inviterId) : undefined;
      if (dismissedInvitesRef.current.has(inviteIdentity(inviteCode, inviterId))) {
        return;
      }
      if (notificationId !== undefined && shownInviteIdRef.current === notificationId) {
        return;
      }
      showPartyInvite(notificationId, payload);
    },
    [showPartyInvite],
  );

  const clearPartyInvite = useCallback(() => {
    shownInviteIdRef.current = undefined;
    setPartyInvite(null);
  }, []);

  useEffect(() => {
    const current = new Set(activeParty?.memberIds ?? []);
    const previous = prevPartyMembersRef.current;
    if (previous.size > 0) {
      const left = [...previous].filter((id) => !current.has(id));
      if (left.length > 0) {
        setInvitedFriends((prev) => {
          let changed = false;
          const next = new Set(prev);
          for (const id of left) {
            if (next.delete(id)) changed = true;
          }
          return changed ? next : prev;
        });
      }
    }
    prevPartyMembersRef.current = current;
  }, [activeParty?.memberIds]);

  const markInvited = useCallback((friendUserId: string) => {
    setInvitedFriends((prev) => {
      if (prev.has(friendUserId)) return prev;
      const next = new Set(prev);
      next.add(friendUserId);
      return next;
    });
  }, []);

  const refresh = useCallback(async () => {
    if (!enabled) return;
    try {
      const [friendList, requests, notifResp] = await Promise.all([
        listFriends(config, accessToken),
        listFriendRequests(config, accessToken),
        requestUserNotifications(config, accessToken),
      ]);
      setFriends(friendList);
      setIncoming(requests.incoming);
      setOutgoing(requests.outgoing);
      const pendingInvite = notifResp.notifications.find((n) => n.type === "party_invite");
      if (pendingInvite) {
        presentPartyInvite(
          pendingInvite.id,
          (pendingInvite.payload ?? {}) as Record<string, unknown>,
        );
      }
    } catch {
      // Presence and lists recover on the next refresh tick.
    }
  }, [config, accessToken, enabled, presentPartyInvite]);

  const socketRef = useRef<SocialSocket | null>(null);

  useEffect(() => {
    if (!enabled) return;
    const socket = new SocialSocket();
    socketRef.current = socket;

    const off = socket.onMessage((event) => {
      if (event.type === "presence") {
        setPresence((prev) => ({ ...prev, ...(event.payload as Record<string, SocialPresenceStatus>) }));
      } else if (event.type === "friend_request" || event.type === "friend_accepted") {
        void refresh();
      } else if (event.type === "party_invite") {
        const payload = event.payload as Record<string, unknown>;
        const notificationId =
          typeof payload.notificationId === "number" ? payload.notificationId : undefined;
        presentPartyInvite(notificationId, payload);
        void refresh();
      } else if (event.type === "party_invite_dismissed") {
        const friendUserId = String(event.payload.friendUserId ?? "");
        if (friendUserId) {
          setInvitedFriends((prev) => {
            if (!prev.has(friendUserId)) return prev;
            const next = new Set(prev);
            next.delete(friendUserId);
            return next;
          });
        }
      }
    });

    const controller = new AbortController();
    socket.connect(config, accessToken, controller.signal);

    return () => {
      off();
      controller.abort();
      socket.close();
      socketRef.current = null;
    };
  }, [config, accessToken, enabled, refresh]);

  useEffect(() => {
    if (enabled) void refresh();
  }, [enabled, refresh]);

  const sendRequest = useCallback(
    async (targetUserId: string) => {
      await sendFriendRequest(config, accessToken, targetUserId);
      await refresh();
    },
    [config, accessToken, refresh],
  );

  const acceptRequest = useCallback(
    async (requesterId: string) => {
      await acceptFriendRequest(config, accessToken, requesterId);
      await refresh();
    },
    [config, accessToken, refresh],
  );

  const declineRequest = useCallback(
    async (requesterId: string) => {
      await declineFriendRequest(config, accessToken, requesterId);
      await refresh();
    },
    [config, accessToken, refresh],
  );

  const removeFriendCb = useCallback(
    async (friendUserId: string) => {
      await removeFriend(config, accessToken, friendUserId);
      await refresh();
    },
    [config, accessToken, refresh],
  );

  const sendPartyInviteCb = useCallback(
    async (friendUserId: string) => {
      if (!activeParty || !activeParty.isOwner) {
        throw new Error("You must own a party to invite friends");
      }
      await sendPartyInvite(config, accessToken, activeParty.id, friendUserId);
    },
    [config, accessToken, activeParty],
  );

  // Marks a party invite as consumed so it can't resurface on the next
  // notification refresh. Accepting an invite must do this too, otherwise the
  // still-unread notification is re-fetched by refresh() and shown again (and
  // again) — producing several invites for what is really a single invite.
  const consumePartyInvite = useCallback(
    (invite: PartyInviteEvent | null) => {
      if (!invite) return;
      if (invite.notificationId) {
        void markUserNotificationRead(config, accessToken, invite.notificationId).catch(
          () => {},
        );
      }
      const identity = inviteIdentity(invite.inviteCode, invite.inviterId);
      if (identity && !dismissedInvitesRef.current.has(identity)) {
        const next = new Set(dismissedInvitesRef.current);
        next.add(identity);
        dismissedInvitesRef.current = next;
        saveDismissedInvites(userId, next);
      }
    },
    [config, accessToken, userId],
  );

  const dismissPartyInvite = useCallback(() => {
    consumePartyInvite(partyInvite);
    clearPartyInvite();
  }, [clearPartyInvite, consumePartyInvite, partyInvite]);

  const acceptPartyInvite = useCallback(
    (inviteCode: string) => {
      consumePartyInvite(partyInvite);
      clearPartyInvite();
      void joinParty?.(inviteCode);
    },
    [clearPartyInvite, consumePartyInvite, joinParty, partyInvite],
  );

  const value: SocialContextValue = {
    enabled,
    accessToken,
    friends,
    incoming,
    outgoing,
    presence,
    activeParty,
    partyInvite,
    invitedFriends,
    markInvited,
    refresh,
    sendRequest,
    acceptRequest,
    declineRequest,
    removeFriend: removeFriendCb,
    sendPartyInvite: sendPartyInviteCb,
    dismissPartyInvite,
    acceptPartyInvite,
  };

  return (
    <SocialContext.Provider value={value}>
      {children}
      {partyInvite ? (
        <div className="pointer-events-none fixed top-24 right-6 z-50 w-[min(92vw,360px)]">
          <div className="pointer-events-auto flex flex-col gap-3 rounded-2xl border border-white/10 bg-black/40 p-4 shadow-2xl">
            <div className="flex items-center gap-2">
              <LobbyPill tone="blue">Party invite</LobbyPill>
              <p className="text-sm font-bold text-white">
                {partyInvite.inviterName
                  ? `${partyInvite.inviterName} invited you to a party`
                  : "You were invited to a party"}
              </p>
            </div>
            <div className="flex items-center justify-end gap-2">
              <LobbyActionButton
                variant="ghost"
                size="sm"
                onClick={dismissPartyInvite}
              >
                Dismiss
              </LobbyActionButton>
              <LobbyActionButton
                variant="primary"
                size="sm"
                onClick={() => acceptPartyInvite(partyInvite.inviteCode)}
              >
                Join party
              </LobbyActionButton>
            </div>
          </div>
        </div>
      ) : null}
    </SocialContext.Provider>
  );
}
