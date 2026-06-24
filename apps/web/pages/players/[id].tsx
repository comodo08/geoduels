import type { GetServerSideProps } from "next";
import { PlayerProfilePage } from "../../features/players/components/PlayerProfilePage";
import { requestPlayerProfile } from "../../features/players/lib/player-client";
import type { PublicPlayerProfile } from "../../features/players/types";
import { normalizeEntityRouteId, toPublicEntityId } from "../../lib/entity-id";
import { createRuntimeConfig } from "../../lib/runtime-config";

type PlayerRouteProps = {
  playerId: string;
  initialProfile: PublicPlayerProfile;
};

export default function PlayerRoute({ playerId, initialProfile }: PlayerRouteProps) {
  return <PlayerProfilePage playerId={playerId} initialProfile={initialProfile} />;
}

export const getServerSideProps: GetServerSideProps<PlayerRouteProps> = async ({ params }) => {
  const rawId = typeof params?.id === "string" ? params.id : "";
  const playerId = normalizeEntityRouteId(rawId);
  if (!playerId) return { notFound: true };
  const canonicalId = toPublicEntityId(playerId);
  if (rawId !== canonicalId) {
    return {
      redirect: {
        destination: `/players/${encodeURIComponent(canonicalId)}`,
        permanent: true,
      },
    };
  }
  try {
    const initialProfile = await requestPlayerProfile(createRuntimeConfig(), playerId);
    return { props: { playerId, initialProfile } };
  } catch {
    return { notFound: true };
  }
};
