// fetchGBFS loads feed data for the viewer UI.
export async function fetchGBFS(url) {
  return fetchJSON('/api/gbfs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  });
}

// fetchValidatorSummary loads a grouped validation summary.
export async function fetchValidatorSummary(url) {
  return fetchJSON('/api/validator-summary', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url, options: { lenientMode: false } }),
  });
}

// fetchFeedProxy loads raw feed data via the proxy endpoint.
export async function fetchFeedProxy(url) {
  return fetchJSON(`/api/proxy?url=${encodeURIComponent(url)}`);
}

// fetchJSON performs a JSON request and raises API errors.
async function fetchJSON(url, options) {
  const res = await fetch(url, options);
  const json = await res.json();
  if (json && json.error) {
    throw new Error(json.error);
  }
  return json;
}
