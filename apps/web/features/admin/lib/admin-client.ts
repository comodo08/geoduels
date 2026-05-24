import { readError } from "../../../lib/http";
import type { RuntimeConfig } from "../../../lib/runtime-config";
import type { MaintenanceStatus } from "../../matchmaking/lib/queue-client";

export async function requestAdminBootstrap(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/bootstrap`, {
    method: "POST",
    credentials: "include",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to bootstrap admin access"));
  }
  return resp.json();
}

export async function requestAdminPlayers(
  config: RuntimeConfig,
  accessToken: string,
  query: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players?query=${encodeURIComponent(query)}`,
    {
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to search players"));
  }
  return resp.json();
}

export async function requestAdminPlayerDetail(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}`,
    {
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load player detail"));
  }
  return resp.json();
}

export async function requestAdminBanPlayer(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
  reason: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}/ban`,
    {
      method: "POST",
      headers: {
        "content-type": "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ reason }),
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to ban player"));
  }
}

export async function requestAdminUnbanPlayer(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}/unban`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to unban player"));
  }
}

export async function requestAdminClearReporterMute(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}/report-mute`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to unmute reporter"));
  }
}

export async function requestAdminPromoteModerator(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}/moderator`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to promote moderator"));
  }
}

export async function requestAdminDemoteModerator(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}/moderator`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to demote moderator"));
  }
}

export async function requestAdminModerationCases(
  config: RuntimeConfig,
  accessToken: string,
  status = "",
) {
  const qs = status ? `?view=${encodeURIComponent(status)}` : "";
  const resp = await fetch(`${config.apiURL}/v1/admin/moderation/cases${qs}`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load moderation cases"));
  }
  return resp.json();
}

export async function requestAdminClaimModerationCase(
  config: RuntimeConfig,
  accessToken: string,
  caseId: number,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/moderation/cases/${encodeURIComponent(caseId)}/claim`,
    { method: "POST", headers: { Authorization: `Bearer ${accessToken}` } },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to claim moderation case"));
  }
  return resp.json();
}

export async function requestAdminReleaseModerationCase(
  config: RuntimeConfig,
  accessToken: string,
  caseId: number,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/moderation/cases/${encodeURIComponent(caseId)}/release`,
    { method: "POST", headers: { Authorization: `Bearer ${accessToken}` } },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to release moderation case"));
  }
  return resp.json();
}

export async function requestAdminPlayerMatches(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/players/${encodeURIComponent(userId)}/matches`,
    { headers: { Authorization: `Bearer ${accessToken}` } },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load match history"));
  }
  return resp.json();
}

export async function requestAdminModerationCase(
  config: RuntimeConfig,
  accessToken: string,
  caseId: number,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/moderation/cases/${encodeURIComponent(caseId)}`,
    { headers: { Authorization: `Bearer ${accessToken}` } },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load moderation case"));
  }
  return resp.json();
}

export async function requestAdminModerationCaseAction(
  config: RuntimeConfig,
  accessToken: string,
  caseId: number,
  action: {
    actionType: string;
    reason?: string;
    status?: string;
    assignedTo?: string;
    muteUserId?: string;
    muteUntil?: string;
  },
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/moderation/cases/${encodeURIComponent(caseId)}/actions`,
    {
      method: "POST",
      headers: {
        "content-type": "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify(action),
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to update moderation case"));
  }
  return resp.json();
}

export async function requestAdminEnforcementActions(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/enforcement/actions`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load enforcement actions"));
  }
  return resp.json();
}

export async function requestAdminRoles(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/roles`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load roles"));
  }
  return resp.json();
}

export async function requestAdminGrantRole(
  config: RuntimeConfig,
  accessToken: string,
  payload: { userId: string; role: string; reason?: string },
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/roles`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(payload),
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to grant role"));
  }
}

export async function requestAdminRevokeRole(
  config: RuntimeConfig,
  accessToken: string,
  userId: string,
  role: string,
  reason = "",
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/roles/${encodeURIComponent(userId)}/${encodeURIComponent(role)}`,
    {
      method: "DELETE",
      headers: {
        "content-type": "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ reason }),
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to revoke role"));
  }
}

export async function requestAdminDebugTestReports(
  config: RuntimeConfig,
  accessToken: string,
  payload: {
    reportedUserId: string;
    count: number;
    category: string;
    reason?: string;
  },
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/debug/test-reports`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(payload),
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to send test reports"));
  }
  return resp.json() as Promise<{
    caseId: number;
    reportsCreated: number;
    reporterUserIds: string[];
  }>;
}

export async function requestAdminIPSignupBans(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/ip-signup-bans`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load IP bans"));
  }
  return resp.json();
}

export async function requestAdminAddIPSignupBan(
  config: RuntimeConfig,
  accessToken: string,
  ipAddress: string,
  reason: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/ip-signup-bans`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify({ ipAddress, reason }),
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to add IP ban"));
  }
}

export async function requestAdminRemoveIPSignupBan(
  config: RuntimeConfig,
  accessToken: string,
  ipAddress: string,
) {
  const resp = await fetch(
    `${config.apiURL}/v1/admin/ip-signup-bans/${encodeURIComponent(ipAddress)}`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${accessToken}` },
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to remove IP ban"));
  }
}

export async function requestAdminMaintenance(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/maintenance`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load maintenance"));
  }
  return resp.json() as Promise<MaintenanceStatus>;
}

export type AdminModerationSettings = {
  discordWebhookUrl: string;
};

export type AdminRankedSeasonSettings = {
  activeSeasonId: string;
};

export type AdminRankedSeasonRolloverResult = {
  previousSeasonId: string;
  activeSeasonId: string;
  badgesAwarded: number;
  playersSeeded: number;
};

export async function requestAdminModerationSettings(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/moderation/settings`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(
      await readError(resp, "Failed to load moderation settings"),
    );
  }
  return resp.json() as Promise<AdminModerationSettings>;
}

export async function requestAdminPutModerationSettings(
  config: RuntimeConfig,
  accessToken: string,
  settings: AdminModerationSettings,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/moderation/settings`, {
    method: "PUT",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(settings),
  });
  if (!resp.ok) {
    throw new Error(
      await readError(resp, "Failed to save moderation settings"),
    );
  }
  return resp.json() as Promise<AdminModerationSettings>;
}

export async function requestAdminRankedSeason(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/seasons`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load season settings"));
  }
  return resp.json() as Promise<AdminRankedSeasonSettings>;
}

export async function requestAdminRolloverRankedSeason(
  config: RuntimeConfig,
  accessToken: string,
  nextSeasonId: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/seasons/rollover`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify({ nextSeasonId }),
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to roll over season"));
  }
  return resp.json() as Promise<AdminRankedSeasonRolloverResult>;
}

export async function requestAdminPutMaintenance(
  config: RuntimeConfig,
  accessToken: string,
  status: MaintenanceStatus,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/maintenance`, {
    method: "PUT",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(status),
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to save maintenance"));
  }
  return resp.json() as Promise<MaintenanceStatus>;
}

export async function requestAdminClearMaintenance(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/maintenance`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to clear maintenance"));
  }
}

export async function requestAdminGetChangelog(
  config: RuntimeConfig,
  accessToken: string,
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/changelog`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to load changelog"));
  }
  return resp.json();
}

export async function requestAdminPutChangelog(
  config: RuntimeConfig,
  accessToken: string,
  content: { eyebrow: string; title: string; markdown: string },
) {
  const resp = await fetch(`${config.apiURL}/v1/admin/changelog`, {
    method: "PUT",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(content),
  });
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to save changelog"));
  }
}

export async function requestAdminUploadCurrentMap(
  config: RuntimeConfig,
  accessToken: string,
  file: File,
  mapKey = "a-source-world",
) {
  const body = new FormData();
  body.append("file", file);
  const resp = await fetch(
    `${config.apiURL}/v1/admin/maps/${encodeURIComponent(mapKey)}/upload`,
    {
      method: "POST",
      headers: { Authorization: `Bearer ${accessToken}` },
      body,
    },
  );
  if (!resp.ok) {
    throw new Error(await readError(resp, "Failed to upload map"));
  }
  return resp.json();
}
