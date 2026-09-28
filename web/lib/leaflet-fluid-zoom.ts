import L from 'leaflet';
import { prepareGroupedTileLevel } from './leaflet-grouped-tiles';

const WHEEL_IDLE_COMMIT_MS = 80;

L.Map.mergeOptions({
  fluidWheelZoom: false,

  fluidWheelZoomLevelsPerPixel: 0.0034,

  fluidWheelZoomEase: 200,

  fluidWheelZoomMinSpeed: 0,

  fluidWheelZoomMaxSpeed: 5
});

function limitCenterSmooth(map: L.Map, center: L.LatLng, zoom: number): L.LatLng {
  if (!map.options.maxBounds) return center;
  const bounds = map.options.maxBounds as L.LatLngBounds;
  const centerPoint = map.project(center, zoom);
  const viewHalf = map.getSize().divideBy(2);
  const projected = L.bounds(
    map.project(bounds.getNorthEast(), zoom),
    map.project(bounds.getSouthWest(), zoom)
  );
  if (!projected.min || !projected.max) return center;
  const minOffset = projected.min.subtract(centerPoint.subtract(viewHalf));
  const maxOffset = projected.max.subtract(centerPoint.add(viewHalf));
  const rebound = (left: number, right: number) =>
    left + right > 0 ? (left - right) / 2 : Math.max(0, left) - Math.max(0, right);
  const dx = rebound(minOffset.x, -maxOffset.x);
  const dy = rebound(minOffset.y, -maxOffset.y);
  if (dx === 0 && dy === 0) return center;
  return map.unproject(centerPoint.add(L.point(dx, dy)), zoom);
}

class FluidWheelZoom extends L.Handler {
  private _map: L.Map;

  _isWheeling = false;
  private _yielded = false;
  private _moved = false;
  private _timeoutId?: ReturnType<typeof setTimeout>;
  private _zoomAnimationId?: number;
  private _prevCenter: L.LatLng = L.latLng(0, 0);
  private _prevZoom = 0;
  private _centerPoint: L.Point = L.point(0, 0);
  private _startLatLng: L.LatLng = L.latLng(0, 0);
  private _wheelMousePosition: L.Point = L.point(0, 0);
  private _anchorLatLng: L.LatLng = L.latLng(0, 0);
  private _intentZoom = 0;
  private _segGoal = 0;
  private _glideZoom = 0;
  private _glideVelocity = 0;
  private _lastStepTime = 0;
  private _preparedTileGoal: number | null = null;

  private _intentDir = 1;

  constructor(map: L.Map) {
    super(map);
    this._map = map;
  }

  private readonly _wheelHandler = (e: Event): void => {
    this._onWheelScroll(e as WheelEvent);
  };

  private readonly _tick = (): void => {
    this._updateWheelZoom();
  };

  addHooks(): void {
    if (this._map.scrollWheelZoom && this._map.scrollWheelZoom.enabled()) {
      this._map.scrollWheelZoom.disable();
    }
    const container = (this._map as unknown as { _container: HTMLElement })._container;
    L.DomEvent.on(container, 'wheel', this._wheelHandler, this);
  }

  removeHooks(): void {
    const container = (this._map as unknown as { _container: HTMLElement })._container;
    L.DomEvent.off(container, 'wheel', this._wheelHandler, this);
    this._abort();
  }

  _onWheelScroll(e: Event): void {
    if (!this._isWheeling) this._onWheelStart(e as WheelEvent);
    this._onWheeling(e as WheelEvent);
  }

  _onWheelStart(_e: WheelEvent): void {
    const map = this._map;

    (map as unknown as { _stop: () => void })._stop();
    const panAnim = (map as unknown as { _panAnim?: { stop: () => void } })._panAnim;
    if (panAnim) panAnim.stop();

    this._isWheeling = true;
    this._yielded = false;
    this._intentDir = 1;
    this._centerPoint = map.getSize().divideBy(2) as L.Point;
    this._startLatLng = map.containerPointToLatLng(this._centerPoint);
    this._moved = false;

    this._intentZoom = map.getZoom();
    this._glideZoom = this._intentZoom;
    this._glideVelocity = 0;
    this._lastStepTime = performance.now();
    this._prevCenter = map.getCenter();
    this._prevZoom = map.getZoom();
    this._preparedTileGoal = null;

    this._zoomAnimationId = requestAnimationFrame(this._tick);
  }

  _onWheeling(e: WheelEvent): void {
    const map = this._map;
    const opts = map.options;

    const raw = L.DomEvent.getWheelDelta(e) * (opts.fluidWheelZoomLevelsPerPixel ?? 0.0034);
    if (raw !== 0) {
      this._intentDir = raw > 0 ? 1 : -1;
      this._intentZoom = this._limitZoom(this._intentZoom + raw);
    }

    this._wheelMousePosition = map.mouseEventToContainerPoint(e);
    this._anchorLatLng = map.containerPointToLatLng(this._wheelMousePosition);

    clearTimeout(this._timeoutId);
    this._timeoutId = setTimeout(this._onWheelIdle.bind(this), WHEEL_IDLE_COMMIT_MS);

    L.DomEvent.preventDefault(e);
    L.DomEvent.stopPropagation(e);
  }

  private _limitZoom(zoom: number): number {
    const map = this._map;
    return Math.max(map.getMinZoom(), Math.min(map.getMaxZoom(), zoom));
  }

  private _cameraTarget(): number {
    return this._limitZoom(this._intentZoom);
  }

  _onWheelIdle(): void {
    if (!this._isWheeling) return;
    if (this._yielded) this._reanchorToLiveCamera();

    this._segGoal = this._cameraTarget();

    const tileLevel = Math.round(this._segGoal);
    if (
      this._preparedTileGoal !== tileLevel &&
      Math.abs(this._segGoal - this._map.getZoom()) > 0.01
    ) {
      prepareGroupedTileLevel(this._map, this._centerAtZoom(tileLevel), tileLevel);
      this._preparedTileGoal = tileLevel;
    }
    const settled =
      Math.abs(this._segGoal - this._map.getZoom()) <= 0.002 &&
      Math.abs(this._glideVelocity) < 0.05;
    if (!settled) {
      this._timeoutId = setTimeout(this._onWheelIdle.bind(this), 50);
      return;
    }
    this._finish();
  }

  _finish(): void {
    if (this._yielded) this._reanchorToLiveCamera();

    this._segGoal = this._cameraTarget();
    const tileLevel = Math.round(this._segGoal);
    if (this._preparedTileGoal !== tileLevel) {
      prepareGroupedTileLevel(this._map, this._centerAtZoom(tileLevel), tileLevel);
      this._preparedTileGoal = tileLevel;
    }
    this._isWheeling = false;
    this._yielded = false;
    this._glideZoom = this._segGoal;
    this._glideVelocity = 0;
    if (this._zoomAnimationId !== undefined) cancelAnimationFrame(this._zoomAnimationId);

    const move = (
      this._map as unknown as {
        _move: (c: L.LatLng, z: number, data?: object) => void;
      }
    )._move.bind(this._map);
    if (this._moved && this._map.getZoom() !== this._segGoal) {
      try {
        move(this._centerAtZoom(this._segGoal), this._segGoal, { flyTo: true });
      } catch (err) {
        if (process.env.NODE_ENV !== 'production') {
          console.warn('[fluid-zoom] final move failed', err);
        }
      }
    }
    (this._map as unknown as { _moveEnd: (noInertia?: boolean) => void })._moveEnd(true);
  }

  _reanchorToLiveCamera(): void {
    const map = this._map;
    try {
      this._centerPoint = map.getSize().divideBy(2) as L.Point;
      this._startLatLng = map.containerPointToLatLng(this._centerPoint);
      if (this._wheelMousePosition) {
        this._anchorLatLng = map.containerPointToLatLng(this._wheelMousePosition);
      }
    } catch {}
  }

  _centerAtZoom(zoom: number): L.LatLng {
    const map = this._map;
    const delta = this._wheelMousePosition.subtract(this._centerPoint);
    let center: L.LatLng;
    if (map.options.fluidWheelZoom === 'center' || (delta.x === 0 && delta.y === 0)) {
      center = this._startLatLng;
    } else {
      center = map.unproject(map.project(this._anchorLatLng, zoom).subtract(delta), zoom);
    }
    return limitCenterSmooth(map, center, zoom);
  }

  _stepSpring(now: number): void {
    const opts = this._map.options;
    const omega = 4000 / Math.max(1, opts.fluidWheelZoomEase ?? 300);
    const maxSpeed = Math.max(0, opts.fluidWheelZoomMaxSpeed ?? 7);
    const minSpeed = Math.max(0, opts.fluidWheelZoomMinSpeed ?? 0);

    let dt = (now - this._lastStepTime) / 1000;
    this._lastStepTime = now;
    if (!(dt > 0)) dt = 0;
    if (dt > 0.05) dt = 0.05;

    const goal = this._segGoal;
    let v = this._glideVelocity;
    if (dt > 0) {
      const x = this._glideZoom - goal;
      const u = v + omega * x;
      const decay = Math.exp(-omega * dt);
      this._glideZoom = goal + (x + u * dt) * decay;
      v = (v - u * omega * dt) * decay;
    }

    const gap = goal - this._glideZoom;
    if (Math.abs(gap) < 1e-4 && Math.abs(v) < 1e-3) {
      this._glideZoom = goal;
      v = 0;
    }
    if (v > maxSpeed) v = maxSpeed;
    else if (v < -maxSpeed) v = -maxSpeed;
    if (minSpeed > 0 && Math.abs(gap) > 1e-4 && Math.abs(v) < minSpeed) {
      v = gap > 0 ? minSpeed : -minSpeed;
    }
    this._glideVelocity = v;
  }

  abortAtCurrentZoom(): void {
    this._abort();
  }

  _abort(): void {
    if (!this._isWheeling) return;
    clearTimeout(this._timeoutId);
    if (this._zoomAnimationId !== undefined) cancelAnimationFrame(this._zoomAnimationId);
    this._isWheeling = false;
    this._yielded = false;
    this._preparedTileGoal = null;
    if (this._moved) {
      this._moved = false;
      (this._map as unknown as { _moveEnd: (noInertia?: boolean) => void })._moveEnd(true);
    }
  }

  yieldCamera(): void {
    if (!this._isWheeling || this._yielded) return;
    this._yielded = true;
  }

  reanchorAfterExternalMove(): void {
    if (!this._isWheeling) return;
    this._yielded = true;
    this._reanchorToLiveCamera();
    this._prevCenter = this._map.getCenter();
    this._prevZoom = this._map.getZoom();
  }

  land(): void {
    if (!this._isWheeling) return;
    this._abort();
  }

  _updateWheelZoom(): void {
    const map = this._map;

    if (
      !this._yielded &&
      (!map.getCenter().equals(this._prevCenter) || map.getZoom() !== this._prevZoom)
    ) {
      this._abort();
      return;
    }

    const prevZoom = this._glideZoom;
    this._segGoal = this._cameraTarget();
    this._stepSpring(performance.now());
    const zoom = this._glideZoom;

    if (zoom !== prevZoom && Math.abs(zoom - map.getZoom()) >= 0.0002) {
      if (this._yielded) this._reanchorToLiveCamera();
      const center = this._centerAtZoom(zoom);

      if (!this._moved) {
        (map as unknown as { _moveStart: (noAnim?: boolean, noMove?: boolean) => void })._moveStart(
          true,
          false
        );
        this._moved = true;
      }

      (
        map as unknown as {
          _move: (c: L.LatLng, z: number, data?: object) => void;
        }
      )._move(center, zoom, { flyTo: true });
      this._prevCenter = map.getCenter();
      this._prevZoom = map.getZoom();
    }

    this._zoomAnimationId = requestAnimationFrame(this._tick);
  }
}

if (!(L.Map as unknown as { FluidWheelZoom?: unknown }).FluidWheelZoom) {
  (L.Map as unknown as { FluidWheelZoom?: unknown }).FluidWheelZoom = FluidWheelZoom;
  L.Map.addInitHook('addHandler', 'fluidWheelZoom', FluidWheelZoom);
}

function fluidZoomActive(map: L.Map): boolean {
  const handler = (map as unknown as { fluidWheelZoom?: FluidWheelZoom }).fluidWheelZoom;
  return !!handler && handler._isWheeling;
}

if (!(L.Map.prototype as unknown as { _fluidZoomPixelOrigin?: boolean })._fluidZoomPixelOrigin) {
  (L.Map.prototype as unknown as { _fluidZoomPixelOrigin: boolean })._fluidZoomPixelOrigin = true;

  const mapProto = L.Map.prototype as unknown as {
    _getNewPixelOrigin: (center: L.LatLng, zoom: number) => L.Point;
  };
  const origGetNewPixelOrigin = mapProto._getNewPixelOrigin;
  mapProto._getNewPixelOrigin = function (
    this: L.Map & { _fluidOriginCache?: { center: L.LatLng; zoom: number; x: number; y: number } },
    center: L.LatLng,
    zoom: number
  ) {
    if (!fluidZoomActive(this)) {
      return origGetNewPixelOrigin.call(this, center, zoom);
    }
    const cached = this._fluidOriginCache;
    if (cached && cached.center === center && cached.zoom === zoom) {
      return L.point(cached.x, cached.y);
    }
    const viewHalf = this.getSize().divideBy(2);
    const origin = this.project(center, zoom)
      .subtract(viewHalf)
      .add((this as unknown as { _getMapPanePos: () => L.Point })._getMapPanePos());
    this._fluidOriginCache = { center, zoom, x: origin.x, y: origin.y };
    return origin;
  };
}

if (!(L.GridLayer.prototype as unknown as { _fluidZoomTransform?: boolean })._fluidZoomTransform) {
  (L.GridLayer.prototype as unknown as { _fluidZoomTransform: boolean })._fluidZoomTransform = true;

  type ZoomLevel = { el: HTMLElement; origin: L.Point; zoom: number };
  const gridProto = L.GridLayer.prototype as unknown as {
    _setZoomTransform: (level: ZoomLevel, center: L.LatLng, zoom: number) => void;
  };
  const origSetZoomTransform = gridProto._setZoomTransform;

  gridProto._setZoomTransform = function (
    this: L.GridLayer,
    level: ZoomLevel,
    center: L.LatLng,
    zoom: number
  ) {
    const map = (this as unknown as { _map?: L.Map })._map;
    if (!fluidZoomActive(map as L.Map)) {
      return origSetZoomTransform.call(this, level, center, zoom);
    }
    const scale = (map as L.Map).getZoomScale(zoom, level.zoom);
    const translate = level.origin
      .multiplyBy(scale)
      .subtract(
        (
          map as unknown as { _getNewPixelOrigin: (c: L.LatLng, z: number) => L.Point }
        )._getNewPixelOrigin(center, zoom)
      );
    if (L.Browser.any3d) {
      level.el.style.transform =
        `translate3d(${translate.x}px,${translate.y}px,0)` + (scale ? ` scale(${scale})` : '');
    } else {
      L.DomUtil.setPosition(level.el, translate);
    }
  };
}

if (!(L.Marker.prototype as unknown as { _fluidZoomPosition?: boolean })._fluidZoomPosition) {
  (L.Marker.prototype as unknown as { _fluidZoomPosition: boolean })._fluidZoomPosition = true;

  const markerProto = L.Marker.prototype as unknown as {
    update: () => void;
    _updateRounded: () => void;
  };
  const origMarkerUpdate = markerProto.update;
  markerProto._updateRounded = origMarkerUpdate;

  markerProto.update = function (this: L.Marker) {
    const map = (this as unknown as { _map?: L.Map })._map;
    if (!fluidZoomActive(map as L.Map)) {
      return origMarkerUpdate.call(this);
    }
    const self = this as unknown as {
      _icon?: HTMLElement;
      _iconAnchor?: L.Point;
      _latlng: L.LatLng;
      _setPos: (pos: L.Point) => void;
    };
    if (self._icon && map) {
      const zoom = map.getZoom();
      const center = map.getCenter();
      self._setPos(
        (
          map as unknown as {
            _latLngToNewLayerPoint: (latlng: L.LatLng, zoom: number, center: L.LatLng) => L.Point;
          }
        )._latLngToNewLayerPoint(self._latlng, zoom, center)
      );
    }
    return this;
  };
}

if (
  !(L.GridLayer.prototype as unknown as { _fluidZoomIdleUpdate?: boolean })._fluidZoomIdleUpdate
) {
  (L.GridLayer.prototype as unknown as { _fluidZoomIdleUpdate: boolean })._fluidZoomIdleUpdate =
    true;

  const gridProto = L.GridLayer.prototype as unknown as {
    _onMoveEnd: (event?: Event) => void;
  };
  const origGridOnMoveEnd = gridProto._onMoveEnd;
  gridProto._onMoveEnd = function (this: L.GridLayer, event?: Event) {
    const map = (this as unknown as { _map?: L.Map })._map;
    if (map && fluidZoomActive(map)) {
      (
        this as unknown as { _setZoomTransforms: (c: L.LatLng, z: number) => void }
      )._setZoomTransforms(map.getCenter(), map.getZoom());
      return;
    }
    return origGridOnMoveEnd.call(this, event);
  };
}

if (!(L.Renderer.prototype as unknown as { _fluidZoomReproject?: boolean })._fluidZoomReproject) {
  (L.Renderer.prototype as unknown as { _fluidZoomReproject: boolean })._fluidZoomReproject = true;

  const rendererProto = L.Renderer.prototype as unknown as {
    _map?: L.Map;
    _onZoom: () => void;
    _onZoomEnd: () => void;
    _update: () => void;
  };
  const origRendererOnZoom = rendererProto._onZoom;
  rendererProto._onZoom = function (this: L.Renderer) {
    const map = (this as unknown as { _map?: L.Map })._map;
    if (!map || !fluidZoomActive(map)) {
      return origRendererOnZoom.call(this);
    }
    const self = this as unknown as {
      _onZoomEnd: () => void;
      _update: () => void;
      _layers: Record<string, unknown>;
    };
    if (!self._layers || Object.keys(self._layers).length === 0) return;
    try {
      self._onZoomEnd();
      self._update();
    } catch {}
  };
}

if (
  !(L.Map.prototype as unknown as { _survivesFluidZoomResize?: boolean })._survivesFluidZoomResize
) {
  (L.Map.prototype as unknown as { _survivesFluidZoomResize: boolean })._survivesFluidZoomResize =
    true;

  const resizeProto = L.Map.prototype as unknown as {
    invalidateSize: (options?: unknown) => void;
    _fluidZoomResizing?: boolean;
  };
  const origInvalidateSize = resizeProto.invalidateSize;

  resizeProto.invalidateSize = function (this: L.Map, options?: unknown) {
    const self = this as L.Map & { fluidWheelZoom?: FluidWheelZoom; _fluidZoomResizing?: boolean };
    const handler = self.fluidWheelZoom;
    if (!handler || !handler._isWheeling) {
      return origInvalidateSize.call(this, options);
    }
    self._fluidZoomResizing = true;
    try {
      origInvalidateSize.call(this, options);
    } finally {
      self._fluidZoomResizing = false;
      handler.reanchorAfterExternalMove();
    }
  };
}

if (!(L.Map.prototype as unknown as { _landsFluidZoom?: boolean })._landsFluidZoom) {
  (L.Map.prototype as unknown as { _landsFluidZoom: boolean })._landsFluidZoom = true;

  const proto = L.Map.prototype as unknown as {
    _stop: () => void;
  };
  const origStop = proto._stop;
  proto._stop = function (this: L.Map) {
    const self = this as L.Map & { fluidWheelZoom?: FluidWheelZoom; _fluidZoomResizing?: boolean };
    const handler = self.fluidWheelZoom;
    if (handler?._isWheeling && !self._fluidZoomResizing) {
      const dragging = !!(L.Draggable as unknown as { _dragging?: boolean })._dragging;
      try {
        if (dragging) handler.abortAtCurrentZoom();
        else handler.land();
      } catch {}
    }
    return origStop.call(this);
  };
}

export default FluidWheelZoom;
