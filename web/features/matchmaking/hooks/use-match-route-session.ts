import { useEffect, useSyncExternalStore } from 'react';
import { getHomeRuntime } from '../../home/state/home-runtime';
import { useRuntimeConfig } from '../../../lib/runtime-config-context';

export function useMatchRouteSession(matchId: string | null) {
  const runtime = getHomeRuntime(useRuntimeConfig());
  const controller = runtime.matchRouteController;
  const state = useSyncExternalStore(
    controller.subscribe,
    controller.getState.bind(controller),
    controller.getState.bind(controller)
  );

  useEffect(() => {
    controller.setTargetMatch(matchId);
    return () => {
      controller.reset();
    };
  }, [controller, matchId]);

  return state;
}
