import Link from "next/link";
import type { MouseEvent, ReactNode } from "react";
import { toPublicEntityId } from "../../lib/entity-id";

export default function PlayerProfileLink({
  userId,
  children,
  className = "",
  disabled = false,
  stopPropagation = false,
  title,
}: {
  userId?: string;
  children: ReactNode;
  className?: string;
  disabled?: boolean;
  stopPropagation?: boolean;
  title?: string;
}) {
  const handleClick = (event: MouseEvent) => {
    if (stopPropagation) event.stopPropagation();
  };
  if (!userId || disabled) {
    return <span className={className}>{children}</span>;
  }
  return (
    <Link
      href={`/players/${encodeURIComponent(toPublicEntityId(userId))}`}
      className={className}
      onClick={handleClick}
      title={title || "View player profile"}
    >
      {children}
    </Link>
  );
}
