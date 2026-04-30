/**
 * Contacts List - List rendering and selection
 */
Object.assign(ContactsManager, {
renderContacts() {
    const container = document.getElementById('contactsList');
    if (!container) return;

    if (this.contacts.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">👥</div>
                <div class="empty-title">No contacts</div>
                <div class="empty-message">
                    ${this.searchQuery ? 'No contacts match your search' : 'Your contacts will appear here'}
                </div>
            </div>
        `;
        return;
    }

    container.innerHTML = this.contacts.map(contact => this.renderContactItem(contact)).join('');

    // Add load more button if there are more contacts
    if (this.hasMore) {
        container.innerHTML += `
            <div class="load-more">
                <button class="btn btn-secondary" data-action="contacts-load-more">
                    Load More
                </button>
            </div>
        `;
    }

    // Re-attach click handlers
    container.querySelectorAll('.contact-item').forEach(item => {
        item.addEventListener('click', () => {
            this.selectContact(item.dataset.contactId);
        });
    });
    this.bindAvatarFallbacks(container);
},

renderContactItem(contact) {
    const isSelected = contact.id === this.selectedContactId;
    const primaryEmail = contact.emails && contact.emails[0] ? contact.emails[0].email : '';
    const displayName = contact.display_name || contact.given_name || 'Unknown';
    // Try photo endpoint first, fall back to UI Avatars. encodeURIComponent
    // protects the URL path; escapeHtml protects the attribute context.
    const photoUrl = `/api/contacts/${encodeURIComponent(contact.id)}/photo`;
    const fallbackUrl = this.getAvatarImageUrl(displayName, primaryEmail, 48);

    return `
        <div class="contact-item ${isSelected ? 'selected' : ''}"
             data-contact-id="${this.escapeHtml(contact.id)}">
            <div class="contact-avatar">
                <img src="${this.escapeHtml(photoUrl)}"
                     alt="${this.escapeHtml(displayName)}"
                     loading="lazy"
                     data-fallback-src="${this.escapeHtml(fallbackUrl)}" />
            </div>
            <div class="contact-info">
                <div class="contact-name">${this.escapeHtml(displayName)}</div>
                ${contact.job_title ? `<div class="contact-title">${this.escapeHtml(contact.job_title)}</div>` : ''}
                ${primaryEmail ? `<div class="contact-email">${this.escapeHtml(primaryEmail)}</div>` : ''}
            </div>
            ${contact.company_name ?
                `<div class="contact-company">${this.escapeHtml(contact.company_name)}</div>` : ''}
        </div>
    `;
},

renderGroupFilter() {
    const select = document.getElementById('contactGroupFilter');
    if (!select) return;

    select.innerHTML = '<option value="">All Contacts</option>';
    this.groups.forEach(group => {
        select.innerHTML += `<option value="${group.id}">${this.escapeHtml(group.name)}</option>`;
    });
},

async selectContact(contactId) {
    // Update selection state
    document.querySelectorAll('.contact-item').forEach(item => {
        item.classList.toggle('selected', item.dataset.contactId === contactId);
    });

    this.selectedContactId = contactId;

    try {
        const contact = await AirAPI.getContact(contactId);
        this.selectedContact = contact;
        this.renderContactDetail(contact);
    } catch (error) {
        console.error('Failed to load contact details:', error);
        this.showDetailError('Failed to load contact details');
    }
},
});
