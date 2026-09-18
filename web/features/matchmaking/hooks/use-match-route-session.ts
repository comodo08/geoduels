import { useEffect, useSyncExternalStore } from 'react';
import { getHomeRuntime } from '../../home/state/home-runtime';
import { getRuntimeConfig } from '../../../lib/runtime-config';

export function useMatchRouteSession(matchId: string | null) {
  const runtime = getHomeRuntime(getRuntimeConfig());
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
