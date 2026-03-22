/**
 * Player Module - Alpine.js data factory
 * Handles: APlayer integration, playback controls, volume management
 * Usage: x-data="playerModule()"
 */

window.playerModule = function () {
    return {
        ap: null,
        currentPlayingId: null,
        currentPlayingSource: null,
        isPlaying: false,
        playMode: 'list',
        volume: 0.7,

        MODES: {
            list: { apMode: 'all', icon: 'fa-list', text: '列表循环' },
            single: { apMode: 'one', icon: 'fa-repeat-1', text: '单曲循环' },
            random: { apMode: 'random', icon: 'fa-shuffle', text: '随机播放' }
        },

        init() {
            this.loadVolume()
            this.setupGlobalPlayer()
        },

        setupGlobalPlayer() {
            if (!this.ap && document.getElementById('aplayer')) {
                this.ap = new APlayer({
                    container: document.getElementById('aplayer'),
                    mini: false,
                    autoplay: false,
                    loop: 'all',
                    order: 'list',
                    preload: 'auto',
                    volume: this.volume,
                    mutex: true,
                    listFolded: false,
                    lrcType: 3
                })

                this.ap.audio.addEventListener('play', () => {
                    this.isPlaying = true
                })

                this.ap.audio.addEventListener('pause', () => {
                    this.isPlaying = false
                })

                this.ap.audio.addEventListener('ended', () => {
                    this.handleSongEnd()
                })

                this.ap.audio.addEventListener('volumechange', () => {
                    this.volume = this.ap.audio.volume
                    this.saveVolume()
                })
            }
        },

        playSong(songData) {
            this.setupGlobalPlayer()
            if (!this.ap) return

            this.currentPlayingId = songData.id
            this.currentPlayingSource = songData.source

            const audioUrl = window.buildDownloadURL(
                songData.id,
                songData.source,
                songData.name,
                songData.artist,
                songData.cover,
                songData.extra
            )

            const existingIndex = this.ap.list.audios.findIndex(audio => audio.id === songData.id)

            if (existingIndex === -1) {
                this.ap.list.add([{
                    name: songData.name,
                    artist: songData.artist,
                    url: audioUrl,
                    cover: songData.cover,
                    lrc: '',
                    id: songData.id
                }])
                this.ap.list.switch(this.ap.list.audios.length - 1)
            } else {
                this.ap.list.switch(existingIndex)
            }

            this.ap.play()
            this.isPlaying = true
        },

        playAllFromButton(btn) {
            const card = btn.closest('.song-card')
            if (!card) return

            this.setupGlobalPlayer()
            const allCards = document.querySelectorAll('.song-card')

            const audios = Array.from(allCards).map(c => ({
                name: c.dataset.name,
                artist: c.dataset.artist,
                url: window.buildDownloadURL(
                    c.dataset.id,
                    c.dataset.source,
                    c.dataset.name,
                    c.dataset.artist,
                    c.dataset.cover,
                    c.dataset.extra
                ),
                cover: c.dataset.cover,
                lrc: '',
                id: c.dataset.id
            }))

            this.ap.list.clear()
            this.ap.list.add(audios)

            const index = Array.from(allCards).indexOf(card)
            this.ap.list.switch(index)
            this.ap.play()

            this.currentPlayingId = card.dataset.id
            this.currentPlayingSource = card.dataset.source
            this.isPlaying = true
        },

        togglePlay(btn) {
            const card = btn.closest('.song-card')
            if (!card) return

            const songId = card.dataset.id

            if (this.currentPlayingId === songId && this.ap) {
                if (this.isPlaying) {
                    this.ap.pause()
                } else {
                    this.ap.play()
                }
            } else {
                this.playSong({
                    id: songId,
                    source: card.dataset.source,
                    name: card.dataset.name,
                    artist: card.dataset.artist,
                    cover: card.dataset.cover,
                    extra: card.dataset.extra
                })
            }
        },

        switchPlayMode() {
            const modeKeys = Object.keys(this.MODES)
            const currentIndex = modeKeys.indexOf(this.playMode)
            this.playMode = modeKeys[(currentIndex + 1) % modeKeys.length]

            if (this.ap) {
                this.ap.mode = this.MODES[this.playMode].apMode
            }
        },

        getPlayModeIcon() {
            return this.MODES[this.playMode]?.icon || 'fa-list'
        },

        getPlayModeText() {
            return this.MODES[this.playMode]?.text || '列表循环'
        },

        handleSongEnd() {
            if (!this.ap) return

            if (this.playMode === 'single') {
                this.ap.play()
            } else if (this.playMode === 'random') {
                const randomIndex = Math.floor(Math.random() * this.ap.list.audios.length)
                this.ap.list.switch(randomIndex)
                this.ap.play()
            }
        },

        isCurrentSong(songId) {
            return this.currentPlayingId === songId
        },

        loadVolume() {
            try {
                const saved = localStorage.getItem('musicdl:volume')
                if (saved) {
                    this.volume = parseFloat(saved)
                }
            } catch (e) {}
        },

        saveVolume() {
            try {
                localStorage.setItem('musicdl:volume', JSON.stringify(this.volume))
            } catch (e) {}
        },

        setVolume(vol) {
            this.volume = vol
            if (this.ap) {
                this.ap.volume(vol, true)
            }
        }
    }
}

// Global instance reference
window.playerModuleInstance = null
document.addEventListener('alpine:init', () => {
    window.playerModuleInstance = window.Alpine.store('player')
})
