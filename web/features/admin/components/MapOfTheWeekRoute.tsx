import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Heart } from "lucide-react";
import { useAuthState } from "../../auth/components/AuthProvider";
import { useRuntimeConfig } from "../../../lib/runtime-config-context";
import { toPublicEntityId } from "../../../lib/entity-id";
import { Button } from "../../../components/ui/button";
import { nominations, likeNomination, type Nomination } from "../lib/curation-client";

export function MapOfTheWeekRoute() {
 const auth = useAuthState(); const config = useRuntimeConfig(); const client = useQueryClient(); const [page,setPage] = useState(1);
 const query = useQuery({queryKey:["motw",auth.userId,page], queryFn:()=>nominations(config,auth.accessToken,page), enabled:auth.isModerator, refetchInterval:30_000});
 const mutation = useMutation({mutationFn:(item:Nomination)=>likeNomination(config,auth.accessToken,item.id,!item.liked),onSuccess:()=>client.invalidateQueries({queryKey:["motw"]})});
 const data=query.data; const pages=Math.max(1,Math.ceil((data?.total||0)/(data?.pageSize||20)));
 return <div className="space-y-5">
  <header><h2 className="text-display-md font-strong">Map of the Week</h2><p className="mt-2 text-content-secondary">Nominate maps from their map page. The most liked map wins each week.</p>
  {data ? <p className="mt-2 text-label text-content-secondary">Selection: {new Date(data.closesAt).toLocaleString()}</p>:null}</header>
  {query.isLoading?<p>Loading nominations…</p>:null}
  {query.error||mutation.error?<p role="alert" className="text-status-danger">{(query.error||mutation.error)?.message}</p>:null}
  {data?.items.map(item=><div key={item.id} className="flex items-center justify-between gap-4 rounded-xl border border-border-default bg-surface-panel p-4">
   <div><Link href={`/maps/${toPublicEntityId(item.mapId)}`} className="font-strong hover:underline">{item.name}</Link><p className="text-body-sm text-content-secondary">{item.authorName}</p></div>
   <Button aria-label={`Like ${item.name}`} aria-pressed={item.liked} disabled={mutation.isPending} variant={item.liked?"primary":"secondary"} onClick={()=>mutation.mutate(item)}><Heart size={16} fill={item.liked?"currentColor":"none"}/>{item.likes}</Button>
  </div>)}
  {data&&!data.items.length?<p className="text-content-secondary">No nominations yet. Without nominations, the top trending community map is selected.</p>:null}
  <div className="flex items-center gap-3"><Button disabled={page<=1} onClick={()=>setPage(page-1)}>Previous</Button><span>Page {page} of {pages}</span><Button disabled={page>=pages} onClick={()=>setPage(page+1)}>Next</Button></div>
 </div>;
}
