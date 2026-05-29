import Head from "next/head";
import { useRouter } from "next/router";
import { useCallback, useEffect, useRef } from "react";
import HomePageView from "../../features/home/page/HomePageView";
import { useHomeModel } from "../../features/home/model/useHomeModel";
import { getSiteURL } from "../../lib/site";

export default function LobbyInviteRoute() {
  const router = useRouter();
  const rawCode = router.isReady && typeof router.query.code === "string"
    ? router.query.code
    : "";
  const lobbyInviteCode = rawCode.trim().toUpperCase();
  const routedLobbyCode = rawCode.toLowerCase() === "new" ? "" : lobbyInviteCode;
  const prevMatchIdRef = useRef("");
  const siteURL = getSiteURL();
  const canonicalURL = lobbyInviteCode
    ? `${siteURL}/lobby/${encodeURIComponent(lobbyInviteCode)}`
    : `${siteURL}/`;
  const title = lobbyInviteCode
    ? `GeoDuels | Lobby ${lobbyInviteCode}`
    : "GeoDuels | Lobby";
  const description =
    "Join a private GeoDuels lobby, invite a friend or guest, and start a duel together.";
  const handlePrivateLobbyEntered = useCallback(
    (inviteCode: string) => {
      const nextPath = `/lobby/${encodeURIComponent(inviteCode)}`;
      if (router.asPath.split("?")[0] !== nextPath) {
        void router.replace(nextPath, undefined, { shallow: true });
      }
    },
    [router],
  );
  const handlePrivateLobbyLeft = useCallback(() => {
    void router.push("/");
  }, [router]);

  const model = useHomeModel({
    routeContext: "home",
    lobbyInviteCode: routedLobbyCode,
    onPrivateLobbyEntered: handlePrivateLobbyEntered,
    onPrivateLobbyLeft: handlePrivateLobbyLeft,
  });

  useEffect(() => {
    if (!router.isReady) return;
    if (!rawCode) return;
    if (rawCode.toLowerCase() === "new") {
      void router.replace("/");
      return;
    }
    if (rawCode !== lobbyInviteCode) {
      void router.replace(`/lobby/${encodeURIComponent(lobbyInviteCode)}`, undefined, {
        shallow: true,
      });
    }
  }, [lobbyInviteCode, rawCode, router]);

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
        <meta name="robots" content="noindex,nofollow" />
        <link rel="canonical" href={canonicalURL} />
        <meta property="og:type" content="website" />
        <meta property="og:site_name" content="GeoDuels" />
        <meta property="og:title" content={title} />
        <meta property="og:description" content={description} />
        <meta property="og:url" content={canonicalURL} />
        <meta property="og:image" content={`${siteURL}/logo.v2.png`} />
      </Head>
      <HomePageView model={model} />
    </>
  );
}
