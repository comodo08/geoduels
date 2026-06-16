import { useRouter } from "next/router";
import LobbyRoutePage from "../../features/home/page/LobbyRoutePage";

export default function MapDetailsRoute() {
  const router = useRouter();
  const mapId = router.isReady && typeof router.query.id === "string" ? router.query.id : "";

  return (
    <LobbyRoutePage
      route="map-details"
      mapId={mapId}
      title="GeoDuels | Map Details"
      description="View GeoDuels map details, country distribution, comments, and play actions."
      canonicalPath={mapId ? `/maps/${encodeURIComponent(mapId)}` : "/maps"}
    />
  );
}
