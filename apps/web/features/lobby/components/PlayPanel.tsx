import type React from "react";
import { forwardRef } from "react";
import { CheckCircle2, Loader2, Play } from "lucide-react";
import { motion } from "framer-motion";
import type { GameRuleset } from "../../matchmaking/lib/queue-client";
import { LobbyActionButton, LobbyPanel } from "./lobby-primitives";

const panelMotion = {
  initial: { opacity: 0, y: 16, scale: 0.97 },
  animate: { opacity: 1, y: 0, scale: 1 },
  exit: { opacity: 0, y: 10, scale: 0.97 },
  transition: { duration: 0.22, ease: [0.16, 1, 0.3, 1] as const },
};

type PlayPanelProps = {
  isQueueing: boolean;
  isSingleplayerLoading: boolean;
  queueError: string;
  queueRulesets: GameRuleset[];
  toggleQueueRuleset: (ruleset: GameRuleset) => void;
  onRankedPlay: () => void;
  cancelQueue: () => void;
  startSingleplayer: () => void | Promise<string>;
  duelDisabled: boolean;
  singleplayerDisabled: boolean;
  queuePaused: boolean;
  playPaused: boolean;
  maintenanceIsActive: boolean;
  primaryButtonLabel: string;
  queueElapsedLabel: string;
  duelModeLabel: string;
  sideCards: React.ReactNode;
};

export const PlayPanel = forwardRef<HTMLDivElement, PlayPanelProps>(function PlayPanel({
  isQueueing,
  isSingleplayerLoading,
  queueError,
  queueRulesets,
  toggleQueueRuleset,
  onRankedPlay,
  cancelQueue,
  startSingleplayer,
  duelDisabled,
  singleplayerDisabled,
  queuePaused,
  playPaused,
  maintenanceIsActive,
  primaryButtonLabel,
  queueElapsedLabel,
  duelModeLabel,
  sideCards,
}, ref) {
  return (
    <motion.div
      ref={ref}
      key="play"
      {...panelMotion}
      className="flex w-full max-w-[1160px] flex-col items-center gap-5 pointer-events-auto lg:grid lg:grid-cols-[minmax(0,480px)_minmax(280px,360px)] lg:items-start lg:justify-center lg:gap-6"
    >
      <div className="flex w-full max-w-[480px] flex-col gap-5 lg:max-w-none">
        <QueueModeCard
          isQueueing={isQueueing}
          queueError={queueError}
          queueRulesets={queueRulesets}
          toggleQueueRuleset={toggleQueueRuleset}
          onRankedPlay={onRankedPlay}
          cancelQueue={cancelQueue}
          duelDisabled={duelDisabled}
          queuePaused={queuePaused}
          playPaused={playPaused}
          maintenanceIsActive={maintenanceIsActive}
          primaryButtonLabel={primaryButtonLabel}
          queueElapsedLabel={queueElapsedLabel}
          duelModeLabel={duelModeLabel}
        />
        <SingleplayerModeCard
          isSingleplayerLoading={isSingleplayerLoading}
          singleplayerDisabled={singleplayerDisabled}
          playPaused={playPaused}
          maintenanceIsActive={maintenanceIsActive}
          startSingleplayer={startSingleplayer}
        />
      </div>

      <div className="flex w-full max-w-[480px] flex-col gap-5 lg:sticky lg:top-8 lg:max-w-none">
        {sideCards}
      </div>
    </motion.div>
  );
});

function QueueModeCard(props: {
  isQueueing: boolean;
  queueError: string;
  queueRulesets: GameRuleset[];
  toggleQueueRuleset: (ruleset: GameRuleset) => void;
  onRankedPlay: () => void;
  cancelQueue: () => void;
  duelDisabled: boolean;
  queuePaused: boolean;
  playPaused: boolean;
  maintenanceIsActive: boolean;
  primaryButtonLabel: string;
  queueElapsedLabel: string;
  duelModeLabel: string;
}) {
  return (
    <LobbyPanel className="lobby-feature-card relative flex w-full flex-col gap-4 p-5 transition-colors duration-500 sm:p-8">
      <div className={`absolute inset-0 pointer-events-none transition-opacity duration-500 ${props.isQueueing ? "opacity-95" : "opacity-80"} bg-[linear-gradient(180deg,rgba(72,128,106,0.28)_0%,rgba(22,42,34,0.78)_100%)]`} />
      <ModeMountains active={props.isQueueing} />
      <ModeHeading eyebrow={props.duelModeLabel} title="Duel" eyebrowClassName="text-[#8cb0a1]" />

      <div className="relative z-10 mx-auto mt-1 flex w-full flex-col px-0 pb-1 sm:mt-2 sm:px-2">
        {props.queueError ? <p className="mb-3 text-center text-xs font-semibold text-red-300">{props.queueError}</p> : null}
        {!props.isQueueing ? (
          <div className="mb-3 overflow-hidden rounded-xl border border-white/10 bg-black/25">
            {([
              ["moving", "Moving"],
              ["nmpz", "NMPZ"],
            ] as const).map(([ruleset, label]) => {
              const selected = props.queueRulesets.includes(ruleset);
              return (
                <button
                  key={ruleset}
                  type="button"
                  aria-pressed={selected}
                  onClick={() => props.toggleQueueRuleset(ruleset)}
                  className={`flex min-h-[44px] w-full items-center justify-between px-4 text-left text-[13px] font-extrabold uppercase tracking-[0.08em] transition ${selected ? "bg-[#22d385]/12 text-[#d7ffec]" : "text-white/70 hover:bg-white/[0.07] hover:text-white"}`}
                >
                  <span>{label}</span>
                  <span className="flex h-[18px] w-[18px] items-center justify-center">
                    {selected ? <CheckCircle2 size={18} strokeWidth={2.5} className="text-[#22d385]" /> : null}
                  </span>
                </button>
              );
            })}
          </div>
        ) : null}

        {!props.isQueueing ? (
          <LobbyActionButton onClick={props.onRankedPlay} disabled={props.duelDisabled} className="w-full rounded-2xl py-[14px] text-base transition-transform hover:scale-[1.01] active:scale-[0.98]">
            <Play fill="currentColor" size={20} className="mr-2.5" />
            {props.queuePaused || props.playPaused || props.maintenanceIsActive ? "Paused" : props.primaryButtonLabel}
          </LobbyActionButton>
        ) : (
          <LobbyActionButton onClick={props.cancelQueue} variant="secondary" className="group w-full rounded-2xl py-[14px] text-sm">
            <Loader2 size={18} className="mr-3 animate-spin text-[#2ad18f] transition-colors group-hover:text-[#3deb9e]" />
            <span className="text-accentPrimary">{props.queueElapsedLabel}</span>
          </LobbyActionButton>
        )}
      </div>
    </LobbyPanel>
  );
}

function SingleplayerModeCard(props: {
  isSingleplayerLoading: boolean;
  singleplayerDisabled: boolean;
  playPaused: boolean;
  maintenanceIsActive: boolean;
  startSingleplayer: () => void | Promise<string>;
}) {
  return (
    <LobbyPanel className="lobby-feature-card relative flex min-h-[240px] w-full flex-col justify-between p-5 transition-colors duration-500 sm:min-h-[260px] sm:p-8" style={{ animationDelay: "-2s" }}>
      <div className="absolute inset-0 pointer-events-none bg-[linear-gradient(180deg,rgba(72,106,128,0.28)_0%,rgba(22,34,42,0.85)_100%)] opacity-80 transition-opacity duration-500" />
      <ModeMountains hueRotate active={false} />
      <ModeHeading eyebrow="Casual" title="Singleplayer" subtitle="Moving allowed" eyebrowClassName="text-[#8caab0]" />

      <div className="relative z-10 mx-auto mt-5 flex h-full w-full flex-col justify-end px-0 pb-1 sm:mt-6 sm:px-2">
        <button
          type="button"
          onClick={() => props.startSingleplayer()}
          disabled={props.singleplayerDisabled}
          className="w-full flex items-center justify-center rounded-2xl bg-[#3b82f6] py-[14px] text-base font-extrabold uppercase tracking-[0.08em] text-white shadow-[0_4px_16px_rgba(59,130,246,0.3)] transition-all duration-200 hover:scale-[1.01] hover:bg-[#4b8df8] hover:shadow-[0_6px_24px_rgba(59,130,246,0.4)] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:scale-100"
        >
          {props.isSingleplayerLoading ? (
            <Loader2 size={20} className="mr-2.5 animate-spin" />
          ) : (
            <Play fill="currentColor" size={20} className="mr-2.5" />
          )}
          {props.isSingleplayerLoading ? "Loading..." : props.playPaused || props.maintenanceIsActive ? "Paused" : "Play"}
        </button>
      </div>
    </LobbyPanel>
  );
}

function ModeMountains({ active, hueRotate = false }: { active: boolean; hueRotate?: boolean }) {
  return (
    <div className={`absolute inset-x-0 bottom-0 pointer-events-none h-full transition-opacity duration-500 ${active ? "opacity-[0.24]" : "opacity-[0.32]"}`}>
      <img
        src="/mountains.v1.svg"
        alt=""
        aria-hidden="true"
        className={`absolute inset-0 h-full w-full object-cover object-center ${hueRotate ? "opacity-50" : ""}`}
        style={{ objectPosition: "center bottom", filter: hueRotate ? "hue-rotate(190deg)" : undefined }}
      />
    </div>
  );
}

function ModeHeading({
  eyebrow,
  title,
  subtitle,
  eyebrowClassName,
}: {
  eyebrow: string;
  title: string;
  subtitle?: string;
  eyebrowClassName: string;
}) {
  return (
    <div className="relative z-10 mt-1 flex flex-col sm:mt-2">
      <span className={`mb-1 text-xs font-bold uppercase tracking-[0.16em] drop-shadow-sm ${eyebrowClassName}`}>
        {eyebrow}
      </span>
      <h2 className="mb-2 text-[36px] font-extrabold leading-tight tracking-tight text-white drop-shadow-md sm:text-[44px]">
        {title}
      </h2>
      {subtitle ? <span className="text-[15px] font-medium text-white/90 drop-shadow-sm sm:text-base">{subtitle}</span> : null}
    </div>
  );
}
