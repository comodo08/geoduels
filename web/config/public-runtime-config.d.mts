export const publicConfigKeys: readonly string[];
export function resolvePublicConfig(env: Record<string, string | undefined>, development?: boolean): Record<string, string>;
export function validatePublicConfig(config: Record<string, string>): void;
