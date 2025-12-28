import { state } from './state.js';
import { fetchFeedProxy } from './api.js';

// bindFeedList wires the feed list to toggling and fetching.
export function bindFeedList() {
  const list = document.getElementById('feeds-list');
  if (!list) return;

  list.addEventListener('click', (event) => {
    const item = event.target.closest('.feed-item');
    if (!item) return;
    const name = item.dataset.feedName;
    const url = item.dataset.feedUrl;
    if (!name || !url) return;
    toggleFeed(item, name, url);
  });
}

// renderFeeds replaces the feed list with current URLs.
export function renderFeeds(feedUrls) {
  const list = document.getElementById('feeds-list');
  if (!list) return;

  if (!feedUrls) {
    list.innerHTML = '—';
    return;
  }

  const html = Object.entries(feedUrls)
    .map(([name, url]) => (
      `
        <div class="feed-item" data-feed-name="${name}" data-feed-url="${url}">
          <div class="feed-header">
            <span class="feed-name">${name.replace(/_/g, ' ')}</span>
            <span class="feed-arrow">▸</span>
          </div>
          <div class="feed-summary"></div>
        </div>
      `
    ))
    .join('');

  list.innerHTML = html;
}

// collapseAllFeeds closes any expanded feed rows.
export function collapseAllFeeds() {
  document.querySelectorAll('.feed-item.open').forEach((item) => {
    item.classList.remove('open');
  });
}

// resetFeedCache clears cached feed payloads.
export function resetFeedCache() {
  state.feedCache = {};
}

async function toggleFeed(item, name, url) {
  const summary = item.querySelector('.feed-summary');
  if (!summary) return;

  if (item.classList.contains('open')) {
    item.classList.remove('open');
    return;
  }

  item.classList.add('open');

  if (state.feedCache[name]) {
    renderFeedSummary(name, state.feedCache[name], summary);
    return;
  }

  summary.innerHTML = '<div class="feed-loading">FETCHING...</div>';

  try {
    const json = await fetchFeedProxy(url);
    state.feedCache[name] = json;
    renderFeedSummary(name, json, summary);
  } catch (error) {
    summary.innerHTML = `<div class="feed-error">FAILED: ${error.message}</div>`;
  }
}

function renderFeedSummary(name, json, container) {
  const data = json.data || {};
  const stats = summarizeFeed(name, data);

  let html = stats
    .map((stat) => (
      `<div class="feed-stat"><span>${stat.label}</span><span class="feed-stat-value">${stat.value}</span></div>`
    ))
    .join('');

  if (json.last_updated) {
    const ts = new Date(json.last_updated * 1000).toISOString().slice(11, 19) + 'Z';
    html += `<div class="feed-stat"><span>updated</span><span class="feed-stat-value">${ts}</span></div>`;
  }

  if (json.ttl !== undefined) {
    html += `<div class="feed-stat"><span>ttl</span><span class="feed-stat-value">${json.ttl}s</span></div>`;
  }

  container.innerHTML = html;
}

function summarizeFeed(name, data) {
  switch (name) {
    case 'system_information':
      return [
        { label: 'name', value: data.name || '—' },
        { label: 'operator', value: data.operator || '—' },
        { label: 'timezone', value: data.timezone || '—' },
        { label: 'language', value: data.language || data.languages?.[0] || '—' },
      ];
    case 'station_information': {
      const stations = data.stations || [];
      const withCap = stations.filter((station) => station.capacity > 0);
      const totalCap = stations.reduce((sum, station) => sum + (station.capacity || 0), 0);
      return [
        { label: 'stations', value: stations.length.toLocaleString() },
        { label: 'with capacity', value: withCap.length.toLocaleString() },
        { label: 'total capacity', value: totalCap.toLocaleString() },
      ];
    }
    case 'station_status': {
      const stations = data.stations || [];
      const online = stations.filter((station) => station.is_installed && station.is_renting).length;
      const bikes = stations.reduce((sum, station) => (
        sum + (station.num_bikes_available || station.num_vehicles_available || 0)
      ), 0);
      const docks = stations.reduce((sum, station) => sum + (station.num_docks_available || 0), 0);
      return [
        { label: 'reporting', value: stations.length.toLocaleString() },
        { label: 'online', value: online.toLocaleString() },
        { label: 'bikes avail', value: bikes.toLocaleString() },
        { label: 'docks avail', value: docks.toLocaleString() },
      ];
    }
    case 'free_bike_status':
    case 'vehicle_status': {
      const vehicles = data.vehicles || data.bikes || [];
      const reserved = vehicles.filter((vehicle) => vehicle.is_reserved).length;
      const disabled = vehicles.filter((vehicle) => vehicle.is_disabled).length;
      return [
        { label: 'vehicles', value: vehicles.length.toLocaleString() },
        { label: 'reserved', value: reserved.toLocaleString() },
        { label: 'disabled', value: disabled.toLocaleString() },
        { label: 'available', value: (vehicles.length - reserved - disabled).toLocaleString() },
      ];
    }
    case 'vehicle_types': {
      const types = data.vehicle_types || [];
      const propulsion = [...new Set(types.map((type) => type.propulsion_type))];
      return [
        { label: 'types', value: types.length },
        { label: 'propulsion', value: propulsion.join(', ') || '—' },
      ];
    }
    case 'system_pricing_plans': {
      const plans = data.plans || [];
      return [
        { label: 'plans', value: plans.length },
        { label: 'names', value: plans.map((plan) => plan.name).slice(0, 3).join(', ') || '—' },
      ];
    }
    case 'geofencing_zones': {
      const zones = data.geofencing_zones?.features || [];
      return [{ label: 'zones', value: zones.length }];
    }
    case 'system_alerts': {
      const alerts = data.alerts || [];
      const active = alerts.filter((alert) => !alert.last_updated || alert.last_updated * 1000 > Date.now()).length;
      return [
        { label: 'alerts', value: alerts.length },
        { label: 'active', value: active },
      ];
    }
    case 'system_regions': {
      const regions = data.regions || [];
      return [{ label: 'regions', value: regions.length }];
    }
    case 'gbfs_versions': {
      const versions = data.versions || [];
      return [{ label: 'versions', value: versions.map((version) => version.version).join(', ') || '—' }];
    }
    default: {
      const keys = Object.keys(data);
      return [
        { label: 'fields', value: keys.length },
        { label: 'keys', value: keys.slice(0, 4).join(', ') || '—' },
      ];
    }
  }
}
