import { useEffect, useState, type SetStateAction } from "react";
import type {
  GameRuleset,
  StreetNamesVisibility,
} from "../../matchmaking/lib/queue-client";

export type DuelStreetNamesChoice = StreetNamesVisibility | "any";

type DuelPreferences = {
  modes: GameRuleset[];
  streetNames: DuelStreetNamesChoice;
};

type SingleplayerPreferences = {
  mode: GameRuleset;
  streetNames: StreetNamesVisibility;
};

const DUEL_STORAGE_KEY = "geoduels.play.duels";
const SINGLEPLAYER_STORAGE_KEY = "geoduels.play.singleplayer";
const LEGACY_STORAGE_KEY = "geoduels.queueRulesets";
const DUEL_CONFIGURED_KEY = "geoduels.play.duels.configured";
const SINGLEPLAYER_CONFIGURED_KEY = "geoduels.play.singleplayer.configured";

const supportedModes = new Set<GameRuleset>(["moving", "no_move", "nmpz"]);

function parseModes(value: unknown): GameRuleset[] {
  if (!Array.isArray(value)) return [];
  return Array.from(
    new Set(value.filter((item): item is GameRuleset => supportedModes.has(item))),
  );
}

function readPreferences() {
  let duel: DuelPreferences = { modes: ["moving"], streetNames: "shown" };
  let singleplayer: SingleplayerPreferences = {
    mode: "moving",
    streetNames: "shown",
  };
  try {
    const storedDuel = JSON.parse(
      window.localStorage.getItem(DUEL_STORAGE_KEY) || "null",
    );
    if (storedDuel && typeof storedDuel === "object") {
      duel = {
        modes: parseModes(storedDuel.modes),
        streetNames:
          storedDuel.streetNames === "hidden" ||
          storedDuel.streetNames === "any"
            ? storedDuel.streetNames
            : "shown",
      };
    } else {
      const legacy = parseModes(
        JSON.parse(window.localStorage.getItem(LEGACY_STORAGE_KEY) || "null"),
      ).filter((mode) => mode !== "no_move");
      if (legacy.length) duel.modes = legacy;
    }

    const storedSingleplayer = JSON.parse(
      window.localStorage.getItem(SINGLEPLAYER_STORAGE_KEY) || "null",
    );
    if (
      storedSingleplayer &&
      typeof storedSingleplayer === "object" &&
      supportedModes.has(storedSingleplayer.mode)
    ) {
      singleplayer = {
        mode: storedSingleplayer.mode,
        streetNames:
          storedSingleplayer.streetNames === "hidden" ? "hidden" : "shown",
      };
    }
  } catch {
  }
  if (!duel.modes.length) duel.modes = ["moving"];
  return { duel, singleplayer };
}

function readConfigured(key: string) {
  try {
    return window.localStorage.getItem(key) === "1";
  } catch {
    return false;
  }
}

export function usePlayPreferences() {
  const [initial] = useState(readPreferences);
  const [duel, setDuelState] = useState<DuelPreferences>(initial.duel);
  const [singleplayer, setSingleplayerState] =
    useState<SingleplayerPreferences>(initial.singleplayer);
  const [duelConfigured, setDuelConfigured] = useState(() =>
    readConfigured(DUEL_CONFIGURED_KEY),
  );
  const [singleplayerConfigured, setSingleplayerConfigured] = useState(() =>
    readConfigured(SINGLEPLAYER_CONFIGURED_KEY),
  );

  useEffect(() => {
    try {
      window.localStorage.setItem(DUEL_STORAGE_KEY, JSON.stringify(duel));
      window.localStorage.removeItem(LEGACY_STORAGE_KEY);
      window.localStorage.setItem(
        SINGLEPLAYER_STORAGE_KEY,
        JSON.stringify(singleplayer),
      );
    } catch {
    }
  }, [duel, singleplayer]);

  const markDuelConfigured = () => {
    setDuelConfigured(true);
    try {
      window.localStorage.setItem(DUEL_CONFIGURED_KEY, "1");
    } catch {
    }
  };

  const markSingleplayerConfigured = () => {
    setSingleplayerConfigured(true);
    try {
      window.localStorage.setItem(SINGLEPLAYER_CONFIGURED_KEY, "1");
    } catch {
    }
  };

  const setDuel = (updater: SetStateAction<DuelPreferences>) => {
    setDuelState(updater);
    markDuelConfigured();
  };

  const setSingleplayer = (
    updater: SetStateAction<SingleplayerPreferences>,
  ) => {
    setSingleplayerState(updater);
    markSingleplayerConfigured();
  };

  return {
    duel,
    setDuel,
    singleplayer,
    setSingleplayer,
    duelConfigured,
    singleplayerConfigured,
    markDuelConfigured,
    markSingleplayerConfigured,
  };
}
