/**
 * Contacts Detail - Detail view rendering
 */
Object.assign(ContactsManager, {
renderContactDetail(contact) {
    const container = document.getElementById('contactDetail');
    if (!container) return;

    const displayName = contact.display_name || contact.given_name || 'Unknown';
    const primaryEmail = contact.emails && contact.emails[0] ? contact.emails[0].email : '';
    // Try photo endpoint first, fall back to UI Avatars
    const photoUrl = `/api/contacts/${contact.id}/photo`;
    const fallbackUrl = this.getAvatarImageUrl(displayName, primaryEmail, 120);

    container.innerHTML = `
        <div class="contact-detail-header">
            <div class="contact-detail-avatar">
                <img src="${photoUrl}"
                     alt="${this.escapeHtml(displayName)}"
                     data-fallback-src="${this.escapeHtml(fallbackUrl)}" />
            </div>
            <div class="contact-detail-name">${this.escapeHtml(displayName)}</div>
            ${contact.job_title ?
                `<div class="contact-detail-title">${this.escapeHtml(contact.job_title)}${contact.company_name ? ` at ${this.escapeHtml(contact.company_name)}` : ''}</div>` :
                (contact.company_name ? `<div class="contact-detail-title">${this.escapeHtml(contact.company_name)}</div>` : '')}
        </div>

        <div class="contact-detail-actions">
            <button class="action-btn primary" data-action="email-contact" data-contact-id="${this.escapeHtml(contact.id)}">
                <span>✉️</span> Email
            </button>
            <button class="action-btn" data-action="edit-contact" data-contact-id="${this.escapeHtml(contact.id)}">
                <span>✏️</span> Edit
            </button>
            <button class="action-btn" data-action="delete-contact" data-contact-id="${this.escapeHtml(contact.id)}">
                <span>🗑️</span> Delete
            </button>
        </div>

        <div class="contact-detail-sections">
            ${this.renderEmailSection(contact.emails)}
            ${this.renderPhoneSection(contact.phone_numbers)}
            ${this.renderAddressSection(contact.addresses)}
            ${this.renderNotesSection(contact.notes)}
            ${this.renderBirthdaySection(contact.birthday)}
        </div>
    `;
    this.bindAvatarFallbacks(container);
},

renderEmailSection(emails) {
    if (!emails || emails.length === 0) return '';

    return `
        <div class="contact-section">
            <div class="section-title">Email</div>
            ${emails.map(e => `
                <div class="section-item">
                    <a href="mailto:${this.escapeHtml(e.email)}" class="section-value">${this.escapeHtml(e.email)}</a>
                    ${e.type ? `<span class="section-label">${this.escapeHtml(e.type)}</span>` : ''}
                </div>
            `).join('')}
        </div>
    `;
},

renderPhoneSection(phones) {
    if (!phones || phones.length === 0) return '';

    return `
        <div class="contact-section">
            <div class="section-title">Phone</div>
            ${phones.map(p => `
                <div class="section-item">
                    <a href="tel:${this.escapeHtml(p.number)}" class="section-value">${this.escapeHtml(p.number)}</a>
                    ${p.type ? `<span class="section-label">${this.escapeHtml(p.type)}</span>` : ''}
                </div>
            `).join('')}
        </div>
    `;
},

renderAddressSection(addresses) {
    if (!addresses || addresses.length === 0) return '';

    return `
        <div class="contact-section">
            <div class="section-title">Address</div>
            ${addresses.map(a => {
                const parts = [a.street_address, a.city, a.state, a.postal_code, a.country].filter(Boolean);
                return `
                    <div class="section-item">
                        <div class="section-value">${parts.map(p => this.escapeHtml(p)).join(', ')}</div>
                        ${a.type ? `<span class="section-label">${this.escapeHtml(a.type)}</span>` : ''}
                    </div>
                `;
            }).join('')}
        </div>
    `;
},

renderNotesSection(notes) {
    if (!notes) return '';

    return `
        <div class="contact-section">
            <div class="section-title">Notes</div>
            <div class="section-item">
                <div class="section-value notes">${this.escapeHtml(notes)}</div>
            </div>
        </div>
    `;
},

renderBirthdaySection(birthday) {
    if (!birthday) return '';

    return `
        <div class="contact-section">
            <div class="section-title">Birthday</div>
            <div class="section-item">
                <div class="section-value">${this.escapeHtml(birthday)}</div>
            </div>
        </div>
    `;
},
});

// Single delegated listener for contact-detail action buttons rendered by renderContactDetail.
// Installed once at module load; replaces per-render onclick attributes.
document.addEventListener('click', function (e) {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    const action = target.dataset.action;
    const contactId = target.dataset.contactId;
    if (!contactId) return;
    switch (action) {
        case 'email-contact':
            ContactsManager.emailContact(contactId);
            break;
        case 'edit-contact':
            ContactsManager.editContact(contactId);
            break;
        case 'delete-contact':
            ContactsManager.deleteContact(contactId);
            break;
    }
});
