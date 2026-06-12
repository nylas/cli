/**
 * Board renderer — palette + workspace cards, rendered exclusively from the
 * server's board state (/api/board).
 */
window.StudioBoard = {
    state: null,

    async load() {
        try {
            const board = await StudioAPI.getBoard();
            this.render(board);
        } catch (error) {
            const boardEl = document.getElementById('board');
            StudioDOM.clear(boardEl);
            StudioDOM.add(boardEl, StudioDOM.el('div', 'empty', 'Failed to load board: ' + (error.message || 'unknown error')));
        }
    },

    render(board) {
        if (!board) {
            return;
        }
        this.state = board;
        this.renderTotals(board.totals || {});
        this.renderPalette(board);
        this.renderWorkspaces(board.workspaces || []);
        this.renderStatus(board);
        StudioAccounts.render(board);
    },

    // displayName strips the connector UUID from auto-generated workspace
    // names ("Workspace for 'Nylas' Connector '<uuid>'" → "Workspace for
    // 'Nylas'"). Full names stay available in tooltips and drawers.
    displayName(name) {
        if (!name) {
            return name;
        }
        return name.replace(/\s*Connector\s+'[0-9a-fA-F-]{36}'/, '').trim();
    },

    // ceilingPolicyID is the plan ceiling: the default workspace's policy.
    ceilingPolicyID(board) {
        for (const ws of board.workspaces || []) {
            if (ws.default && ws.policy) {
                return ws.policy.id;
            }
        }
        return '';
    },

    renderTotals(totals) {
        const D = StudioDOM;
        const el = D.clear(document.getElementById('totals'));
        const parts = [
            { label: (totals.accounts || 0) + ' accounts', view: 'accounts' },
            { label: (totals.workspaces || 0) + ' workspaces', view: 'board' },
            { label: (totals.policies || 0) + ' policies', view: 'board' },
            { label: (totals.rules || 0) + ' rules', view: 'board' },
            { label: (totals.lists || 0) + ' lists', view: 'board' }
        ];
        parts.forEach((part, index) => {
            if (index > 0) {
                D.add(el, D.el('span', '', ' · '));
            }
            const link = D.el('button', 'total-link', part.label);
            link.type = 'button';
            link.addEventListener('click', () => StudioViews.show(part.view));
            D.add(el, link);
        });
    },

    // collectResources dedupes palette entries from workspace attachments,
    // rule list references, and the orphan/unused sections.
    collectResources(board) {
        const policies = new Map();
        const rules = new Map();
        const lists = new Map();
        const ruleAttached = new Set();

        for (const ws of board.workspaces || []) {
            if (ws.policy) {
                policies.set(ws.policy.id, ws.policy);
            }
            for (const rule of ws.rules || []) {
                rules.set(rule.id, rule);
                ruleAttached.add(rule.id);
                for (const list of rule.lists || []) {
                    lists.set(list.id, list);
                }
            }
        }
        for (const policy of board.orphan_policies || []) {
            policies.set(policy.id, policy);
        }
        for (const rule of board.orphan_rules || []) {
            rules.set(rule.id, rule);
        }
        for (const list of board.unused_lists || []) {
            lists.set(list.id, list);
        }
        return { policies, rules, lists, ruleAttached };
    },

    renderPalette(board) {
        const D = StudioDOM;
        const palette = D.clear(document.getElementById('palette'));
        const { policies, rules, lists, ruleAttached } = this.collectResources(board);
        const ceilingID = this.ceilingPolicyID(board);

        D.add(palette, D.el('div', 'palette-label', 'Policies · ' + policies.size));
        for (const policy of policies.values()) {
            const locked = policy.id === ceilingID;
            const chip = D.el('div', locked ? 'chip chip-policy locked' : 'chip chip-policy',
                (locked ? '🔒 ' : '🛡 ') + (policy.name || policy.id));
            if (locked) {
                D.add(chip, D.el('span', 'chip-tag', 'plan ceiling'));
            } else if (policy.missing) {
                D.add(chip, D.el('span', 'chip-tag', 'missing'));
            }
            chip.addEventListener('click', () => StudioDrawer.showPolicy(policy, locked));
            if (!policy.missing) {
                StudioDragDrop.makeChipDraggable(chip, 'policy', policy);
            }
            D.add(palette, chip);
        }

        D.add(palette, D.el('div', 'palette-label', 'Rules · ' + rules.size));
        for (const rule of rules.values()) {
            const chip = D.el('div', 'chip chip-rule', '⚡ ' + (rule.name || rule.id));
            if (!ruleAttached.has(rule.id)) {
                D.add(chip, D.el('span', 'chip-tag', 'unattached'));
            }
            chip.addEventListener('click', () => StudioDrawer.showRule(rule));
            if (!rule.missing) {
                StudioDragDrop.makeChipDraggable(chip, 'rule', rule);
            }
            D.add(palette, chip);
        }

        D.add(palette, D.el('div', 'palette-label', 'Lists · ' + lists.size));
        for (const list of lists.values()) {
            const chip = D.el('div', 'chip chip-list', '📋 ' + (list.name || list.id));
            D.add(chip, D.el('span', 'chip-tag', String(list.items_count || 0)));
            chip.addEventListener('click', () => StudioDrawer.showList(list));
            D.add(palette, chip);
        }
    },

    renderWorkspaces(workspaces) {
        const D = StudioDOM;
        const board = D.clear(document.getElementById('board'));
        if (!workspaces.length) {
            D.add(board, D.el('div', 'empty', 'No workspaces yet. Create an agent account to get one automatically.'));
            return;
        }

        for (const ws of workspaces) {
            board.appendChild(this.workspaceCard(ws));
        }
    },

    workspaceCard(ws) {
        const D = StudioDOM;
        const card = D.el('div', 'ws-card');
        StudioDragDrop.makeWorkspaceDroppable(card, ws);

        const head = D.el('div', 'ws-head');
        const name = D.el('span', 'ws-name', this.displayName(ws.name) || ws.id);
        name.title = ws.name || ws.id;
        name.style.cursor = 'pointer';
        name.addEventListener('click', () => StudioDrawer.showWorkspace(ws));
        D.add(head, name);
        if (ws.default) {
            D.add(head, D.el('span', 'badge badge-default', 'default'));
        }
        if ((ws.accounts || []).length > 1 && ws.auto_group) {
            D.add(head, D.el('span', 'badge badge-shared', '⚠ shared · ' + ws.accounts.length));
        }
        D.add(card, head);

        D.add(card, D.el('div', 'section-label', 'Policy'));
        D.add(card, this.policySlot(ws));

        const rules = ws.rules || [];
        D.add(card, D.el('div', 'section-label', 'Rules · ' + rules.length));
        if (!rules.length) {
            D.add(card, D.el('div', 'slot slot-empty', 'No rules attached'));
        }
        for (const rule of rules) {
            D.add(card, this.ruleSlot(rule));
        }

        const accounts = ws.accounts || [];
        D.add(card, D.el('div', 'section-label', 'Accounts · ' + accounts.length));
        const row = D.el('div', 'acct-row');
        if (!accounts.length) {
            D.add(row, D.el('span', 'acct-chip', 'no accounts'));
        }
        for (const ref of accounts) {
            const chip = D.el('span', 'acct-chip');
            const dot = D.el('span', StudioAccountOps.dotClass(ref.status));
            dot.title = ref.status || 'unknown status';
            D.add(chip, dot, D.el('span', '', '✉ ' + ref.email));
            chip.addEventListener('click', () => {
                const full = (this.state.accounts || []).find((a) => a.id === ref.id);
                StudioDrawer.showAccount(full || ref);
            });
            StudioDragDrop.makeChipDraggable(chip, 'account', { id: ref.id, name: ref.email });
            D.add(row, chip);
        }
        D.add(card, row);

        return card;
    },

    policySlot(ws) {
        const D = StudioDOM;
        if (!ws.policy) {
            return D.el('div', 'slot slot-empty', 'No policy attached');
        }
        if (ws.policy.missing) {
            const slot = D.el('div', 'slot slot-warn', '⚠ Policy ' + ws.policy.id + ' no longer exists');
            slot.addEventListener('click', () => StudioDrawer.showPolicy(ws.policy, false));
            return slot;
        }
        const locked = ws.policy.id === this.ceilingPolicyID(this.state);
        const slot = D.el('div', locked ? 'slot slot-policy locked' : 'slot slot-policy',
            (locked ? '🔒 ' : '🛡 ') + (ws.policy.name || ws.policy.id));
        slot.addEventListener('click', () => StudioDrawer.showPolicy(ws.policy, locked));
        return slot;
    },

    ruleSlot(rule) {
        const D = StudioDOM;
        if (rule.missing) {
            const slot = D.el('div', 'slot slot-warn', '⚠ Rule ' + rule.id + ' no longer exists');
            slot.addEventListener('click', () => StudioDrawer.showRule(rule));
            return slot;
        }
        const slot = D.el('div', rule.enabled ? 'slot slot-rule' : 'slot slot-rule disabled');
        D.add(slot, D.el('div', '', '⚡ ' + (rule.name || rule.id) + (rule.enabled ? '' : ' (disabled)')));
        for (const list of rule.lists || []) {
            D.add(slot, D.el('div', 'slot-sub', list.missing
                ? '↳ ⚠ list ' + list.id + ' no longer exists'
                : '↳ 📋 ' + (list.name || list.id) + ' · ' + (list.items_count || 0) + ' items'));
        }
        slot.addEventListener('click', () => StudioDrawer.showRule(rule));
        return slot;
    },

    renderStatus(board) {
        const D = StudioDOM;
        const bar = D.clear(document.getElementById('statusbar'));
        D.add(bar, D.el('span', 'status-dot'));
        D.add(bar, D.el('span', '', 'Synced just now'));

        const warnings = [];
        const orphanRules = (board.orphan_rules || []).length;
        const orphanPolicies = (board.orphan_policies || []).length;
        const unusedLists = (board.unused_lists || []).length;
        if (orphanRules) {
            warnings.push(orphanRules + ' unattached rule(s)');
        }
        if (orphanPolicies) {
            warnings.push(orphanPolicies + ' unattached polic(y/ies)');
        }
        if (unusedLists) {
            warnings.push(unusedLists + ' unused list(s)');
        }
        if (warnings.length) {
            D.add(bar, D.el('span', 'status-warn', '⚠ ' + warnings.join(' · ')));
        }
    }
};

document.addEventListener('DOMContentLoaded', () => {
    StudioViews.init();
    StudioBoard.load();
    const newBtn = document.getElementById('newBtn');
    if (newBtn) {
        newBtn.addEventListener('click', (event) => {
            event.stopPropagation();
            StudioBuilders.openNewMenu(newBtn);
        });
        document.addEventListener('click', () => {
            document.getElementById('newMenu').classList.remove('open');
        });
    }
});
