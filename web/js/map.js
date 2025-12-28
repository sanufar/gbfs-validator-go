import { state } from './state.js';

// initMap initializes the base map and mouse coordinate display.
export function initMap() {
  state.map = new maplibregl.Map({
    container: 'map',
    style: {
      version: 8,
      sources: {
        carto: {
          type: 'raster',
          tiles: ['https://basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png'],
          tileSize: 256,
          attribution: '&copy; <a href="https://carto.com/">CARTO</a>',
        },
      },
      layers: [{ id: 'base', type: 'raster', source: 'carto' }],
    },
    center: [-73.98, 40.75],
    zoom: 12,
    attributionControl: false,
  });

  state.map.on('mousemove', (e) => {
    const coords = document.getElementById('coords');
    coords.textContent = `${e.lngLat.lat.toFixed(5)} ${e.lngLat.lng.toFixed(5)}`;
  });
}

// updateLayers renders station and vehicle layers on the map.
export function updateLayers(data, onStationClick, onVehicleClick) {
  const layers = [];

  if (data.stations?.length) {
    layers.push(new deck.ScatterplotLayer({
      id: 'stations',
      data: data.stations,
      getPosition: (x) => [x.lon, x.lat],
      getFillColor: (x) => getStationColor(x),
      getRadius: 2.5,
      radiusUnits: 'pixels',
      radiusMinPixels: 1.5,
      radiusMaxPixels: 4,
      pickable: true,
      onHover: handleHover,
      onClick: (info) => {
        if (info.object) {
          onStationClick(info.object);
        }
      },
    }));
  }

  if (data.vehicles?.length) {
    layers.push(new deck.ScatterplotLayer({
      id: 'vehicles',
      data: data.vehicles,
      getPosition: (x) => [x.lon, x.lat],
      getFillColor: [0, 255, 255, 200],
      getRadius: 1.5,
      radiusUnits: 'pixels',
      radiusMinPixels: 1,
      radiusMaxPixels: 3,
      pickable: true,
      onHover: handleHover,
      onClick: (info) => {
        if (info.object) {
          onVehicleClick(info.object);
        }
      },
    }));
  }

  if (!state.deckOverlay) {
    state.deckOverlay = new deck.MapboxOverlay({ layers });
    state.map.addControl(state.deckOverlay);
  } else {
    state.deckOverlay.setProps({ layers });
  }
}

// fitBounds adjusts the map viewport to show stations and vehicles.
export function fitBounds(data) {
  const points = [
    ...(data.stations || []).map((s) => [s.lon, s.lat]),
    ...(data.vehicles || []).map((v) => [v.lon, v.lat]),
  ];

  if (!points.length) return;

  const bounds = points.reduce(
    (acc, [lon, lat]) => [
      [Math.min(acc[0][0], lon), Math.min(acc[0][1], lat)],
      [Math.max(acc[1][0], lon), Math.max(acc[1][1], lat)],
    ],
    [[Infinity, Infinity], [-Infinity, -Infinity]],
  );

  state.map.fitBounds(bounds, { padding: 30, maxZoom: 14 });
}

// getStationColor maps station availability to a color.
function getStationColor(station) {
  const online = (station.is_installed === true || station.is_installed === 1)
    && (station.is_renting === true || station.is_renting === 1);
  if (!online) return [80, 80, 80, 220];

  const available = station.num_bikes_available || station.num_vehicles_available || 0;
  const capacity = station.capacity || (available + (station.num_docks_available || 0));
  if (!capacity) return [80, 80, 80, 220];

  const ratio = available / capacity;
  if (ratio > 0.5) return [0, 255, 0, 220];
  if (ratio > 0.2) return [255, 255, 0, 220];
  if (ratio > 0) return [255, 0, 0, 220];
  return [80, 80, 80, 220];
}

// handleHover updates the tooltip for stations or vehicles.
function handleHover(info) {
  const tooltip = document.getElementById('tooltip');
  if (!info.object) {
    tooltip.style.display = 'none';
    return;
  }

  const obj = info.object;
  const isStation = info.layer.id === 'stations';

  tooltip.innerHTML = isStation
    ? `
      <div class="tooltip-name">${obj.name || obj.station_id}</div>
      <div class="tooltip-row"><span class="label">avail</span><span>${obj.num_bikes_available ?? obj.num_vehicles_available ?? 0}</span></div>
      <div class="tooltip-row"><span class="label">docks</span><span>${obj.num_docks_available ?? 0}</span></div>
      <div class="tooltip-row"><span class="label">cap</span><span>${obj.capacity || '—'}</span></div>
    `
    : `
      <div class="tooltip-name">${(obj.vehicle_id || obj.bike_id || '').slice(0, 12)}</div>
      <div class="tooltip-row"><span class="label">type</span><span>${obj.vehicle_type_id || 'bike'}</span></div>
    `;

  tooltip.style.display = 'block';
  tooltip.style.left = `${info.x + 10}px`;
  tooltip.style.top = `${info.y + 10}px`;
}
