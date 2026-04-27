/* Templates Manager - Email templates functionality */

const TemplatesManager = {
    templates: [],
    currentTemplate: null,
    currentVariables: {},

    async init() {
        await this.loadTemplates();
        this.setupTemplateButton();
        console.log('%c📋 Templates module loaded', 'color: #22c55e;');
    },

    // Methods called from HTML templates
    open() {
        this.showTemplatesPicker();
    },

    close() {
        const picker = document.getElementById('templatesPicker');
        if (picker) {
            picker.classList.remove('active');
        }
    },

    showCreate() {
        this.showCreateTemplate();
    },

    hideCreate() {
        const modal = document.getElementById('createTemplateModal');
        if (modal) {
            modal.classList.remove('active');
        }
    },

    bindCreateTemplateForm(modal) {
        const form = modal?.querySelector('.template-form');
        if (!form || form.dataset.boundSubmit === 'true') return;

        form.dataset.boundSubmit = 'true';
        form.addEventListener('submit', event => {
            event.preventDefault();
            this.save();
        });
    },

    filter(query) {
        const templatesList = document.getElementById('templatesList');
        if (!templatesList) return;

        const lowerQuery = (query || '').toLowerCase();
        templatesList.querySelectorAll('.template-item').forEach(item => {
            const name = item.querySelector('.template-name')?.textContent?.toLowerCase() || '';
            item.style.display = name.includes(lowerQuery) ? '' : 'none';
        });
    },

    cancelVariables() {
        const picker = document.getElementById('variablesPicker');
        if (picker) {
            picker.classList.remove('active');
        }
        this.currentTemplate = null;
        this.currentVariables = {};
    },

    applyVariables() {
        const picker = document.getElementById('variablesPicker');
        if (!picker || !this.currentTemplate) return;

        const variables = {};
        picker.querySelectorAll('[data-var]').forEach(input => {
            variables[input.dataset.var] = input.value || '';
        });

        // Expand template with variables
        const expanded = { ...this.currentTemplate };
        let body = expanded.body || '';
        let subject = expanded.subject || '';

        for (const [key, value] of Object.entries(variables)) {
            const regex = new RegExp(`{{${key}}}`, 'g');
            body = body.replace(regex, value);
            subject = subject.replace(regex, value);
        }

        expanded.body = body;
        expanded.subject = subject;

        this.applyTemplate(expanded);
        this.cancelVariables();
    },

    async save() {
        const nameInput = document.getElementById('templateName');
        const categorySelect = document.getElementById('templateCategory');
        const subjectInput = document.getElementById('templateSubject');
        const bodyInput = document.getElementById('templateBody');

        const name = nameInput?.value?.trim();
        const category = categorySelect?.value || '';
        const subject = subjectInput?.value?.trim() || '';
        const body = bodyInput?.value?.trim();

        if (!name || !body) {
            if (typeof showToast === 'function') {
                showToast('warning', 'Required', 'Name and body are required');
            }
            return;
        }

        try {
            const result = await AirAPI.createTemplate({ name, category, subject, body });
            this.templates.push(result.template || result);

            if (typeof showToast === 'function') {
                showToast('success', 'Created', 'Template saved');
            }

            this.hideCreate();
        } catch (error) {
            console.error('Failed to create template:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to save template');
            }
        }
    },

    setupTemplateButton() {
        // Add template button to compose toolbar
        const composeForm = document.getElementById('composeForm');
        if (!composeForm) return;

        const toolbar = composeForm.querySelector('.compose-toolbar');
        if (!toolbar || toolbar.querySelector('.template-btn')) return;

        const templateBtn = document.createElement('button');
        templateBtn.type = 'button';
        templateBtn.className = 'template-btn';
        templateBtn.innerHTML = '📋 Templates';
        templateBtn.title = 'Insert template';
        templateBtn.addEventListener('click', () => this.showTemplatesPicker());

        toolbar.insertBefore(templateBtn, toolbar.firstChild);
    },

    showTemplatesPicker() {
        // Use the existing picker from HTML template
        let picker = document.getElementById('templatesPicker');

        if (!picker) {
            // Fallback: create picker if not in template
            picker = document.createElement('div');
            picker.className = 'templates-picker';
            picker.id = 'templatesPicker';
            picker.innerHTML = `
                <div class="templates-header">
                    <h4>Email Templates</h4>
                    <button class="templates-close" data-action="templates-close">&times;</button>
                </div>
                <div class="templates-search">
                    <input type="text" id="templatesSearch" placeholder="Search templates...">
                </div>
                <div class="templates-list" id="templatesList"></div>
                <div class="templates-footer">
                    <button class="btn-primary" data-action="templates-show-create">+ Create Template</button>
                </div>
            `;
            document.body.appendChild(picker);
        }

        // Populate templates list
        this.renderTemplatesList();

        // Show picker
        setTimeout(() => picker.classList.add('active'), 10);
    },

    renderTemplatesList() {
        const templatesList = document.getElementById('templatesList');
        if (!templatesList) return;

        if (this.templates.length === 0) {
            templatesList.innerHTML = `
                <div class="no-templates">
                    <p>No templates yet</p>
                    <p>Create your first template to speed up email composition</p>
                </div>
            `;
            return;
        }

        templatesList.innerHTML = this.templates.map(t => `
            <div class="template-item" data-action="insert-template" data-template-id="${this.escapeHtml(t.id)}">
                <div class="template-name">${this.escapeHtml(t.name)}</div>
                <div class="template-preview">${this.escapeHtml((t.body || '').substring(0, 100))}${(t.body || '').length > 100 ? '...' : ''}</div>
            </div>
        `).join('');
    },

    async insertTemplate(templateId) {
        try {
            const template = this.templates.find(t => t.id === templateId);
            if (!template) return;

            // Close the templates picker
            this.close();

            // Extract variables from template body ({{variableName}} pattern)
            const variableMatches = (template.body || '').match(/\{\{(\w+)\}\}/g) || [];
            const variables = [...new Set(variableMatches.map(v => v.replace(/\{\{|\}\}/g, '')))];

            // Check for variables
            if (variables.length > 0) {
                template.variables = variables;
                this.promptForVariables(template);
            } else {
                this.applyTemplate(template);
            }
        } catch (error) {
            console.error('Failed to insert template:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to insert template');
            }
        }
    },

    promptForVariables(template) {
        // Store current template for applyVariables method
        this.currentTemplate = template;

        // Use existing picker from HTML or create one
        let picker = document.getElementById('variablesPicker');

        if (!picker) {
            picker = document.createElement('div');
            picker.className = 'variables-picker';
            picker.id = 'variablesPicker';
            document.body.appendChild(picker);
        }

        // Populate variables fields
        const variablesFields = picker.querySelector('#variablesFields') || picker;
        variablesFields.innerHTML = `
            <div class="variables-form">
                <div class="variables-header">
                    <h4>Fill in variables for: ${this.escapeHtml(template.name)}</h4>
                </div>
                ${template.variables.map(v => `
                    <div class="variable-field">
                        <label>${this.escapeHtml(v)}</label>
                        <input type="text" data-var="${this.escapeHtml(v)}" placeholder="Enter ${this.escapeHtml(v)}">
                    </div>
                `).join('')}
                <div class="variables-actions">
                    <button class="btn-secondary" data-action="templates-cancel-variables">Cancel</button>
                    <button class="btn-primary" data-action="templates-apply-variables">Apply</button>
                </div>
            </div>
        `;

        setTimeout(() => picker.classList.add('active'), 10);

        // Focus first input
        const firstInput = picker.querySelector('input[data-var]');
        if (firstInput) firstInput.focus();
    },

    applyTemplate(template) {
        if (typeof ComposeManager === 'undefined') return;

        const els = ComposeManager.getElements();

        if (template.subject && els.subject) {
            els.subject.value = template.subject;
        }
        if (template.body && els.body) {
            // Append to existing content or replace
            const currentBody = els.body.value;
            els.body.value = currentBody ? currentBody + '\n\n' + template.body : template.body;
        }

        if (typeof showToast === 'function') {
            showToast('success', 'Template applied', template.name);
        }
    },

    showCreateTemplate() {
        // Close templates picker first
        this.close();

        // Use existing modal from HTML template
        let modal = document.getElementById('createTemplateModal');

        if (!modal) {
            // Fallback: create modal if not in template
            modal = document.createElement('div');
            modal.className = 'create-template-modal';
            modal.id = 'createTemplateModal';
            modal.innerHTML = `
                <div class="create-template">
                    <h3>Create Template</h3>
                    <form class="template-form">
                        <div class="form-field">
                            <label for="templateName">Template Name</label>
                            <input type="text" id="templateName" placeholder="e.g., Meeting Follow-up" required>
                        </div>
                        <div class="form-field">
                            <label for="templateCategory">Category</label>
                            <select id="templateCategory">
                                <option value="">No category</option>
                                <option value="work">Work</option>
                                <option value="personal">Personal</option>
                                <option value="sales">Sales</option>
                                <option value="support">Support</option>
                            </select>
                        </div>
                        <div class="form-field">
                            <label for="templateSubject">Subject Line</label>
                            <input type="text" id="templateSubject" placeholder="Use {{name}} for variables">
                        </div>
                        <div class="form-field">
                            <label for="templateBody">Body</label>
                            <textarea id="templateBody" placeholder="Hi {{name}},&#10;&#10;Use {{variable}} for dynamic content..." required></textarea>
                        </div>
                        <div class="template-actions">
                            <button type="button" class="btn-secondary" data-action="templates-hide-create">Cancel</button>
                            <button type="submit" class="btn-primary">Save Template</button>
                        </div>
                    </form>
                </div>
            `;
            document.body.appendChild(modal);
        }
        this.bindCreateTemplateForm(modal);

        // Clear form fields
        const nameInput = document.getElementById('templateName');
        const categorySelect = document.getElementById('templateCategory');
        const subjectInput = document.getElementById('templateSubject');
        const bodyInput = document.getElementById('templateBody');

        if (nameInput) nameInput.value = '';
        if (categorySelect) categorySelect.value = '';
        if (subjectInput) subjectInput.value = '';
        if (bodyInput) bodyInput.value = '';

        // Show modal
        setTimeout(() => modal.classList.add('active'), 10);

        // Focus name input
        if (nameInput) nameInput.focus();
    },

    async loadTemplates() {
        try {
            const result = await AirAPI.getTemplates();
            this.templates = result.templates || result || [];
            // Ensure templates is always an array
            if (!Array.isArray(this.templates)) {
                this.templates = [];
            }
        } catch (error) {
            // Silently fail - templates are optional
            console.log('%c📋 Templates: none available (this is OK)', 'color: #a1a1aa;');
            this.templates = [];
        }
    },

    escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
};

// =============================================================================
// INITIALIZATION
// =============================================================================

document.addEventListener('DOMContentLoaded', () => {
    // Initialize all productivity modules
    setTimeout(() => {
        SplitInboxManager.init();
        SnoozeManager.init();
        ScheduledSendManager.init();
        UndoSendManager.init();
        TemplatesManager.init();
    }, 500); // Wait for other modules to load first
});

// Override the old snooze handler
window.showSnoozePicker = () => SnoozeManager.showPicker();
window.handleSnooze = (time) => SnoozeManager.snoozeSelected(time);

// Export managers globally
window.SplitInboxManager = SplitInboxManager;
window.SnoozeManager = SnoozeManager;
window.ScheduledSendManager = ScheduledSendManager;
window.UndoSendManager = UndoSendManager;
window.TemplatesManager = TemplatesManager;

console.log('%c🚀 Productivity module loaded', 'color: #22c55e;');

// Single delegated listener for template list items rendered by renderTemplatesList.
// Installed once at module load; no per-render handler attachment needed.
document.addEventListener('click', function (e) {
    const target = e.target.closest('[data-action="insert-template"]');
    if (!target) return;
    TemplatesManager.insertTemplate(target.dataset.templateId);
});
