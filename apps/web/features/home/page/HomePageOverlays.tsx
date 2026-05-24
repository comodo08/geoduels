import EndMatchOverlay from "../../../components/ui/EndMatchOverlay";
import NicknameOnboardingModal from "../../../components/home/NicknameOnboardingModal";
import GuestVerificationOverlay from "./GuestVerificationOverlay";
import type {
  HomeActions,
  HomeAuthView,
  HomeOverlaysView,
} from "../model/types";

type HomePageOverlaysProps = {
  auth: HomeAuthView;
  overlays: HomeOverlaysView;
  actions: Pick<
    HomeActions,
    | "setNicknameInput"
    | "submitOnboardingNickname"
    | "leaveGame"
    | "reportPlayer"
    | "startSingleplayer"
    | "dismissNotification"
    | "submitGuestVerificationToken"
    | "markGuestVerificationExpired"
    | "cancelGuestVerification"
  >;
};

export default function HomePageOverlays({
  auth,
  overlays,
  actions,
}: HomePageOverlaysProps) {
  const activeNotification = overlays.notifications?.[0];
  return (
    <>
      <NicknameOnboardingModal
        open={overlays.onboardingOpen}
        nicknameInput={auth.nicknameInput}
        nicknameError={auth.nicknameError}
        nicknameSaving={auth.nicknameSaving}
        onChangeNickname={actions.setNicknameInput}
        onSubmit={() => void actions.submitOnboardingNickname()}
      />
      <GuestVerificationOverlay
        verification={overlays.guestVerification}
        onToken={actions.submitGuestVerificationToken}
        onExpired={actions.markGuestVerificationExpired}
        onCancel={actions.cancelGuestVerification}
      />
      {activeNotification?.type === "mmr_refund" ? (
        <div className="fixed inset-0 z-[1000] flex items-center justify-center bg-black/70 px-4">
          <div className="w-full max-w-sm rounded-2xl border border-[#2ad18f]/30 bg-[#0b1620] p-5 text-white shadow-2xl">
            <p className="text-xs font-black uppercase tracking-[0.16em] text-[#2ad18f]">
              Rating refunded
            </p>
            <h2 className="mt-2 text-2xl font-black">
              +{activeNotification.payload.refundDelta || 0} MMR
            </h2>
            <p className="mt-3 text-sm leading-6 text-[#b9c9d8]">
              A player you lost to was banned for cheating. Your rating has been
              recalculated from your current MMR and refunded.
            </p>
            <button
              type="button"
              onClick={() =>
                void actions.dismissNotification(activeNotification.id)
              }
              className="mt-5 min-h-11 w-full rounded-xl bg-[#2ad18f] px-4 text-sm font-black text-[#08111b]"
            >
              Got it
            </button>
          </div>
        </div>
      ) : null}
      {overlays.endMatch.open && (
        <EndMatchOverlay
          onLeaveGame={actions.leaveGame}
          mode={overlays.endMatch.mode}
          outcome={overlays.endMatch.outcome}
          selfName={overlays.endMatch.selfName}
          opponentName={overlays.endMatch.opponentName}
          opponentUserId={overlays.endMatch.opponentUserId}
          selfElo={overlays.endMatch.selfElo}
          opponentElo={overlays.endMatch.opponentElo}
          selfEloDelta={overlays.endMatch.selfEloDelta}
          opponentEloDelta={overlays.endMatch.opponentEloDelta}
          selfHP={overlays.endMatch.selfHP}
          oppHP={overlays.endMatch.oppHP}
          selfAvatarUrl={overlays.endMatch.selfAvatarUrl}
          oppAvatarUrl={overlays.endMatch.oppAvatarUrl}
          selfFallback={overlays.endMatch.selfFallback}
          oppFallback={overlays.endMatch.oppFallback}
          selfAvatarColor={overlays.endMatch.selfAvatarColor}
          oppAvatarColor={overlays.endMatch.oppAvatarColor}
          selfIsAdmin={overlays.endMatch.selfIsAdmin}
          opponentIsAdmin={overlays.endMatch.opponentIsAdmin}
          selfSelectedBadge={overlays.endMatch.selfSelectedBadge}
          opponentSelectedBadge={overlays.endMatch.opponentSelectedBadge}
          totalScore={overlays.endMatch.totalScore}
          roundResults={overlays.endMatch.roundResults}
          resultPlayerNames={overlays.endMatch.resultPlayerNames}
          resultPlayerAvatars={overlays.endMatch.resultPlayerAvatars}
          resultPlayerFallbacks={overlays.endMatch.resultPlayerFallbacks}
          resultPlayerBorderColors={overlays.endMatch.resultPlayerBorderColors}
          participantsById={overlays.endMatch.participantsById}
          selfParticipant={overlays.endMatch.selfParticipant}
          opponentParticipant={overlays.endMatch.opponentParticipant}
          onReportPlayer={actions.reportPlayer}
          onPlayAgain={
            overlays.endMatch.mode === "singleplayer"
              ? actions.startSingleplayer
              : undefined
          }
          asPage
        />
      )}
    </>
  );
}
