/**
 * Settings Module - Alpine.js data factory
 * Handles: user settings, cookies management, download URL builder
 * Usage: x-data="settingsModule()"
 */

window.settingsModule = function () {
    return {
        embedDownload: false,
        cookies: {},
        showSettingsModal: false,

        init() {
            this.loadSettings()
        },

        loadSettings() {
            try {
                const raw = localStorage.getItem('musicdl:web_settings')
                if (raw) {
                    const parsed = JSON.parse(raw)
                    this.embedDownload = parsed.embedDownload || false
                }
            } catch (e) {}
        },

        saveSettings() {
            try {
                localStorage.setItem(
                    'musicdl:web_settings',
                    JSON.stringify({
                        embedDownload: this.embedDownload
                    })
                )

                this.saveCookies()
                this.showSettingsModal = false
            } catch (e) {
                console.error('Failed to save settings:', e)
            }
        },

        saveCookies() {
            const cookiesToSave = {}
            Object.keys(this.cookies).forEach(source => {
                if (this.cookies[source]) {
                    cookiesToSave[source] = this.cookies[source]
                }
            })

            if (Object.keys(cookiesToSave).length === 0) return

            fetch(`${window.API_ROOT}/cookies`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ cookies: cookiesToSave })
            }).catch(err => console.error(err))
        },

        openSettings() {
            this.showSettingsModal = true
        },

        closeSettings() {
            this.showSettingsModal = false
        }
    }
}

/**
 * Build download URL with proper encoding
 * @param {string} id - Song ID
 * @param {string} source - Source provider
 * @param {string} name - Song name
 * @param {string} artist - Artist name
 * @param {string} cover - Cover image URL
 * @param {object|string} extra - Extra metadata
 * @returns {string} Download URL
 */
window.buildDownloadURL = function (id, source, name, artist, cover = '', extra = '') {
    const params = new URLSearchParams({
        id: String(id || ''),
        source: String(source || ''),
        name: String(name || ''),
        artist: String(artist || '')
    })

    if (cover) {
        params.set('cover', cover)
    }

    if (extra && extra !== '{}' && extra !== 'null') {
        params.set('extra', typeof extra === 'string' ? extra : JSON.stringify(extra))
    }

    let embedDownload = false
    try {
        const raw = localStorage.getItem('musicdl:web_settings')
        if (raw) {
            const parsed = JSON.parse(raw)
            embedDownload = parsed.embedDownload || false
        }
    } catch (e) {}

    if (embedDownload) {
        params.set('embed', '1')
    }

    return `${window.API_ROOT}/download?${params.toString()}`
}
