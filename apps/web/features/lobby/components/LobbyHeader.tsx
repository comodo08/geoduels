import Link from "next/link";
import type React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { HelpCircle, Shield } from "lucide-react";
import AvatarBadge from "../../../components/ui/AvatarBadge";
import PlayerBadge, { type PlayerBadgeInfo } from "../../../components/ui/PlayerBadge";
import PlayerNameWithBadge from "../../../components/ui/PlayerNameWithBadge";
import { RatingTrophyIcon } from "../../../components/ui/PlayerIdentity";
import type { LobbyContentRoute } from "../lib/lobby-ui";
import { NAV_ITEMS, lobbyRouteStorageKey } from "../lib/lobby-ui";

export function LobbyHeader({
  displayName,
  isAdmin,
  isQueueing,
  maintenanceBanner,
  mmr,
  selectedBadge,
  setOpenHelp,
  setOpenProfile,
  showPartyPanel,
  signInButton,
  userAvatar,
  userAvatarFallback,
  userEmail,
  userId,
  visualNavIndex,
  visualNavRoute,
  currentNavRoute,
}: {
  displayName: string;
  isAdmin: boolean;
  isQueueing: boolean;
  maintenanceBanner: React.ReactNode;
  mmr: number;
  selectedBadge: PlayerBadgeInfo | null;
  setOpenHelp: () => void;
  setOpenProfile: () => void;
  showPartyPanel: boolean;
  signInButton: React.ReactNode;
  userAvatar?: string;
  userAvatarFallback: string;
  userEmail: string;
  userId: string;
  visualNavIndex: number;
  visualNavRoute: LobbyContentRoute;
  currentNavRoute: LobbyContentRoute;
}) {
  return (
    <header className="sticky top-0 z-20 px-4 pb-4 pt-4 sm:px-6 sm:pb-5 sm:pt-5 lg:px-8 lg:pb-6 lg:pt-6">
      <AnimatePresence>{maintenanceBanner}</AnimatePresence>
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-3 sm:gap-4 lg:gap-6">
        <div className="flex items-center gap-3 sm:gap-5">
          <button onClick={setOpenHelp} className="text-[#a9bfd4] transition-colors hover:text-white" aria-label="Help">
            <HelpCircle size={20} strokeWidth={2} className="sm:h-[22px] sm:w-[22px]" />
          </button>
          {isAdmin ? (
            <Link
              href="/admin"
              prefetch={false}
              className="inline-flex items-center gap-2 rounded-full border border-[#2ad18f]/35 bg-[#2ad18f]/10 px-3 py-1.5 text-[11px] font-bold uppercase tracking-[0.12em] text-white transition hover:bg-[#2ad18f]/18 sm:text-[12px]"
            >
              <Shield size={14} />
              Admin
            </Link>
          ) : null}
        </div>

        <div className="flex min-w-0 items-center justify-center">
          <Link href="/" aria-label="GeoDuels home" className="inline-flex">
            <img src="/logo.v2.png" alt="GeoDuels" width={140} height={38} className="h-auto w-[112px] sm:w-[140px]" />
          </Link>
        </div>

        {userId && userEmail ? (
          <button
            type="button"
            className="group flex min-w-0 items-center justify-self-end gap-2.5 cursor-pointer sm:gap-3"
            onClick={setOpenProfile}
          >
            <div className="flex min-w-0 max-w-[7.5rem] flex-col items-end justify-center sm:max-w-none">
              <PlayerNameWithBadge
                name={displayName || userEmail || "Player"}
                isAdmin={isAdmin}
                selectedBadge={null}
                nameClassName="text-[12px] font-bold leading-tight text-white transition-colors group-hover:text-emerald-100 sm:text-[15px]"
              />
              <div className="mt-0.5 flex items-center text-[10px] font-bold text-[#2ad18f] sm:text-[12px]">
                <RatingTrophyIcon className="mr-1 h-3 w-3" />
                {mmr}
                <PlayerBadge badge={selectedBadge} size="sm" className="ml-1" />
              </div>
            </div>
            <AvatarBadge
              avatarUrl={userAvatar}
              fallback={userAvatarFallback}
              alt={displayName || userEmail || "Player"}
              size="sm"
              className="h-9 w-9 border-[1.5px] border-white/20 bg-[#162130] transition-colors group-hover:border-white/40 sm:h-[42px] sm:w-[42px]"
            />
          </button>
        ) : (
          <div className="pointer-events-auto justify-self-end">{signInButton}</div>
        )}
      </div>

      {!showPartyPanel ? (
        <div className="flex justify-center pt-5 sm:pt-6">
          <div className="relative flex h-9 w-full max-w-[340px] items-center justify-center pointer-events-auto sm:h-10 sm:max-w-[400px] lg:max-w-[440px]">
            {NAV_ITEMS.map((item, idx) => {
              const isActive = item.route === visualNavRoute;
              const offset = idx - visualNavIndex;

              return (
                <motion.div
                  key={item.route}
                  initial={false}
                  animate={{
                    x: offset * 104,
                    scale: isActive ? 1.05 : 0.95,
                    opacity: isActive ? 1 : 0.4,
                  }}
                  transition={{ type: "spring", stiffness: 350, damping: 35 }}
                  className={`absolute font-bold text-[15px] tracking-[0.18em] transition-colors duration-200 sm:text-[16px] lg:text-[17px] ${
                    isQueueing ? "cursor-not-allowed text-[#a9bfd4]/50" : "cursor-pointer"
                  }`}
                  style={{
                    color: isActive ? (isQueueing ? "#8cb0a1" : "#ffffff") : "#a9bfd4",
                    transformOrigin: "center",
                  }}
                >
                  {isQueueing ? (
                    <span className="cursor-not-allowed">{item.label}</span>
                  ) : (
                    <Link
                      href={item.href}
                      onClick={() => {
                        try {
                          window.sessionStorage.setItem(lobbyRouteStorageKey, currentNavRoute);
                        } catch {
                          // Navigation still works if session storage is unavailable.
                        }
                      }}
                    >
                      {item.label}
                    </Link>
                  )}
                </motion.div>
              );
            })}
          </div>
        </div>
      ) : null}
    </header>
  );
}
