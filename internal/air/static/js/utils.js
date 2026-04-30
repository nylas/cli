// ====================================
// UTILITY FUNCTIONS
// ====================================

// Debounce function for performance
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Throttle function for scroll/resize events
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// Format relative time
function formatRelativeTime(date) {
    const now = new Date();
    const diff = now - new Date(date);
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return new Date(date).toLocaleDateString();
}

// Escape HTML to prevent XSS. Escapes &, <, >, ", and ' so the result is
// safe in both element and attribute contexts.
function escapeHtml(text) {
    if (text == null) return '';
    return String(text)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
}

// Sanitize untrusted HTML before assigning it to innerHTML.
//
// Uses DOMParser so the browser handles malformed markup, entity tricks,
// and weird tag boundaries that defeat regex-based strippers (e.g.
// "<scr<script>ipt>", "<img src=x onerror=...>", SVG payloads).
//
// Removes: <script>, <iframe>, <object>, <embed>, <link>, <meta>, <base>,
// <form>, all on* event-handler attributes, and any href/src whose value
// after trimming starts with javascript:, data: (except data:image/*),
// vbscript:, or file:.
//
// Always render the return value via innerHTML — the browser has already
// parsed and sanitized the structure.
const DANGEROUS_TAGS = new Set([
    'SCRIPT', 'IFRAME', 'OBJECT', 'EMBED', 'LINK', 'META', 'BASE',
    'FORM', 'INPUT', 'TEXTAREA', 'BUTTON', 'SELECT', 'OPTION'
]);

function sanitizeHtml(html) {
    if (typeof html !== 'string' || html === '') return '';

    const doc = new DOMParser().parseFromString(html, 'text/html');
    const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_ELEMENT, null);

    const toRemove = [];
    let node = walker.currentNode;
    while ((node = walker.nextNode())) {
        if (DANGEROUS_TAGS.has(node.tagName)) {
            toRemove.push(node);
            continue;
        }
        // Strip event handlers and dangerous URLs.
        for (const attr of Array.from(node.attributes)) {
            const name = attr.name.toLowerCase();
            if (name.startsWith('on')) {
                node.removeAttribute(attr.name);
                continue;
            }
            if (name === 'href' || name === 'src' || name === 'xlink:href' || name === 'action' || name === 'formaction') {
                if (!isSafeUrl(attr.value)) {
                    node.removeAttribute(attr.name);
                }
            }
        }
    }
    toRemove.forEach((n) => n.remove());

    return doc.body.innerHTML;
}

// isSafeUrl returns true for relative, anchor, http(s), mailto:, tel:, and
// data:image/* URLs. Everything else (javascript:, vbscript:, data:text/html,
// file:) is rejected.
function isSafeUrl(rawUrl) {
    const url = String(rawUrl).trim();
    if (url === '') return true;
    const m = url.match(/^([a-z][a-z0-9+.-]*):/i);
    if (!m) return true; // relative or anchor
    const scheme = m[1].toLowerCase();
    if (scheme === 'http' || scheme === 'https' || scheme === 'mailto' || scheme === 'tel') return true;
    if (scheme === 'data') {
        // Allow only data:image/* (PNG, JPEG, GIF, SVG, WEBP) — never data:text/html.
        return /^data:image\//i.test(url);
    }
    return false;
}

// Generate unique ID
function generateId() {
    return 'id_' + Math.random().toString(36).substr(2, 9);
}

// stableHashIndex returns a deterministic integer in [0, modulo) derived from
// the input string. Used so that the same person/event always maps to the
// same avatar gradient instead of flickering on each re-render.
function stableHashIndex(seed, modulo) {
    if (modulo <= 0) return 0;
    const s = seed == null ? '' : String(seed);
    let hash = 0;
    for (let i = 0; i < s.length; i++) {
        hash = (hash * 31 + s.charCodeAt(i)) | 0;
    }
    return Math.abs(hash) % modulo;
}

// gradientFor returns a CSS var() reference (--gradient-1 .. --gradient-5)
// that is stable for a given seed string.
function gradientFor(seed) {
    return `var(--gradient-${stableHashIndex(seed, 5) + 1})`;
}

// Check if element is in viewport
function isInViewport(element) {
    const rect = element.getBoundingClientRect();
    return (
        rect.top >= 0 &&
        rect.left >= 0 &&
        rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
        rect.right <= (window.innerWidth || document.documentElement.clientWidth)
    );
}

// Lazy load images using Intersection Observer
function initLazyLoading() {
    const lazyImages = document.querySelectorAll('img[data-src]');

    if ('IntersectionObserver' in window) {
        const imageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src;
                    img.removeAttribute('data-src');
                    observer.unobserve(img);
                }
            });
        }, {
            rootMargin: '50px 0px',
            threshold: 0.01
        });

        lazyImages.forEach(img => imageObserver.observe(img));
    } else {
        // Fallback for browsers without IntersectionObserver
        lazyImages.forEach(img => {
            img.src = img.dataset.src;
            img.removeAttribute('data-src');
        });
    }
}

// Local Storage helpers with error handling
const storage = {
    get(key, defaultValue = null) {
        try {
            const item = localStorage.getItem(key);
            return item ? JSON.parse(item) : defaultValue;
        } catch (e) {
            console.warn('Storage get error:', e);
            return defaultValue;
        }
    },
    set(key, value) {
        try {
            localStorage.setItem(key, JSON.stringify(value));
            return true;
        } catch (e) {
            console.warn('Storage set error:', e);
            return false;
        }
    },
    remove(key) {
        try {
            localStorage.removeItem(key);
            return true;
        } catch (e) {
            console.warn('Storage remove error:', e);
            return false;
        }
    }
};

// Announce message to screen readers
function announce(message, priority = 'polite') {
    const announcer = document.getElementById('announcer');
    if (announcer) {
        announcer.setAttribute('aria-live', priority);
        announcer.textContent = '';
        setTimeout(() => {
            announcer.textContent = message;
        }, 100);
    }
}

// Check online status
function isOnline() {
    return navigator.onLine;
}

// Initialize online/offline detection
function initOnlineDetection() {
    window.addEventListener('online', () => {
        showToast('success', 'Back Online', 'Your connection has been restored');
        document.body.classList.remove('offline');
    });

    window.addEventListener('offline', () => {
        showToast('warning', 'Offline', 'You are currently offline');
        document.body.classList.add('offline');
    });
}

// Initialize utilities
document.addEventListener('DOMContentLoaded', () => {
    initLazyLoading();
    initOnlineDetection();
});

console.log('%c📦 Utils module loaded', 'color: #22c55e;');
