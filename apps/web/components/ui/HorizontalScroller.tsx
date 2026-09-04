import * as React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { cn } from "../../lib/cn";
import { ButtonLink, IconButton } from "./button";

const dragThresholdPx = 6;

export function HorizontalScroller({
  label,
  children,
  className,
  itemClassName,
  viewAllHref,
  draggable = false,
}: {
  label: string;
  children: React.ReactNode;
  className?: string;
  itemClassName?: string;
  viewAllHref?: string;
  draggable?: boolean;
}) {
  const viewportRef = React.useRef<HTMLDivElement>(null);
  const [canScrollBack, setCanScrollBack] = React.useState(false);
  const [canScrollForward, setCanScrollForward] = React.useState(false);
  const [dragging, setDragging] = React.useState(false);

  const updateControls = React.useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    setCanScrollBack(viewport.scrollLeft > 1);
    setCanScrollForward(viewport.scrollLeft + viewport.clientWidth < viewport.scrollWidth - 1);
  }, []);

  React.useEffect(() => {
    updateControls();
    const viewport = viewportRef.current;
    if (!viewport) return;
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(updateControls);
    observer?.observe(viewport);
    window.addEventListener("resize", updateControls);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", updateControls);
    };
  }, [children, updateControls]);

  React.useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport || !draggable) return;

    let activePointerId: number | null = null;
    let startClientX = 0;
    let startClientY = 0;
    let startScrollLeft = 0;
    let draggingRow = false;
    let suppressNextClick: ((event: MouseEvent) => void) | null = null;

    const clearSuppressedClick = () => {
      if (suppressNextClick) {
        viewport.removeEventListener("click", suppressNextClick, { capture: true });
        suppressNextClick = null;
      }
    };

    const abortDrag = () => {
      draggingRow = false;
      viewport.style.scrollBehavior = "";
      setDragging(false);
    };

    const endDrag = (pointerId: number | null) => {
      if (!draggingRow) return;
      if (pointerId !== null && viewport.hasPointerCapture(pointerId)) {
        viewport.releasePointerCapture(pointerId);
      }
      abortDrag();
      clearSuppressedClick();
      suppressNextClick = (event: MouseEvent) => {
        if (event.detail === 0) return;
        event.preventDefault();
        event.stopPropagation();
      };
      viewport.addEventListener("click", suppressNextClick, { capture: true, once: true });
    };

    const onPointerDown = (event: PointerEvent) => {
      if (event.pointerType !== "mouse" || event.button !== 0) return;
      clearSuppressedClick();
      if (draggingRow) abortDrag();
      activePointerId = event.pointerId;
      startClientX = event.clientX;
      startClientY = event.clientY;
      startScrollLeft = viewport.scrollLeft;
    };

    const onPointerMove = (event: PointerEvent) => {
      if (event.pointerId !== activePointerId) return;
      const deltaX = event.clientX - startClientX;
      const deltaY = event.clientY - startClientY;
      if (!draggingRow) {
        if (Math.abs(deltaX) < dragThresholdPx && Math.abs(deltaY) < dragThresholdPx) return;
        if (Math.abs(deltaX) <= Math.abs(deltaY)) {
          activePointerId = null;
          return;
        }
        draggingRow = true;
        viewport.setPointerCapture(event.pointerId);
        viewport.style.scrollBehavior = "auto";
        setDragging(true);
      }
      viewport.scrollLeft = startScrollLeft - deltaX;
    };

    const onPointerEnd = (event: PointerEvent) => {
      if (event.pointerId !== activePointerId) return;
      activePointerId = null;
      endDrag(event.pointerId);
    };

    const onDragStart = (event: DragEvent) => {
      if (activePointerId !== null) event.preventDefault();
    };

    const onLostPointerCapture = (event: PointerEvent) => {
      if (event.pointerId !== activePointerId) return;
      activePointerId = null;
      abortDrag();
    };

    viewport.addEventListener("pointerdown", onPointerDown);
    viewport.addEventListener("pointermove", onPointerMove);
    viewport.addEventListener("pointerup", onPointerEnd);
    viewport.addEventListener("pointercancel", onPointerEnd);
    viewport.addEventListener("lostpointercapture", onLostPointerCapture);
    viewport.addEventListener("dragstart", onDragStart);
    return () => {
      viewport.removeEventListener("pointerdown", onPointerDown);
      viewport.removeEventListener("pointermove", onPointerMove);
      viewport.removeEventListener("pointerup", onPointerEnd);
      viewport.removeEventListener("pointercancel", onPointerEnd);
      viewport.removeEventListener("lostpointercapture", onLostPointerCapture);
      viewport.removeEventListener("dragstart", onDragStart);
      clearSuppressedClick();
      viewport.style.scrollBehavior = "";
    };
  }, [draggable]);

  const move = (direction: -1 | 1) => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const reducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    viewport.scrollBy({
      left: direction * Math.max(viewport.clientWidth * 0.82, 240),
      behavior: reducedMotion ? "auto" : "smooth",
    });
  };

  return (
    <div className={cn("min-w-0", className)}>
      <div className="mb-3 flex items-center justify-between gap-4">
        <h2 className="text-heading-md font-strong leading-heading text-content-primary">{label}</h2>
        <div className="flex shrink-0 items-center gap-2" aria-label={`${label} navigation`}>
          {viewAllHref ? (
            <ButtonLink href={viewAllHref} variant="ghost" size="sm">
              View all
            </ButtonLink>
          ) : null}
          <IconButton aria-label={`Previous ${label}`} size="icon-md" disabled={!canScrollBack} onClick={() => move(-1)}>
            <ChevronLeft size={17} />
          </IconButton>
          <IconButton aria-label={`Next ${label}`} size="icon-md" disabled={!canScrollForward} onClick={() => move(1)}>
            <ChevronRight size={17} />
          </IconButton>
        </div>
      </div>
      <div
        ref={viewportRef}
        role="region"
        aria-label={label}
        tabIndex={0}
        onScroll={updateControls}
        className={cn(
          "scrollbar-hidden overflow-x-auto overscroll-x-contain scroll-smooth focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-border-focus/60",
          draggable && "cursor-grab select-none",
          dragging && "cursor-grabbing",
        )}
      >
        <div className="flex w-max min-w-full snap-x snap-proximity gap-4 pb-1">
          {React.Children.map(children, (child) => (
            <div className={cn("snap-start", itemClassName)}>{child}</div>
          ))}
        </div>
      </div>
    </div>
  );
}
