import { useEffect, useState } from 'react';

type Props = {
  avatarUrl?: string;
  fallback: string;
  alt: string;
  opponent?: boolean;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
  avatarColor?: string;
};

const sizeClass: Record<NonNullable<Props['size']>, string> = {
  sm: 'h-9 w-9 text-sm',
  md: 'h-11 w-11 text-base',
  lg: 'h-14 w-14 text-lg',
  xl: 'h-20 w-20 text-2xl'
};

export default function AvatarBadge({
  avatarUrl,
  fallback,
  alt,
  opponent = false,
  size = 'md',
  className = '',
  avatarColor
}: Props) {
  const [imgFailed, setImgFailed] = useState(false);

  useEffect(() => {
    setImgFailed(false);
  }, [avatarUrl]);

  const base = avatarColor
    ? ''
    : opponent
      ? 'bg-gradient-to-br from-orange-300 via-orange-500 to-red-600'
      : 'bg-gradient-to-br from-emerald-200 via-emerald-400 to-teal-500';

  return (
    <div
      className={`relative grid aspect-square shrink-0 place-items-center overflow-hidden rounded-full border border-white/20 ${base} ${sizeClass[size]} ${className}`}
      style={avatarColor ? { backgroundColor: avatarColor } : undefined}
    >
      {avatarUrl && !imgFailed ? (
        // Using img keeps this simple for remote avatar URLs.
        <img
          src={avatarUrl}
          alt={alt}
          className="h-full w-full object-cover"
          onError={() => setImgFailed(true)}
        />
      ) : (
        <span className={`font-extrabold ${avatarColor ? 'text-white font-hud' : 'text-slate-900'}`}>{fallback.slice(0, 1).toUpperCase()}</span>
      )}
    </div>
  );
}
