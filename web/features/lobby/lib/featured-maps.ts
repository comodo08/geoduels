import type { CustomMap } from "../../maps/lib/maps-client";

export function createSeededRandom(seed: string) {
  let state = 2166136261;
  for (let index = 0; index < seed.length; index += 1) {
    state ^= seed.charCodeAt(index);
    state = Math.imul(state, 16777619);
  }
  return () => {
    state += 0x6d2b79f5;
    let value = state;
    value = Math.imul(value ^ (value >>> 15), value | 1);
    value ^= value + Math.imul(value ^ (value >>> 7), value | 61);
    return ((value ^ (value >>> 14)) >>> 0) / 4294967296;
  };
}

export function featuredMapDay(date = new Date()) {
  return date.toISOString().slice(0, 10);
}

function featuredMapWeight(map: CustomMap) {
  // Logarithmic weighting makes large regional maps more likely without letting
  // location count turn the result into a deterministic size ranking.
  return Math.max(1, Math.log1p(Math.max(0, map.locationCount)));
}

export function selectFeaturedOfficialMaps(
  maps: CustomMap[],
  limit: number,
  random: () => number = Math.random,
) {
  return maps
    .filter(
      (map) =>
        map.status === "ready" &&
        (map.official || map.system) &&
        !map.modeMoving &&
        !map.modeNoMove &&
        !map.modeNmpz,
    )
    .map((map) => ({
      map,
      // Weighted reservoir sampling without replacement. Every eligible map
      // retains a non-zero chance; higher weights only improve its odds.
      key: -Math.log(Math.max(Number.EPSILON, random())) / featuredMapWeight(map),
    }))
    .sort((left, right) => left.key - right.key)
    .slice(0, Math.max(0, limit))
    .map(({ map }) => map);
}

export function selectHomepageMaps(official: CustomMap[], awarded: CustomMap[], limit: number, random: () => number) {
 const unique=(maps:CustomMap[])=>[...new Map(maps.map(map=>[map.id,map])).values()];
 const shuffle=(maps:CustomMap[])=>{const items=[...maps];for(let i=items.length-1;i>0;i--){const j=Math.floor(random()*(i+1));[items[i],items[j]]=[items[j],items[i]];}return items;};
 const eligible=(map:CustomMap)=>map.status==="ready";
 const winners=unique(awarded.filter(eligible));
 const current=winners.find(map=>map.motwCurrent);
 const cap=Math.floor(limit/2);
 const selectedWinners=[...(current&&cap>0?[current]:[]),...shuffle(winners.filter(map=>map.id!==current?.id))].slice(0,cap);
 const selectedIDs=new Set(selectedWinners.map(map=>map.id));
 const regular=shuffle(unique(official.filter(map=>eligible(map)&&!map.motwCount&&!selectedIDs.has(map.id))));
 const rest=shuffle([...selectedWinners.filter(map=>map.id!==current?.id),...regular.slice(0,Math.max(0,limit-selectedWinners.length))]);
 return [...(current&&selectedIDs.has(current.id)?[current]:[]),...rest].slice(0,limit);
}
