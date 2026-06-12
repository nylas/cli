/**
 * Universal inspector drawer — one pattern for every resource type.
 * Phase 3: details + delete. Edit forms arrive with the builders phase.
 */
window.StudioDrawer = {
    open(title, subtitle, build) {
        const D = StudioDOM;
        const drawer = document.getElementById('drawer');
        const backdrop = document.getElementById('drawerBackdrop');
        D.clear(drawer);

        const head = D.el('div', 'drawer-head');
        D.add(head, D.el('span', 'drawer-title', title));
        const close = D.el('button', 'drawer-close', '✕');
        close.addEventListener('click', () => this.close());
        D.add(head, close);
        D.add(drawer, head);
        if (subtitle) {
            D.add(drawer, D.el('div', 'drawer-sub', subtitle));
        }
        build(drawer);

        drawer.classList.add('open');
        backdrop.classList.add('open');
        backdrop.onclick = () => this.close();
    },

    close() {
        document.getElementById('drawer').classList.remove('open');
        document.getElementById('drawerBackdrop').classList.remove('open');
    },

    dangerZone(drawer, label, note, action) {
        const D = StudioDOM;
        const actions = D.el('div', 'drawer-actions');
        const del = D.el('button', 'btn btn-danger', label);
        del.addEventListener('click', action);
        D.add(actions, del);
        D.add(drawer, actions, D.el('div', 'danger-note', note));
    },

    async confirmAndRun(question, run) {
        if (!window.confirm(question)) {
            return;
        }
        try {
            const result = await run();
            this.close();
            StudioBoard.render(result.board);
        } catch (error) {
            window.alert(error.message || 'The change failed');
        }
    },

    showAccount(acct) {
        const D = StudioDOM;
        this.open(acct.email, acct.status ? '● ' + acct.status : '', (drawer) => {
            D.add(drawer, D.el('div', 'section-label', 'Identity'));
            D.add(drawer, D.kv('Grant ID', acct.id, true));
            if (acct.workspace_name || acct.workspace_id) {
                D.add(drawer, D.kv('Workspace', acct.workspace_name || acct.workspace_id));
            }

            D.add(drawer, D.el('div', 'section-label', 'Governance'));
            if (acct.policy) {
                D.add(drawer, D.kv('Policy', acct.policy.missing ? '⚠ deleted (' + acct.policy.id + ')' : acct.policy.name || acct.policy.id));
            } else {
                D.add(drawer, D.kv('Policy', 'none attached'));
            }
            D.add(drawer, D.kv('Rules', String((acct.rules || []).length)));

            D.add(drawer, D.el('div', 'section-label', 'Quick actions'));
            const actions = D.el('div', 'drawer-actions');
            const test = D.el('button', 'btn btn-ghost', '✈ Send test email');
            test.addEventListener('click', () => StudioAccountOps.sendTest(acct, test));
            const rotate = D.el('button', 'btn btn-ghost', '⟳ Rotate app password');
            rotate.addEventListener('click', () => {
                this.close();
                StudioAccountOps.rotate(acct);
            });
            const move = D.el('button', 'btn btn-ghost', '⇄ Move to workspace…');
            move.addEventListener('click', () => {
                this.close();
                StudioAccountOps.move(acct);
            });
            D.add(actions, test, rotate, move);
            D.add(drawer, actions);
            D.add(drawer, D.el('div', 'danger-note', 'OTP extraction: run `nylas otp get` with this account active.'));

            this.dangerZone(drawer, 'Delete account…',
                'Revokes the grant. Mailbox contents become unreachable.',
                () => this.confirmAndRun(
                    'Delete ' + acct.email + '? This revokes the grant and its mailbox becomes unreachable.',
                    () => StudioAPI.deleteAccount(acct.id)));
        });
    },

    showPolicy(policy, locked) {
        const D = StudioDOM;
        const title = (locked ? '🔒 ' : '🛡 ') + (policy.name || policy.id);
        const sub = locked ? 'Plan ceiling — read-only' : 'Custom policy';
        this.open(title, sub, (drawer) => {
            D.add(drawer, D.kv('Policy ID', policy.id, true));
            if (policy.missing) {
                D.add(drawer, D.el('div', 'slot slot-warn', '⚠ This policy no longer exists but is still referenced.'));
            }
            if (!locked && !policy.missing) {
                const actions = D.el('div', 'drawer-actions');
                const edit = D.el('button', 'btn', '✎ Edit policy');
                edit.addEventListener('click', async () => {
                    this.close();
                    let full = policy;
                    try {
                        full = await StudioAPI.getPolicy(policy.id);
                    } catch (_error) { /* fall back to summary */ }
                    await StudioBuilders.policyForm(full);
                });
                D.add(actions, edit);
                D.add(drawer, actions);

                this.dangerZone(drawer, 'Delete policy…',
                    'Workspaces referencing it fall back to no policy.',
                    () => this.confirmAndRun(
                        'Delete policy "' + (policy.name || policy.id) + '"?',
                        () => StudioAPI.deletePolicy(policy.id)));
            }
        });
    },

    showRule(rule) {
        const D = StudioDOM;
        this.open('⚡ ' + (rule.name || rule.id), rule.trigger ? 'Trigger: ' + rule.trigger : '', (drawer) => {
            D.add(drawer, D.kv('Rule ID', rule.id, true));
            D.add(drawer, D.kv('Enabled', rule.enabled ? 'yes' : 'no'));
            if (rule.missing) {
                D.add(drawer, D.el('div', 'slot slot-warn', '⚠ This rule no longer exists but is still attached.'));
            }
            const lists = rule.lists || [];
            if (lists.length) {
                D.add(drawer, D.el('div', 'section-label', 'Lists'));
                for (const list of lists) {
                    D.add(drawer, D.kv(list.missing ? '⚠ deleted' : (list.name || list.id),
                        list.missing ? list.id : list.type + ' · ' + list.items_count + ' items'));
                }
            }
            if (!rule.missing) {
                this.dangerZone(drawer, 'Delete rule…',
                    'Detaches the rule from every workspace first.',
                    () => this.confirmAndRun(
                        'Delete rule "' + (rule.name || rule.id) + '"?',
                        () => StudioAPI.deleteRule(rule.id)));
            }
        });
    },

    showList(list) {
        const D = StudioDOM;
        this.open('📋 ' + (list.name || list.id), list.type ? 'Type: ' + list.type + ' (immutable)' : '', (drawer) => {
            D.add(drawer, D.kv('List ID', list.id, true));
            D.add(drawer, D.kv('Items', String(list.items_count || 0)));
            if (list.missing) {
                D.add(drawer, D.el('div', 'slot slot-warn', '⚠ This list no longer exists but is still referenced.'));
                return;
            }

            D.add(drawer, D.el('div', 'section-label', 'Items'));
            const itemsHost = D.el('div', 'list-items');
            D.add(drawer, itemsHost);
            const renderItems = (items) => {
                D.clear(itemsHost);
                if (!items.length) {
                    D.add(itemsHost, D.el('div', 'danger-note', 'No items yet.'));
                }
                for (const item of items) {
                    const chip = D.el('span', 'acct-chip', item + '  ✕');
                    chip.addEventListener('click', () => {
                        StudioModal.confirm('Remove "' + item + '"?',
                            'Rules using this list stop matching the value immediately.',
                            async () => {
                                const result = await StudioAPI.removeListItems(list.id, [item]);
                                StudioBoard.render(result.board);
                                const fresh = await StudioAPI.getListItems(list.id);
                                renderItems(fresh.items || []);
                            });
                    });
                    D.add(itemsHost, chip);
                }
            };
            StudioAPI.getListItems(list.id)
                .then((resp) => renderItems(resp.items || []))
                .catch(() => D.add(itemsHost, D.el('div', 'danger-note', 'Failed to load items.')));

            const addRow = D.el('div', 'field-row');
            const input = StudioModal.input('Add item (' + list.type + ')');
            const add = D.el('button', 'btn btn-ghost btn-inline', '＋');
            add.addEventListener('click', async () => {
                const value = input.value.trim();
                if (!value) {
                    return;
                }
                try {
                    const result = await StudioAPI.addListItems(list.id, [value]);
                    StudioBoard.render(result.board);
                    input.value = '';
                    const fresh = await StudioAPI.getListItems(list.id);
                    renderItems(fresh.items || []);
                } catch (error) {
                    StudioDragDrop.toast(error.message || 'Add failed');
                }
            });
            D.add(addRow, input, add);
            D.add(drawer, addRow);

            this.dangerZone(drawer, 'Delete list…',
                'Rules referencing it stop matching.',
                () => this.confirmAndRun(
                    'Delete list "' + (list.name || list.id) + '"? Rules referencing it stop matching.',
                    () => StudioAPI.deleteList(list.id)));
        });
    },

    showWorkspace(ws) {
        const D = StudioDOM;
        const traits = [];
        if (ws.default) { traits.push('default'); }
        if (ws.auto_group) { traits.push('auto-group'); }
        this.open(ws.name || ws.id, traits.join(' · '), (drawer) => {
            D.add(drawer, D.kv('Workspace ID', ws.id, true));
            D.add(drawer, D.kv('Policy', ws.policy ? (ws.policy.name || ws.policy.id) : 'none'));
            D.add(drawer, D.kv('Rules', String((ws.rules || []).length)));
            D.add(drawer, D.kv('Accounts', String((ws.accounts || []).length)));
            if (!ws.default && (ws.accounts || []).length === 0) {
                this.dangerZone(drawer, 'Delete workspace…',
                    'Only empty, non-default workspaces can be deleted.',
                    () => this.confirmAndRun(
                        'Delete workspace "' + (ws.name || ws.id) + '"?',
                        () => StudioAPI.deleteWorkspace(ws.id)));
            }
        });
    }
};
