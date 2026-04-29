/**
 * App Search - Recent searches manager and advanced search
 */
        // ====================================
        // RECENT SEARCHES MANAGER
        // ====================================

        const RecentSearches = {
            STORAGE_KEY: 'nylas_air_recent_searches',
            MAX_ITEMS: 10,

            // Get all recent searches from localStorage
            getAll() {
                return storage.get(this.STORAGE_KEY, []);
            },

            // Add a search to recent searches
            add(query) {
                if (!query || query.trim().length === 0) return;

                const trimmedQuery = query.trim();
                let searches = this.getAll();

                // Remove if already exists (will re-add at top)
                searches = searches.filter(s => s.query.toLowerCase() !== trimmedQuery.toLowerCase());

                // Add to beginning
                searches.unshift({
                    query: trimmedQuery,
                    timestamp: Date.now()
                });

                // Keep only MAX_ITEMS
                searches = searches.slice(0, this.MAX_ITEMS);

                storage.set(this.STORAGE_KEY, searches);
                this.render();
            },

            // Remove a specific search
            remove(query) {
                let searches = this.getAll();
                searches = searches.filter(s => s.query !== query);
                storage.set(this.STORAGE_KEY, searches);
                this.render();
            },

            // Clear all recent searches
            clear() {
                storage.remove(this.STORAGE_KEY);
                this.render();
                showToast('info', 'Cleared', 'Recent searches cleared');
            },

            // Render recent searches in the UI
            render() {
                const container = document.getElementById('recentSearchesList');
                const group = document.getElementById('recentSearchesGroup');
                if (!container || !group) return;

                const searches = this.getAll();

                if (searches.length === 0) {
                    group.style.display = 'none';
                    return;
                }

                group.style.display = 'block';
                container.innerHTML = searches.map(search => `
                    <div class="search-suggestion-item recent-search-item" data-action="execute-search" data-search-query="${escapeHtml(search.query)}">
                        <div class="search-suggestion-icon">🕐</div>
                        <div class="search-suggestion-content">
                            <div class="search-suggestion-text">${escapeHtml(search.query)}</div>
                            <div class="search-suggestion-meta">${this.formatTime(search.timestamp)}</div>
                        </div>
                        <button class="remove-recent-btn" data-action="remove-recent-search" data-search-query="${escapeHtml(search.query)}" title="Remove">
                            <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                <path d="M18 6L6 18M6 6l12 12"/>
                            </svg>
                        </button>
                    </div>
                `).join('');
            },

            // Format timestamp to relative time
            formatTime(timestamp) {
                const now = Date.now();
                const diff = now - timestamp;
                const minutes = Math.floor(diff / 60000);
                const hours = Math.floor(diff / 3600000);
                const days = Math.floor(diff / 86400000);

                if (minutes < 1) return 'Just now';
                if (minutes < 60) return `${minutes}m ago`;
                if (hours < 24) return `${hours}h ago`;
                if (days < 7) return `${days}d ago`;
                return new Date(timestamp).toLocaleDateString();
            }
        };

        // ====================================
        // ADVANCED SEARCH
        // ====================================

        let searchSelectedIndex = -1;

        function openSearch() {
            const overlay = document.getElementById('searchOverlay');
            overlay.classList.add('active');
            searchSelectedIndex = -1;

            // Render recent searches when opening
            RecentSearches.render();

            // Show recent searches, hide results section
            const recentGroup = document.getElementById('recentSearchesGroup');
            const resultsSection = document.getElementById('searchResultsSection');
            if (recentGroup) recentGroup.style.display = 'block';
            if (resultsSection) resultsSection.innerHTML = '';

            setTimeout(() => {
                const input = document.getElementById('searchInput');
                if (input) {
                    input.focus();
                    input.select();
                }
            }, 100);
        }

        function closeSearch() {
            const overlay = document.getElementById('searchOverlay');
            if (overlay) overlay.classList.remove('active');

            const input = document.getElementById('searchInput');
            if (input) input.value = '';

            searchSelectedIndex = -1;
        }

        function handleSearchInput(query) {
            const recentGroup = document.getElementById('recentSearchesGroup');
            const resultsSection = document.getElementById('searchResultsSection');

            if (!query || query.length === 0) {
                // Show recent searches when input is empty
                if (recentGroup) recentGroup.style.display = 'block';
                if (resultsSection) resultsSection.innerHTML = '';
                RecentSearches.render();
                return;
            }

            // Hide recent searches when typing
            if (recentGroup) recentGroup.style.display = 'none';

            // Show search results (demo mode for now)
            const escapedQuery = escapeHtml(query);
            if (resultsSection) {
                resultsSection.innerHTML = `
                    <div class="search-suggestion-group">
                        <div class="search-suggestion-title">Results for "${escapedQuery}"</div>
                        <div class="search-suggestion-item"  data-action="execute-search" data-search-query="${escapedQuery}">
                            <div class="search-suggestion-icon">📧</div>
                            <div class="search-suggestion-content">
                                <div class="search-suggestion-text">Email containing <mark>${escapedQuery}</mark></div>
                                <div class="search-suggestion-meta">From Sarah Chen - 2 hours ago</div>
                            </div>
                        </div>
                        <div class="search-suggestion-item"  data-action="execute-search" data-search-query="${escapedQuery}">
                            <div class="search-suggestion-icon">📧</div>
                            <div class="search-suggestion-content">
                                <div class="search-suggestion-text">Re: <mark>${escapedQuery}</mark> discussion</div>
                                <div class="search-suggestion-meta">From Alex Johnson - Yesterday</div>
                            </div>
                        </div>
                        <div class="search-suggestion-item"  data-action="execute-search" data-search-query="${escapedQuery}">
                            <div class="search-suggestion-icon">📅</div>
                            <div class="search-suggestion-content">
                                <div class="search-suggestion-text">Meeting: <mark>${escapedQuery}</mark> review</div>
                                <div class="search-suggestion-meta">Tomorrow at 2:00 PM</div>
                            </div>
                        </div>
                    </div>
                `;
            }
        }

        function handleSearchKeydown(event) {
            const items = document.querySelectorAll('#searchSuggestions .search-suggestion-item');

            if (event.key === 'ArrowDown') {
                event.preventDefault();
                searchSelectedIndex = Math.min(searchSelectedIndex + 1, items.length - 1);
                updateSearchSelection(items);
            } else if (event.key === 'ArrowUp') {
                event.preventDefault();
                searchSelectedIndex = Math.max(searchSelectedIndex - 1, -1);
                updateSearchSelection(items);
            } else if (event.key === 'Enter') {
                event.preventDefault();
                if (searchSelectedIndex >= 0 && items[searchSelectedIndex]) {
                    items[searchSelectedIndex].click();
                } else {
                    // Execute search with current input value
                    const query = document.getElementById('searchInput').value;
                    if (query.trim()) {
                        executeSearch(query.trim());
                    }
                }
            } else if (event.key === 'Escape') {
                closeSearch();
            }
        }

        function updateSearchSelection(items) {
            items.forEach((item, index) => {
                if (index === searchSelectedIndex) {
                    item.classList.add('selected');
                    item.scrollIntoView({ block: 'nearest' });
                } else {
                    item.classList.remove('selected');
                }
            });
        }

        function executeSearch(query) {
            if (!query || query.trim().length === 0) return;

            // Save to recent searches
            RecentSearches.add(query);

            // Close search overlay
            closeSearch();

            // Show searching toast
            showToast('info', 'Searching', `Finding items matching "${query}"...`);

            // TODO: Integrate with actual search API
            // For now, filter emails if EmailListManager is available
            if (typeof EmailListManager !== 'undefined') {
                EmailListManager.loadEmails({ search: query });
            }
        }

        function toggleSearchFilter(btn) {
            // Toggle single selection for filter chips
            const chips = btn.parentElement.querySelectorAll('.search-filter-chip');
            chips.forEach(chip => chip.classList.remove('active'));
            btn.classList.add('active');

            // Re-run search with new filter if there's a query
            const query = document.getElementById('searchInput').value;
            if (query.trim()) {
                handleSearchInput(query);
            }
        }

        // Wire up search input event listeners (replaces inline oninput/onkeydown)
        document.addEventListener('DOMContentLoaded', function() {
            const searchInput = document.getElementById('searchInput');
            if (searchInput) {
                searchInput.addEventListener('input', function() {
                    handleSearchInput(this.value);
                });
                searchInput.addEventListener('keydown', function(event) {
                    handleSearchKeydown(event);
                });
            }
        });
