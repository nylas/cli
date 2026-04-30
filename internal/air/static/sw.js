// ====================================
// NYLAS AIR SERVICE WORKER
// Provides offline support and caching
// ====================================

// Bump this whenever shipped JS/CSS changes — the activate handler
// deletes any cache that doesn't match, so users picking up new builds
// don't keep stale assets via stale-while-revalidate. v2 ships the
// invite-card subject heuristic, attendee chips, and inline-calendar
// attachment row.
const CACHE_NAME = 'nylas-air-v2';
const STATIC_ASSETS = [
    '/',
    '/css/style.css',
    '/js/app.js',
    '/js/api.js',
    '/js/email.js',
    '/js/calendar.js',
    '/js/contacts.js',
    '/js/compose.js',
    '/js/settings.js',
    '/js/shortcuts.js',
    '/js/theme.js',
    '/js/utils.js',
    '/js/mobile.js'
];

// Install event - cache static assets
self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(cache => {
                console.log('[SW] Caching static assets');
                return cache.addAll(STATIC_ASSETS);
            })
            .then(() => self.skipWaiting())
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', event => {
    event.waitUntil(
        caches.keys()
            .then(cacheNames => {
                return Promise.all(
                    cacheNames
                        .filter(name => name !== CACHE_NAME)
                        .map(name => {
                            console.log('[SW] Removing old cache:', name);
                            return caches.delete(name);
                        })
                );
            })
            .then(() => self.clients.claim())
    );
});

// Fetch event - network first for API, cache first for static
self.addEventListener('fetch', event => {
    const url = new URL(event.request.url);

    // Skip non-GET requests
    if (event.request.method !== 'GET') {
        return;
    }

    // API requests - network only (no caching)
    if (url.pathname.startsWith('/api/')) {
        event.respondWith(
            fetch(event.request)
                .catch(() => {
                    return new Response(
                        JSON.stringify({ error: 'Offline', message: 'Network unavailable' }),
                        { headers: { 'Content-Type': 'application/json' }, status: 503 }
                    );
                })
        );
        return;
    }

    // Static assets - stale-while-revalidate
    event.respondWith(
        caches.match(event.request)
            .then(cachedResponse => {
                const fetchPromise = fetch(event.request)
                    .then(networkResponse => {
                        // Update cache with fresh response
                        if (networkResponse.ok) {
                            const responseClone = networkResponse.clone();
                            caches.open(CACHE_NAME)
                                .then(cache => cache.put(event.request, responseClone));
                        }
                        return networkResponse;
                    })
                    .catch(() => cachedResponse);

                // Return cached response immediately, or wait for network
                return cachedResponse || fetchPromise;
            })
    );
});
