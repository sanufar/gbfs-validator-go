import { PRESETS, state } from './state.js';
import { fetchGBFS, fetchValidatorSummary } from './api.js';
import { bindFeedList, collapseAllFeeds, resetFeedCache } from './feeds.js';
import { bindLayout } from './layout.js';
import { initMap, updateLayers, fitBounds } from './map.js';
import {
  clearValidationErrors,
  renderValidationErrors,
  resetValidation,
  setLoading,
  setValidationStatus,
  updateTimestamp,
  updateUI,
} from './ui.js';
import { bindStreetViewControls, showStreetView } from './streetview.js';

// initApp wires UI events and initializes the map.
function initApp() {
  const loadBtn = document.getElementById('load-btn');
  const validateBtn = document.getElementById('validate-btn');
  const urlInput = document.getElementById('url');
  const presetButtons = document.querySelectorAll('.preset');
  const feedCollapseBtn = document.getElementById('feeds-collapse-btn');

  loadBtn.addEventListener('click', () => loadFeed());
  validateBtn.addEventListener('click', () => validateFeed());
  feedCollapseBtn.addEventListener('click', () => collapseAllFeeds());

  presetButtons.forEach((button) => {
    button.addEventListener('click', () => {
      const key = button.dataset.preset;
      if (!key || !PRESETS[key]) return;
      urlInput.value = PRESETS[key];
      loadFeed();
    });
  });

  urlInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') loadFeed();
  });

  bindLayout();
  bindFeedList();
  bindStreetViewControls();
  initMap();
}

// loadFeed fetches and renders a GBFS feed.
async function loadFeed() {
  const url = document.getElementById('url').value.trim();
  if (!url) return;

  state.currentUrl = url;
  setLoading(true);
  resetValidation();
  resetFeedCache();

  try {
    const data = await fetchGBFS(url);
    state.data = data;
    updateUI(data);
    updateLayers(data, handleStationClick, handleVehicleClick);
    fitBounds(data);
    updateTimestamp();
  } catch (error) {
    alert(error.message);
  } finally {
    setLoading(false);
  }
}

// validateFeed runs the validation summary endpoint.
async function validateFeed() {
  if (!state.currentUrl) {
    alert('Load a feed first');
    return;
  }

  setValidationStatus({ state: 'loading' });
  clearValidationErrors();

  try {
    const result = await fetchValidatorSummary(state.currentUrl);
    const errors = result.summary?.errorsCount || 0;
    const valid = !result.summary?.hasErrors;
    setValidationStatus({ valid, errors, version: result.summary?.version?.validated || '—' });
    if (!valid) {
      renderValidationErrors(result.filesSummary || []);
    }
  } catch {
    setValidationStatus({ state: 'offline' });
  }
}

// handleStationClick opens street view for a station.
function handleStationClick(station) {
  const name = station.name || station.station_id;
  showStreetView(station.lat, station.lon, name);
}

// handleVehicleClick opens street view for a vehicle.
function handleVehicleClick(vehicle) {
  const label = vehicle.vehicle_id || vehicle.bike_id || 'Vehicle';
  showStreetView(vehicle.lat, vehicle.lon, label);
}

document.addEventListener('DOMContentLoaded', initApp);
