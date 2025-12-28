let configPromise;
let mapsPromise;

// loadConfig fetches UI configuration from the server.
export async function loadConfig() {
  if (configPromise) return configPromise;
  configPromise = (async () => {
    try {
      const res = await fetch('/api/config');
      if (!res.ok) return {};
      return await res.json();
    } catch {
      return {};
    }
  })();
  return configPromise;
}

// ensureGoogleMapsLoaded injects the Maps script once and resolves when ready.
export function ensureGoogleMapsLoaded(apiKey) {
  if (window.google && window.google.maps) {
    return Promise.resolve();
  }
  if (mapsPromise) return mapsPromise;
  if (!apiKey) {
    return Promise.reject(new Error('Missing Google Maps API key'));
  }

  mapsPromise = new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = `https://maps.googleapis.com/maps/api/js?key=${apiKey}`;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Failed to load Google Maps'));
    document.head.appendChild(script);
  });

  return mapsPromise;
}
