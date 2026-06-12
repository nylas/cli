/**
 * Accounts view — a first-class, searchable list of every agent account —
 * plus the shared account operations (test email, rotate, move, delete)
 * used by both this view and the account drawer.
 */
window.StudioAccountOps = {
    // Grants report status "valid" when healthy (not "active").
    dotClass(status) {
        return status === 'valid' || status === 'active' ? 'acct-dot ok' : 'acct-dot warn';
    },

    async sendTest(acct, btn) {
        if (btn) {
            btn.disabled = true;
        }
        try {
            const result = await StudioAPI.sendTestEmail(acct.id);
            StudioDragDrop.toast('Test email sent to ' + (result.to || acct.email));
        } catch (error) {
            StudioDragDrop.toast(error.message || 'Test email failed');
        } finally {
            if (btn) {
                btn.disabled = false;
            }
        }
    },

    rotate(acct) {
        const D = StudioDOM;
        const password = StudioBuilders.generatePassword();
        StudioModal.confirm('Rotate app password for ' + acct.email + '?',
            'Mail clients using the old password stop authenticating immediately.',
            async () => {
                try {
                    const result = await StudioAPI.rotatePassword(acct.id, password);
                    if (result && result.board) {
                        StudioBoard.render(result.board);
                    }
                    StudioModal.open('New app password', 'Copy it now — it is not shown again.', (modal) => {
                        D.add(modal, D.kv('Password', password, true));
                        const actions = D.el('div', 'modal-actions');
                        const done = D.el('button', 'btn', 'Done');
                        done.addEventListener('click', () => StudioModal.close());
                        D.add(actions, done);
                        D.add(modal, actions);
                    });
                } catch (error) {
                    StudioDragDrop.toast(error.message || 'Rotation failed');
                }
            });
    },

    move(acct) {
        const full = ((StudioBoard.state || {}).accounts || []).find((a) => a.id === acct.id) || acct;
        const workspaces = ((StudioBoard.state || {}).workspaces || []).filter((ws) => ws.id !== full.workspace_id);
        if (!workspaces.length) {
            StudioDragDrop.toast('No other workspace to move to — create one first.');
            return;
        }
        const options = workspaces.map((ws) => ({ value: ws.id, label: StudioBoard.displayName(ws.name) || ws.id }));
        StudioModal.open('Move ' + full.email, 'The target workspace’s policy and rules govern the account immediately.', (modal) => {
            const select = StudioModal.select(options, options[0].value);
            StudioDOM.add(modal, StudioModal.field('Target workspace', select));
            StudioModal.actions(modal, 'Move account', async () => {
                const previous = full.workspace_id || '';
                const result = await StudioAPI.moveAccount(full.id, select.value);
                StudioDragDrop.toast('✉ ' + full.email + ' moved',
                    previous ? () => StudioAPI.moveAccount(full.id, previous) : null);
                return result;
            });
        });
    },

    remove(acct) {
        return StudioDrawer.confirmAndRun(
            'Delete ' + acct.email + '? This revokes the grant and its mailbox becomes unreachable.',
            () => StudioAPI.deleteAccount(acct.id));
    }
};

window.StudioAccounts = {
    query: '',

    render(board) {
        const D = StudioDOM;
        const host = D.clear(document.getElementById('accountsView'));
        const accounts = board.accounts || [];

        const head = D.el('div', 'accounts-head');
        D.add(head, D.el('span', 'accounts-title', 'Accounts · ' + accounts.length));
        const search = StudioModal.input('Search by email, workspace, or policy…', this.query);
        search.classList.add('accounts-search');
        search.addEventListener('input', () => {
            this.query = search.value;
            this.renderRows(board);
        });
        D.add(head, search);
        D.add(host, head);

        const rows = D.el('div', 'accounts-rows');
        rows.id = 'accountRows';
        D.add(host, rows);
        this.renderRows(board);
    },

    matches(acct) {
        const q = this.query.trim().toLowerCase();
        if (!q) {
            return true;
        }
        const policy = acct.policy ? (acct.policy.name || acct.policy.id) : '';
        return [acct.email, acct.workspace_name, acct.workspace_id, policy]
            .some((field) => (field || '').toLowerCase().includes(q));
    },

    renderRows(board) {
        const D = StudioDOM;
        const rows = D.clear(document.getElementById('accountRows'));
        const accounts = (board.accounts || []).filter((acct) => this.matches(acct));

        if (!accounts.length) {
            const empty = D.el('div', 'empty');
            D.add(empty, D.el('div', '', this.query ? 'No accounts match your search.' : 'No agent accounts yet.'));
            if (!this.query) {
                const cta = D.el('button', 'btn', '＋ New account');
                cta.addEventListener('click', () => StudioBuilders.accountForm());
                D.add(empty, cta);
            }
            D.add(rows, empty);
            return;
        }

        for (const acct of accounts) {
            rows.appendChild(this.row(acct));
        }
    },

    row(acct) {
        const D = StudioDOM;
        const row = D.el('div', 'acct-card');

        const main = D.el('div', 'acct-main');
        const dot = D.el('span', StudioAccountOps.dotClass(acct.status));
        dot.title = acct.status || 'unknown status';
        const email = D.el('span', 'acct-email', acct.email);
        D.add(main, dot, email);
        if (acct.shared_with > 1) {
            D.add(main, D.el('span', 'badge badge-shared', '⚠ shared · ' + acct.shared_with));
        }
        D.add(row, main);

        const meta = D.el('div', 'acct-meta');
        const wsName = StudioBoard.displayName(acct.workspace_name) || acct.workspace_id || 'no workspace';
        D.add(meta, D.el('span', '', (acct.status || 'unknown') + ' · ' + wsName));
        const ceiling = StudioBoard.ceilingPolicyID(StudioBoard.state || {});
        const policy = acct.policy
            ? ((acct.policy.id === ceiling ? '🔒 ' : '🛡 ') + (acct.policy.name || acct.policy.id))
            : 'no policy';
        D.add(meta, D.el('span', '', policy + ' · ' + (acct.rules || []).length + ' ⚡'));
        D.add(row, meta);

        const actions = D.el('div', 'acct-actions');
        const test = D.el('button', 'btn btn-ghost btn-inline', '✈ Test');
        test.addEventListener('click', () => StudioAccountOps.sendTest(acct, test));
        const rotate = D.el('button', 'btn btn-ghost btn-inline', '⟳ Rotate');
        rotate.addEventListener('click', () => StudioAccountOps.rotate(acct));
        const move = D.el('button', 'btn btn-ghost btn-inline', '⇄ Move');
        move.addEventListener('click', () => StudioAccountOps.move(acct));
        const del = D.el('button', 'btn btn-ghost btn-inline btn-danger-ghost', '🗑 Delete');
        del.addEventListener('click', () => StudioAccountOps.remove(acct));
        D.add(actions, test, rotate, move, del);
        D.add(row, actions);

        for (const target of [email, meta]) {
            target.style.cursor = 'pointer';
            target.addEventListener('click', () => StudioDrawer.showAccount(acct));
        }
        return row;
    }
};

window.StudioViews = {
    show(view) {
        document.getElementById('palette').hidden = view !== 'board';
        document.getElementById('board').hidden = view !== 'board';
        document.getElementById('accountsView').hidden = view !== 'accounts';
        for (const tab of document.querySelectorAll('.tab')) {
            tab.classList.toggle('active', tab.dataset.view === view);
        }
        const hash = view === 'accounts' ? '#accounts' : '#board';
        if (location.hash !== hash) {
            history.replaceState(null, '', hash);
        }
    },

    init() {
        for (const tab of document.querySelectorAll('.tab')) {
            tab.addEventListener('click', () => this.show(tab.dataset.view));
        }
        window.addEventListener('hashchange', () =>
            this.show(location.hash === '#accounts' ? 'accounts' : 'board'));
        if (location.hash === '#accounts') {
            this.show('accounts');
        }
    }
};
