/**
 * Modal framework — confirms with consequence text and create/edit forms.
 */
window.StudioModal = {
    open(title, subtitle, build) {
        const D = StudioDOM;
        const backdrop = document.getElementById('modalBackdrop');
        const modal = D.clear(document.getElementById('modal'));

        D.add(modal, D.el('div', 'modal-title', title));
        if (subtitle) {
            D.add(modal, D.el('div', 'modal-sub', subtitle));
        }
        build(modal);

        backdrop.classList.add('open');
        backdrop.onclick = (event) => {
            if (event.target === backdrop) {
                this.close();
            }
        };
    },

    close() {
        document.getElementById('modalBackdrop').classList.remove('open');
    },

    confirm(title, message, onConfirm) {
        const D = StudioDOM;
        this.open(title, '', (modal) => {
            D.add(modal, D.el('p', 'modal-text', message));
            const actions = D.el('div', 'modal-actions');
            const yes = D.el('button', 'btn', 'Continue');
            yes.addEventListener('click', async () => {
                this.close();
                try {
                    await Promise.resolve(onConfirm());
                } catch (error) {
                    StudioDragDrop.toast((error && error.message) || 'The change failed');
                }
            });
            const no = D.el('button', 'btn btn-ghost', 'Cancel');
            no.addEventListener('click', () => this.close());
            D.add(actions, yes, no);
            D.add(modal, actions);
        });
    },

    field(label, input) {
        const D = StudioDOM;
        const wrap = D.el('label', 'field');
        D.add(wrap, D.el('span', 'field-label', label), input);
        return wrap;
    },

    input(placeholder, value) {
        const node = StudioDOM.el('input', 'input');
        node.placeholder = placeholder || '';
        if (value !== undefined) {
            node.value = value;
        }
        return node;
    },

    select(options, selected) {
        const node = StudioDOM.el('select', 'input');
        for (const option of options) {
            const opt = StudioDOM.el('option', '', option.label);
            opt.value = option.value;
            if (option.value === selected) {
                opt.selected = true;
            }
            node.appendChild(opt);
        }
        return node;
    },

    error(modal, message) {
        const existing = modal.querySelector('.modal-error');
        if (existing) {
            existing.textContent = message;
            return;
        }
        const node = StudioDOM.el('div', 'modal-error', message);
        modal.insertBefore(node, modal.querySelector('.modal-actions'));
    },

    actions(modal, submitLabel, onSubmit) {
        const D = StudioDOM;
        const actions = D.el('div', 'modal-actions');
        const submit = D.el('button', 'btn', submitLabel);
        submit.addEventListener('click', async () => {
            submit.disabled = true;
            try {
                const result = await onSubmit();
                this.close();
                if (result && result.board) {
                    StudioBoard.render(result.board);
                }
            } catch (error) {
                this.error(modal, error.message || 'The request failed');
            } finally {
                submit.disabled = false;
            }
        });
        const cancel = D.el('button', 'btn btn-ghost', 'Cancel');
        cancel.addEventListener('click', () => this.close());
        D.add(actions, submit, cancel);
        D.add(modal, actions);
        return submit;
    }
};
