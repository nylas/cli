/**
 * App Core - Toast system, animations, view switching, and modal toggles
 */
        // Toast System with action button support
        function showToast(type, title, message, options = null) {
            const container = document.getElementById('toastContainer');
            if (!container) return;

            const toast = document.createElement('div');
            toast.className = `toast ${type}`;

            const icons = { success: '✅', info: '💡', warning: '⏰', error: '❌' };

            // Build toast content
            const iconSpan = document.createElement('span');
            iconSpan.className = 'toast-icon';
            iconSpan.textContent = icons[type] || '💬';

            const messageDiv = document.createElement('div');
            messageDiv.className = 'toast-message';
            // Use textContent for XSS prevention - user data may be in title/message
            const strong = document.createElement('strong');
            strong.textContent = title;
            messageDiv.appendChild(strong);
            messageDiv.appendChild(document.createTextNode(' — ' + message));

            toast.appendChild(iconSpan);
            toast.appendChild(messageDiv);

            // Add action button if provided
            if (options && options.action && options.onAction) {
                const actionBtn = document.createElement('button');
                actionBtn.className = 'toast-action';
                actionBtn.textContent = options.action;
                actionBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    options.onAction();
                    toast.remove();
                });
                toast.appendChild(actionBtn);
            }

            container.appendChild(toast);

            // Auto-dismiss after duration (longer if has action)
            const duration = options && options.action ? 6000 : 4000;
            const dismissTimeout = setTimeout(() => {
                toast.style.opacity = '0';
                toast.style.transform = 'translateY(20px)';
                setTimeout(() => toast.remove(), 300);
            }, duration);

            // Allow manual dismiss by clicking
            toast.addEventListener('click', () => {
                clearTimeout(dismissTimeout);
                toast.style.opacity = '0';
                toast.style.transform = 'translateY(20px)';
                setTimeout(() => toast.remove(), 300);
            });
        }

        // Send Animation
        function showSendAnimation() {
            const anim = document.getElementById('sendAnimation');
            anim.classList.add('active');
            setTimeout(() => anim.classList.remove('active'), 1000);
        }

        // View Switching
        function showView(view, event) {
            const navView = view === 'rulesPolicy' ? 'email' : view;

            // Update nav tabs
            document.querySelectorAll('.nav-tab').forEach(tab => {
                tab.classList.remove('active');
                tab.setAttribute('aria-selected', 'false');
            });

            // Find and activate the clicked tab
            if (event && event.target) {
                const clickedTab = event.target.closest('.nav-tab');
                if (clickedTab) {
                    clickedTab.classList.add('active');
                    clickedTab.setAttribute('aria-selected', 'true');
                }
            } else {
                const fallbackTab = document.querySelector(`.nav-tab[data-view="${navView}"]`);
                if (fallbackTab) {
                    fallbackTab.classList.add('active');
                    fallbackTab.setAttribute('aria-selected', 'true');
                }
            }

            // Hide all views
            document.querySelectorAll('[data-air-view]').forEach((viewEl) => {
                viewEl.classList.remove('active');
            });

            // Show selected view
            const targetView = document.getElementById(view + 'View');
            if (targetView) {
                targetView.classList.add('active');

                // Load notetakers when view is shown
                if (view === 'notetaker' && typeof NotetakerModule !== 'undefined') {
                    NotetakerModule.loadNotetakers();
                }
                if (view === 'rulesPolicy' && typeof RulesPolicyManager !== 'undefined') {
                    RulesPolicyManager.loadAll();
                }
            }

            // Update mobile nav if present
            document.querySelectorAll('.mobile-nav-item').forEach(item => {
                item.classList.remove('active');
            });
            const mobileNavItems = document.querySelectorAll('.mobile-nav-item');
            const mobileIndex = navView === 'email' ? 1 : navView === 'calendar' ? 2 : 3;
            if (mobileNavItems[mobileIndex]) {
                mobileNavItems[mobileIndex].classList.add('active');
            }

            // Announce view change for screen readers
            if (typeof announce === 'function') {
                const labels = {
                    email: 'Email',
                    calendar: 'Calendar',
                    contacts: 'Contacts',
                    notetaker: 'Notetaker',
                    rulesPolicy: 'Policy and Rules'
                };
                announce(`Switched to ${labels[view] || view} view`);
            }

            // Lazy load data for the view (only loads once)
            if (view === 'calendar' && typeof CalendarManager !== 'undefined') {
                CalendarManager.init(); // Will only load data once due to isInitialized flag
            }
            if (view === 'contacts' && typeof ContactsManager !== 'undefined') {
                ContactsManager.loadContacts();
            }
        }

        function toggleCommandPalette() {
            const palette = document.getElementById('commandPalette');
            palette.classList.toggle('hidden');
            if (!palette.classList.contains('hidden')) {
                palette.querySelector('input').focus();
            }
        }

        function toggleCompose() {
            // Use ComposeManager if available
            if (typeof ComposeManager !== 'undefined') {
                if (ComposeManager.isOpen) {
                    ComposeManager.close();
                } else {
                    ComposeManager.open();
                }
            } else {
                // Fallback to simple toggle
                document.getElementById('composeModal').classList.toggle('hidden');
            }
        }
