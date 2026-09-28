export type StaffRole = "admin" | "judge" | "moderator" | "lanista";
export function staffRoles(user?: {roles?: StaffRole[]; isAdmin?: boolean; isModerator?: boolean} | null): StaffRole[] {
 if (user?.roles) return user.roles;
 // v1's moderator flag means judge, never the new team-curation role.
 return [...(user?.isAdmin ? ["admin" as const] : []), ...(user?.isModerator ? ["judge" as const] : [])];
}
