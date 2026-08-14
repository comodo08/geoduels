import { useMemo, useRef, useState } from "react";
import {
  FloatingPortal,
  autoUpdate,
  flip,
  offset,
  shift,
  useClick,
  useDismiss,
  useFloating,
  useInteractions,
  useListNavigation,
  useRole,
} from "@floating-ui/react";
import { Check, Loader2, Plus } from "lucide-react";
import { circularIconButtonClassName } from "../../../components/ui/button";
import { CountryFlag } from "../../../components/ui/CountryFlag";
import { cn } from "../../../lib/cn";
import { COUNTRY_CODE_ALLOWLIST } from "../../../lib/country-flag";
import { COUNTRY_NAMES } from "../../../lib/country-names";

const FLAG_CODES = [...COUNTRY_CODE_ALLOWLIST].sort();

function countryName(code: string): string {
  return COUNTRY_NAMES[code] || code;
}

type FlagSelectProps = {
  value: string;
  onSelect: (code: string) => void;
  isSaving?: boolean;
  disabled?: boolean;
  className?: string;
};

export function FlagSelect({
  value,
  onSelect,
  isSaving = false,
  disabled = false,
  className,
}: FlagSelectProps) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [query, setQuery] = useState("");
  const [previewCode, setPreviewCode] = useState<string | null>(null);
  const listRef = useRef<Array<HTMLButtonElement | null>>([]);

  const resetMenu = () => {
    setQuery("");
    setActiveIndex(null);
    setPreviewCode(null);
  };

  const closeMenu = () => {
    setOpen(false);
    resetMenu();
  };

  const normalizedQuery = query.trim().toLowerCase();
  const filtered = useMemo(() => {
    if (!normalizedQuery) return FLAG_CODES;
    return FLAG_CODES.filter((code) => {
      const name = countryName(code);
      return (
        code.toLowerCase().startsWith(normalizedQuery) ||
        name.toLowerCase().startsWith(normalizedQuery)
      );
    });
  }, [normalizedQuery]);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange: (next) => {
      setOpen(next);
      if (!next) resetMenu();
    },
    placement: "bottom-start",
    whileElementsMounted: autoUpdate,
    middleware: [offset(6), flip({ padding: 8 }), shift({ padding: 8 })],
  });

  const click = useClick(context, { enabled: !disabled && !isSaving });
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "menu" });
  const listNav = useListNavigation(context, {
    listRef,
    activeIndex,
    onNavigate: setActiveIndex,
  });
  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
  ]);

  const displayCode = previewCode || value;
  const displayName = countryName(displayCode);

  const commit = (code: string) => {
    closeMenu();
    onSelect(code);
  };

  return (
    <div className={cn("relative inline-block", className)}>
      <button
        type="button"
        disabled={disabled || isSaving}
        aria-label={
          value ? `National flag: ${value}. Change flag.` : "Select national flag"
        }
        className={
          value
            ? "inline-flex h-10 min-h-10 w-10 items-center justify-center rounded-full p-0 text-white/75 transition hover:bg-white/[0.08] focus-visible:bg-white/[0.08] disabled:cursor-not-allowed disabled:opacity-50"
            : circularIconButtonClassName()
        }
        ref={refs.setReference}
        {...getReferenceProps()}
      >
        {isSaving ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : displayCode ? (
          <CountryFlag code={displayCode} size="md" title={displayName} />
        ) : (
          <Plus className="h-4 w-4" />
        )}
      </button>

      {open ? (
        <FloatingPortal>
          <div
            ref={refs.setFloating}
            style={floatingStyles}
            {...getFloatingProps()}
            className="glass-panel z-[2300] w-64 rounded-3xl p-3 shadow-2xl shadow-black/40"
          >
            <input
              autoFocus
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                setActiveIndex(null);
              }}
              placeholder="Search"
              aria-label="Search flags"
              className="mb-2.5 w-full rounded-xl border border-white/10 bg-slate-800/60 px-2.5 py-1.5 text-sm text-white placeholder:text-slate-500 outline-none transition focus:border-accentPrimary/40 focus:ring-1 focus:ring-accentPrimary/30"
            />
            <div className="grid max-h-60 grid-cols-6 gap-1 overflow-y-auto scroll-smooth overscroll-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
              {filtered.map((code, index) => {
                const name = countryName(code);
                const selected = code === value;
                return (
                  <button
                    key={code}
                    type="button"
                    ref={(element) => {
                      listRef.current[index] = element;
                    }}
                    title={name}
                    aria-label={`Flag ${name}`}
                    aria-current={selected || undefined}
                    className={cn(
                      "relative flex h-8 w-full items-center justify-center rounded-lg p-0.5 transition",
                      activeIndex === index
                        ? "bg-white/10 ring-1 ring-white/15"
                        : "hover:bg-white/[0.06]",
                    )}
                    {...getItemProps()}
                    onMouseEnter={() => setPreviewCode(code)}
                    onMouseLeave={() => setPreviewCode(null)}
                    onClick={() => commit(code)}
                  >
                    <CountryFlag code={code} title={name} />
                    {selected ? (
                      <span className="absolute -bottom-1 -right-1 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-accentPrimary text-slate-900">
                        <Check className="h-2.5 w-2.5" strokeWidth={3} />
                      </span>
                    ) : null}
                  </button>
                );
              })}
              {!filtered.length ? (
                <p className="col-span-6 px-1 py-3 text-center text-xs text-slate-500">
                  No matches
                </p>
              ) : null}
            </div>
            <button
              type="button"
              onClick={() => commit("")}
              className="mt-3 w-full rounded-xl border border-white/10 bg-white/[0.03] px-2 py-2 text-xs font-semibold text-slate-300 transition hover:bg-white/10"
            >
              None
            </button>
          </div>
        </FloatingPortal>
      ) : null}
    </div>
  );
}
