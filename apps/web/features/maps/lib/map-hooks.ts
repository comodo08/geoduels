import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { RuntimeConfig } from "../../../lib/runtime-config";
import {
  archiveMap,
  createMapComment,
  deleteMapComment,
  getMap,
  listMaps,
  publishMap,
  setMapFavorite,
  uploadMapRevision,
  validateMapFile,
  type MapComment,
  type MapDetails,
  type MapSort,
  type MapScope,
} from "./maps-client";

function markCommentDeleted(comment: MapComment, commentId: string): { item: MapComment; changed: boolean } {
  let changed = false;
  const replies = comment.replies?.map((reply) => {
    const updated = markCommentDeleted(reply, commentId);
    changed = changed || updated.changed;
    return updated.item;
  });
  if (comment.id === commentId) {
    changed = true;
    return {
      item: {
        ...comment,
        body: "Comment deleted.",
        status: "deleted",
        replies,
      },
      changed,
    };
  }
  return { item: replies ? { ...comment, replies } : comment, changed };
}

function markMapDetailsCommentDeleted(details: MapDetails | undefined, commentId: string): MapDetails | undefined {
  if (!details) return details;
  let changed = false;
  const comments = details.comments.map((comment) => {
    const updated = markCommentDeleted(comment, commentId);
    changed = changed || updated.changed;
    return updated.item;
  });
  return changed ? { ...details, comments } : details;
}

export function useMapList(
  config: RuntimeConfig,
  accessToken: string | undefined,
  userId: string,
  scope: MapScope,
  sort: MapSort,
  search = "",
) {
  const trimmedSearch = search.trim();
  return useQuery({
    queryKey: ["maps", scope, sort, trimmedSearch, userId, accessToken],
    queryFn: () => listMaps(config, accessToken, { scope, sort: scope === "community" ? sort : undefined, search: trimmedSearch }),
    enabled: scope === "official" || scope === "community" || !!accessToken,
    staleTime: 15_000,
  });
}

export function useMapDetails(config: RuntimeConfig, accessToken: string | undefined, mapId: string) {
  return useQuery({
    queryKey: ["map-details", mapId, accessToken],
    queryFn: () => getMap(config, accessToken, mapId),
    enabled: !!mapId,
    staleTime: 10_000,
  });
}

export function useFavoriteMap(config: RuntimeConfig, accessToken: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ mapId, favorite }: { mapId: string; favorite: boolean }) =>
      setMapFavorite(config, accessToken || "", mapId, favorite),
    onSuccess: (_item, vars) => {
      void queryClient.invalidateQueries({ queryKey: ["maps"] });
      void queryClient.invalidateQueries({ queryKey: ["map-details", vars.mapId] });
    },
  });
}

export function useMapComments(config: RuntimeConfig, accessToken: string | undefined, mapId: string) {
  const queryClient = useQueryClient();
  return {
    createComment: useMutation({
      mutationFn: (input: { body: string; parentId?: string }) =>
        createMapComment(config, accessToken || "", mapId, input),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: ["map-details", mapId] });
        void queryClient.invalidateQueries({ queryKey: ["maps"] });
      },
    }),
    deleteComment: useMutation({
      mutationFn: ({ commentId }: { commentId: string }) =>
        deleteMapComment(config, accessToken || "", mapId, commentId),
      onSuccess: (_item, vars) => {
        queryClient.setQueriesData<MapDetails>(
          { queryKey: ["map-details", mapId] },
          (details) => markMapDetailsCommentDeleted(details, vars.commentId),
        );
        void queryClient.invalidateQueries({ queryKey: ["map-details", mapId] });
        void queryClient.invalidateQueries({ queryKey: ["maps"] });
      },
    }),
  };
}

export function useMapManagement(config: RuntimeConfig, accessToken: string | undefined, onUploadError: (message: string) => void) {
  const queryClient = useQueryClient();
  return {
    archiveMap: useMutation({
      mutationFn: (mapId: string) => archiveMap(config, accessToken || "", mapId),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["maps"] }),
    }),
    publishMap: useMutation({
      mutationFn: (mapId: string) => publishMap(config, accessToken || "", mapId),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["maps"] }),
    }),
    uploadRevision: useMutation({
      mutationFn: async ({ mapId, file }: { mapId: string; file: File }) => {
        await validateMapFile(file);
        return uploadMapRevision(config, accessToken || "", mapId, file);
      },
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["maps"] }),
      onError: (error) => onUploadError(error instanceof Error ? error.message : "Revision upload failed"),
    }),
  };
}
