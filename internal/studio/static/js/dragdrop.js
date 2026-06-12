/**
 * Drag-drop verbs: policy chip → workspace (set policy), rule chip →
 * workspace (attach). Drops onto shared auto-group workspaces confirm first,
 * naming the affected accounts; every applied drop offers Undo via toast.
 */
window.StudioDragDrop = {
    toastTimer: null,

    makeChipDraggable(chip, kind, resource) {
        chip.draggable = true;
        chip.addEventListener('dragstart', (event) => {
            event.dataTransfer.effectAllowed = 'copy';
            event.dataTransfer.setData('text/plain', JSON.stringify({
                kind,
                id: resource.id,
                name: resource.name || resource.id
            }));
            chip.style.opacity = '0.4';
        });
        chip.addEventListener('dragend', () => {
            chip.style.opacity = '';
        });
    },

    makeWorkspaceDroppable(card, ws) {
        card.addEventListener('dragover', (event) => {
            event.preventDefault();
            event.dataTransfer.dropEffect = 'copy';
            card.classList.add('drop-target');
        });
        card.addEventListener('dragleave', () => {
            card.classList.remove('drop-target');
        });
        card.addEventListener('drop', (event) => {
            event.preventDefault();
            card.classList.remove('drop-target');
            let payload;
            try {
                payload = JSON.parse(event.dataTransfer.getData('text/plain'));
            } catch (_error) {
                return;
            }
            if (!this.validPayload(payload)) {
                return;
            }
            this.handleDrop(payload, ws);
        });
    },

    // validPayload accepts only drops describing a policy, rule, or account
    // that exists in the current board state — arbitrary dragged text is
    // ignored.
    validPayload(payload) {
        if (!payload || typeof payload.id !== 'string' || typeof payload.kind !== 'string') {
            return false;
        }
        if (payload.kind === 'account') {
            return (StudioBoard.state.accounts || []).some((acct) => acct.id === payload.id);
        }
        const { policies, rules } = StudioBoard.collectResources(StudioBoard.state);
        if (payload.kind === 'policy') {
            return policies.has(payload.id);
        }
        if (payload.kind === 'rule') {
            return rules.has(payload.id);
        }
        return false;
    },

    handleDrop(payload, ws) {
        if (payload.kind === 'policy') {
            if (ws.policy && ws.policy.id === payload.id) {
                return;
            }
            const previous = ws.policy ? ws.policy.id : '';
            this.confirmShared(ws, 'Attach policy "' + payload.name + '"', () => {
                this.apply(
                    () => StudioAPI.patchWorkspace(ws.id, { policy_id: payload.id }),
                    '🛡 ' + payload.name + ' attached to ' + (ws.name || ws.id),
                    previous ? () => StudioAPI.patchWorkspace(ws.id, { policy_id: previous }) : null
                );
            });
            return;
        }

        if (payload.kind === 'rule') {
            const attached = (ws.rules || []).some((rule) => rule.id === payload.id);
            if (attached) {
                return;
            }
            this.confirmShared(ws, 'Attach rule "' + payload.name + '"', () => {
                this.apply(
                    () => StudioAPI.patchWorkspace(ws.id, { add_rule_ids: [payload.id] }),
                    '⚡ ' + payload.name + ' attached to ' + (ws.name || ws.id),
                    () => StudioAPI.patchWorkspace(ws.id, { remove_rule_ids: [payload.id] })
                );
            });
            return;
        }

        if (payload.kind === 'account') {
            const acct = (StudioBoard.state.accounts || []).find((a) => a.id === payload.id);
            if (!acct || acct.workspace_id === ws.id) {
                return;
            }
            const previous = acct.workspace_id || '';
            this.apply(
                () => StudioAPI.moveAccount(payload.id, ws.id),
                '✉ ' + payload.name + ' moved to ' + (StudioBoard.displayName(ws.name) || ws.id),
                previous ? () => StudioAPI.moveAccount(payload.id, previous) : null
            );
        }
    },

    // confirmShared interrupts drops onto auto-group workspaces shared by
    // multiple accounts: the change affects them all.
    confirmShared(ws, actionLabel, proceed) {
        const accounts = ws.accounts || [];
        if (!ws.auto_group || accounts.length <= 1) {
            proceed();
            return;
        }
        const emails = accounts.map((account) => account.email).join(', ');
        StudioModal.confirm(
            actionLabel + '?',
            'This workspace is shared — the change affects ' + accounts.length + ' accounts: ' + emails,
            proceed
        );
    },

    async apply(run, message, undo) {
        try {
            const result = await run();
            StudioBoard.render(result.board);
            this.toast(message, undo);
        } catch (error) {
            this.toast(error.message || 'The change failed');
        }
    },

    toast(message, undo) {
        const D = StudioDOM;
        const host = D.clear(document.getElementById('toast'));
        D.add(host, D.el('span', '', message));
        if (undo) {
            const btn = D.el('button', 'toast-undo', 'Undo');
            btn.addEventListener('click', async () => {
                host.classList.remove('show');
                try {
                    const result = await undo();
                    StudioBoard.render(result.board);
                } catch (error) {
                    this.toast(error.message || 'Undo failed');
                }
            });
            D.add(host, btn);
        }
        host.classList.add('show');
        clearTimeout(this.toastTimer);
        this.toastTimer = setTimeout(() => host.classList.remove('show'), 6000);
    }
};
