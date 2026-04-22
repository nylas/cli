/* Email Folders - Folder management */
const EmailFoldersManager = typeof EmailListManager !== 'undefined' ? EmailListManager : {};

Object.assign(EmailFoldersManager, {
async loadFolders() {
    try {
        const data = await AirAPI.getFolders();
        this.folders = data.folders || [];
        this.renderFolders(this.folders);
        // Find and store inbox folder ID for initial load
        const inboxFolder = this.findFolderByName('Inbox') || this.findFolderByName('INBOX');
        if (inboxFolder) {
            this.inboxFolderId = inboxFolder.id;
        }
    } catch (error) {
        console.error('Failed to load folders:', error);
        // Keep using template-rendered folders
    }
},

// Find folder by name (case-insensitive) - works with both Google and Microsoft
findFolderByName(name) {
    if (!this.folders) return null;
    const lowerName = name.toLowerCase();
    return this.folders.find(f => (f.name || '').toLowerCase() === lowerName);
},

getVisibleFolders(folders) {
    // Primary folder names to show directly (in order)
    // Use names instead of IDs to support both Google (ID=INBOX) and Microsoft (ID=long-string, name=Inbox)
    const primaryFolderNames = ['inbox', 'starred', 'sent', 'sent items', 'draft', 'drafts', 'archive', 'trash', 'deleted items', 'spam', 'junk', 'junk email'];

    // Helper to get normalized folder name for comparison
    const getNormalizedName = (folder) => {
        const name = (folder.name || folder.id || '').toLowerCase();
        return name;
    };

    // Helper to check if folder is a primary folder
    const isPrimaryFolder = (folder) => {
        const name = getNormalizedName(folder);
        const id = (folder.id || '').toLowerCase();
        // Check both name and ID (Google uses ID as name)
        return primaryFolderNames.includes(name) || primaryFolderNames.includes(id);
    };

    // Helper to get sort order for primary folders
    const getPrimarySortOrder = (folder) => {
        const name = getNormalizedName(folder);
        const id = (folder.id || '').toLowerCase();
        // Map variations to canonical order
        if (name === 'inbox' || id === 'inbox') return 0;
        if (name === 'starred' || id === 'starred') return 1;
        if (name === 'sent' || name === 'sent items' || id === 'sent') return 2;
        if (name === 'draft' || name === 'drafts' || id === 'draft') return 3;
        if (name === 'archive' || id === 'archive') return 4;
        if (name === 'trash' || name === 'deleted items' || id === 'trash') return 5;
        if (name === 'spam' || name === 'junk' || name === 'junk email' || id === 'spam' || id === 'junk') return 6;
        return 99;
    };

    // Filter out Gmail category folders and system folders
    const filteredFolders = folders.filter(f => {
        const id = (f.id || '').toUpperCase();
        const name = (f.name || '').toLowerCase();
        if (id.startsWith('CATEGORY_')) return false;
        if (id === 'UNREAD' || id === 'CHAT' || id === 'IMPORTANT' || id === 'SNOOZED' || id === 'SCHEDULED') return false;
        // Microsoft: filter out some system folders
        if (name === 'conversation history' || name === 'outbox' || name === 'scheduled') return false;
        return true;
    });

    // Separate primary and other folders
    const primaryFolders = [];
    const otherFolders = [];

    filteredFolders.forEach(f => {
        if (isPrimaryFolder(f)) {
            primaryFolders.push(f);
        } else {
            otherFolders.push(f);
        }
    });

    // Sort primary folders by predefined order
    primaryFolders.sort((a, b) => {
        return getPrimarySortOrder(a) - getPrimarySortOrder(b);
    });

    // Sort other folders alphabetically
    otherFolders.sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id));

    return [...primaryFolders, ...otherFolders];
},

renderFolders(folders) {
    const folderList = document.getElementById('folderList') || document.querySelector('.folder-group');
    if (!folderList) return;

    const visibleFolders = this.getVisibleFolders(folders);
    folderList.innerHTML = '';

    visibleFolders.forEach(folder => {
        const folderName = (folder.name || '').toLowerCase();
        const folderId = (folder.id || '').toLowerCase();
        // Check if this folder is the current one, or if it's inbox and no folder is selected yet
        const isInbox = folderName === 'inbox' || folderId === 'inbox';
        const isActive = folder.id === this.currentFolder || (isInbox && !this.currentFolder);
        const item = this.createFolderElement(folder, isActive);
        folderList.appendChild(item);
    });
},

createFolderElement(folder, isActive = false) {
    // Icon mapping by name (normalized) - works with both Google and Microsoft
    const iconsByName = {
        'inbox': '📥',
        'sent': '📤',
        'sent items': '📤',
        'draft': '📝',
        'drafts': '📝',
        'trash': '🗑️',
        'deleted items': '🗑️',
        'spam': '⚠️',
        'junk': '⚠️',
        'junk email': '⚠️',
        'starred': '⭐',
        'snoozed': '🕐',
        'scheduled': '📅',
        'archive': '📦'
    };

    const folderName = (folder.name || '').toLowerCase();
    const folderId = (folder.id || '').toLowerCase();
    // Check both name and ID for icon lookup
    const icon = iconsByName[folderName] || iconsByName[folderId] || '📁';

    // Clean up display name
    let displayName = folder.name || folder.id;
    // Capitalize first letter, lowercase rest
    displayName = displayName.charAt(0).toUpperCase() + displayName.slice(1).toLowerCase();

    const div = document.createElement('div');
    div.className = `folder-item${isActive ? ' active' : ''}`;
    div.setAttribute('data-folder-id', folder.id);
    div.setAttribute('data-folder-name', displayName);
    div.setAttribute('role', 'listitem');
    div.setAttribute('tabindex', '0');
    if (isActive) div.setAttribute('aria-current', 'true');

    const count = folder.unread_count || folder.total_count || 0;
    div.innerHTML = `
        <span class="folder-icon">${icon}</span>
        <span>${this.escapeHtml(displayName)}</span>
        ${count > 0 ? `<span class="folder-count${folder.unread_count > 0 ? ' unread' : ''}">${count}</span>` : ''}
    `;

    return div;
},

escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
},
});

if (typeof module !== 'undefined' && module.exports) {
    module.exports = EmailFoldersManager;
}
