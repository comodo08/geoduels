import { avatarSource, defaultAvatars, type AvatarKind } from "../../../lib/avatar";
import { useEffect, useState } from 'react';

type Props = {
  avatarUrl?: string;
  fallback: string;
  alt: string;
  opponent?: boolean;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
  avatarColor?: string;
  kind?: AvatarKind;
};

const sizeClass: Record<NonNullable<Props['size']>, string> = {
  sm: 'h-9 w-9 text-body-sm',
  md: 'h-11 w-11 text-body',
  lg: 'h-14 w-14 text-heading-sm',
  xl: 'h-20 w-20 text-heading-md'
};

export default function AvatarBadge({
  avatarUrl,
  fallback,
  alt,
  opponent = false,
  size = 'md',
  className = '',
  avatarColor,
  kind = "player"
}: Props) {
  const [imgFailed, setImgFailed] = useState(false);

  useEffect(() => {
    setImgFailed(false);
  }, [avatarUrl]);

  const base = avatarColor
    ? ''
    : opponent
      ? 'bg-gradient-to-br from-brand-orange via-brand-orange to-status-danger'
      : 'bg-gradient-to-br from-action-primary via-action-primary to-status-success';

  return (
    <div
      className={`relative grid place-items-center overflow-hidden rounded-full border border-border-strong ${base} ${sizeClass[size]} ${className}`}
      style={avatarColor ? { backgroundColor: avatarColor } : undefined}
    >
      <img src={imgFailed ? defaultAvatars[kind] : avatarSource(avatarUrl,kind)} alt={alt} className="h-full w-full object-cover" onError={()=>setImgFailed(true)} />
    </div>
  );
}
