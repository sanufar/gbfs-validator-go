import { renderFeeds } from './feeds.js';

// setLoading toggles the loading overlay and disables the submit button.
export function setLoading(isLoading) {
  document.getElementById('loading').classList.toggle('active', isLoading);
  document.getElementById('load-btn').disabled = isLoading;
}

// resetValidation clears validation state in the UI.
export function resetValidation() {
  document.getElementById('v-dot').className = 'status-indicator';
  document.getElementById('v-status').textContent = 'Not validated';
  document.getElementById('v-version').textContent = '—';
  document.getElementById('errors-wrap').classList.add('hidden');
}

// updateUI refreshes system and summary data on the sidebar.
export function updateUI(data) {
  document.getElementById('version').textContent = `v${data.version}`;
  document.getElementById('system-name').textContent = data.systemInfo?.name || '—';
  document.getElementById('system-name-inline').textContent = data.systemInfo?.name || '—';
  document.getElementById('system-meta').textContent =
    [data.systemInfo?.operator, data.systemInfo?.timezone].filter(Boolean).join(' · ') || '—';

  const stations = data.stations || [];
  const vehicles = data.vehicles || [];
  const bikes = stations.reduce((sum, station) => (
    sum + (station.num_bikes_available || station.num_vehicles_available || 0)
  ), 0);
  const docks = stations.reduce((sum, station) => sum + (station.num_docks_available || 0), 0);
  const capacity = stations.reduce((sum, station) => sum + (station.capacity || 0), 0);

  document.getElementById('d-stations').textContent = stations.length.toLocaleString();
  document.getElementById('d-vehicles').textContent = vehicles.length.toLocaleString();
  document.getElementById('d-bikes').textContent = bikes.toLocaleString();
  document.getElementById('d-docks').textContent = docks.toLocaleString();
  document.getElementById('d-capacity').textContent = capacity.toLocaleString();
  document.getElementById('d-util').textContent = capacity
    ? `${((bikes / capacity) * 100).toFixed(1)}%`
    : '—';

  renderFeeds(data.feedUrls);
}

// updateTimestamp shows the latest load time.
export function updateTimestamp() {
  document.getElementById('time').textContent = `${new Date().toISOString().slice(11, 19)}Z`;
}

// setValidationStatus updates the validation indicator and summary text.
export function setValidationStatus(status) {
  const dot = document.getElementById('v-dot');
  const label = document.getElementById('v-status');

  if (status.state === 'loading') {
    dot.className = 'status-indicator loading';
    label.textContent = 'Checking...';
    return;
  }

  if (status.state === 'offline') {
    dot.className = 'status-indicator';
    label.textContent = 'Offline';
    document.getElementById('v-version').textContent = '—';
    return;
  }

  dot.className = `status-indicator ${status.valid ? 'valid' : 'invalid'}`;
  label.textContent = status.valid ? 'Valid' : `${status.errors} errors`;
  document.getElementById('v-version').textContent = status.version || '—';
}

// renderValidationErrors renders grouped validation errors.
export function renderValidationErrors(filesSummary) {
  const wrap = document.getElementById('errors-wrap');
  const list = document.getElementById('errors-list');

  if (!filesSummary || !filesSummary.length) {
    wrap.classList.add('hidden');
    return;
  }

  wrap.classList.remove('hidden');
  list.innerHTML = filesSummary
    .filter((file) => file.hasErrors)
    .flatMap((file) => file.groupedErrors.map((err) => (
      `
        <div class="error-row">
          <span class="error-file">${file.file.replace('.json', '')}</span>
          <span class="error-msg" title="${err.message}">${err.message}</span>
          <span class="error-count">${err.count}</span>
        </div>
      `
    )))
    .join('');
}

// clearValidationErrors hides validation error output.
export function clearValidationErrors() {
  document.getElementById('errors-wrap').classList.add('hidden');
}
