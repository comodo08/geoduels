import 'leaflet';

declare module 'leaflet' {
  interface MapOptions {

    fluidWheelZoom?: boolean | 'center';

    fluidWheelZoomLevelsPerPixel?: number;

    fluidWheelZoomEase?: number;

    fluidWheelZoomMinSpeed?: number;

    fluidWheelZoomMaxSpeed?: number;
  }
}
