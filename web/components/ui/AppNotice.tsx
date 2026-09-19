import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { IconButton } from "./button";
import { Notice } from "./patterns";

const AUTO_HIDE_MS = 4000;

type AppNoticeContextValue = {
  show: (message: string) => void;
};

const AppNoticeContext = createContext<AppNoticeContextValue>({
  show: () => undefined,
});

export function AppNoticeProvider({ children }: { children: ReactNode }) {
  const reducedMotion = useReducedMotion();
  const [notice, setNotice] = useState<{ id: number; message: string } | null>(null);
  const nextId = useRef(0);

  const show = useCallback((message: string) => {
    const text = message.trim();
    if (!text) return;
    nextId.current += 1;
    setNotice({ id: nextId.current, message: text });
  }, []);

  const dismiss = useCallback(() => setNotice(null), []);

  useEffect(() => {
    if (!notice) return;
    const timer = window.setTimeout(dismiss, AUTO_HIDE_MS);
    return () => window.clearTimeout(timer);
  }, [dismiss, notice]);

  const value = useMemo(() => ({ show }), [show]);

  return (
    <AppNoticeContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed inset-x-0 top-3 z-popover flex justify-center px-4">
        <AnimatePresence mode="wait">
          {notice ? (
            <motion.div
              key={notice.id}
              initial={reducedMotion ? { opacity: 0 } : { opacity: 0, y: -12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={reducedMotion ? { opacity: 0 } : { opacity: 0, y: -8 }}
              transition={{ duration: reducedMotion ? 0.12 : 0.22, ease: [0.16, 1, 0.3, 1] }}
              className="pointer-events-auto w-full max-w-md"
            >
              <Notice
                tone="muted"
                action={
                  <IconButton aria-label="Dismiss notice" size="icon-sm" onClick={dismiss} className="shrink-0">
                    <X size={14} />
                  </IconButton>
                }
              >
                {notice.message}
              </Notice>
            </motion.div>
          ) : null}
        </AnimatePresence>
      </div>
    </AppNoticeContext.Provider>
  );
}

export function useAppNotice() {
  return useContext(AppNoticeContext);
}

/** Show an overlay notice whenever `message` becomes a non-empty string. */
export function useShowAppNoticeOnValue(message: string | null | undefined) {
  const { show } = useAppNotice();
  useEffect(() => {
    const text = message?.trim();
    if (text) show(text);
  }, [message, show]);
}

/** Show an overlay notice when `active` is true. Pass `key` to re-fire the same copy. */
export function useShowAppNoticeWhen(active: boolean, message: string, key?: number) {
  const { show } = useAppNotice();
  useEffect(() => {
    if (active) show(message);
  }, [active, key, message, show]);
}
