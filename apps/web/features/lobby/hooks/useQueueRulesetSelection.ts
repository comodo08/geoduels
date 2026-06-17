import { useEffect, useState } from "react";
import type { GameRuleset } from "../../matchmaking/lib/queue-client";

const storageKey = "geoduels.queueRulesets";

export function useQueueRulesetSelection() {
  const [queueRulesets, setQueueRulesets] = useState<GameRuleset[]>(["moving"]);

  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(storageKey);
      const parsed = raw ? JSON.parse(raw) : null;
      if (Array.isArray(parsed)) {
        const next = parsed.filter(
          (item): item is GameRuleset => item === "moving" || item === "nmpz",
        );
        setQueueRulesets(Array.from(new Set(next)));
      }
    } catch {
      setQueueRulesets(["moving"]);
    }
  }, []);

  useEffect(() => {
    try {
      window.localStorage.setItem(storageKey, JSON.stringify(queueRulesets));
    } catch {
      // Storage failures should not block queue defaults.
    }
  }, [queueRulesets]);

  const toggleQueueRuleset = (ruleset: GameRuleset) => {
    setQueueRulesets((current) => {
      if (current.includes(ruleset)) {
        return current.filter((item) => item !== ruleset);
      }
      return [...current, ruleset];
    });
  };

  return { queueRulesets, toggleQueueRuleset };
}
