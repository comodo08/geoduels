import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import { HorizontalScroller } from "../HorizontalScroller";

const POINTER_ID = 1;

beforeAll(() => {
  const captured = new WeakMap<Element, Set<number>>();
  const captureSet = (element: Element) => {
    let ids = captured.get(element);
    if (!ids) {
      ids = new Set<number>();
      captured.set(element, ids);
    }
    return ids;
  };
  Element.prototype.setPointerCapture = function setPointerCapture(pointerId: number) {
    captureSet(this).add(pointerId);
  };
  Element.prototype.hasPointerCapture = function hasPointerCapture(pointerId: number) {
    return captureSet(this).has(pointerId);
  };
  Element.prototype.releasePointerCapture = function releasePointerCapture(pointerId: number) {
    captureSet(this).delete(pointerId);
  };
});

afterEach(cleanup);

type PointerInit = {
  pointerId?: number;
  pointerType?: string;
  button?: number;
  clientX?: number;
  clientY?: number;
};

function pointerEvent(type: string, init: PointerInit = {}) {
  const event = new MouseEvent(type, {
    bubbles: true,
    cancelable: true,
    button: init.button ?? 0,
    clientX: init.clientX ?? 0,
    clientY: init.clientY ?? 0,
  });
  Object.defineProperty(event, "pointerId", { value: init.pointerId ?? POINTER_ID });
  Object.defineProperty(event, "pointerType", { value: init.pointerType ?? "mouse" });
  return event;
}

function dispatch(target: Element, event: Event) {
  act(() => {
    target.dispatchEvent(event);
  });
}

function clickEvent() {
  return new MouseEvent("click", { bubbles: true, cancelable: true, detail: 1 });
}

function renderScroller({ draggable = true }: { draggable?: boolean } = {}) {
  render(
    <HorizontalScroller label="Trending Maps" itemClassName="w-80" draggable={draggable}>
      <div>Card A</div>
      <div>Card B</div>
    </HorizontalScroller>,
  );
  return screen.getByRole("region", { name: "Trending Maps" });
}

describe("HorizontalScroller drag", () => {
  it("scrolls the viewport when a mouse drag passes the threshold", () => {
    const viewport = renderScroller();
    expect(viewport).toHaveClass("cursor-grab");

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 200, clientY: 100 }));

    expect(viewport.scrollLeft).toBe(100);
    expect(viewport).toHaveClass("cursor-grabbing");
  });

  it("ignores movement that stays under the drag threshold", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 296, clientY: 100 }));

    expect(viewport.scrollLeft).toBe(0);
    expect(viewport).not.toHaveClass("cursor-grabbing");
  });

  it("abandons a mostly-vertical gesture so the page can still scroll", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 290, clientY: 140 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 220, clientY: 140 }));

    expect(viewport.scrollLeft).toBe(0);
    expect(viewport).not.toHaveClass("cursor-grabbing");
  });

  it("does not drag for non-primary buttons or non-mouse pointers", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100, button: 2 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 100, clientY: 100 }));
    expect(viewport.scrollLeft).toBe(0);

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100, pointerType: "touch" }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 100, clientY: 100 }));
    expect(viewport.scrollLeft).toBe(0);
  });

  it("suppresses only the click that follows a drag", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 200, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointerup", { clientX: 200, clientY: 100 }));

    const afterDrag = clickEvent();
    dispatch(viewport, afterDrag);
    expect(afterDrag.defaultPrevented).toBe(true);

    const laterClick = clickEvent();
    dispatch(viewport, laterClick);
    expect(laterClick.defaultPrevented).toBe(false);
  });

  it("leaves clicks alone when no drag happened", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointerup", { clientX: 300, clientY: 100 }));

    const click = clickEvent();
    dispatch(viewport, click);
    expect(click.defaultPrevented).toBe(false);
  });

  it("ends the drag when the pointer capture is lost", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 200, clientY: 100 }));
    expect(viewport).toHaveClass("cursor-grabbing");

    dispatch(viewport, pointerEvent("lostpointercapture", { clientX: 200, clientY: 100 }));

    expect(viewport).not.toHaveClass("cursor-grabbing");
    expect(viewport).toHaveClass("cursor-grab");
  });

  it("re-applies the threshold after a lost capture instead of dragging from a stale origin", () => {
    const viewport = renderScroller();

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 200, clientY: 100 }));
    expect(viewport.scrollLeft).toBe(100);

    dispatch(viewport, pointerEvent("lostpointercapture", { clientX: 200, clientY: 100 }));

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 500, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 503, clientY: 100 }));

    expect(viewport.scrollLeft).toBe(100);
    expect(viewport).not.toHaveClass("cursor-grabbing");
  });

  it("does not attach drag handling when draggable is false", () => {
    const viewport = renderScroller({ draggable: false });
    expect(viewport).not.toHaveClass("cursor-grab");

    dispatch(viewport, pointerEvent("pointerdown", { clientX: 300, clientY: 100 }));
    dispatch(viewport, pointerEvent("pointermove", { clientX: 100, clientY: 100 }));

    expect(viewport.scrollLeft).toBe(0);
  });
});
