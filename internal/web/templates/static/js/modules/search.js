/**
 * Search Module - Alpine.js data factory
 * Handles: search form, source selection, recommendations
 * Usage: x-data="searchModule()"
 */

window.searchModule = function () {
    return {
        searchType: 'song',
        keyword: '',
        selectedSources: [],
        allSources: [],
        error: '',

        SOURCE_NAMES: {
            netease: '网易云',
            qq: 'QQ 音乐',
            kugou: '酷狗',
            kuwo: '酷我',
            migu: '咪咕',
            fivesing: '5sing',
            jamendo: 'Jamendo',
            joox: 'JOOX',
            qianqian: '千千',
            soda: 'Soda',
            bilibili: 'B 站'
        },

        EXCLUDED_DEFAULT: ['bilibili', 'joox', 'jamendo', 'fivesing'],

        PLAYLIST_SUPPORTED: ['netease', 'qq', 'kugou', 'kuwo', 'bilibili', 'soda', 'fivesing'],

        RECOMMEND_SUPPORTED: ['netease', 'qq', 'kugou', 'kuwo'],

        init() {
            this.allSources = window.allSources || []
            this.selectedSources = window.selectedSources || this.getDefaultSources()
        },

        getDefaultSources() {
            return this.allSources.filter(s => !this.EXCLUDED_DEFAULT.includes(s))
        },

        getSourceName(source) {
            return this.SOURCE_NAMES[source] || source
        },

        selectAll() {
            this.selectedSources = [...this.allSources]
        },

        clearAll() {
            this.selectedSources = []
        },

        performSearch() {
            if (!this.keyword.trim()) {
                this.error = '请输入搜索内容'
                return
            }

            const sources = this.selectedSources.join(',')
            const url = `${window.API_ROOT}/search?q=${encodeURIComponent(this.keyword)}&type=${this.searchType}&sources=${sources}`
            window.location.href = url
        },

        goToRecommend() {
            const selected = this.selectedSources.filter(s => this.RECOMMEND_SUPPORTED.includes(s))

            if (selected.length === 0) {
                alert('请至少选择一个支持推荐歌单的音乐源')
                return
            }

            const params = new URLSearchParams()
            selected.forEach(s => params.append('sources', s))
            window.location.href = `/recommend?${params.toString()}`
        },

        isSourceSupported(source) {
            return this.PLAYLIST_SUPPORTED.includes(source)
        }
    }
}
