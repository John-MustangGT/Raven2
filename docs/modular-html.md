# Raven Modular Frontend Structure

## Overview

The monolithic `index.html` has been refactored into a modular Vue.js application with separate components and services. This improves maintainability, reusability, and code organization.

## File Structure

```
web/
├── index.html                    # Main HTML template (loads components)
├── styles.css                   # Global styles (unchanged)
├── favicon.ico                  # Favicon (unchanged)
├── favicon.svg                  # SVG favicon (unchanged)
└── js/
    ├── app.js                   # Main Vue application
    ├── utils.js                 # Utility functions
    ├── api.js                   # API service layer
    └── components/
        ├── sidebar.js           # Sidebar navigation
        ├── header.js            # Top header bar
        ├── dashboard-view.js    # Dashboard page
        ├── hosts-view.js        # Hosts management page
        ├── checks-view.js       # Checks management page
        ├── alerts-view.js       # Alerts page
        ├── about-view.js        # About/build info page
        ├── settings-view.js     # Settings page
        ├── host-modal.js        # Host add/edit modal
        ├── check-modal.js       # Check add/edit modal
        └── notification.js      # Toast notifications
```

## Component Architecture

### Core Components

1. **SidebarComponent** (`js/components/sidebar.js`)
   - Navigation menu
   - Collapsible/responsive behavior
   - Logo with configurable link

2. **HeaderComponent** (`js/components/header.js`)
   - Page title
   - Connection status
   - Theme toggle
   - Action buttons (context-dependent)

3. **View Components** (Various `*-view.js` files)
   - Each major page is a separate component
   - Self-contained logic and templates
   - Communicates with parent via events

4. **Modal Components** (`*-modal.js` files)
   - Reusable form dialogs
   - Host and Check creation/editing
   - Form validation and submission

### Services & Utilities

1. **RavenAPI** (`js/api.js`)
   - Centralized API calls
   - Error handling
   - Consistent data formatting

2. **RavenUtils** (`js/utils.js`)
   - Common utility functions
   - Date/time formatting
   - Status conversion helpers
   - Form templates

3. **Main App** (`js/app.js`)
   - Root Vue application
   - State management
   - Component coordination
   - WebSocket handling

## Key Benefits

### 1. **Maintainability**
- Each component has a single responsibility
- Easy to locate and modify specific functionality
- Clear separation of concerns

### 2. **Reusability**
- Components can be reused across different contexts
- Consistent UI patterns
- Shared utility functions

### 3. **Testability**
- Components can be tested in isolation
- Clear API boundaries
- Mocked dependencies

### 4. **Scalability**
- Easy to add new views or components
- Minimal impact when modifying existing features
- Clear dependency management

## Migration Guide

### From Monolithic to Modular

1. **HTML Template**: The main `index.html` now only contains:
   - Component placeholders
   - Script includes
   - Basic structure

2. **JavaScript**: The large inline script has been split into:
   - Component definitions
   - Service functions
   - Main application logic

3. **State Management**: All reactive state remains in the main app component, with child components receiving props and emitting events.

### Adding New Components

1. Create component file in `js/components/`
2. Define component with template and logic
3. Register in main app (`js/app.js`)
4. Add to HTML template if needed

Example:
```javascript
// js/components/my-component.js
window.MyComponent = {
    props: ['data'],
    emits: ['action'],
    template: `<div>{{ data }}</div>`
};

// js/app.js (components section)
components: {
    'my-component': window.MyComponent,
    // ... other components
}
```

## Development Workflow

### Local Development
1. Serve files from `web/` directory
2. All scripts load independently
3. Browser caching applies per file
4. Hot-reload friendly

### Production Deployment
- All files can be served as-is
- Consider bundling/minification for performance
- Maintain file structure for clarity

## Browser Compatibility

- Modern browsers supporting ES6+
- Vue 3 compatibility requirements
- No build step required
- Progressive enhancement friendly

## Performance Considerations

### Loading Strategy
- Components load after main libraries
- Lazy loading possible for large views
- Minimal initial payload

### Caching
- Individual files can be cached separately
- Component updates don't invalidate entire app
- CDN-friendly structure

### Memory Management
- Components properly clean up event listeners
- WebSocket connection management
- Reactive data cleanup

## Security Notes

- All API calls go through centralized service
- XSS protection through Vue's template system
- CSRF protection via existing patterns
- Input validation in components

## Future Enhancements

### Possible Additions
1. **State Management**: Vuex/Pinia for complex state
2. **Routing**: Vue Router for URL management
3. **Build System**: Webpack/Vite for optimization
4. **TypeScript**: Type safety for larger codebase
5. **Testing**: Jest/Vitest for component testing

### Module System
Current implementation uses global objects (`window.*`) for simplicity. Could be enhanced with:
- ES6 modules
- AMD/CommonJS
- Bundle splitting
- Tree shaking

This modular structure maintains the existing functionality while providing a solid foundation for future development and maintenance.
