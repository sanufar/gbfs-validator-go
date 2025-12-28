export const PRESETS = {
  nyc: 'https://gbfs.citibikenyc.com/gbfs/2.3/gbfs.json',
  chi: 'https://gbfs.lyft.com/gbfs/2.3/chi/gbfs.json',
  dc: 'https://gbfs.capitalbikeshare.com/gbfs/2.3/gbfs.json',
  sf: 'https://gbfs.lyft.com/gbfs/2.3/bay/gbfs.json',
  bos: 'https://gbfs.lyft.com/gbfs/2.3/bos/gbfs.json',
};

export const state = {
  map: null,
  deckOverlay: null,
  data: null,
  currentUrl: '',
  miniMap: null,
  streetViewExpanded: false,
  feedCache: {},
  googleMapsLoaded: false,
};
