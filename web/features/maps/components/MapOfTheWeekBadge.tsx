import { Flame } from "lucide-react";
import type { CustomMap } from "../lib/maps-client";
export function MapOfTheWeekBadge({map}:{map:Pick<CustomMap,"motwCount"|"motwCurrent"|"motwLastAt">}) {
 if(!map.motwCount)return null;
 const label=map.motwCurrent?"Current Map of the Week":`Previous Map of the Week${map.motwLastAt?` · ${new Date(map.motwLastAt).toLocaleDateString()}`:""}`;
 return <span title={label} aria-label={label} className={`inline-flex shrink-0 items-center ${map.motwCurrent?"text-brand-orange":"text-status-warning"}`}><Flame size={20} fill={map.motwCurrent?"currentColor":"none"}/></span>;
}
