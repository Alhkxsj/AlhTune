/**
 * Navbar Module - Alpine.js data factory
 * Handles: toast notifications, navbar interactions
 * Usage: x-data="navbarModule()"
 */

window.navbarModule = function () {
    return {
        showSettings: false,
        showFavorites: false,
        toasts: [],

        init() {
            // Auto-hide toasts after 3 seconds
            setInterval(() => {
                if (this.toasts.length > 0) {
                    this.toasts.shift()
                }
            }, 3000)
        },

        showToast(message, type = 'info') {
            const iconMap = {
                success: 'fa-check-circle',
                error: 'fa-circle-exclamation',
                warning: 'fa-triangle-exclamation',
                info: 'fa-info-circle'
            }

            this.toasts.push({
                message,
                type,
                icon: iconMap[type] || iconMap.info
            })
        }
    }
}
