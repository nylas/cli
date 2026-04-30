/**
 * Contacts Utils - Utility functions and initialization
 */
Object.assign(ContactsManager, {
getInitials(name) {
    if (!name) return '?';
    const parts = name.split(' ').filter(Boolean);
    if (parts.length >= 2) {
        return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    }
    return name.substring(0, 2).toUpperCase();
},

getAvatarColor(name) {
    const colors = [
        '#e11d48', '#db2777', '#c026d3', '#9333ea',
        '#7c3aed', '#6366f1', '#2563eb', '#0284c7',
        '#0891b2', '#059669', '#16a34a', '#65a30d',
        '#ca8a04', '#ea580c', '#dc2626'
    ];
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
        hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    return colors[Math.abs(hash) % colors.length];
},

escapeHtml(text) {
    if (!text) return '';
    return String(text)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
},

// Generate avatar image URL using UI Avatars API as fallback
getAvatarImageUrl(name, email, size = 80) {
    // Use initials with consistent background color
    const initials = this.getInitials(name || '?');
    const bgColor = this.getAvatarColor(name || '').replace('#', '');
    // UI Avatars API generates professional-looking avatars
    return `https://ui-avatars.com/api/?name=${encodeURIComponent(initials)}&background=${bgColor}&color=fff&size=${size}&font-size=0.4&bold=true`;
},

bindAvatarFallbacks(container) {
    if (!container) return;
    container.querySelectorAll('img[data-fallback-src]').forEach(img => {
        if (img.dataset.fallbackBound === 'true') return;
        img.dataset.fallbackBound = 'true';
        img.addEventListener('error', () => {
            const fallbackSrc = img.dataset.fallbackSrc;
            if (!fallbackSrc || img.src === fallbackSrc) return;
            img.src = fallbackSrc;
        }, { once: true });
    });
}
});

// Debounce utility
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

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('contactsList') || document.querySelector('[data-tab="contacts"]')) {
        ContactsManager.init();
    }
});
