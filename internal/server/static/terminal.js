// Terminal helper functions for termblog
// This file is kept for potential future use

// Utility function to debounce resize events
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Export for potential module use
if (typeof module !== "undefined" && module.exports) {
    module.exports = { debounce };
}
