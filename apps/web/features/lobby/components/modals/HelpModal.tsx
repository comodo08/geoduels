import type React from "react";
import AppModalShell from "../../../../components/ui/AppModalShell";
import { LobbyPanel } from "../lobby-primitives";

export function HelpModal(props: { onClose: () => void }) {
  return (
    <AppModalShell title="Help" onClose={props.onClose}>
      <div className="space-y-5 text-[15px] leading-relaxed text-[#a9bfd4]">
        <HelpCard title="1. Rules of the Game">
          You and your opponent will be dropped into the same random street
          view location somewhere in the world. Your goal is to figure out
          where you are and place your guess on the map.
        </HelpCard>
        <HelpCard title="2. How to Join">
          Click "PLAY" on the main menu to enter the matchmaking queue. We'll
          automatically find you an opponent with a similar skill rating (MMR).
        </HelpCard>
        <HelpCard title="3. How Duels Work">
          Both players start with 6,000 HP. The first person to guess starts a
          countdown timer. When the round ends, whoever is closer to the actual
          location deals damage to the other player based on the distance
          difference. The game ends when a player's HP hits 0!
        </HelpCard>
      </div>
    </AppModalShell>
  );
}

function HelpCard(props: { title: string; children: React.ReactNode }) {
  return (
    <LobbyPanel className="rounded-xl p-4">
      <h3 className="mb-2 font-bold text-white tracking-wide">{props.title}</h3>
      <p>{props.children}</p>
    </LobbyPanel>
  );
}
