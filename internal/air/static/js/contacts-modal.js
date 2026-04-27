/**
 * Contacts Modal - Modal management and form handling
 */
Object.assign(ContactsManager, {
showCreateModal() {
    this.editingContactId = null;
    document.getElementById('contactModalTitle').textContent = 'New Contact';
    document.getElementById('contactForm').reset();
    document.getElementById('contactId').value = '';

    // Reset email/phone rows to single empty row
    this.resetMultiInputs();

    document.getElementById('contactModalOverlay').classList.remove('hidden');
},

closeModal() {
    document.getElementById('contactModalOverlay').classList.add('hidden');
    this.editingContactId = null;
},

resetMultiInputs() {
    // Reset emails to single row
    const emailsContainer = document.getElementById('contactEmails');
    emailsContainer.innerHTML = `
        <div class="contact-multi-row">
            <input type="email" class="contact-input contact-email-input" placeholder="Email address">
            <select class="contact-type-select">
                <option value="personal">Personal</option>
                <option value="work">Work</option>
                <option value="other">Other</option>
            </select>
            <button type="button" class="contact-add-btn" data-action="contact-add-email-row">+</button>
        </div>
    `;

    // Reset phones to single row
    const phonesContainer = document.getElementById('contactPhones');
    phonesContainer.innerHTML = `
        <div class="contact-multi-row">
            <input type="tel" class="contact-input contact-phone-input" placeholder="Phone number">
            <select class="contact-type-select">
                <option value="mobile">Mobile</option>
                <option value="home">Home</option>
                <option value="work">Work</option>
                <option value="other">Other</option>
            </select>
            <button type="button" class="contact-add-btn" data-action="contact-add-phone-row">+</button>
        </div>
    `;
},

addEmailRow() {
    const container = document.getElementById('contactEmails');
    const row = document.createElement('div');
    row.className = 'contact-multi-row';
    row.innerHTML = `
        <input type="email" class="contact-input contact-email-input" placeholder="Email address">
        <select class="contact-type-select">
            <option value="personal">Personal</option>
            <option value="work">Work</option>
            <option value="other">Other</option>
        </select>
        <button type="button" class="contact-remove-btn" data-action="contact-remove-row">−</button>
    `;
    container.appendChild(row);
},

addPhoneRow() {
    const container = document.getElementById('contactPhones');
    const row = document.createElement('div');
    row.className = 'contact-multi-row';
    row.innerHTML = `
        <input type="tel" class="contact-input contact-phone-input" placeholder="Phone number">
        <select class="contact-type-select">
            <option value="mobile">Mobile</option>
            <option value="home">Home</option>
            <option value="work">Work</option>
            <option value="other">Other</option>
        </select>
        <button type="button" class="contact-remove-btn" data-action="contact-remove-row">−</button>
    `;
    container.appendChild(row);
},

async saveContact() {
    const contactId = document.getElementById('contactId').value;

    // Collect form data
    const contact = {
        given_name: document.getElementById('contactGivenName').value.trim(),
        surname: document.getElementById('contactSurname').value.trim(),
        nickname: document.getElementById('contactNickname').value.trim(),
        company_name: document.getElementById('contactCompany').value.trim(),
        job_title: document.getElementById('contactJobTitle').value.trim(),
        notes: document.getElementById('contactNotes').value.trim(),
        birthday: document.getElementById('contactBirthday').value || null,
        emails: [],
        phone_numbers: []
    };

    // Collect emails
    document.querySelectorAll('#contactEmails .contact-multi-row').forEach(row => {
        const email = row.querySelector('.contact-email-input').value.trim();
        const type = row.querySelector('.contact-type-select').value;
        if (email) {
            contact.emails.push({ email, type });
        }
    });

    // Collect phone numbers
    document.querySelectorAll('#contactPhones .contact-multi-row').forEach(row => {
        const number = row.querySelector('.contact-phone-input').value.trim();
        const type = row.querySelector('.contact-type-select').value;
        if (number) {
            contact.phone_numbers.push({ number, type });
        }
    });

    // Validate - need at least a name or email
    if (!contact.given_name && !contact.surname && contact.emails.length === 0) {
        if (typeof showToast === 'function') {
            showToast('warning', 'Required', 'Please enter a name or email address');
        }
        return;
    }

    try {
        if (contactId) {
            // Update existing contact
            await AirAPI.updateContact(contactId, contact);
            if (typeof showToast === 'function') {
                showToast('success', 'Updated', 'Contact updated successfully');
            }
        } else {
            // Create new contact
            await AirAPI.createContact(contact);
            if (typeof showToast === 'function') {
                showToast('success', 'Created', 'Contact created successfully');
            }
        }

        this.closeModal();
        this.isInitialized = false; // Force reload
        await this.loadContacts();
    } catch (error) {
        console.error('Failed to save contact:', error);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', error.message || 'Failed to save contact');
        }
    }
},
});
