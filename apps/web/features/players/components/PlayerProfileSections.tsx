import { Check, Loader2, Pencil, Settings, X } from "lucide-react";
import AvatarBadge from "../../../components/ui/AvatarBadge";
import { Button } from "../../../components/ui/button";
import { IconMetric } from "../../../components/ui/IconMetric";
import { Input } from "../../../components/ui/input";
import { MmrDisplay } from "../../../components/ui/MmrDisplay";
import PlayerBadge from "../../../components/ui/PlayerBadge";
import { Surface } from "../../../components/ui/Surface";
import { Tabs } from "../../../components/ui/Tabs";
import { Textarea } from "../../../components/ui/textarea";
import type { useProfileEditor } from "../hooks/use-profile-editor";
import type { PlayerMatchSummary, PublicPlayerProfile } from "../types";
import { CountryFlag } from "../../../components/ui/CountryFlag";
import {
  MatchHistoryRow,
  ProfileBadgeCollection,
  profileMetrics,
} from "./PlayerProfilePrimitives";
import { FlagSelect } from "./FlagSelect";

type Editor = ReturnType<typeof useProfileEditor>;

export function ProfileOverview({
  profile,
  editor,
  owner,
  onSettings,
}: {
  profile: PublicPlayerProfile;
  editor: Editor;
  owner: boolean;
  onSettings: () => void;
}) {
  const winRate = profile.gamesPlayed
    ? Math.round((profile.wins / profile.gamesPlayed) * 100)
    : 0;
  const rankedWinRate = profile.rankedGamesPlayed
    ? Math.round((profile.rankedWins / profile.rankedGamesPlayed) * 100)
    : 0;
  const duelStats = [
    [profileMetrics.games, "Duels played", profile.gamesPlayed],
    [profileMetrics.wins, "Duel wins", profile.wins],
    [profileMetrics.winRate, "Duel win rate", `${winRate}%`],
  ] as const;
  const seasonStats = [
    [profileMetrics.rankedGames, "Season duels", profile.rankedGamesPlayed],
    [profileMetrics.rankedWins, "Season wins", profile.rankedWins],
    [profileMetrics.rankedWinRate, "Season win rate", `${rankedWinRate}%`],
  ] as const;

  return (
    <Surface variant="gameGlass" className="rounded-3xl p-5 sm:p-7">
      <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
        <AvatarBadge
          avatarUrl={profile.avatarUrl}
          fallback={(profile.displayName || "?").slice(0, 1).toUpperCase()}
          alt={profile.displayName}
          size="xl"
          className="shrink-0 border-white/20 bg-slate-900"
        />
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2.5">
            {editor.editingName ? (
              <Input
                variant="game"
                value={editor.nickname}
                onChange={(event) =>
                  editor.setNickname(event.target.value.slice(0, 14))
                }
                onKeyDown={(event) => {
                  if (event.key === "Escape") editor.cancelName();
                  if (event.key === "Enter" && editor.nickname.trim().length >= 2)
                    editor.saveName();
                }}
                autoFocus
                className="max-w-sm flex-1 text-2xl font-black sm:text-3xl"
              />
            ) : (
              <h1 className="truncate text-3xl font-black text-white sm:text-4xl">
                {profile.displayName}
              </h1>
            )}
            {owner ? (
              <FlagSelect
                value={editor.flagCode}
                onSelect={editor.selectFlag}
                isSaving={editor.flagMutation.isPending}
              />
            ) : (
              <CountryFlag code={profile.flagCode} size="md" />
            )}
            <PlayerBadge badge={profile.selectedBadge} size="md" />
            {owner ? (
              editor.editingName ? (
                <>
                  <Button
                    variant="icon"
                    size="icon"
                    aria-label="Save display name"
                    disabled={
                      editor.nickname.trim().length < 2 ||
                      editor.nicknameMutation.isPending
                    }
                    onClick={editor.saveName}
                    className="h-9 min-h-9 w-9"
                  >
                    {editor.nicknameMutation.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Check className="h-4 w-4" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Cancel display name"
                    onClick={editor.cancelName}
                    className="h-9 min-h-9 w-9"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </>
              ) : (
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Edit display name"
                  onClick={() => editor.setEditingName(true)}
                  className="h-9 min-h-9 w-9"
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              )
            ) : null}
          </div>
          <MmrDisplay value={profile.mmr} size="md" label className="mt-3" />
          {owner ? (
            <MutationError
              mutation={editor.flagMutation}
              fallback="Failed to update national flag"
              className="mt-2"
            />
          ) : null}
          <MutationError
            mutation={editor.nicknameMutation}
            fallback="Failed to update display name"
          />
        </div>
        {owner ? (
          <Button onClick={onSettings} className="shrink-0 rounded-xl">
            <Settings className="h-4 w-4" />
            Account settings
          </Button>
        ) : null}
      </div>
      <div className="mt-9 grid grid-cols-2 gap-2.5 sm:grid-cols-3 lg:grid-cols-5">
        <div className="col-span-2 sm:col-span-3">
          {owner && editor.editingAbout ? (
            <div className="flex flex-col gap-2">
              <Textarea
                variant="game"
                value={editor.about}
                maxLength={150}
                rows={4}
                onChange={(event) => editor.setAbout(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Escape") editor.cancelAbout();
                }}
                autoFocus
                placeholder="Tell the world about yourself"
                className="resize-none"
              />
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs font-semibold text-[#8caab0]">
                  {editor.about.length}/150
                </span>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label="Cancel bio"
                    onClick={editor.cancelAbout}
                    disabled={editor.aboutMutation.isPending}
                  >
                    <X className="h-3.5 w-3.5" />
                    Cancel
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    aria-label="Save bio"
                    onClick={editor.saveAbout}
                    disabled={editor.aboutMutation.isPending}
                  >
                    {editor.aboutMutation.isPending ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Check className="h-3.5 w-3.5" />
                    )}
                    Save
                  </Button>
                </div>
              </div>
            </div>
          ) : profile.about ? (
            <div className="flex items-start gap-2">
              <p className="min-w-0 break-words whitespace-pre-wrap text-sm font-medium leading-normal text-white">
                {profile.about}
              </p>
              {owner ? (
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Edit bio"
                  onClick={() => editor.setEditingAbout(true)}
                  className="-mt-0.5 h-6 min-h-6 w-6 shrink-0"
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              ) : null}
            </div>
          ) : owner ? (
            <div className="flex items-center gap-2">
              <p className="text-sm text-[#8caab0]">No bio yet.</p>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Edit bio"
                onClick={() => editor.setEditingAbout(true)}
                className="h-8 min-h-8 w-8 shrink-0"
              >
                <Pencil className="h-4 w-4" />
              </Button>
            </div>
          ) : null}
          <MutationError
            mutation={editor.aboutMutation}
            fallback="Failed to update bio"
            className="mb-3"
          />
        </div>
      </div>
      <div className="mt-6 border-t border-white/10 pt-5">
        <div className="flex flex-col gap-2.5 lg:flex-row lg:items-stretch">
          <div className="grid flex-1 grid-cols-2 gap-2.5 sm:grid-cols-3">
            {duelStats.map(([icon, label, value], index) => (
              <IconMetric
                key={label}
                icon={icon}
                label={label}
                value={value}
                className={index === 2 ? "col-span-2 sm:col-span-1" : undefined}
              />
            ))}
          </div>
          <div
            aria-hidden
            className="h-px w-full flex-none bg-white/10 lg:h-auto lg:w-px"
          />
          <div className="grid flex-1 grid-cols-2 gap-2.5 sm:grid-cols-3">
            {seasonStats.map(([icon, label, value], index) => (
              <IconMetric
                key={label}
                icon={icon}
                label={label}
                value={value}
                className={index === 2 ? "col-span-2 sm:col-span-1" : undefined}
              />
            ))}
          </div>
        </div>
      </div>
    </Surface>
  );
}

export function ProfileBadges({
  profile,
  editor,
  owner,
}: {
  profile: PublicPlayerProfile;
  editor: Editor;
  owner: boolean;
}) {
  return (
    <Surface variant="gameGlass" className="rounded-3xl p-4 sm:p-6">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-black text-white">Earned badges</h2>
        {owner ? (
          editor.choosingBadge ? (
            <div className="flex gap-2">
              <Button variant="ghost" size="sm" onClick={editor.cancelBadge}>
                Cancel
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={editor.saveBadge}
                disabled={editor.badgeMutation.isPending}
              >
                {editor.badgeMutation.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : null}
                Save
              </Button>
            </div>
          ) : (
            <Button
              size="sm"
              onClick={() => editor.setChoosingBadge(true)}
            >
              Choose displayed badge
            </Button>
          )
        ) : null}
      </div>
      <MutationError
        mutation={editor.badgeMutation}
        fallback="Failed to update displayed badge"
        className="mb-3"
      />
      <ProfileBadgeCollection
        badges={profile.badges || []}
        editing={editor.choosingBadge}
        selectedBadgeId={
          editor.choosingBadge ? editor.badgeId : profile.selectedBadge?.id
        }
        onSelect={editor.setBadgeId}
      />
    </Surface>
  );
}

export function ProfileHistory({
  matches,
  query,
  filter,
  onFilterChange,
}: {
  matches: PlayerMatchSummary[];
  filter: "all" | "ranked";
  onFilterChange: (filter: "all" | "ranked") => void;
  query: {
    isLoading: boolean;
    isError: boolean;
    hasNextPage: boolean;
    isFetchingNextPage: boolean;
    fetchNextPage: () => unknown;
  };
}) {
  return (
    <Surface variant="gameGlass" className="rounded-3xl p-4 sm:p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <h2 className="text-xl font-black text-white">Match history</h2>
        <div className="flex flex-wrap items-center gap-3">
          <Tabs
            value={filter}
            onChange={onFilterChange}
            items={[
              { id: "all", label: "All" },
              { id: "ranked", label: "Ranked" },
            ]}
            className="grid-cols-2"
          />
          {matches.length ? (
            <span className="text-xs font-bold text-[#8caab0]">
              {matches.length} shown
            </span>
          ) : null}
        </div>
      </div>
      {query.isLoading ? <HistorySkeleton /> : null}
      {query.isError ? (
        <p className="mt-4 text-sm text-red-300">
          Match history is temporarily unavailable.
        </p>
      ) : null}
      {!query.isLoading && !query.isError && !matches.length ? (
        <p className="mt-4 text-sm text-[#8caab0]">
          {filter === "ranked"
            ? "No ranked matches yet."
            : "No match history yet."}
        </p>
      ) : null}
      <div className="mt-4 space-y-2">
        {matches.map((match) => (
          <MatchHistoryRow key={match.matchId} match={match} />
        ))}
      </div>
      {query.hasNextPage ? (
        <Button
          onClick={() => void query.fetchNextPage()}
          disabled={query.isFetchingNextPage}
          className="mt-4 w-full rounded-xl"
        >
          {query.isFetchingNextPage ? "Loading…" : "Load more"}
        </Button>
      ) : null}
    </Surface>
  );
}

function MutationError({
  mutation,
  fallback,
  className = "mt-2",
}: {
  mutation: { isError: boolean; error: unknown };
  fallback: string;
  className?: string;
}) {
  if (!mutation.isError) return null;
  return (
    <p className={`${className} text-xs font-semibold text-red-300`}>
      {mutation.error instanceof Error ? mutation.error.message : fallback}
    </p>
  );
}

function HistorySkeleton() {
  return (
    <div className="mt-4 space-y-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <div
          key={index}
          className="h-[76px] animate-pulse rounded-xl bg-white/[0.05]"
        />
      ))}
    </div>
  );
}
