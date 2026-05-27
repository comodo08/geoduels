export type PlayerBadgeInfo = {
  id: string;
  kind: string;
  label: string;
  description: string;
  imageUrl: string;
  rarity?: string;
  seasonId?: string;
  rank?: number;
  owned?: boolean;
  unobtainable?: boolean;
};

type PlayerBadgeProps = {
  badge?: PlayerBadgeInfo | null;
  size?: "sm" | "md" | "lg";
  muted?: boolean;
  className?: string;
};

const sizeClass = {
  sm: "h-7 w-7",
  md: "h-9 w-9",
  lg: "h-14 w-14",
};

const rankTextClass = {
  sm: "text-[10px]",
  md: "text-[12px]",
  lg: "text-[18px]",
};

export function badgeTitle(badge?: PlayerBadgeInfo | null) {
  if (!badge) return "";
  return `${badge.label}${badge.description ? ` - ${badge.description}` : ""}`;
}

export default function PlayerBadge({
  badge,
  size = "sm",
  muted = false,
  className = "",
}: PlayerBadgeProps) {
  if (!badge) return null;
  const rankLabel =
    badge.kind === "season_rank" && badge.rank ? `#${badge.rank}` : "";
  return (
    <span
      className={`relative inline-flex shrink-0 items-center justify-center ${sizeClass[size]} ${muted ? "grayscale opacity-45" : ""} ${className}`.trim()}
      aria-label={badgeTitle(badge)}
    >
      <img
        src={badge.imageUrl}
        alt=""
        className="h-full w-full object-contain drop-shadow-[0_4px_10px_rgba(0,0,0,0.35)]"
      />
      {rankLabel ? (
        <span
          className={`font-hud absolute inset-0 flex items-center justify-center font-black leading-none text-white drop-shadow-[0_1px_2px_rgba(0,0,0,0.9)] ${rankTextClass[size]}`}
        >
          {rankLabel}
        </span>
      ) : null}
    </span>
  );
}
