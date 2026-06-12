/**
 * Create flows: the "+ New" menu and per-resource builders. The rule builder
 * is sentence-shaped and constrained by the live API matrix so invalid
 * combinations cannot be expressed.
 */
window.StudioBuilders = {
    // The rule matrix mirrors the API: inbound exposes from.* only; outbound
    // adds recipient.* and outbound.type. in_list requires a type-matched list.
    MATRIX: {
        triggers: ['inbound', 'outbound'],
        fields: {
            inbound: ['from.address', 'from.domain', 'from.tld'],
            outbound: ['from.address', 'from.domain', 'from.tld', 'recipient.address', 'recipient.domain', 'recipient.tld', 'outbound.type']
        },
        operators: {
            'outbound.type': ['is', 'is_not'],
            default: ['is', 'is_not', 'contains', 'in_list']
        },
        listTypeFor: { address: 'address', domain: 'domain', tld: 'tld' },
        actions: ['archive', 'mark_as_read', 'mark_as_starred', 'mark_as_spam', 'block', 'trash', 'assign_to_folder']
    },

    RECIPES: [
        {
            label: 'Block listed domains',
            rule: { trigger: 'inbound', name: 'Block listed domains', conditions: [{ field: 'from.domain', operator: 'in_list', value: '' }], actions: ['mark_as_spam'] }
        },
        {
            label: 'Archive newsletters',
            rule: { trigger: 'inbound', name: 'Archive newsletters', conditions: [{ field: 'from.address', operator: 'contains', value: 'newsletter' }], actions: ['archive', 'mark_as_read'] }
        }
    ],

    openNewMenu(anchor) {
        const D = StudioDOM;
        const menu = D.clear(document.getElementById('newMenu'));
        const entries = [
            ['Agent account', () => this.accountForm()],
            ['Workspace', () => this.workspaceForm()],
            ['Policy', () => this.policyForm()],
            ['Rule', () => this.ruleForm()],
            ['List', () => this.listForm()]
        ];
        for (const recipe of this.RECIPES) {
            entries.push(['Recipe: ' + recipe.label, () => this.ruleForm(recipe.rule)]);
        }
        for (const [label, action] of entries) {
            const item = D.el('div', 'menu-item', label);
            item.addEventListener('click', () => {
                menu.classList.remove('open');
                Promise.resolve(action()).catch((error) => {
                    StudioDragDrop.toast((error && error.message) || 'Failed to open the form');
                });
            });
            D.add(menu, item);
        }
        const rect = anchor.getBoundingClientRect();
        menu.style.top = (rect.bottom + 6) + 'px';
        menu.style.right = (window.innerWidth - rect.right) + 'px';
        menu.classList.toggle('open');
    },

    // randomIndex draws an unbiased index via rejection sampling.
    randomIndex(max) {
        const limit = Math.floor(0x100000000 / max) * max;
        const buf = new Uint32Array(1);
        do {
            crypto.getRandomValues(buf);
        } while (buf[0] >= limit);
        return buf[0] % max;
    },

    generatePassword() {
        const upper = 'ABCDEFGHJKLMNPQRSTUVWXYZ';
        const lower = 'abcdefghjkmnpqrstuvwxyz';
        const digits = '23456789';
        const all = upper + lower + digits + '!#%+?';
        // Guarantee one of each required class, fill to 24, then shuffle so
        // the guaranteed characters sit at random positions.
        const chars = [
            upper[this.randomIndex(upper.length)],
            lower[this.randomIndex(lower.length)],
            digits[this.randomIndex(digits.length)]
        ];
        while (chars.length < 24) {
            chars.push(all[this.randomIndex(all.length)]);
        }
        for (let i = chars.length - 1; i > 0; i--) {
            const j = this.randomIndex(i + 1);
            [chars[i], chars[j]] = [chars[j], chars[i]];
        }
        return chars.join('');
    },

    workspaceOptions() {
        return (StudioBoard.state.workspaces || []).map((ws) => ({ value: ws.id, label: ws.name || ws.id }));
    },

    accountForm() {
        const M = StudioModal;
        const D = StudioDOM;
        M.open('New agent account', 'A Nylas-hosted mailbox for your agent', (modal) => {
            const email = M.input('agent@yourapp.nylas.email');
            D.add(modal, M.field('Email', email));

            const password = M.input('App password (optional, enables IMAP/SMTP)');
            const gen = D.el('button', 'btn btn-ghost btn-inline', '⟳ Generate');
            gen.addEventListener('click', () => { password.value = this.generatePassword(); });
            const row = D.el('div', 'field-row');
            D.add(row, password, gen);
            D.add(modal, M.field('Mail client access', row));

            const wsOptions = [{ value: '', label: 'Default workspace (automatic)' }].concat(this.workspaceOptions());
            const workspace = M.select(wsOptions, '');
            D.add(modal, M.field('Workspace', workspace));

            D.add(modal, D.el('div', 'modal-note', 'Without a workspace, the account lands in the default workspace and runs at your plan\'s limits unless a policy is attached.'));

            M.actions(modal, 'Create account', () => {
                const body = { email: email.value.trim() };
                if (password.value.trim()) {
                    body.app_password = password.value.trim();
                }
                if (workspace.value) {
                    body.workspace_id = workspace.value;
                }
                return StudioAPI.createAccount(body);
            });
        });
    },

    workspaceForm() {
        const M = StudioModal;
        const D = StudioDOM;
        M.open('New workspace', 'Group accounts under their own policy and rules', (modal) => {
            const name = M.input('Workspace name');
            D.add(modal, M.field('Name', name));
            M.actions(modal, 'Create workspace', () => StudioAPI.createWorkspace({ name: name.value.trim() }));
        });
    },

    policyForm(existing) {
        const M = StudioModal;
        const D = StudioDOM;
        const limitFields = [
            ['limit_count_daily_message_per_grant', 'Daily messages / account'],
            ['limit_attachment_size_limit', 'Attachment size (bytes)'],
            ['limit_attachment_count_limit', 'Attachment count'],
            ['limit_inbox_retention_period', 'Inbox retention (days)'],
            ['limit_spam_retention_period', 'Spam retention (days)']
        ];

        M.open(existing ? 'Edit policy' : 'New policy',
            'Blank limits default to your plan\'s maximum; values above it are rejected by the API', (modal) => {
            const name = M.input('Policy name', existing ? existing.name : '');
            D.add(modal, M.field('Name', name));

            const inputs = {};
            for (const [key, label] of limitFields) {
                const input = M.input('plan maximum');
                input.type = 'number';
                if (existing && existing.limits && existing.limits[key] !== undefined && existing.limits[key] !== null) {
                    input.value = String(existing.limits[key]);
                }
                inputs[key] = input;
                D.add(modal, M.field(label, input));
            }

            M.actions(modal, existing ? 'Save policy' : 'Create policy', () => {
                const body = { name: name.value.trim() };
                const limits = {};
                for (const key of Object.keys(inputs)) {
                    if (inputs[key].value === '') {
                        continue;
                    }
                    limits[key] = Number(inputs[key].value);
                }
                if (Object.keys(limits).length) {
                    body.limits = limits;
                }
                return existing ? StudioAPI.updatePolicy(existing.id, body) : StudioAPI.createPolicy(body);
            });
        });
    },

    availableLists(forField) {
        const wanted = this.MATRIX.listTypeFor[forField.split('.')[1]];
        const { lists } = StudioBoard.collectResources(StudioBoard.state);
        return Array.from(lists.values())
            .filter((list) => !list.missing && list.type === wanted)
            .map((list) => ({ value: list.id, label: list.name + ' (' + list.type + ')' }));
    },

    ruleForm(preset) {
        const M = StudioModal;
        const D = StudioDOM;
        const matrix = this.MATRIX;
        const state = {
            trigger: (preset && preset.trigger) || 'inbound',
            conditions: (preset && preset.conditions && preset.conditions.map((c) => ({ ...c }))) || [{ field: 'from.domain', operator: 'is', value: '' }],
            actions: (preset && preset.actions && preset.actions.slice()) || ['archive']
        };

        M.open('New rule', 'When mail arrives, if conditions match, then act', (modal) => {
            const name = M.input('Rule name', (preset && preset.name) || '');
            D.add(modal, M.field('Name', name));

            const trigger = M.select(matrix.triggers.map((t) => ({ value: t, label: t })), state.trigger);
            D.add(modal, M.field('When', trigger));

            const match = M.select([{ value: 'all', label: 'all conditions match' }, { value: 'any', label: 'any condition matches' }], 'all');
            D.add(modal, M.field('If', match));

            const conditionsHost = D.el('div', 'conditions');
            D.add(modal, conditionsHost);

            const renderConditions = () => {
                D.clear(conditionsHost);
                const fields = matrix.fields[trigger.value] || matrix.fields.inbound;
                state.conditions.forEach((condition, index) => {
                    if (!fields.includes(condition.field)) {
                        condition.field = fields[0];
                    }
                    const row = D.el('div', 'condition-row');

                    const fieldSel = M.select(fields.map((f) => ({ value: f, label: f })), condition.field);
                    fieldSel.addEventListener('change', () => {
                        condition.field = fieldSel.value;
                        renderConditions();
                    });

                    const operators = matrix.operators[condition.field] || matrix.operators.default;
                    if (!operators.includes(condition.operator)) {
                        condition.operator = operators[0];
                    }
                    const opSel = M.select(operators.map((o) => ({ value: o, label: o.replace('_', ' ') })), condition.operator);
                    opSel.addEventListener('change', () => {
                        condition.operator = opSel.value;
                        renderConditions();
                    });

                    let valueNode;
                    if (condition.operator === 'in_list') {
                        const lists = this.availableLists(condition.field);
                        if (lists.length) {
                            valueNode = M.select(lists, condition.value);
                            valueNode.addEventListener('change', () => { condition.value = valueNode.value; });
                            if (!condition.value) {
                                condition.value = lists[0].value;
                            }
                        } else {
                            valueNode = D.el('span', 'condition-hint', 'No ' + matrix.listTypeFor[condition.field.split('.')[1]] + '-type lists yet — create one first');
                            condition.value = '';
                        }
                    } else {
                        valueNode = M.input(condition.field === 'outbound.type' ? 'compose or reply' : 'value', condition.value);
                        valueNode.addEventListener('input', () => { condition.value = valueNode.value; });
                    }

                    const remove = D.el('button', 'btn btn-ghost btn-inline', '✕');
                    remove.addEventListener('click', () => {
                        state.conditions.splice(index, 1);
                        if (!state.conditions.length) {
                            state.conditions.push({ field: fields[0], operator: 'is', value: '' });
                        }
                        renderConditions();
                    });

                    D.add(row, fieldSel, opSel, valueNode, remove);
                    D.add(conditionsHost, row);
                });

                const add = D.el('button', 'btn btn-ghost btn-inline', '＋ add condition');
                add.addEventListener('click', () => {
                    const fields = matrix.fields[trigger.value];
                    state.conditions.push({ field: fields[0], operator: 'is', value: '' });
                    renderConditions();
                });
                D.add(conditionsHost, add);
            };
            trigger.addEventListener('change', renderConditions);
            renderConditions();

            D.add(modal, D.el('div', 'field-label', 'Then'));
            const actionsHost = D.el('div', 'actions-row');
            D.add(modal, actionsHost);
            const renderActions = () => {
                D.clear(actionsHost);
                for (const action of matrix.actions) {
                    const selected = state.actions.some((a) => a === action || a.startsWith(action + '='));
                    const pill = D.el('button', selected ? 'pill selected' : 'pill', action.replace(/_/g, ' '));
                    pill.addEventListener('click', () => {
                        if (selected) {
                            state.actions = state.actions.filter((a) => a !== action && !a.startsWith(action + '='));
                        } else if (action === 'assign_to_folder') {
                            const folder = window.prompt('Folder ID to assign matching mail to:');
                            if (folder && folder.trim()) {
                                state.actions.push(action + '=' + folder.trim());
                            }
                        } else {
                            state.actions.push(action);
                        }
                        renderActions();
                    });
                    D.add(actionsHost, pill);
                }
            };
            renderActions();

            const workspace = M.select(this.workspaceOptions(), (StudioBoard.state.workspaces[0] || {}).id);
            D.add(modal, M.field('Attach to workspace', workspace));

            M.actions(modal, 'Create & attach', () => {
                if (!name.value.trim()) {
                    throw new Error('rule name is required');
                }
                if (!state.actions.length) {
                    throw new Error('pick at least one action');
                }
                // Reject — never silently drop — incomplete conditions: an
                // empty in_list (no matching-type list yet) must block submit.
                for (const c of state.conditions) {
                    if (c.value === '') {
                        throw new Error(c.operator === 'in_list'
                            ? 'an in_list condition has no list selected — create a matching-type list first'
                            : 'every condition needs a value');
                    }
                }
                const conditions = state.conditions.map((c) => ({
                    field: c.field,
                    operator: c.operator,
                    value: c.operator === 'in_list' ? [c.value] : c.value
                }));
                // Split on the FIRST '=' only: folder IDs may themselves
                // contain '=' (matches the CLI's strings.Cut semantics).
                const actions = state.actions.map((a) => {
                    const sep = a.indexOf('=');
                    if (sep === -1) {
                        return { type: a };
                    }
                    return { type: a.slice(0, sep), value: a.slice(sep + 1) };
                });
                return StudioAPI.createRule({
                    workspace_id: workspace.value,
                    name: name.value.trim(),
                    trigger: trigger.value,
                    match: { operator: match.value, conditions },
                    actions
                });
            });
        });
    },

    listForm() {
        const M = StudioModal;
        const D = StudioDOM;
        M.open('New list', 'Typed values for rule in_list conditions (type is immutable)', (modal) => {
            const name = M.input('List name');
            D.add(modal, M.field('Name', name));
            const type = M.select([
                { value: 'domain', label: 'domain — matches from.domain / recipient.domain' },
                { value: 'tld', label: 'tld — matches from.tld / recipient.tld' },
                { value: 'address', label: 'address — matches from.address / recipient.address' }
            ], 'domain');
            D.add(modal, M.field('Type', type));
            const items = M.input('Items, comma-separated (optional)');
            D.add(modal, M.field('Seed items', items));

            M.actions(modal, 'Create list', () => {
                const body = { name: name.value.trim(), type: type.value };
                const seed = items.value.split(',').map((v) => v.trim()).filter(Boolean);
                if (seed.length) {
                    body.items = seed;
                }
                return StudioAPI.createList(body);
            });
        });
    }
};
