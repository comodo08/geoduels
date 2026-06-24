export type Player = {
  userId: string;
  email?: string;
  displayName: string;
  avatarUrl?: string;
  mmr: number;
  gamesPlayed: number;
  wins: number;
  rankedGamesPlayed: number;
  isGuest?: boolean;
  isAdmin: boolean;
  isModerator: boolean;
  isBanned: boolean;
  banReason?: string;
  bannedAt?: string;
  lastIpAddress?: string;
  reportMutedUntil?: string;
  identities?: AdminUserIdentity[];
};

export type AdminUserIdentity = {
  provider: string;
  providerUserId: string;
  email?: string;
  providerName?: string;
  lastSeenAt?: string;
  deletedAt?: string;
};

export type ModerationCase = {
  id: number;
  targetUserId: string;
  targetDisplayName: string;
  status: string;
  queue?: string;
  source?: string;
  priority: string;
  score: number;
  riskScore?: number;
  riskBreakdown?: Record<string, unknown>;
  confidence?: number;
  reportCount: number;
  uniqueReporterCount: number;
  categories: Record<string, number>;
  assignedTo?: string;
  latestActivityAt: string;
};

export type PlayerReport = {
  id: number;
  matchId: string;
  reporterUserId: string;
  reporterName: string;
  category: string;
  reason?: string;
  reporterWeight: number;
  createdAt: string;
};

export type ModerationEvidence = {
  id: number;
  evidenceType: string;
  matchId?: string;
  roundId?: string;
  detectorVersion?: string;
  ruleId?: string;
  score: number;
  weight: number;
  payload?: Record<string, unknown>;
};

export type ModerationTimelineItem = {
  id: number;
  eventType: string;
  actorUserId?: string;
  reasonCode?: string;
  body?: string;
  createdAt: string;
};

export type ModerationMatch = {
  matchId: string;
  mode?: string;
  startedAt?: string;
  endedAt?: string;
  winnerUserId?: string;
  roundCount: number;
  players: Array<{
    userId: string;
    displayName: string;
    totalScore: number;
    finalHp: number;
  }>;
};

export type MatchHistory = {
  matchId: string;
  mode: string;
  startedAt?: string;
  endedAt: string;
  winnerUserId?: string;
};

export type PlayerDetail = {
  player: Player;
  stats: {
    totalMatches: number;
    rankedMatches: number;
    duelMatches: number;
    singleplayerRuns: number;
    wins: number;
    losses: number;
  };
  eloHistory: Array<{
    date: string;
    mmr: number;
    delta: number;
    played: number;
  }>;
};

export type EnforcementAction = {
  id: number;
  targetUserId: string;
  targetName?: string;
  actorUserId?: string;
  actorName?: string;
  sourceCaseId?: number;
  actionType: string;
  reasonCode?: string;
  reasonNote?: string;
  createdAt: string;
};

export type UserRoleGrant = {
  userId: string;
  displayName?: string;
  email?: string;
  role: string;
  grantedBy?: string;
  grantedAt: string;
  reason?: string;
};

export type IPBan = {
  id: number;
  ipAddress: string;
  reason?: string;
  createdAt: string;
};
