import { state } from './state.js';

// bindLayout wires sidebar, panels, and mobile gestures.
export function bindLayout() {
  bindSidebarToggle();
  bindSidebarResize();
  bindCollapsiblePanels();
  bindErrorExpansion();
  bindMobileCollapseDrag();
}

// bindSidebarToggle toggles the desktop sidebar.
export function bindSidebarToggle() {
  const toggle = document.getElementById('sidebar-toggle');
  if (!toggle) return;

  toggle.addEventListener('click', () => {
    const app = document.querySelector('.app');
    const isCollapsed = app.classList.toggle('sidebar-collapsed');

    if (isCollapsed) {
      app.style.setProperty('--sidebar-width', '0px');
      toggle.style.left = '0px';
    } else {
      app.style.setProperty('--sidebar-width', '280px');
      toggle.style.left = '280px';
    }

    setTimeout(() => {
      if (state.map) state.map.resize();
      if (state.miniMap) state.miniMap.resize();
    }, 200);
  });
}

// bindSidebarResize allows resizing the sidebar on desktop.
function bindSidebarResize() {
  const handle = document.getElementById('resize-handle');
  const toggle = document.getElementById('sidebar-toggle');
  const app = document.querySelector('.app');
  let isResizing = false;

  if (!handle || !toggle || !app) return;

  handle.addEventListener('mousedown', () => {
    isResizing = true;
    app.classList.add('resizing');
    handle.classList.add('dragging');
    document.body.style.cursor = 'ew-resize';
    document.body.style.userSelect = 'none';
  });

  document.addEventListener('mousemove', (event) => {
    if (!isResizing) return;
    const width = Math.max(0, Math.min(500, event.clientX));
    app.style.setProperty('--sidebar-width', `${width}px`);
    toggle.style.left = `${width}px`;

    if (width === 0) {
      app.classList.add('sidebar-collapsed');
    } else {
      app.classList.remove('sidebar-collapsed');
    }
  });

  document.addEventListener('mouseup', () => {
    if (!isResizing) return;
    isResizing = false;
    app.classList.remove('resizing');
    handle.classList.remove('dragging');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    if (state.map) state.map.resize();
  });
}

// bindCollapsiblePanels toggles panels on mobile.
function bindCollapsiblePanels() {
  document.querySelectorAll('.panel.collapsible .panel-title').forEach((title) => {
    title.addEventListener('click', (event) => {
      if (event.target.closest('button')) return;
      const panel = title.closest('.panel');
      panel.classList.toggle('open');
    });
  });
}

// bindErrorExpansion toggles error rows.
function bindErrorExpansion() {
  const list = document.getElementById('errors-list');
  if (!list) return;

  list.addEventListener('click', (event) => {
    const row = event.target.closest('.error-row');
    if (row) row.classList.toggle('expanded');
  });
}

// bindMobileCollapseDrag enables swipe-like UI for mobile layouts.
function bindMobileCollapseDrag() {
  const app = document.querySelector('.app');
  const handle = document.getElementById('mobile-drag-handle');
  const pullTab = document.getElementById('mobile-pull-tab');
  const isMobile = () => window.matchMedia('(max-width: 900px)').matches;
  let startY = null;
  let dragging = false;
  let fromPullTab = false;
  let lastDy = 0;
  const maxCover = 180;

  if (!app || !handle || !pullTab) return;

  const getY = (event) => (event.touches ? event.touches[0].clientY : event.clientY);

  const startDrag = (event, pull) => {
    if (!isMobile()) return;
    dragging = true;
    fromPullTab = pull;
    startY = getY(event);
    lastDy = 0;
    app.classList.add('dragging');
    event.preventDefault();
  };

  const onMove = (event) => {
    if (!dragging) return;
    const dy = getY(event) - startY;
    lastDy = dy;

    if (fromPullTab) {
      const clamped = Math.max(0, Math.min(160, dy));
      app.style.setProperty('--mobile-drag-offset', `${clamped}px`);
      app.style.setProperty('--mobile-cover-height', '0px');
    } else {
      const cover = Math.max(0, Math.min(maxCover, -dy));
      app.style.setProperty('--mobile-drag-offset', '0px');
      app.style.setProperty('--mobile-cover-height', `${cover}px`);
    }

    event.preventDefault();
  };

  const endDrag = () => {
    if (!dragging) return;
    const dy = lastDy;

    if (fromPullTab) {
      if (Math.abs(dy) < 10) {
        app.classList.remove('mobile-sidebar-collapsed');
        app.classList.toggle('mobile-panels-collapsed');
      } else if (dy > 120) {
        app.classList.remove('mobile-sidebar-collapsed');
        app.classList.remove('mobile-panels-collapsed');
      } else if (dy > 50) {
        app.classList.remove('mobile-sidebar-collapsed');
        app.classList.add('mobile-panels-collapsed');
      }
    } else {
      if (dy < -120) {
        app.classList.add('mobile-sidebar-collapsed');
      } else if (dy < -50) {
        app.classList.add('mobile-panels-collapsed');
        app.classList.remove('mobile-sidebar-collapsed');
      } else if (dy > 50) {
        app.classList.remove('mobile-sidebar-collapsed');
        app.classList.remove('mobile-panels-collapsed');
      }
    }

    app.style.setProperty('--mobile-drag-offset', '0px');
    app.style.setProperty('--mobile-cover-height', '0px');
    app.classList.remove('dragging');

    setTimeout(() => {
      if (state.map) state.map.resize();
      if (state.miniMap) state.miniMap.resize();
    }, 200);

    dragging = false;
    startY = null;
    fromPullTab = false;
    lastDy = 0;
  };

  handle.addEventListener('touchstart', (event) => startDrag(event, false), { passive: false });
  handle.addEventListener('mousedown', (event) => startDrag(event, false));
  pullTab.addEventListener('touchstart', (event) => startDrag(event, true), { passive: false });
  pullTab.addEventListener('mousedown', (event) => startDrag(event, true));

  document.addEventListener('touchmove', onMove, { passive: false });
  document.addEventListener('mousemove', onMove);
  document.addEventListener('touchend', endDrag);
  document.addEventListener('mouseup', endDrag);
}
