import Head from 'next/head';
import { useRouter } from 'next/router';
import { useCallback, useEffect, useRef } from 'react';
import HomePageView from '../features/home/page/HomePageView';
import { useHomeModel } from '../features/home/model/useHomeModel';
import { getSiteURL } from '../lib/site';

export default function HomePage() {
  const router = useRouter();
  const lobbyInviteCode =
    router.isReady && typeof router.query.lobby === 'string'
      ? router.query.lobby
      : '';
  const handlePrivateLobbyEntered = useCallback(
    (inviteCode: string) => {
      void router.push(`/lobby/${encodeURIComponent(inviteCode)}`);
    },
    [router],
  );
  const model = useHomeModel({
    routeContext: 'home',
    lobbyInviteCode,
    onPrivateLobbyEntered: handlePrivateLobbyEntered
  });
  const prevMatchIdRef = useRef(model.view.meta.activeMatchId);
  const siteURL = getSiteURL();
  const canonicalURL = `${siteURL}/`;
  const title = 'GeoDuels | Play';
  const description =
    'Play the best free GeoGuessr alternative with ranked duels, online 1v1 games, singleplayer, or 2v2 with friends!'; 

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
        <meta property="og:image" content={`${siteURL}/logo.v1.png`} />
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content={title} />
        <meta name="twitter:description" content={description} />
        <meta name="twitter:image" content={`${siteURL}/logo.v1.png`} />
      </Head>
      <HomePageView model={model} />
    </>
  );
}
