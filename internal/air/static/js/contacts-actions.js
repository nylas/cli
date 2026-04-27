/**
 * Contacts Actions - Contact operations (email, edit, delete)
 */
Object.assign(ContactsManager, {
async emailContact(contactId) {
    const contact = this.contacts.find(c => c.id === contactId) || this.selectedContact;
    if (!contact || !contact.emails || contact.emails.length === 0) {
        if (typeof showToast === 'function') {
            showToast('warning', 'No Email', 'This contact has no email address');
        }
        return;
    }

    // Open compose with this contact's email
    if (typeof ComposeManager !== 'undefined') {
        ComposeManager.open('new');
        const els = ComposeManager.getElements();
        if (els.to) {
            els.to.value = contact.emails[0].email;
        }
    }
},

async editContact(contactId) {
    try {
        const contact = await AirAPI.getContact(contactId);
        this.editingContactId = contactId;

        document.getElementById('contactModalTitle').textContent = 'Edit Contact';
        document.getElementById('contactId').value = contactId;
        document.getElementById('contactGivenName').value = contact.given_name || '';
        document.getElementById('contactSurname').value = contact.surname || '';
        document.getElementById('contactNickname').value = contact.nickname || '';
        document.getElementById('contactCompany').value = contact.company_name || '';
        document.getElementById('contactJobTitle').value = contact.job_title || '';
        document.getElementById('contactNotes').value = contact.notes || '';
        document.getElementById('contactBirthday').value = contact.birthday || '';

        // Populate emails
        const emailsContainer = document.getElementById('contactEmails');
        if (contact.emails && contact.emails.length > 0) {
            emailsContainer.innerHTML = contact.emails.map((e, i) => `
                <div class="contact-multi-row">
                    <input type="email" class="contact-input contact-email-input" placeholder="Email address" value="${this.escapeHtml(e.email || '')}">
                    <select class="contact-type-select">
                        <option value="personal" ${e.type === 'personal' ? 'selected' : ''}>Personal</option>
                        <option value="work" ${e.type === 'work' ? 'selected' : ''}>Work</option>
                        <option value="other" ${e.type === 'other' ? 'selected' : ''}>Other</option>
                    </select>
                    ${i === 0 ?
                        '<button type="button" class="contact-add-btn" data-action="contact-add-email-row">+</button>' :
                        '<button type="button" class="contact-remove-btn" data-action="contact-remove-row">−</button>'}
                </div>
            `).join('');
        } else {
            this.resetMultiInputs();
        }

        // Populate phones
        const phonesContainer = document.getElementById('contactPhones');
        if (contact.phone_numbers && contact.phone_numbers.length > 0) {
            phonesContainer.innerHTML = contact.phone_numbers.map((p, i) => `
                <div class="contact-multi-row">
                    <input type="tel" class="contact-input contact-phone-input" placeholder="Phone number" value="${this.escapeHtml(p.number || '')}">
                    <select class="contact-type-select">
                        <option value="mobile" ${p.type === 'mobile' ? 'selected' : ''}>Mobile</option>
                        <option value="home" ${p.type === 'home' ? 'selected' : ''}>Home</option>
                        <option value="work" ${p.type === 'work' ? 'selected' : ''}>Work</option>
                        <option value="other" ${p.type === 'other' ? 'selected' : ''}>Other</option>
                    </select>
                    ${i === 0 ?
                        '<button type="button" class="contact-add-btn" data-action="contact-add-phone-row">+</button>' :
                        '<button type="button" class="contact-remove-btn" data-action="contact-remove-row">−</button>'}
                </div>
            `).join('');
        }

        document.getElementById('contactModalOverlay').classList.remove('hidden');
    } catch (error) {
        console.error('Failed to load contact for editing:', error);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to load contact details');
        }
    }
},

async deleteContact(contactId) {
    if (!confirm('Are you sure you want to delete this contact?')) {
        return;
    }

    try {
        await AirAPI.deleteContact(contactId);

        if (typeof showToast === 'function') {
            showToast('success', 'Deleted', 'Contact deleted successfully');
        }

        // Clear selection if deleted contact was selected
        if (this.selectedContactId === contactId) {
            this.selectedContactId = null;
            this.selectedContact = null;
            const detailContainer = document.getElementById('contactDetail');
            if (detailContainer) {
                detailContainer.innerHTML = `
                    <div class="empty-state">
                        <div class="empty-icon">👥</div>
                        <div class="empty-title">Select a contact</div>
                        <div class="empty-message">Choose a contact to view details</div>
                    </div>
                `;
            }
        }

        // Reload contacts list
        await this.loadContacts();
    } catch (error) {
        console.error('Failed to delete contact:', error);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to delete contact');
        }
    }
},
});
