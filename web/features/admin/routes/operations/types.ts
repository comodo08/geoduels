import { useRuntimeConfig } from "../../../../lib/runtime-config-context";

export type OperationsSectionProps = {
  config: ReturnType<typeof useRuntimeConfig>;
  accessToken: string;
  leaf: string;
  canManageAdmin: boolean;
  refreshAdminData: () => Promise<void>;
};
