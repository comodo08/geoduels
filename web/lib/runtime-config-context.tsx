import { createContext, useContext, useState, type ReactNode } from 'react';
import type { RuntimeConfig } from './runtime-config';

const RuntimeConfigContext = createContext<RuntimeConfig | null>(null);

export function RuntimeConfigProvider({ config, children }: { config: RuntimeConfig; children: ReactNode }) {
  // Keep controller configuration stable across client navigation. A full
  // document load picks up deployment changes from the server again.
  const [value] = useState(config);
  return <RuntimeConfigContext.Provider value={value}>{children}</RuntimeConfigContext.Provider>;
}

export function useRuntimeConfig() {
  const config = useContext(RuntimeConfigContext);
  if (!config) throw new Error('RuntimeConfigProvider is required.');
  return config;
}
