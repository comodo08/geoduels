import { useEffect, useState } from "react";
import type { PublicPlayerProfile } from "../types";
import { useProfileOwnerActions } from "./use-profile-mutations";

const MAX_ABOUT_LENGTH = 150;

export function useProfileEditor(
  profile: PublicPlayerProfile | undefined,
  accessToken: string,
  onNicknameSaved?: (nickname: string) => void,
) {
  const actions = useProfileOwnerActions(accessToken);
  const [editingName, setEditingName] = useState(false);
  const [nickname, setNickname] = useState(profile?.displayName || "");
  const [choosingBadge, setChoosingBadge] = useState(false);
  const [badgeId, setBadgeId] = useState(profile?.selectedBadge?.id || "");
  const [flagCode, setFlagCode] = useState(profile?.flagCode || "");
  const [editingAbout, setEditingAbout] = useState(false);
  const [about, setAbout] = useState(profile?.about || "");

  useEffect(() => {
    if (!profile) return;
    if (!editingName) setNickname(profile.displayName);
    if (!choosingBadge) setBadgeId(profile.selectedBadge?.id || "");
    setFlagCode(profile.flagCode || "");
    if (!editingAbout) setAbout(profile.about || "");
  }, [choosingBadge, editingName, editingAbout, profile]);

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
  const cancelBadge = () => {
    setBadgeId(profile?.selectedBadge?.id || "");
    setChoosingBadge(false);
  };
  const saveBadge = () =>
    actions.badgeMutation.mutate(badgeId, {
      onSuccess: () => setChoosingBadge(false),
    });
  const selectFlag = (code: string) => {
    const previous = profile?.flagCode || "";
    setFlagCode(code);
    actions.flagMutation.mutate(code, {
      onError: () => setFlagCode(previous),
    });
  };
  const cancelAbout = () => {
    setAbout(profile?.about || "");
    setEditingAbout(false);
  };
  const saveAbout = () =>
    actions.aboutMutation.mutate(about.trim().slice(0, MAX_ABOUT_LENGTH), {
      onSuccess: () => setEditingAbout(false),
    });

  return {
    ...actions,
    editingName,
    setEditingName,
    nickname,
    setNickname,
    cancelName,
    saveName,
    choosingBadge,
    setChoosingBadge,
    badgeId,
    setBadgeId,
    cancelBadge,
    saveBadge,
    flagCode,
    setFlagCode,
    selectFlag,
    editingAbout,
    setEditingAbout,
    about,
    setAbout,
    cancelAbout,
    saveAbout,
  };
}
