import { useEffect, useState } from "react";
import type { PublicPlayerProfile } from "../types";
import { useProfileOwnerActions } from "./use-profile-mutations";

export function useProfileEditor(
  profile: PublicPlayerProfile | undefined,
  accessToken: string,
  onNicknameSaved?: (nickname: string) => void,
) {
  const actions = useProfileOwnerActions(accessToken);
  const [editingName, setEditingName] = useState(false);
  const [nickname, setNickname] = useState(profile?.displayName || "");

  useEffect(() => {
    if (!profile) return;
    if (!editingName) setNickname(profile.displayName);
  }, [editingName, profile]);

  const cancelName = () => {
    setNickname(profile?.displayName || "");
    setEditingName(false);
  };
  const saveName = () =>
    actions.nicknameMutation.mutate(nickname.trim(), {
      onSuccess: () => {
        setEditingName(false);
        onNicknameSaved?.(nickname.trim());
      },
    });
  // Selecting the currently displayed badge clears the selection.
  const selectBadge = (badgeId: string) =>
    actions.badgeMutation.mutate(
      profile?.selectedBadge?.id === badgeId ? "" : badgeId,
    );

  return {
    ...actions,
    editingName,
    setEditingName,
    nickname,
    setNickname,
    cancelName,
    saveName,
    selectBadge,
  };
}
