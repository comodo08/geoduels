import Head from "next/head";
import { useRouter } from "next/router";
import { useCallback, useEffect, useRef } from "react";
import type { LobbyContentRoute } from "../../../components/ui/LobbyScreen";
import { getSiteURL } from "../../../lib/site";
import { useHomeModel } from "../model/useHomeModel";
import HomePageView from "./HomePageView";

type LobbyRoutePageProps = {
  route: LobbyContentRoute;
  mapId?: string;
  title: string;
  description: string;
  canonicalPath: string;
};

export default function LobbyRoutePage({
  route,
  mapId = "",
  title,
  description,
  canonicalPath,
}: LobbyRoutePageProps) {
  const router = useRouter();
  const prevMatchIdRef = useRef("");
  const siteURL = getSiteURL();
  const canonicalURL = `${siteURL}${canonicalPath}`;
  const handlePrivateLobbyEntered = useCallback(
    (inviteCode: string) => {
      void router.push(`/party/${encodeURIComponent(inviteCode)}`);
    },
    [router],
  );
  const model = useHomeModel({
    routeContext: "home",
    onPrivateLobbyEntered: handlePrivateLobbyEntered,
  });

  useEffect(() => {
    const nextMatchId = model.view.meta.activeMatchId;
    const prevMatchId = prevMatchIdRef.current;
    prevMatchIdRef.current = nextMatchId;
    if (!nextMatchId || nextMatchId === prevMatchId) {
      return;
    }
    void router.push(`/match/${encodeURIComponent(nextMatchId)}`);
  }, [model.view.meta.activeMatchId, router]);

  return (
    <>
      <Head>
        <title>{title}</title>
        <meta name="description" content={description} />
        <meta name="robots" content="index,follow" />
        <link rel="canonical" href={canonicalURL} />
        <meta property="og:type" content="website" />
        <meta property="og:site_name" content="GeoDuels" />
        <meta property="og:title" content={title} />
        <meta property="og:description" content={description} />
        <meta property="og:url" content={canonicalURL} />
        <meta property="og:image" content={`${siteURL}/logo.v2.png`} />
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content={title} />
        <meta name="twitter:description" content={description} />
        <meta name="twitter:image" content={`${siteURL}/logo.v2.png`} />
      </Head>
      <HomePageView model={model} lobbyRoute={route} mapId={mapId} />
    </>
  );
}
