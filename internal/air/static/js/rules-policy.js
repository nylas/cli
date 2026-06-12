/**
 * Policy & Rules Module - Nylas-managed mailbox inspection
 */
window.RulesPolicyManager = {
    policiesLoaded: false,
    rulesLoaded: false,
    workspaceLoaded: false,
    listsLoaded: false,

    async loadAll(force = false) {
        await Promise.all([
            this.loadWorkspace(force),
            this.loadPolicies(force),
            this.loadRules(force),
            this.loadLists(force),
        ]);
    },

    async loadWorkspace(force = false) {
        if (this.workspaceLoaded && !force) {
            return;
        }

        const container = document.getElementById('workspaceDetails');
        if (!container || typeof AirAPI === 'undefined') {
            return;
        }

        container.innerHTML = this.loadingMarkup('Loading workspace...');

        try {
            const response = await AirAPI.getWorkspace();
            this.renderWorkspace(response.workspace || null);
            this.workspaceLoaded = true;
        } catch (error) {
            console.error('Failed to load workspace:', error);
            this.renderError(container, 'workspace', error);
        }
    },

    async loadLists(force = false) {
        if (this.listsLoaded && !force) {
            return;
        }

        const container = document.getElementById('agentListList');
        if (!container || typeof AirAPI === 'undefined') {
            return;
        }

        container.innerHTML = this.loadingMarkup('Loading lists...');

        try {
            const response = await AirAPI.getAgentLists();
            this.renderLists(response.lists || []);
            this.listsLoaded = true;
        } catch (error) {
            console.error('Failed to load lists:', error);
            this.renderError(container, 'lists', error);
        }
    },

    async loadPolicies(force = false) {
        if (this.policiesLoaded && !force) {
            return;
        }

        const container = document.getElementById('policyList');
        if (!container || typeof AirAPI === 'undefined') {
            return;
        }

        container.innerHTML = this.loadingMarkup('Loading policies...');

        try {
            const response = await AirAPI.getPolicies();
            this.renderPolicies(response.policies || []);
            this.policiesLoaded = true;
        } catch (error) {
            console.error('Failed to load policies:', error);
            this.renderError(container, 'policies', error);
        }
    },

    async loadRules(force = false) {
        if (this.rulesLoaded && !force) {
            return;
        }

        const container = document.getElementById('ruleList');
        if (!container || typeof AirAPI === 'undefined') {
            return;
        }

        container.innerHTML = this.loadingMarkup('Loading rules...');

        try {
            const response = await AirAPI.getRules();
            this.renderRules(response.rules || []);
            this.rulesLoaded = true;
        } catch (error) {
            console.error('Failed to load rules:', error);
            this.renderError(container, 'rules', error);
        }
    },

    async refreshPolicies() {
        this.policiesLoaded = false;
        await this.loadPolicies(true);
    },

    async refreshRules() {
        this.rulesLoaded = false;
        await this.loadRules(true);
    },

    async refreshWorkspace() {
        this.workspaceLoaded = false;
        await this.loadWorkspace(true);
    },

    async refreshLists() {
        this.listsLoaded = false;
        await this.loadLists(true);
    },

    renderWorkspace(workspace) {
        const container = document.getElementById('workspaceDetails');
        if (!container) {
            return;
        }

        if (!workspace) {
            container.innerHTML = this.emptyMarkup(
                '🗂️',
                'No workspace attached',
                'This Nylas account is not associated with a workspace right now.'
            );
            container.classList.add('rules-policy-empty');
            return;
        }

        container.classList.remove('rules-policy-empty');

        const ruleIDs = Array.isArray(workspace.rule_ids) ? workspace.rule_ids : [];
        const tags = [];
        if (workspace.default === true) {
            tags.push('Default workspace');
        }
        if (workspace.auto_group === true) {
            tags.push('Auto group');
        }
        if (workspace.domain) {
            tags.push(`Domain ${workspace.domain}`);
        }

        container.innerHTML = `
            <article class="rules-policy-card">
                <div class="rules-policy-card-header">
                    <div>
                        <h3 class="rules-policy-card-title">${this.escape(workspace.name || workspace.workspace_id || 'Unnamed workspace')}</h3>
                        <p class="rules-policy-card-meta">${this.escape(workspace.workspace_id || 'No workspace ID')}</p>
                    </div>
                    <span class="rules-policy-pill">Workspace</span>
                </div>
                ${tags.length ? `
                <div class="rules-policy-section">
                    <div class="rules-policy-section-label">Scope</div>
                    <div class="rules-policy-tags">${tags.map((tag) => `<span class="rules-policy-tag">${this.escape(tag)}</span>`).join('')}</div>
                </div>` : ''}
                <div class="rules-policy-section">
                    <div class="rules-policy-section-label">Attached Policy</div>
                    <div class="rules-policy-tags">
                        ${workspace.policy_id ? `<span class="rules-policy-tag rules-policy-tag-mono">${this.escape(workspace.policy_id)}</span>` : '<span class="rules-policy-tag">No policy attached</span>'}
                    </div>
                </div>
                <div class="rules-policy-section">
                    <div class="rules-policy-section-label">Attached Rules</div>
                    <div class="rules-policy-tags">
                        ${ruleIDs.length ? ruleIDs.map((id) => `<span class="rules-policy-tag rules-policy-tag-mono">${this.escape(id)}</span>`).join('') : '<span class="rules-policy-tag">No rules attached</span>'}
                    </div>
                </div>
            </article>
        `;
    },

    renderLists(lists) {
        const container = document.getElementById('agentListList');
        if (!container) {
            return;
        }

        if (!lists.length) {
            container.innerHTML = this.emptyMarkup(
                '📋',
                'No lists configured',
                'This application does not have any lists for rule in_list conditions yet.'
            );
            container.classList.add('rules-policy-empty');
            return;
        }

        container.classList.remove('rules-policy-empty');
        container.innerHTML = lists.map((list) => {
            const itemsCount = typeof list.items_count === 'number' ? list.items_count : 0;

            return `
                <article class="rules-policy-card">
                    <div class="rules-policy-card-header">
                        <div>
                            <h3 class="rules-policy-card-title">${this.escape(list.name || list.id || 'Unnamed list')}</h3>
                            <p class="rules-policy-card-meta">${this.escape(list.id || 'No list ID')}</p>
                        </div>
                        <span class="rules-policy-pill">${this.escape(list.type || 'list')}</span>
                    </div>
                    ${list.description ? `<p class="rules-policy-description">${this.escape(list.description)}</p>` : ''}
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Items</div>
                        <div class="rules-policy-tags">
                            <span class="rules-policy-tag">${this.escape(`${itemsCount} item${itemsCount === 1 ? '' : 's'}`)}</span>
                        </div>
                    </div>
                </article>
            `;
        }).join('');
    },

    renderPolicies(policies) {
        const container = document.getElementById('policyList');
        if (!container) {
            return;
        }

        const assignedMailbox = this.getAssignedMailbox();

        if (!policies.length) {
            container.innerHTML = this.emptyMarkup(
                '🛡️',
                'No policies configured',
                'This Nylas account does not expose any managed policies right now.'
            );
            container.classList.add('rules-policy-empty');
            return;
        }

        container.classList.remove('rules-policy-empty');
        container.innerHTML = policies.map((policy) => {
            const tags = [];
            if (Array.isArray(policy.rules) && policy.rules.length) {
                tags.push(`${policy.rules.length} linked rule${policy.rules.length === 1 ? '' : 's'}`);
            }
            if (policy.application_id) {
                tags.push(`App ${policy.application_id}`);
            }
            if (policy.organization_id) {
                tags.push(`Org ${policy.organization_id}`);
            }

            const limitTags = this.policyLimitTags(policy);
            const optionTags = this.policyOptionTags(policy);

            return `
                <article class="rules-policy-card">
                    <div class="rules-policy-card-header">
                        <div>
                            <h3 class="rules-policy-card-title">${this.escape(policy.name || policy.id || 'Unnamed policy')}</h3>
                            <p class="rules-policy-card-meta">${this.escape(policy.id || 'No policy ID')}</p>
                        </div>
                        <span class="rules-policy-pill">Policy</span>
                    </div>
                    ${(assignedMailbox.email || assignedMailbox.grantID) ? `
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Assigned Mailbox</div>
                        <div class="rules-policy-tags">
                            ${assignedMailbox.email ? `<span class="rules-policy-tag">${this.escape(assignedMailbox.email)}</span>` : ''}
                            ${assignedMailbox.grantID ? `<span class="rules-policy-tag rules-policy-tag-mono">${this.escape(assignedMailbox.grantID)}</span>` : ''}
                        </div>
                    </div>` : ''}
                    ${tags.length ? `
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Scope</div>
                        <div class="rules-policy-tags">${tags.map((tag) => `<span class="rules-policy-tag">${this.escape(tag)}</span>`).join('')}</div>
                    </div>` : ''}
                    ${limitTags.length ? `
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Limits</div>
                        <div class="rules-policy-tags">${limitTags.map((tag) => `<span class="rules-policy-tag">${this.escape(tag)}</span>`).join('')}</div>
                    </div>` : ''}
                    ${optionTags.length ? `
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Options</div>
                        <div class="rules-policy-tags">${optionTags.map((tag) => `<span class="rules-policy-tag">${this.escape(tag)}</span>`).join('')}</div>
                    </div>` : ''}
                </article>
            `;
        }).join('');
    },

    renderRules(rules) {
        const container = document.getElementById('ruleList');
        if (!container) {
            return;
        }

        if (!rules.length) {
            container.innerHTML = this.emptyMarkup(
                '⚙️',
                'No rules configured',
                'This Nylas account does not expose any managed rules right now.'
            );
            container.classList.add('rules-policy-empty');
            return;
        }

        container.classList.remove('rules-policy-empty');
        container.innerHTML = rules.map((rule) => {
            const enabled = rule.enabled !== false;
            const conditions = Array.isArray(rule.match?.conditions) ? rule.match.conditions : [];
            const actions = Array.isArray(rule.actions) ? rule.actions : [];

            return `
                <article class="rules-policy-card">
                    <div class="rules-policy-card-header">
                        <div>
                            <h3 class="rules-policy-card-title">${this.escape(rule.name || rule.id || 'Unnamed rule')}</h3>
                            <p class="rules-policy-card-meta">${this.escape(rule.id || 'No rule ID')}</p>
                        </div>
                        <span class="rules-policy-pill${enabled ? '' : ' muted'}">${enabled ? 'Enabled' : 'Disabled'}</span>
                    </div>
                    ${rule.description ? `<p class="rules-policy-description">${this.escape(rule.description)}</p>` : ''}
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Trigger</div>
                        <div class="rules-policy-tags">
                            <span class="rules-policy-tag">${this.escape(rule.trigger || 'unspecified')}</span>
                            ${typeof rule.priority === 'number' ? `<span class="rules-policy-tag">Priority ${this.escape(String(rule.priority))}</span>` : ''}
                        </div>
                    </div>
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Match Conditions</div>
                        <div class="rules-policy-tags">
                            ${conditions.length ? conditions.map((condition) => `<span class="rules-policy-tag">${this.escape(this.formatCondition(condition))}</span>`).join('') : '<span class="rules-policy-tag">No conditions</span>'}
                        </div>
                    </div>
                    <div class="rules-policy-section">
                        <div class="rules-policy-section-label">Actions</div>
                        <div class="rules-policy-tags">
                            ${actions.length ? actions.map((action) => `<span class="rules-policy-tag">${this.escape(this.formatAction(action))}</span>`).join('') : '<span class="rules-policy-tag">No actions</span>'}
                        </div>
                    </div>
                </article>
            `;
        }).join('');
    },

    renderError(container, resourceName, error) {
        container.classList.add('rules-policy-error');
        const message = error?.message || `Failed to load ${resourceName}.`;
        container.innerHTML = this.emptyMarkup(
            '⚠️',
            `Unable to load ${resourceName}`,
            message
        );
    },

    loadingMarkup(message) {
        return this.emptyMarkup('⏳', 'Loading', message);
    },

    emptyMarkup(icon, title, message) {
        return `
            <div class="empty-state">
                <div class="empty-icon">${icon}</div>
                <div class="empty-title">${this.escape(title)}</div>
                <div class="empty-message">${this.escape(message)}</div>
            </div>
        `;
    },

    policyLimitTags(policy) {
        const tags = [];
        const limits = policy.limits || {};

        if (typeof limits.limit_attachment_size_limit === 'number') {
            tags.push(`Attachment size ${this.humanBytes(limits.limit_attachment_size_limit)}`);
        }
        if (typeof limits.limit_attachment_count_limit === 'number') {
            tags.push(`Attachment count ${limits.limit_attachment_count_limit}`);
        }
        if (typeof limits.limit_storage_total === 'number') {
            tags.push(`Storage ${this.humanBytes(limits.limit_storage_total)}`);
        }
        if (typeof limits.limit_count_daily_message_per_grant === 'number') {
            tags.push(`Daily messages ${limits.limit_count_daily_message_per_grant}`);
        }
        if (typeof limits.limit_inbox_retention_period === 'number') {
            tags.push(`Inbox retention ${limits.limit_inbox_retention_period}d`);
        }
        if (typeof limits.limit_spam_retention_period === 'number') {
            tags.push(`Spam retention ${limits.limit_spam_retention_period}d`);
        }

        return tags;
    },

    policyOptionTags(policy) {
        const tags = [];
        const options = policy.options || {};
        const spamDetection = policy.spam_detection || {};

        if (Array.isArray(options.additional_folders) && options.additional_folders.length) {
            tags.push(`${options.additional_folders.length} extra folder${options.additional_folders.length === 1 ? '' : 's'}`);
        }
        if (options.use_cidr_aliasing === true) {
            tags.push('CIDR aliasing');
        }
        if (spamDetection.use_list_dnsbl === true) {
            tags.push('DNSBL checks');
        }
        if (spamDetection.use_header_anomaly_detection === true) {
            tags.push('Header anomaly detection');
        }
        if (typeof spamDetection.spam_sensitivity === 'number') {
            tags.push(`Spam sensitivity ${spamDetection.spam_sensitivity}`);
        }

        return tags;
    },

    formatCondition(condition) {
        const value = this.compactValue(condition.value);
        return `${condition.field || 'field'} ${condition.operator || 'is'} ${value}`;
    },

    getAssignedMailbox() {
        const view = document.getElementById('rulesPolicyView');
        if (!view) {
            return { email: '', grantID: '' };
        }

        return {
            email: view.dataset.accountEmail || '',
            grantID: view.dataset.grantId || '',
        };
    },

    formatAction(action) {
        const value = this.compactValue(action.value);
        return value ? `${action.type || 'action'} → ${value}` : (action.type || 'action');
    },

    compactValue(value) {
        if (value === null || value === undefined || value === '') {
            return '';
        }
        if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
            return String(value);
        }
        try {
            return JSON.stringify(value);
        } catch (_error) {
            return String(value);
        }
    },

    humanBytes(bytes) {
        if (!bytes || bytes < 1024) {
            return `${bytes || 0} B`;
        }

        const units = ['KB', 'MB', 'GB', 'TB'];
        let value = bytes;
        let unitIndex = -1;
        while (value >= 1024 && unitIndex < units.length - 1) {
            value /= 1024;
            unitIndex += 1;
        }
        return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unitIndex]}`;
    },

    escape(value) {
        return String(value)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }
};
