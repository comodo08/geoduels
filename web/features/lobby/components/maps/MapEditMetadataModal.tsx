import { Save } from "lucide-react";
import { useState } from "react";
import AppModalShell from "../../../../components/ui/AppModalShell";
import type { CustomMap, MapUpdateInput } from "../../../maps/lib/maps-client";
import { MapMetadataFields, thumbnailCategoryFromKey } from "../MapMetadataFields";
import { Button } from "../../../../components/ui/button";
import { useAppNotice } from "../../../../components/ui/AppNotice";

type MapEditMetadataModalProps = {
  map: CustomMap;
  onClose: () => void;
  onSave: (mapId: string, input: MapUpdateInput) => Promise<CustomMap>;
};

export function MapEditMetadataModal({ map, onClose, onSave }: MapEditMetadataModalProps) {
  const [mapName, setMapName] = useState(map.displayName);
  const [mapDescription, setMapDescription] = useState(map.description || "");
  const [mapDifficulty, setMapDifficulty] = useState<CustomMap["difficulty"]>(map.difficulty);
  const [mapVisibility, setMapVisibility] = useState(map.visibility);
  const [mapThumbnailKey, setMapThumbnailKey] = useState(map.thumbnailKey);
  const [mapThumbnailCategory, setMapThumbnailCategory] = useState(thumbnailCategoryFromKey(map.thumbnailKey));
  const [mapThumbnailSearch, setMapThumbnailSearch] = useState("");
  const [saving, setSaving] = useState(false);
  const { show } = useAppNotice();
  const saveDisabled = saving || !mapName.trim();

  const save = async () => {
    if (saveDisabled) return;
    setSaving(true);
    try {
      await onSave(map.id, {
        displayName: mapName,
        description: mapDescription,
        difficulty: mapDifficulty,
        visibility: mapVisibility,
        thumbnailKey: mapThumbnailKey,
        thumbnailVariant: map.thumbnailVariant,
      });
      onClose();
    } catch (err) {
      show(err instanceof Error ? err.message : "Map update failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <AppModalShell
      title="Edit Map"
      onClose={onClose}
      placement="center"
      maxWidthClassName="max-w-[900px]"
      contentClassName="space-y-4"
    >
      {map.publishedAt ? (
        <p className="text-body-sm text-content-secondary">Changes update the public map listing immediately.</p>
      ) : null}
      <MapMetadataFields
        disabled={saving}
        mapName={mapName}
        setMapName={setMapName}
        mapDescription={mapDescription}
        setMapDescription={setMapDescription}
        mapDifficulty={mapDifficulty}
        setMapDifficulty={setMapDifficulty}
        mapVisibility={mapVisibility}
        setMapVisibility={setMapVisibility}
        mapThumbnailKey={mapThumbnailKey}
        setMapThumbnailKey={setMapThumbnailKey}
        mapThumbnailCategory={mapThumbnailCategory}
        setMapThumbnailCategory={setMapThumbnailCategory}
        mapThumbnailSearch={mapThumbnailSearch}
        setMapThumbnailSearch={setMapThumbnailSearch}
      />
      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="ghost" disabled={saving} onClick={onClose}>
          Cancel
        </Button>
        <Button type="button" variant="primary" disabled={saveDisabled} onClick={save} loading={saving} loadingLabel="Saving map" icon={<Save size={17} />}>
          Save
        </Button>
      </div>
    </AppModalShell>
  );
}
