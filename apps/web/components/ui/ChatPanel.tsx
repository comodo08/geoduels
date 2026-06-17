import { AnimatePresence, motion } from 'framer-motion';
import { MessageCircle, Send, X } from 'lucide-react';
import { useEffect, useRef, useState, type FormEvent } from 'react';
import type { ChatEmote, ChatMessage } from './types';

const chatEmotes: Array<{ emote: ChatEmote; label: string; glyph: string }> = [
  { emote: 'skull', label: 'Skull', glyph: '💀' },
  { emote: 'sob', label: 'Sob', glyph: '😭' },
  { emote: 'thinking', label: 'Thinking', glyph: '🤔' },
  { emote: 'sunglasses', label: 'Sunglasses', glyph: '😎' }
];

function emoteGlyph(emote?: ChatEmote) {
  return chatEmotes.find((item) => item.emote === emote)?.glyph || '';
}

export default function ChatPanel({
  messages,
  selfUserId,
  onSendMessage,
  onSendEmote,
  className = "absolute left-3 top-24 z-40 w-[min(calc(100vw-1.5rem),21rem)] md:left-4 md:top-28",
}: {
  messages: ChatMessage[];
  selfUserId: string;
  onSendMessage: (body: string) => boolean;
  onSendEmote: (emote: ChatEmote) => boolean;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const [body, setBody] = useState('');
  const [previewMessage, setPreviewMessage] = useState<ChatMessage | null>(null);
  const [previewVisible, setPreviewVisible] = useState(false);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const initialMessagesSeenRef = useRef(false);
  const latestMessageIdRef = useRef<string | null>(null);
  const previewTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!open) return;
    const node = scrollRef.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [messages, open]);

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    });
  }, [open]);

  useEffect(() => {
    const latestMessage = messages[messages.length - 1];
    if (!initialMessagesSeenRef.current) {
      initialMessagesSeenRef.current = true;
      latestMessageIdRef.current = latestMessage?.id ?? null;
      return;
    }

    if (!latestMessage) {
      latestMessageIdRef.current = null;
      return;
    }

    if (latestMessageIdRef.current === latestMessage.id) return;
    latestMessageIdRef.current = latestMessage.id;

    if (latestMessage.senderUserId === selfUserId) return;

    if (previewTimerRef.current) clearTimeout(previewTimerRef.current);
    setPreviewMessage(latestMessage);
    setPreviewVisible(true);
    previewTimerRef.current = setTimeout(() => {
      setPreviewVisible(false);
      previewTimerRef.current = null;
    }, 4200);
  }, [messages, selfUserId]);

  useEffect(() => {
    return () => {
      if (previewTimerRef.current) clearTimeout(previewTimerRef.current);
    };
  }, []);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const sent = onSendMessage(body);
    if (sent) setBody('');
  };

  return (
    <div className={className}>
      {open ? (
        <div className="relative rounded-[14px] border border-white/10 bg-[rgba(7,12,18,0.75)] p-3 text-white">
          <button
            type="button"
            onClick={() => setOpen(false)}
            aria-label="Close chat"
            className="absolute right-3 top-4 z-10 flex h-8 w-8 items-center justify-center rounded-full text-white/75 transition hover:bg-white/10 hover:text-white"
          >
            <X size={15} strokeWidth={2.5} />
          </button>
          <div
            ref={scrollRef}
            className="scrollbar-hidden flex max-h-48 flex-col gap-1 overflow-y-auto pr-10 text-[13px] font-semibold leading-snug [text-shadow:0_1px_5px_rgba(0,0,0,0.7)]"
          >
            {messages.length === 0 ? (
              <p className="py-8 text-center text-sm font-semibold text-white/45 [text-shadow:none]">No messages yet</p>
            ) : (
              messages.map((message) => {
                const self = message.senderUserId === selfUserId;
                return (
                  <p key={message.id} className="break-words text-white/90">
                    <span className={`mr-1 font-bold ${self ? 'text-[#7effbd]' : 'text-[#9fd4ff]'}`}>
                      {message.senderDisplayName}
                    </span>
                    <span className={message.kind === 'emote' ? 'text-lg leading-none' : ''}>
                      {message.kind === 'emote' ? emoteGlyph(message.emote) : message.body}
                    </span>
                  </p>
                );
              })
            )}
          </div>
          <div className="mt-3 flex gap-2">
            {chatEmotes.map((item) => (
              <button
                key={item.emote}
                type="button"
                onClick={() => onSendEmote(item.emote)}
                aria-label={item.label}
                title={item.label}
                className="flex h-10 w-10 items-center justify-center rounded-full border border-white/10 bg-white/5 text-xl [text-shadow:0_1px_5px_rgba(0,0,0,0.65)] transition hover:bg-white/[0.12]"
              >
                {item.glyph}
              </button>
            ))}
          </div>
          <form onSubmit={submit} className="mt-3 flex gap-2">
            <input
              ref={inputRef}
              value={body}
              onChange={(event) => setBody(event.target.value.slice(0, 180))}
              maxLength={180}
              className="min-w-0 flex-1 rounded-[14px] border border-white/10 bg-black/20 px-3 py-2 text-sm font-semibold text-white outline-none placeholder:text-white/35 focus:border-[#22d385]/50"
              placeholder="Message"
            />
            <button
              type="submit"
              aria-label="Send message"
              disabled={!body.trim()}
              className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-accentPrimary text-white transition hover:bg-accentPrimaryDeep disabled:cursor-not-allowed disabled:opacity-45"
            >
              <Send size={16} strokeWidth={2.5} />
            </button>
          </form>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setOpen(true)}
          aria-label="Open chat"
          className="flex h-11 max-w-[min(calc(100vw-1.5rem),19rem)] items-center gap-2 rounded-pill border border-white/10 bg-[rgba(7,12,18,0.25)] px-3 text-left text-white/85 transition hover:bg-[rgba(7,12,18,0.35)] hover:text-white"
        >
          <MessageCircle size={17} strokeWidth={2.4} className="flex-shrink-0 text-white/65" />
          <span className="relative block min-w-0 flex-1 overflow-hidden pr-1 text-sm font-semibold leading-none">
            <AnimatePresence mode="wait" initial={false}>
              {previewVisible && previewMessage ? (
                <motion.span
                  key={previewMessage.id}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: 0.28, ease: 'easeOut' }}
                  className="block truncate"
                >
                  {previewMessage.kind === 'emote' ? emoteGlyph(previewMessage.emote) : previewMessage.body || ''}
                </motion.span>
              ) : (
                <motion.span
                  key="placeholder"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.2, ease: 'easeOut' }}
                  className="block truncate text-white/65"
                >
                  Message...
                </motion.span>
              )}
            </AnimatePresence>
          </span>
        </button>
      )}
    </div>
  );
}
