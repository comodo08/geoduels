import ChatPanel from "../../../components/ui/ChatPanel";
import type { HomeActions, HomeChatView } from "../model/types";

type HomePageChatDockProps = {
  chat: HomeChatView;
  actions: Pick<HomeActions, "sendChatMessage" | "sendChatEmote">;
};

export default function HomePageChatDock({ chat, actions }: HomePageChatDockProps) {
  if (!chat.conversationId || !chat.selfUserId) return null;

  return (
    <div className="app-layer-chat fixed left-3 top-24 flex w-[min(calc(100vw-1.5rem),23rem)] flex-col items-start gap-2 md:left-4 md:top-28">
      <ChatPanel
        messages={chat.messages}
        selfUserId={chat.selfUserId}
        onSendMessage={actions.sendChatMessage}
        onSendEmote={actions.sendChatEmote}
        className="relative w-full"
      />
      {chat.opponentLeftNotice ? (
        <div className="pointer-events-none inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-black/20 px-3 py-1.5 text-[11px] font-bold text-amber-200/90">
          {chat.opponentLeftNoticeName || 'Opponent'} left the game
        </div>
      ) : null}
    </div>
  );
}
