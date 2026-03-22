/**
 * List Module - Alpine.js data factory
 * Handles: batch operations, song selection, play controls
 * Usage: x-data="listModule()"
 */

window.listModule = function () {
    return {
        batchMode: false,
        selectedSongs: [],
        selectAllChecked: false,

        init() {
            this.selectedSongs = []
            this.batchMode = false
            this.selectAllChecked = false
        },

        toggleBatchMode() {
            this.batchMode = !this.batchMode
            if (!this.batchMode) {
                this.selectedSongs = []
                this.selectAllChecked = false
            }
        },

        toggleSelectAll(checked) {
            this.selectAllChecked = checked
        },

        toggleSongSelection(songData, checked) {
            if (checked) {
                this.selectedSongs.push(songData)
            } else {
                this.selectedSongs = this.selectedSongs.filter(s => s.id !== songData.id)
            }
        },

        getSelectedCount() {
            return this.selectedSongs.length
        },

        playAll(btn) {
            if (window.playerModuleInstance) {
                window.playerModuleInstance.playAllFromButton(btn)
            }
        },

        batchDownload() {
            this.selectedSongs.forEach(song => {
                const url = window.buildDownloadURL(
                    song.id,
                    song.source,
                    song.name,
                    song.artist,
                    song.cover,
                    song.extra
                )
                window.open(url, '_blank')
            })
        }
    }
}
