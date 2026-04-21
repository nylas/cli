/**
 * Smart Compose Module
 * AI-powered email composition with autocomplete suggestions
 * Inspired by Gmail's Smart Compose and Superhuman
 */

const SmartCompose = {
    // Configuration
    config: {
        debounceMs: 300,
        minChars: 10,
        maxSuggestionLength: 100,
        enabled: true,
    },

    // State
    state: {
        isActive: false,
        currentSuggestion: '',
        lastText: '',
        debounceTimer: null,
        textarea: null,
        abortController: null,
        requestId: 0,
    },

    /**
     * Initialize smart compose on a textarea
     * @param {HTMLTextAreaElement} textarea - The compose textarea
     */
    init(textarea) {
        if (!textarea) return;

        this.state.textarea = textarea;
        this.setupListeners();
        this.createSuggestionOverlay();
        console.log('%c✨ Smart Compose initialized', 'color: #8b5cf6;');
    },

    /**
     * Setup event listeners
     */
    setupListeners() {
        const { textarea } = this.state;

        // Input handler with debounce
        textarea.addEventListener('input', () => {
            this.handleInput();
        });

        // Tab to accept suggestion
        textarea.addEventListener('keydown', (e) => {
            if (e.key === 'Tab' && this.state.currentSuggestion) {
                e.preventDefault();
                this.acceptSuggestion();
            } else if (e.key === 'Escape') {
                this.clearSuggestion();
            }
        });

        // Clear on blur
        textarea.addEventListener('blur', () => {
            this.clearSuggestion();
        });
    },

    /**
     * Handle input event
     */
    handleInput() {
        clearTimeout(this.state.debounceTimer);
        this.state.debounceTimer = null;
        this.cancelPendingRequest();
        this.hideSuggestion();

        const text = this.state.textarea.value;

        // Clear if too short
        if (text.length < this.config.minChars) {
            this.clearSuggestion();
            return;
        }

        // Debounce API call
        this.state.debounceTimer = setTimeout(() => {
            this.fetchSuggestion(text);
        }, this.config.debounceMs);
    },

    /**
     * Fetch AI suggestion from server
     * @param {string} text - Current text
     */
    async fetchSuggestion(text) {
        if (!this.config.enabled) return;

        // Don't fetch if text unchanged
        if (text === this.state.lastText) return;
        this.state.lastText = text;
        this.cancelPendingRequest();

        const controller = new AbortController();
        const requestId = ++this.state.requestId;
        this.state.abortController = controller;

        try {
            const response = await fetch('/api/ai/complete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    text: text,
                    maxLength: this.config.maxSuggestionLength,
                }),
                signal: controller.signal,
            });

            if (!response.ok) {
                this.hideSuggestion();
                return;
            }

            const data = await response.json();
            if (requestId !== this.state.requestId || text !== this.state.textarea.value) {
                return;
            }

            if (data.suggestion) {
                this.showSuggestion(data.suggestion);
            } else {
                this.hideSuggestion();
            }
        } catch (error) {
            if (error && error.name === 'AbortError') {
                return;
            }
            console.error('Smart compose error:', error);
        } finally {
            if (this.state.abortController === controller) {
                this.state.abortController = null;
            }
        }
    },

    /**
     * Create suggestion overlay element
     */
    createSuggestionOverlay() {
        const overlay = document.createElement('div');
        overlay.className = 'smart-compose-overlay';
        overlay.setAttribute('aria-hidden', 'true');

        const ghost = document.createElement('span');
        ghost.className = 'smart-compose-ghost';
        overlay.appendChild(ghost);

        const suggestion = document.createElement('span');
        suggestion.className = 'smart-compose-suggestion';
        overlay.appendChild(suggestion);

        // Position relative to textarea
        const wrapper = this.state.textarea.parentElement;
        wrapper.style.position = 'relative';
        wrapper.appendChild(overlay);

        this.overlay = overlay;
        this.ghostEl = ghost;
        this.suggestionEl = suggestion;
    },

    /**
     * Show suggestion in overlay
     * @param {string} suggestion - Suggested text
     */
    showSuggestion(suggestion) {
        if (!suggestion) return;

        this.state.currentSuggestion = suggestion;
        this.state.isActive = true;

        // Show current text as ghost
        this.ghostEl.textContent = this.state.textarea.value;
        this.suggestionEl.textContent = suggestion;

        this.overlay.classList.add('visible');
    },

    /**
     * Accept current suggestion
     */
    acceptSuggestion() {
        if (!this.state.currentSuggestion) return;

        const { textarea } = this.state;
        textarea.value += this.state.currentSuggestion;

        this.clearSuggestion();

        // Trigger input event for other listeners
        textarea.dispatchEvent(new Event('input', { bubbles: true }));

        // Track acceptance for learning
        this.trackAcceptance();
    },

    /**
     * Clear suggestion
     */
    clearSuggestion() {
        this.cancelPendingRequest();
        this.state.lastText = '';
        this.hideSuggestion();
    },

    hideSuggestion() {
        this.state.currentSuggestion = '';
        this.state.isActive = false;

        if (this.overlay) {
            this.overlay.classList.remove('visible');
        }
    },

    cancelPendingRequest() {
        if (this.state.debounceTimer) {
            clearTimeout(this.state.debounceTimer);
            this.state.debounceTimer = null;
        }
        if (this.state.abortController) {
            this.state.abortController.abort();
            this.state.abortController = null;
        }
        this.state.requestId += 1;
    },

    /**
     * Track suggestion acceptance for ML
     */
    trackAcceptance() {
        // Could send to analytics/ML pipeline
        console.log('Smart compose suggestion accepted');
    },

    /**
     * Enable/disable smart compose
     * @param {boolean} enabled
     */
    setEnabled(enabled) {
        this.config.enabled = enabled;
        if (!enabled) {
            this.clearSuggestion();
        }
    },

    /**
     * Get keyboard shortcut hint
     * @returns {string}
     */
    getHint() {
        return this.state.isActive
            ? 'Press Tab to accept suggestion'
            : '';
    },
};

// Export for use
if (typeof window !== 'undefined') {
    window.SmartCompose = SmartCompose;
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = SmartCompose;
}
