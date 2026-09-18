export type MatchmakingStatus = 'idle' | 'ready' | 'queueing' | 'matched_connecting' | 'in_match' | 'recovering' | 'abandoned';

export type RecoverOutcome = 'ready' | 'queueing' | 'matched' | 'abandoned';

export type MatchmakingState = {
  status: MatchmakingStatus;
  intentVersion: number;
  activeRecoverRequestID: number | null;
  queueStartedAt: number | null;
};

export type MatchmakingAction =
  | { type: 'set_status'; status: MatchmakingStatus; bumpIntent?: boolean }
  | { type: 'join_requested'; startedAt?: number }
  | { type: 'leave_requested' }
  | { type: 'queue_status'; status: string; queuedAt?: number }
  | { type: 'match_found' }
  | { type: 'game_connected' }
  | { type: 'queue_error' }
  | { type: 'ws_closed' }
  | { type: 'recover_started'; requestID: number }
  | {
      type: 'recover_resolved';
      requestID: number;
      intentVersionAtStart: number;
      outcome: RecoverOutcome;
      hasSnapshot: boolean;
    }
  | { type: 'recover_failed'; requestID: number };

export const initialMatchmakingState: MatchmakingState = {
  status: 'idle',
  intentVersion: 0,
  activeRecoverRequestID: null,
  queueStartedAt: null
};

export function matchmakingReducer(state: MatchmakingState, action: MatchmakingAction): MatchmakingState {
  switch (action.type) {
    case 'set_status':
      return {
        ...state,
        status: action.status,
        queueStartedAt: action.status === 'queueing' ? (state.queueStartedAt ?? null) : null,
        intentVersion: action.bumpIntent === false ? state.intentVersion : state.intentVersion + 1
      };
    case 'join_requested':
      return { ...state, status: 'queueing', queueStartedAt: action.startedAt ?? null, intentVersion: state.intentVersion + 1 };
    case 'leave_requested':
      return { ...state, status: 'ready', queueStartedAt: null, intentVersion: state.intentVersion + 1 };
    case 'queue_status': {
      const normalized = action.status === 'queued' ? 'queueing' : action.status;
      if (normalized === 'left') {
        return { ...state, status: 'ready', queueStartedAt: null };
      }
      if (normalized === 'matched') {
        return { ...state, status: 'matched_connecting', queueStartedAt: null };
      }
      if (normalized === 'queueing') {
        return { ...state, status: normalized, queueStartedAt: action.queuedAt ?? state.queueStartedAt };
      }
      if (normalized === 'ready' || normalized === 'idle' || normalized === 'recovering' || normalized === 'abandoned') {
        return { ...state, status: normalized, queueStartedAt: null };
      }
      return state;
    }
    case 'match_found':
      return { ...state, status: 'matched_connecting', queueStartedAt: null };
    case 'game_connected':
      return { ...state, status: 'in_match', queueStartedAt: null };
    case 'queue_error':
      if (state.status === 'queueing') {
        return { ...state, status: 'ready', queueStartedAt: null };
      }
      return state;
    case 'ws_closed':
      if (state.status === 'queueing' || state.status === 'matched_connecting' || state.status === 'in_match') {
        return { ...state, status: 'recovering' };
      }
      return state;
    case 'recover_started':
      return { ...state, activeRecoverRequestID: action.requestID };
    case 'recover_resolved': {
      if (state.activeRecoverRequestID !== action.requestID) {
        return state;
      }
      const next = { ...state, activeRecoverRequestID: null };
      if (action.outcome === 'matched') {
        return { ...next, status: 'matched_connecting' };
      }
      if (action.outcome === 'abandoned') {
        return { ...next, status: 'abandoned' };
      }
      if (action.intentVersionAtStart != state.intentVersion) {
        return next;
      }
      if (action.outcome === 'queueing') {
        return { ...next, status: 'queueing', queueStartedAt: state.queueStartedAt };
      }
      if (!action.hasSnapshot) {
        return { ...next, status: 'ready' };
      }
      return next;
    }
    case 'recover_failed':
      if (state.activeRecoverRequestID !== action.requestID) {
        return state;
      }
      return { ...state, activeRecoverRequestID: null };
    default:
      return state;
  }
}
