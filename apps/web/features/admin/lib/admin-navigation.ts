import type { NextRouter } from "next/router";
import {
  Archive,
  Bell,
  Bug,
  ClipboardList,
  FileText,
  Gavel,
  History,
  KeyRound,
  PlayCircle,
  Search,
  Shield,
  UserCog,
  Users,
  Wrench,
} from "lucide-react";

export const moderationViews = new Set([
  "active",
  "mine",
  "unclaimed",
  "watching",
  "auto-detection",
  "escalated",
  "archive",
]);

export const adminNav = [
  {
    title: "Moderation",
    items: [
      { href: "/admin/moderation/active", label: "Active Cases", icon: ClipboardList },
      { href: "/admin/moderation/mine", label: "Mine", icon: UserCog },
      { href: "/admin/moderation/unclaimed", label: "Unclaimed", icon: PlayCircle },
      { href: "/admin/moderation/watching", label: "Watching", icon: Search },
      { href: "/admin/moderation/escalated", label: "Escalated", icon: Gavel },
      { href: "/admin/moderation/auto-detection", label: "Auto Detection", icon: Shield },
      { href: "/admin/moderation/archive", label: "Archive", icon: Archive },
    ],
  },
  {
    title: "Players",
    items: [
      { href: "/admin/players", label: "Player Search", icon: Users },
      { href: "/admin/enforcement", label: "Enforcement", icon: Gavel },
    ],
  },
  {
    title: "Operations",
    items: [
      { href: "/admin/operations/maintenance", label: "Maintenance", icon: Wrench },
      { href: "/admin/operations/seasons", label: "Seasons", icon: History },
      { href: "/admin/operations/notifications", label: "Notifications", icon: Bell },
      { href: "/admin/content/changelog", label: "Changelog", icon: FileText },
    ],
  },
  {
    title: "Access",
    items: [{ href: "/admin/access/roles", label: "Roles", icon: KeyRound }],
  },
  {
    title: "Debug",
    items: [{ href: "/admin/debug/test-reports", label: "Test Reports", icon: Bug }],
  },
];

export function pathFromRouter(router: NextRouter) {
  const rawPath = router.query.path;
  if (Array.isArray(rawPath) && rawPath.length > 0) return rawPath;
  const tab = router.query.tab;
  if (typeof tab === "string") return [tab];
  return ["moderation", "active"];
}
