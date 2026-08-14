import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getRuntimeConfig } from "../../../lib/runtime-config";
import { getHomeRuntime } from "../../home/state/home-runtime";
import {
  requestDeleteAvatar,
  requestUpdateAvatar,
  requestUpdateAbout,
  requestUpdateNickname,
  requestUpdateSelectedBadge,
  requestUpdateProfileFlag,
} from "../../auth/lib/auth-client";

type AvatarPayload = {
  user?: { avatar_url?: string };
};

export function useProfileOwnerActions(accessToken: string) {
  const config = getRuntimeConfig();
  const queryClient = useQueryClient();
  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ["player-profile"] }),
      queryClient.invalidateQueries({ queryKey: ["optional-viewer"] }),
      queryClient.invalidateQueries({ queryKey: ["leaderboard"] }),
      queryClient.invalidateQueries({ queryKey: ["me"] }),
    ]);
  const applyAvatar = (payload: AvatarPayload | undefined) => {
    if (payload?.user && typeof payload.user.avatar_url === "string") {
      getHomeRuntime(config).sessionController.applyUserAvatar(
        payload.user.avatar_url,
      );
    }
  };

  return {
    nicknameMutation: useMutation({
      mutationFn: (nickname: string) =>
        requestUpdateNickname(config, accessToken, nickname),
      onSuccess: refresh,
    }),
    badgeMutation: useMutation({
      mutationFn: (badgeId: string) =>
        requestUpdateSelectedBadge(config, accessToken, badgeId),
      onSuccess: refresh,
    }),
    flagMutation: useMutation({
      mutationFn: (flagCode: string) =>
        requestUpdateProfileFlag(config, accessToken, flagCode),
      onSuccess: refresh,
    }),
    avatarUploadMutation: useMutation({
      mutationFn: (file: File) =>
        requestUpdateAvatar(config, accessToken, file),
      onSuccess: (payload) => {
        applyAvatar(payload as AvatarPayload);
        refresh();
      },
    }),
    avatarResetMutation: useMutation({
      mutationFn: () => requestDeleteAvatar(config, accessToken),
      onSuccess: (payload) => {
        applyAvatar(payload as AvatarPayload);
        refresh();
      },
    }),
    aboutMutation: useMutation({
      mutationFn: (about: string) =>
        requestUpdateAbout(config, accessToken, about),
      onSuccess: refresh,
    }),
  };
}
