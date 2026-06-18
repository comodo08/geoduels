import { useRouter } from "next/router";
import { getLobbyLayout } from "../../features/home/page/LobbyApplicationLayout";
import LobbyRoutePage from "../../features/home/page/LobbyRoutePage";
import type { NextPageWithLayout } from "../_app";

const MapDetailsRoute: NextPageWithLayout = function MapDetailsRoute() {
  const router = useRouter();
  const mapId = router.isReady && typeof router.query.id === "string" ? router.query.id : "";

  return (
    <LobbyRoutePage
      title="GeoDuels | Map Details"
      description="View GeoDuels map details, country distribution, comments, and play actions."
      canonicalPath={mapId ? `/maps/${encodeURIComponent(mapId)}` : "/maps"}
    />
  );
};

MapDetailsRoute.getLayout = getLobbyLayout;

export default MapDetailsRoute;
