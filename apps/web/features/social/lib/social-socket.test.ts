import { describe, expect, it } from "vitest";
import { diffPresence } from "./social-socket";

describe("diffPresence", () => {
  it("reports all entries as changed when previous is empty", () => {
    const next = { a: "online", b: "offline" };
    expect(diffPresence({}, next)).toEqual(next);
  });

  it("omits unchanged statuses and includes only changed ones", () => {
    const previous = { a: "online", b: "offline" };
    const next = { a: "online", b: "away", c: "online" };
    expect(diffPresence(previous, next)).toEqual({ b: "away", c: "online" });
  });

  it("treats a missing previous value as offline", () => {
    const previous = { a: "online" };
    expect(diffPresence(previous, { a: "online" })).toEqual({});
    expect(diffPresence(previous, { a: "offline" })).toEqual({ a: "offline" });
  });
});
