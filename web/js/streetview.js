import { state } from './state.js';
import { ensureGoogleMapsLoaded, loadConfig } from './config.js';

let panorama = null;
let streetViewService = null;

// bindStreetViewControls wires street view buttons.
export function bindStreetViewControls() {
  const expandBtn = document.getElementById('sv-expand-btn');
  const closeBtn = document.getElementById('sv-close-btn');
  const swapBtn = document.getElementById('mini-map-swap-btn');

  if (expandBtn) expandBtn.addEventListener('click', toggleStreetViewExpand);
  if (closeBtn) closeBtn.addEventListener('click', closeStreetView);
  if (swapBtn) swapBtn.addEventListener('click', toggleStreetViewExpand);
}

// showStreetView loads street view imagery for a location.
export async function showStreetView(lat, lon, label) {
  const panel = document.getElementById('streetview-panel');
  const viewer = document.getElementById('streetview-content');
  const stationName = document.getElementById('sv-station-name');

  panel.classList.remove('hidden');
  stationName.textContent = label;
  cleanupViewers();
  viewer.innerHTML = '<div class="streetview-msg" style="color:var(--yellow);">LOADING...</div>';

  const config = await loadConfig();
  try {
    await ensureGoogleMapsLoaded(config.googleMapsApiKey);
  } catch (error) {
    viewer.innerHTML = `<div class="streetview-msg" style="color:var(--muted);">${error.message}</div>`;
    return;
  }

  if (!streetViewService) {
    streetViewService = new google.maps.StreetViewService();
  }

  try {
    const result = await streetViewService.getPanorama({
      location: { lat, lng: lon },
      radius: 50,
      source: google.maps.StreetViewSource.OUTDOOR,
    });

    viewer.innerHTML = '';
    panorama = new google.maps.StreetViewPanorama(viewer, {
      pano: result.data.location.pano,
      pov: { heading: 0, pitch: 0 },
      zoom: 1,
      addressControl: false,
      showRoadLabels: false,
      motionTracking: false,
      motionTrackingControl: false,
    });
  } catch {
    viewer.innerHTML = '<div class="streetview-msg" style="color:var(--muted);">NO IMAGERY AVAILABLE</div>';
  }
}

// toggleStreetViewExpand expands or collapses the street view panel.
export function toggleStreetViewExpand() {
  const panel = document.getElementById('streetview-panel');
  const miniMapEl = document.getElementById('mini-map');

  state.streetViewExpanded = !state.streetViewExpanded;

  if (state.streetViewExpanded) {
    panel.classList.add('expanded');
    miniMapEl.style.display = 'block';
    initMiniMap();
  } else {
    panel.classList.remove('expanded');
    miniMapEl.style.display = 'none';
  }

  setTimeout(() => {
    if (state.map) state.map.resize();
    if (state.miniMap) state.miniMap.resize();
  }, 250);
}

// closeStreetView hides the street view panel.
export function closeStreetView() {
  const panel = document.getElementById('streetview-panel');
  panel.classList.add('hidden');
  panel.classList.remove('expanded');
  document.getElementById('mini-map').style.display = 'none';
  state.streetViewExpanded = false;
  cleanupViewers();
}

// initMiniMap creates the mini map overlay once.
function initMiniMap() {
  if (state.miniMap) return;

  state.miniMap = new maplibregl.Map({
    container: 'mini-map-content',
    style: {
      version: 8,
      sources: {
        carto: {
          type: 'raster',
          tiles: ['https://basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png'],
          tileSize: 256,
        },
      },
      layers: [{ id: 'base', type: 'raster', source: 'carto' }],
    },
    center: state.map ? state.map.getCenter() : [-73.98, 40.75],
    zoom: state.map ? state.map.getZoom() : 12,
    attributionControl: false,
  });

  if (state.map) {
    state.map.on('move', () => {
      if (state.miniMap && state.streetViewExpanded) {
        state.miniMap.setCenter(state.map.getCenter());
        state.miniMap.setZoom(state.map.getZoom());
      }
    });
  }
}

// cleanupViewers clears any existing panorama views.
function cleanupViewers() {
  if (panorama) {
    panorama = null;
  }
  document.getElementById('streetview-content').innerHTML = '';
}
