import type { RuntimeConfig } from "../../../lib/runtime-config";
import { apiFetch, authHeaders, expectJSON, readError } from "../../../lib/http";
export type Nomination = {id: number; mapId: string; name: string; authorName: string; thumbnailKey: string; likes: number; liked: boolean; nominatedAt: string};
export type NominationPage = {items: Nomination[]; total: number; page: number; pageSize: number; closesAt: string};
export async function nominations(config: RuntimeConfig, token: string, page: number): Promise<NominationPage> {
 return expectJSON(await apiFetch(config, `/staff/motw?page=${page}`, {headers: authHeaders(token)}), "Nominations unavailable");
}
export async function nominate(config: RuntimeConfig, token: string, mapId: string) {
 const response = await apiFetch(config, `/staff/motw/maps/${encodeURIComponent(mapId)}`, {method: "POST", headers: authHeaders(token)});
 if (!response.ok) throw new Error(await readError(response, "Could not nominate map"));
}
export async function likeNomination(config: RuntimeConfig, token: string, id: number, liked: boolean) {
 const response = await apiFetch(config, `/staff/motw/nominations/${id}/like`, {method: "PUT", headers: {...authHeaders(token), "Content-Type": "application/json"}, body: JSON.stringify({liked})});
 if (!response.ok) throw new Error(await readError(response, "Could not update like"));
}
