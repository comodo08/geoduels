# Current Architecture

This document describes the current runtime implemented in this repository. Older "runtime-core" / NATS notes are historical and should not be treated as the source of truth for the running system.

## Runtime roles

- `apps/web`: Next.js browser client and route shell.
- `services/api`: auth, profile, leaderboard, singleplayer bootstrap, match-route bootstrap/session endpoints.
- `services/match-coordinator`: duel queue websocket endpoint, assignment, recovery, online count, maintenance exposure.
- `services/realtime-gateway`: public websocket gateway for `/ws/{node}`.
- `services/gameplay-node`: authoritative in-memory duel and singleplayer execution for assigned matches.
- `workers/location-ingest`: bootstraps official datasets; user uploads are handled by the API.

## Data ownership

- PostgreSQL is the durable source of truth for users, identities, sessions, stats, ranks, runtime match metadata, and final match snapshots.
- Redis is used for queue state, gameplay-node registration, route assignment, presence, and maintenance status.
- `pkg/persistence` owns Postgres persistence behavior.
- `pkg/coordinator` owns Redis-backed node registration, assignment, and presence.
- `pkg/duel` and `pkg/singleplayer` own match rules and round progression.
- `pkg/gameticket` issues short-lived gameplay admission tickets.

## End-to-end flows

### Auth

1. Browser loads `apps/web`.
2. Web bootstraps auth through `GET /v1/auth/session`.
3. `services/api` rotates the session cookie and returns a short-lived app access JWT.
4. The browser keeps the access JWT in memory only.

### Duel

1. Browser opens websocket matchmaking to `services/match-coordinator` at `/queue`.
2. `match-coordinator` authenticates the app JWT, manages queue state, and selects a gameplay node.
3. `match-coordinator` pins the selected map revision, persists a deterministic round plan, and creates the match on the chosen `gameplay-node`.
4. `match-coordinator` returns match assignment plus a short-lived gameplay ticket.
5. Browser connects to `/ws/{node}` through `services/realtime-gateway`.
6. `realtime-gateway` resolves the registered gameplay pod for that route and proxies the websocket to the exact `gameplay-node`.
7. `gameplay-node` runs the authoritative match loop and broadcasts snapshots.

### Singleplayer

1. Browser asks `services/api` to start a singleplayer session.
2. `api` selects a non-draining gameplay node and creates the match there.
3. `api` returns assignment plus gameplay ticket.
4. Browser connects through `realtime-gateway` to the assigned node.

### Match route bootstrap

- `/match/[id]` is the canonical route for a specific match.
- Cold loads resolve through `GET /v1/matches/{id}/bootstrap`.
- Already-authenticated refreshes can resolve through `GET /v1/matches/{id}/session`.
- The route can resolve to:
  - live reconnect with a fresh gameplay ticket
  - final match history
  - replaced
  - forbidden
  - missing

## Routing and scaling boundaries

- `match-coordinator` is responsible for duel creation and assignment.
- `api` is responsible for singleplayer creation and route bootstrap/session endpoints.
- `realtime-gateway` is only a routing/proxy layer.
- `gameplay-node` is the in-memory authority for the matches assigned to it.
- Queueing and realtime simulation are intentionally separate scaling boundaries.

## Reconnect and session model

- Browser auth and gameplay admission are intentionally separate:
  - app JWT for API and queue access
  - gameplay ticket for a specific match/node websocket connection
- Disconnects are handled by the match runtime on the gameplay node.
- Reconnect flows recover through assignment lookup plus a newly minted gameplay ticket.

## Maintenance and drain behavior

- Gameplay nodes register themselves in Redis with:
  - public websocket route
  - internal pod URL
  - active match count
  - draining flag
- During shutdown, a gameplay node marks itself draining, stops accepting new match creation, and waits for active matches to finish before exit.
- `match-coordinator` and `api` both exclude draining gameplay nodes from new assignment decisions.
- `realtime-gateway` stops accepting new websocket upgrades during shutdown and waits for active proxied sockets to drain.
- `api`, `match-coordinator`, `realtime-gateway`, and `gameplay-node` all fail readiness while draining so Kubernetes can remove them from rotation.

## Maintenance status

- Redis key `system:maintenance` stores optional maintenance status.
- `match-coordinator` exposes that status through `/queue/online`.
- `queuePaused` blocks new duel queueing.
- `playPaused` blocks all new play session creation.
- The web lobby renders:
  - a warning banner when `phase=warning`
  - a blocking overlay when `phase=active`

## Location pipeline

- Custom-map JSON is streamed through the API, validated, normalized into immutable PostgreSQL revisions, and then discarded.
- Match launch selects and persists the full bounded round plan before assigning a gameplay node.
- `gameplay-node` consumes the supplied in-memory plan and does not query or preload map catalogs.
- Redis is not used for map contents, lobby map settings, or per-match location deduplication.
