import { AnimatePresence, motion } from 'framer-motion';

type Props = {
  mm: string;
  ss: string;
  isRoundTimerRunning?: boolean;
  timerProgressPct: number;
  isTimerCritical: boolean;
  isTimerPulseActive: boolean;
  hasTopCompass?: boolean;
};

export default function GameHUD({
  mm,
  ss,
  isRoundTimerRunning = true,
  timerProgressPct,
  isTimerCritical,
  isTimerPulseActive,
  hasTopCompass = false,
}: Props) {
  const ringColor = isTimerCritical ? '#ff6d42' : '#2ad18f';
  const progress = Number(Math.max(0, Math.min(100, timerProgressPct)).toFixed(2));
  const width = 120;
  const height = 48;
  const stroke = 4;
  const inset = stroke / 2;
  const radius = (height - stroke) / 2;
  const centerX = width / 2;
  const rightX = width - inset - radius;
  const leftX = inset + radius;
  const topY = inset;
  const bottomY = height - inset;
  const progressPath = `M ${centerX} ${topY} H ${leftX} A ${radius} ${radius} 0 0 0 ${leftX} ${bottomY} H ${rightX} A ${radius} ${radius} 0 0 0 ${rightX} ${topY} H ${centerX}`;

  return (
    <div
      className={`pointer-events-none absolute inset-x-0 top-[91px] z-40 animate-hudSlideIn ${
        hasTopCompass ? "md:top-14" : "md:top-4"
      }`}
    >
      <div className="absolute left-1/2 -translate-x-1/2">
        <div className="flex items-center gap-2.5 md:gap-3">
          <AnimatePresence initial={false}>
            {isRoundTimerRunning && (
              <motion.div
                key="round-timer"
                className="relative shrink-0 overflow-visible"
                initial={{ width: 0, opacity: 0, x: 10 }}
                animate={{ width, opacity: 1, x: 0 }}
                exit={{ width: 0, opacity: 0, x: 10 }}
                transition={{ duration: 0.22, ease: 'easeOut' }}
                style={{ height }}
              >
                <div className="relative shrink-0" style={{ width, height }}>
                  {isTimerPulseActive && (
                    <div
                      data-testid="timer-pulse-glow"
                      className="pointer-events-none absolute inset-0 rounded-pill animate-timerCritical"
                    />
                  )}
                  <div
                    data-testid="timer-pill"
                    className="font-hud relative grid place-items-center rounded-pill shadow-elev-2 backdrop-blur-hud bg-hudBg tracking-[0.08em] text-ink overflow-hidden"
                    style={{ width, height, fontSize: 20 }}
                  >
                    <svg
                      className="pointer-events-none absolute inset-0"
                      viewBox={`0 0 ${width} ${height}`}
                      aria-hidden="true"
                    >
                      <path
                        d={progressPath}
                        fill="none"
                        stroke={ringColor}
                        strokeWidth={stroke}
                        strokeLinecap="round"
                        strokeDasharray={`${progress} 100`}
                        pathLength={100}
                      />
                    </svg>
                    <span className="relative z-10">
                      {mm}:{ss}
                    </span>
                  </div>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}
