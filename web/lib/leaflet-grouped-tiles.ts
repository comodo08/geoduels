import L from 'leaflet';

const TILE_FADE_MS = 220;
const TILE_PRUNE_AFTER_FADE_MS = 270;
const PINCH_TARGET_STABLE_MS = 90;
const PINCH_PREPARE_MIN_INTERVAL_MS = 180;
const PINCH_SNAP_EPSILON = 0.01;
const PINCH_DIRECTION_THRESHOLD = 0.01;

type GridLayerWithState = L.GridLayer & {
  options: L.GridLayerOptions;
  _map?: L.Map;
  _container?: HTMLElement;
  _tiles: Record<
    string,
    { current?: boolean; loaded?: Date & number; active?: boolean; el?: HTMLElement | undefined }
  >;
  _fadeFrame?: number;
  _noPrune?: boolean;
  _tileZoom?: number | undefined;
  _loading?: boolean;
  _groupedTileReleaseTimer?: ReturnType<typeof setTimeout>;
  _groupedTileLoadPruneTimer?: ReturnType<typeof setTimeout>;
  _groupedTileRelease?: (() => void) | undefined;
  _noTilesToLoad: () => boolean;
  _onOpaqueTile: (tile: unknown) => void;
  _setView: (center: L.LatLng, zoom: number, noPrune?: boolean, noUpdate?: boolean) => void;
  _setZoomTransforms: (center: L.LatLng, zoom: number) => void;
  _pruneTiles: () => void;
  _tileCoordsToKey: (coords: L.Coords) => string;
  isLoading: () => boolean;
};

type MapWithState = L.Map & {
  _fadeAnimated?: boolean;
  _animatingZoom?: boolean;
  _suspendTileUpdates?: boolean;
  _groupedControlZoomState?: { active?: boolean };
  touchZoom?: { _zooming?: boolean };
};

type TouchZoomHandlerWithState = {
  _map?: MapWithState;
  _zooming?: boolean;
  _moved?: boolean;
  _center?: L.LatLng;
  _zoom?: number;
  _startZoom?: number;
  _groupedTiles?: boolean;
  _groupedPinchTimer?: ReturnType<typeof setTimeout>;
  _groupedPinchCandidate?: number | null;
  _groupedPinchPreparedTarget?: number | null;
  _groupedPinchPreparedAt?: number;
  _groupedPinchDirection?: number;
  _groupedPinchDirectionTravel?: number;
  _groupedPinchLastZoom?: number;
  _onTouchStart: (event: Event) => unknown;
  _onTouchMove: (event: Event) => unknown;
  _onTouchEnd: (event: Event) => unknown;
  removeHooks: () => void;
};
if (!(L.GridLayer.prototype as unknown as { _groupedTiles?: boolean })._groupedTiles) {
  (L.GridLayer.prototype as unknown as { _groupedTiles: boolean })._groupedTiles = true;

  const proto = L.GridLayer.prototype as unknown as Record<string, unknown>;

  const dpr = typeof window === 'undefined' ? 1 : window.devicePixelRatio || 1;
  const useChromiumSeamBlend =
    L.Browser.chrome && !L.Browser.mobile && Math.abs(dpr * 4 - Math.round(dpr * 4)) < 0.001;

  const origInitTile = proto._initTile as (this: L.GridLayer, tile: HTMLElement) => void;
  proto._initTile = function (this: L.GridLayer, tile: HTMLElement) {
    origInitTile.call(this, tile);

    if (tile.tagName === 'IMG') (tile as HTMLImageElement).decoding = 'async';
    if ((this.options as L.GridLayerOptions).updateWhenZooming === false) {
      tile.style.mixBlendMode = useChromiumSeamBlend && tile.tagName === 'IMG' ? '' : 'normal';
    }
  } as unknown as typeof origInitTile;

  const origUpdateOpacity = proto._updateOpacity as (this: L.GridLayer) => void;
  proto._updateOpacity = function (this: L.GridLayer) {
    const self = this as GridLayerWithState;
    const map = this._map as MapWithState | undefined;
    if (!map) return;
    if (!map._fadeAnimated) {
      return origUpdateOpacity.call(this);
    }

    L.DomUtil.setOpacity(
      (this as unknown as { _container?: HTMLElement })._container as HTMLElement,
      (this.options as L.GridLayerOptions).opacity ?? 1
    );

    const now = +new Date();
    let nextFrame = false;
    let willPrune = false;

    for (const key in self._tiles) {
      const tile = self._tiles[key];
      if (!tile.current || !tile.loaded) continue;

      const fade = Math.min(1, (now - tile.loaded) / TILE_FADE_MS);
      if (tile.el) L.DomUtil.setOpacity(tile.el, fade);
      if (fade < 1) {
        nextFrame = true;
      } else {
        if (tile.active) {
          willPrune = true;
        } else {
          self._onOpaqueTile(tile);
        }
        tile.active = true;
      }
    }

    if (willPrune && !self._noPrune) self._pruneTiles();

    if (nextFrame) {
      L.Util.cancelAnimFrame(self._fadeFrame as number);
      self._fadeFrame = L.Util.requestAnimFrame(origUpdateOpacity.bind(this), this);
    }
  } as typeof origUpdateOpacity;

  const origTileReady = proto._tileReady as (
    this: L.GridLayer,
    coords: L.Coords,
    err: Error | undefined,
    tile: HTMLImageElement
  ) => void;
  proto._tileReady = function (
    this: L.GridLayer,
    coords: L.Coords,
    err: Error | undefined,
    tile: HTMLImageElement
  ) {
    const self = this as GridLayerWithState;
    const map = this._map as MapWithState | undefined;
    if (!map?._fadeAnimated) {
      return origTileReady.call(this, coords, err, tile);
    }

    if (err) {
      this.fire('tileerror', { error: err, tile, coords });
    }

    const key = self._tileCoordsToKey(coords);
    const entry = self._tiles[key];
    if (!entry) return;

    const loadedAt = +new Date();
    (entry as { loaded: number }).loaded = loadedAt;
    if (entry.el) L.DomUtil.setOpacity(entry.el, 0);
    L.Util.cancelAnimFrame(self._fadeFrame as number);
    self._fadeFrame = L.Util.requestAnimFrame(origUpdateOpacity.bind(this), this);

    if (!err) {
      if (entry.el) L.DomUtil.addClass(entry.el, 'leaflet-tile-loaded');
      this.fire('tileload', { tile: entry.el, coords });
    }

    if (self._noTilesToLoad()) {
      self._loading = false;
      this.fire('load');

      clearTimeout(self._groupedTileLoadPruneTimer);
      self._groupedTileLoadPruneTimer = setTimeout(() => {
        delete self._groupedTileLoadPruneTimer;
        if (!this._map || self._noPrune) return;
        try {
          self._pruneTiles();
        } catch {}
      }, TILE_PRUNE_AFTER_FADE_MS);
    }
  } as typeof origTileReady;

  const origOnMoveEnd = proto._onMoveEnd as (this: L.GridLayer, event?: Event) => void;
  proto._onMoveEnd = function (this: L.GridLayer, event?: Event) {
    const self = this as GridLayerWithState;
    const map = this._map as MapWithState | undefined;
    if (!map || map._animatingZoom) return;

    if (map._suspendTileUpdates) return;

    if (map._groupedControlZoomState?.active) return;

    if (map.touchZoom?._zooming) {
      if (Math.abs(map.getZoom() - (self._tileZoom ?? map.getZoom())) <= 1) {
        return origOnMoveEnd.call(this, event);
      }
      return;
    }

    if ((this.options as L.GridLayerOptions).updateWhenZooming === false) {
      const zoom = map.getZoom();
      const tileZoom = Math.round(zoom);
      if (self._tileZoom !== tileZoom) {
        self._setView(map.getCenter(), zoom, true, false);

        self._noPrune = false;
        return;
      }
    }

    return origOnMoveEnd.call(this, event);
  } as typeof origOnMoveEnd;
}

export function prepareGroupedTileLevel(
  map: L.Map,
  targetCenter: L.LatLng,
  targetZoom: number
): void {
  const m = map as MapWithState;
  if (typeof m.eachLayer !== 'function') return;

  const liveCenter = map.getCenter();
  const liveZoom = map.getZoom();

  map.eachLayer((layer) => {
    if (
      !(layer instanceof L.GridLayer) ||
      (layer.options as L.GridLayerOptions).updateWhenZooming !== false
    ) {
      return;
    }
    const self = layer as GridLayerWithState;

    try {
      const roundedTarget = Math.round(targetZoom);

      if (self._tileZoom === roundedTarget) {
        self._setZoomTransforms(liveCenter, liveZoom);
        return;
      }

      clearTimeout(self._groupedTileReleaseTimer);
      delete self._groupedTileReleaseTimer;
      clearTimeout(self._groupedTileLoadPruneTimer);
      delete self._groupedTileLoadPruneTimer;
      if (self._groupedTileRelease) {
        layer.off('load', self._groupedTileRelease);
        delete self._groupedTileRelease;
      }

      self._setView(targetCenter, targetZoom, true, false);
      self._setZoomTransforms(liveCenter, liveZoom);

      const release = () => {
        if (self._groupedTileRelease !== release) return;
        layer.off('load', release);
        delete self._groupedTileRelease;
        clearTimeout(self._groupedTileReleaseTimer);

        self._groupedTileReleaseTimer = setTimeout(() => {
          delete self._groupedTileReleaseTimer;
          if (!(layer as unknown as { _map?: L.Map })._map) return;
          self._noPrune = false;
          try {
            self._pruneTiles();
          } catch {}
        }, TILE_PRUNE_AFTER_FADE_MS);
      };

      if (self.isLoading()) {
        self._groupedTileRelease = release;
        layer.once('load', release);
      } else {
        self._groupedTileRelease = release;
        release();
      }
    } catch {}
  });
}

function clearPinchPrepare(handler: TouchZoomHandlerWithState, resetTarget = false): void {
  clearTimeout(handler._groupedPinchTimer);
  delete handler._groupedPinchTimer;
  if (resetTarget) {
    handler._groupedPinchCandidate = null;
    handler._groupedPinchPreparedTarget = null;
  }
}

function pinchTarget(handler: TouchZoomHandlerWithState): number | null {
  const map = handler._map;
  if (!map || !Number.isFinite(handler._zoom)) return null;
  const zoom = handler._zoom as number;

  const direction =
    handler._groupedPinchDirection || Math.sign(zoom - (handler._startZoom ?? zoom));
  const target =
    direction >= 0
      ? Math.ceil((zoom - PINCH_SNAP_EPSILON) / 1) * 1
      : Math.floor((zoom + PINCH_SNAP_EPSILON) / 1) * 1;
  return Math.max(map.getMinZoom(), Math.min(map.getMaxZoom(), target));
}

function schedulePinchPrepare(handler: TouchZoomHandlerWithState): void {
  if (!handler._zooming || !handler._moved || !handler._center) return;
  const target = pinchTarget(handler);
  if (target == null || target === handler._groupedPinchCandidate) return;
  clearPinchPrepare(handler);
  handler._groupedPinchCandidate = target;
  const elapsed = Date.now() - (handler._groupedPinchPreparedAt || 0);

  const outward = target < (handler._zoom as number);
  const delay = outward
    ? Math.max(0, PINCH_PREPARE_MIN_INTERVAL_MS - elapsed)
    : Math.max(PINCH_TARGET_STABLE_MS, PINCH_PREPARE_MIN_INTERVAL_MS - elapsed);
  handler._groupedPinchTimer = setTimeout(() => {
    delete handler._groupedPinchTimer;
    if (!handler._zooming || pinchTarget(handler) !== target) return;
    prepareGroupedTileLevel(handler._map as L.Map, handler._center as L.LatLng, target);
    handler._groupedPinchPreparedTarget = target;
    handler._groupedPinchPreparedAt = Date.now();
  }, delay);
}

if ((L.Map as unknown as { TouchZoom?: unknown }).TouchZoom) {
  const touchZoom = (L.Map as unknown as { TouchZoom: new () => unknown }).TouchZoom
    .prototype as unknown as TouchZoomHandlerWithState;
  touchZoom._groupedTiles = true;

  const origTouchStart = touchZoom._onTouchStart;
  touchZoom._onTouchStart = function (event: Event) {
    clearPinchPrepare(this, true);
    this._groupedPinchPreparedAt = 0;
    this._groupedPinchDirection = 0;
    this._groupedPinchDirectionTravel = 0;
    const result = origTouchStart.call(this, event);
    this._groupedPinchLastZoom = this._startZoom;
    return result;
  };

  const origTouchMove = touchZoom._onTouchMove;
  touchZoom._onTouchMove = function (event: Event) {
    const result = origTouchMove.call(this, event);
    if (Number.isFinite(this._zoom)) {
      const delta = (this._zoom as number) - (this._groupedPinchLastZoom ?? (this._zoom as number));
      this._groupedPinchDirectionTravel = (this._groupedPinchDirectionTravel ?? 0) + delta;
      if (Math.abs(this._groupedPinchDirectionTravel) >= PINCH_DIRECTION_THRESHOLD) {
        this._groupedPinchDirection = Math.sign(this._groupedPinchDirectionTravel);
        this._groupedPinchDirectionTravel = 0;
      }
      this._groupedPinchLastZoom = this._zoom;
    }
    schedulePinchPrepare(this);
    return result;
  };

  const origTouchEnd = touchZoom._onTouchEnd;
  touchZoom._onTouchEnd = function (event: Event) {
    const target = pinchTarget(this);
    const shouldPrepare =
      !!this._zooming &&
      !!this._moved &&
      !!this._center &&
      target != null &&
      target !== this._groupedPinchPreparedTarget;
    clearPinchPrepare(this);

    if (shouldPrepare) {
      prepareGroupedTileLevel(this._map as L.Map, this._center as L.LatLng, target as number);
      this._groupedPinchPreparedTarget = target;
    }

    if (target != null) {
      this._zoom = target;
    }
    return origTouchEnd.call(this, event);
  };

  const origRemoveHooks = touchZoom.removeHooks;
  touchZoom.removeHooks = function () {
    clearPinchPrepare(this, true);
    return origRemoveHooks.call(this);
  };
}

export {};
