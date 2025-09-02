// js/keyboard-navigation.js - Add keyboard navigation support for detail views
window.KeyboardNavigation = {
    init(app) {
        document.addEventListener('keydown', (event) => {
            // Only handle keyboard shortcuts when not in form inputs
            if (event.target.tagName === 'INPUT' || 
                event.target.tagName === 'TEXTAREA' || 
                event.target.tagName === 'SELECT' ||
                event.target.contentEditable === 'true') {
                return;
            }

            // Handle different key combinations
            switch (event.key) {
                case 'Escape':
                    this.handleEscape(app, event);
                    break;
                case 'Enter':
                    this.handleEnter(app, event);
                    break;
                case 'Backspace':
                    if (event.ctrlKey || event.metaKey) {
                        this.handleBack(app, event);
                    }
                    break;
                case 'r':
                    if (event.ctrlKey || event.metaKey) {
                        event.preventDefault();
                        this.handleRefresh(app, event);
                    }
                    break;
                case 'h':
                    if (event.ctrlKey || event.metaKey) {
                        event.preventDefault();
                        this.handleHome(app, event);
                    }
                    break;
                case '/':
                    if (!event.ctrlKey && !event.metaKey) {
                        this.handleSearch(app, event);
                    }
                    break;
                case '?':
                    if (!event.ctrlKey && !event.metaKey) {
                        this.showKeyboardHelp();
                    }
                    break;
            }

            // Number keys for quick navigation
            if (event.key >= '1' && event.key <= '6' && !event.ctrlKey && !event.metaKey) {
                this.handleQuickNavigation(app, parseInt(event.key), event);
            }

            // Arrow keys for table navigation
            if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(event.key)) {
                this.handleArrowNavigation(app, event);
            }
        });

        // Add keyboard navigation hints to the UI
        this.addKeyboardHints();
    },

    handleEscape(app, event) {
        event.preventDefault();
        
        // Close modals first
        if (app.showHostModal || app.showCheckModal) {
            app.closeHostModal();
            app.closeCheckModal();
            return;
        }

        // Navigate back from detail views
        if (app.currentView === 'host-detail') {
            app.backToHosts();
        } else if (app.currentView === 'alert-detail') {
            app.backToAlerts();
        }
    },

    handleEnter(app, event) {
        // If a row is focused, navigate to its detail view
        const focusedRow = document.activeElement;
        if (focusedRow && focusedRow.classList.contains('clickable-row')) {
            event.preventDefault();
            focusedRow.click();
        }
    },

    handleBack(app, event) {
        event.preventDefault();
        app.goBack();
    },

    handleRefresh(app, event) {
        event.preventDefault();
        app.refreshData();
    },

    handleHome(app, event) {
        event.preventDefault();
        app.setView('dashboard');
    },

    handleSearch(app, event) {
        event.preventDefault();
        
        // Focus on search input if available
        const searchInput = document.querySelector('.search-input');
        if (searchInput) {
            searchInput.focus();
            searchInput.select();
        }
    },

    handleQuickNavigation(app, number, event) {
        event.preventDefault();
        
        const viewMap = {
            1: 'dashboard',
            2: 'hosts', 
            3: 'checks',
            4: 'alerts',
            5: 'about',
            6: 'settings'
        };

        const targetView = viewMap[number];
        if (targetView) {
            app.setView(targetView);
        }
    },

    handleArrowNavigation(app, event) {
        const rows = Array.from(document.querySelectorAll('.clickable-row'));
        if (rows.length === 0) return;

        const currentRow = document.activeElement;
        const currentIndex = rows.indexOf(currentRow);

        let newIndex = -1;

        switch (event.key) {
            case 'ArrowUp':
                event.preventDefault();
                newIndex = currentIndex <= 0 ? rows.length - 1 : currentIndex - 1;
                break;
            case 'ArrowDown':
                event.preventDefault();
                newIndex = currentIndex >= rows.length - 1 ? 0 : currentIndex + 1;
                break;
        }

        if (newIndex >= 0 && newIndex < rows.length) {
            rows[newIndex].focus();
        } else if (currentIndex === -1 && rows.length > 0) {
            // If no row is focused, focus the first one
            rows[0].focus();
        }
    },

    showKeyboardHelp() {
        const helpModal = document.createElement('div');
        helpModal.className = 'modal-overlay active';
        helpModal.innerHTML = `
            <div class="modal" style="max-width: 600px;">
                <div class="modal-header">
                    <h3 class="modal-title">
                        <i class="fas fa-keyboard"></i>
                        Keyboard Shortcuts
                    </h3>
                    <button class="close-btn" onclick="this.closest('.modal-overlay').remove()">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div style="padding: 1rem;">
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 2rem;">
                        <div>
                            <h4 style="margin-bottom: 1rem; color: var(--primary-color);">Navigation</h4>
                            <div class="keyboard-shortcuts">
                                <div><kbd>1</kbd> Dashboard</div>
                                <div><kbd>2</kbd> Hosts</div>
                                <div><kbd>3</kbd> Checks</div>
                                <div><kbd>4</kbd> Alerts</div>
                                <div><kbd>5</kbd> About</div>
                                <div><kbd>6</kbd> Settings</div>
                                <div><kbd>Ctrl</kbd> + <kbd>H</kbd> Home (Dashboard)</div>
                                <div><kbd>Ctrl</kbd> + <kbd>←</kbd> Back</div>
                            </div>
                        </div>
                        <div>
                            <h4 style="margin-bottom: 1rem; color: var(--primary-color);">Actions</h4>
                            <div class="keyboard-shortcuts">
                                <div><kbd>Ctrl</kbd> + <kbd>R</kbd> Refresh</div>
                                <div><kbd>/</kbd> Focus Search</div>
                                <div><kbd>Esc</kbd> Close/Back</div>
                                <div><kbd>Enter</kbd> Open Selected</div>
                                <div><kbd>↑</kbd> <kbd>↓</kbd> Navigate Rows</div>
                                <div><kbd>?</kbd> Show This Help</div>
                            </div>
                        </div>
                    </div>
                    
                    <div style="margin-top: 2rem; padding-top: 1rem; border-top: 1px solid var(--border-color);">
                        <p style="color: var(--text-muted); font-size: 0.875rem;">
                            <i class="fas fa-info-circle"></i>
                            Keyboard shortcuts are available throughout the application. 
                            Press <kbd>Tab</kbd> to navigate between interactive elements.
                        </p>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(helpModal);

        // Auto-close after 10 seconds or on click outside
        const autoClose = setTimeout(() => {
            if (helpModal.parentNode) {
                helpModal.remove();
            }
        }, 10000);

        helpModal.addEventListener('click', (event) => {
            if (event.target === helpModal) {
                clearTimeout(autoClose);
                helpModal.remove();
            }
        });
    },

    addKeyboardHints() {
        // Add keyboard hints to clickable rows
        const style = document.createElement('style');
        style.textContent = `
            .keyboard-shortcuts {
                display: flex;
                flex-direction: column;
                gap: 0.5rem;
            }
            
            .keyboard-shortcuts div {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 0.25rem 0;
            }
            
            kbd {
                background: var(--light-bg);
                border: 1px solid var(--border-color);
                border-radius: 0.25rem;
                padding: 0.125rem 0.375rem;
                font-family: monospace;
                font-size: 0.75rem;
                color: var(--text-primary);
                box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
                margin: 0 0.125rem;
            }
            
            [data-theme="dark"] kbd {
                background: var(--darker-bg);
                border-color: var(--border-color);
                color: var(--text-primary);
            }
            
            .clickable-row {
                position: relative;
            }
            
            .clickable-row:focus {
                outline: 2px solid var(--primary-color);
                outline-offset: -2px;
                background: rgba(59, 130, 246, 0.1) !important;
            }
            
            .clickable-row::after {
                content: "Press Enter to open";
                position: absolute;
                right: 1rem;
                top: 50%;
                transform: translateY(-50%);
                background: var(--primary-color);
                color: white;
                padding: 0.25rem 0.5rem;
                border-radius: 0.25rem;
                font-size: 0.75rem;
                opacity: 0;
                transition: opacity 0.2s ease;
                pointer-events: none;
                z-index: 10;
            }
            
            .clickable-row:focus::after {
                opacity: 1;
            }
            
            /* Hide keyboard hints on mobile */
            @media (max-width: 768px) {
                .clickable-row::after {
                    display: none;
                }
            }
            
            /* Keyboard navigation indicator */
            .keyboard-nav-active .clickable-row {
                outline: 1px solid var(--border-color);
                outline-offset: -1px;
            }
        `;
        document.head.appendChild(style);

        // Add tabindex to make rows keyboard navigable
        this.makeRowsKeyboardNavigable();
    },

    makeRowsKeyboardNavigable() {
        // Use a MutationObserver to watch for new rows being added
        const observer = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                mutation.addedNodes.forEach((node) => {
                    if (node.nodeType === Node.ELEMENT_NODE) {
                        const rows = node.querySelectorAll ? 
                            node.querySelectorAll('.clickable-row') : 
                            [];
                        rows.forEach((row, index) => {
                            if (!row.hasAttribute('tabindex')) {
                                row.setAttribute('tabindex', '0');
                                row.setAttribute('role', 'button');
                                row.setAttribute('aria-label', `Row ${index + 1}, click to view details`);
                            }
                        });
                    }
                });
            });
        });

        observer.observe(document.body, {
            childList: true,
            subtree: true
        });

        // Initial setup for existing rows
        document.querySelectorAll('.clickable-row').forEach((row, index) => {
            row.setAttribute('tabindex', '0');
            row.setAttribute('role', 'button');
            row.setAttribute('aria-label', `Row ${index + 1}, click to view details`);
        });
    },

    // Utility to check if keyboard navigation should be active
    isKeyboardNavigationActive() {
        return document.body.classList.contains('keyboard-nav-active');
    },

    // Enable keyboard navigation mode
    enableKeyboardNavigation() {
        document.body.classList.add('keyboard-nav-active');
    },

    // Disable keyboard navigation mode
    disableKeyboardNavigation() {
        document.body.classList.remove('keyboard-nav-active');
    }
};

// Initialize keyboard navigation when the DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Wait for Vue app to be ready
    setTimeout(() => {
        const app = document.querySelector('#app').__vue_app__;
        if (app) {
            window.KeyboardNavigation.init(app.config.globalProperties);
        }
    }, 1000);
});

// Detect keyboard usage and enable navigation hints
document.addEventListener('keydown', (event) => {
    if (event.key === 'Tab') {
        window.KeyboardNavigation.enableKeyboardNavigation();
    }
});

document.addEventListener('mousedown', () => {
    window.KeyboardNavigation.disableKeyboardNavigation();
});`
