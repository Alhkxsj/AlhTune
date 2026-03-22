// Main application entry point
import { loadWebSettings, saveWebSettings, webSettings } from './modules/settings.js';
import { buildDownloadURL, refreshDownloadLinks } from './modules/download.js';
import { 
    currentPlayingId, 
    currentPlayingSource,
    switchPlayMode, 
    updatePlayModeButton, 
    syncAllPlayButtons,
    handleSongCardClick,
    onPlayerPlay,
    onPlayerPause,
    onPlayerEnded
} from './modules/player.js';
import { addToCollection, removeFromCollection, createCollection, getCollections } from './modules/collection.js';

const API_ROOT = window.API_ROOT;
const INSPECT_REQUEST_DELAY_MS = 100;

// Make functions globally available for HTML onclick handlers
window.switchPlayMode = switchPlayMode;
window.addToCollection = addToCollection;
window.removeFromCollection = removeFromCollection;
window.createCollection = createCollection;
window.getCollections = getCollections;
window.currentPlayingId = currentPlayingId;
window.currentPlayingSource = currentPlayingSource;

// Expose player callbacks for APlayer events
window.updatePlayerInfo = function(songData) {
    // Update player UI with song data
    const playerEl = document.getElementById('player-song-name');
    if (playerEl) playerEl.textContent = songData.name || '未知曲目';
    
    const artistEl = document.getElementById('player-artist');
    if (artistEl) artistEl.textContent = songData.artist || '';
    
    const coverEl = document.getElementById('player-cover');
    if (coverEl && songData.cover) coverEl.src = songData.cover;
};

window.onApPlay = onPlayerPlay;
window.onApPause = onPlayerPause;
window.onApEnded = onPlayerEnded;

document.addEventListener('DOMContentLoaded', function() {
    // Load saved settings
    loadWebSettings();

    // Setup source checkboxes
    setupSourceCheckboxes();
    
    // Setup search type toggle
    setupSearchTypeToggle();
    
    // Setup song cards
    setupSongCards();
    
    // Setup settings toggles
    setupSettingsToggles();
    
    // Refresh download links with current settings
    refreshDownloadLinks();
    
    // Sync play buttons
    syncAllPlayButtons();
    
    // Update play mode button
    updatePlayModeButton();
    
    // Setup play mode button
    const playModeBtn = document.getElementById('play-mode-btn');
    if (playModeBtn) {
        playModeBtn.addEventListener('click', switchPlayMode);
    }
    
    // Setup APlayer event listeners if available
    setupAPlayerListeners();
});

function setupSourceCheckboxes() {
    const btnAll = document.getElementById('btn-all');
    const btnNone = document.getElementById('btn-none');
    const checkboxes = document.querySelectorAll('.source-checkbox');
    
    if (btnAll) {
        btnAll.onclick = () => {
            checkboxes.forEach(cb => {
                if (!cb.disabled) cb.checked = true;
            });
        };
    }
    
    if (btnNone) {
        btnNone.onclick = () => {
            checkboxes.forEach(cb => {
                if (!cb.disabled) cb.checked = false;
            });
        };
    }
}

function setupSearchTypeToggle() {
    const initialTypeEl = document.querySelector('input[name="type"]:checked');
    if (initialTypeEl) {
        toggleSearchType(initialTypeEl.value);
    }
    
    document.querySelectorAll('input[name="type"]').forEach(radio => {
        radio.addEventListener('change', (e) => {
            toggleSearchType(e.target.value);
        });
    });
}

function toggleSearchType(type) {
    const checkboxes = document.querySelectorAll('.source-checkbox');
    checkboxes.forEach(cb => {
        const isSupported = cb.dataset.supported === "true";
        
        if (type === 'playlist') {
            if (!isSupported) {
                cb.disabled = true;
                cb.checked = false;
            } else {
                cb.disabled = false;
            }
        } else {
            cb.disabled = false;
        }
    });
}

function setupSongCards() {
    const cards = document.querySelectorAll('.song-card');
    
    // Queue song inspection
    cards.forEach((card, index) => {
        queueInspectSong(card, index * INSPECT_REQUEST_DELAY_MS);
    });
    
    // Setup cover click for VideoGen
    cards.forEach(card => {
        const coverWrap = card.querySelector('.cover-wrapper');
        if (!coverWrap) return;
        
        coverWrap.style.cursor = 'pointer';
        coverWrap.title = '点击生成视频';
        
        coverWrap.onclick = (e) => {
            e.stopPropagation();
            openVideoGen(card);
        };
    });
    
    // Setup card click for playback
    cards.forEach(card => {
        card.addEventListener('click', (e) => {
            // Ignore if clicking on buttons
            if (e.target.closest('.btn-play') || 
                e.target.closest('.btn-download') ||
                e.target.closest('.btn-collection')) {
                return;
            }
            
            handleSongCardClick(card);
        });
    });
    
    // Setup play buttons
    cards.forEach(card => {
        const playBtn = card.querySelector('.btn-play');
        if (playBtn) {
            playBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                handleSongCardClick(card);
            });
        }
    });
}

function setupSettingsToggles() {
    const embedToggle = document.getElementById('setting-embed-download');
    if (embedToggle) {
        embedToggle.checked = webSettings.embedDownload;
        embedToggle.addEventListener('change', (e) => {
            webSettings.embedDownload = e.target.checked;
            saveWebSettings();
            refreshDownloadLinks();
        });
    }
}

function setupAPlayerListeners() {
    // Wait for APlayer to be initialized
    const checkAPlayer = setInterval(() => {
        if (window.ap) {
            clearInterval(checkAPlayer);
            
            // Listen to APlayer events
            window.ap.audio.addEventListener('play', window.onApPlay);
            window.ap.audio.addEventListener('pause', window.onApPause);
            window.ap.audio.addEventListener('ended', window.onApEnded);
        }
    }, 100);
}

function queueInspectSong(card, delay) {
    setTimeout(() => {
        inspectSong(card);
    }, delay);
}

async function inspectSong(card) {
    const id = card.dataset.id;
    const source = card.dataset.source;
    
    if (!id || !source) return;
    
    try {
        const response = await fetch(`${API_ROOT}/inspect?id=${encodeURIComponent(id)}&source=${encodeURIComponent(source)}`, {
            method: 'GET'
        });
        
        if (response.ok) {
            const data = await response.json();
            
            // Update card with inspected data
            if (data.duration) {
                card.dataset.duration = data.duration;
                const durationEl = card.querySelector('.duration');
                if (durationEl) durationEl.textContent = formatDuration(data.duration);
            }
            
            if (data.playable === false) {
                card.classList.add('unplayable');
                const playBtn = card.querySelector('.btn-play');
                if (playBtn) {
                    playBtn.disabled = true;
                    playBtn.title = '无法播放';
                }
            }
        }
    } catch (error) {
        console.error('Song inspection failed:', error);
    }
}

function formatDuration(seconds) {
    if (!seconds || isNaN(seconds)) return '--:--';
    
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
}

function openVideoGen(card) {
    if (window.VideoGen) {
        const img = card.querySelector('.cover-wrapper img');
        const currentCover = img ? img.src : (card.dataset.cover || '');

        window.VideoGen.openRenderWindow({
            id: card.dataset.id,
            source: card.dataset.source,
            name: card.dataset.name,
            artist: card.dataset.artist,
            cover: currentCover,
            duration: parseInt(card.dataset.duration) || 0
        });
    } else {
        console.error("VideoGen library not loaded.");
        alert("视频生成组件加载失败，请刷新页面重试");
    }
}

// Go to recommendation page
window.goToRecommend = function() {
    const supported = ['netease', 'qq', 'kugou', 'kuwo'];
    const selected = [];
    
    document.querySelectorAll('.source-checkbox:checked').forEach(cb => {
        if (supported.includes(cb.value)) {
            selected.push(cb.value);
        }
    });
    
    if (selected.length === 0) {
        alert('请至少选择一个支持推荐歌单的音乐源');
        return;
    }
    
    const params = new URLSearchParams();
    selected.forEach(s => params.append('sources', s));
    
    window.location.href = `/recommend?${params.toString()}`;
};

// Batch operations
window.batchAddToCollection = async function(collectionId) {
    const selectedCards = document.querySelectorAll('.song-card.selected');
    
    if (selectedCards.length === 0) {
        alert('请先选择要添加的歌曲');
        return;
    }
    
    let successCount = 0;
    
    for (const card of selectedCards) {
        const songData = {
            id: card.dataset.id,
            source: card.dataset.source,
            name: card.dataset.name,
            artist: card.dataset.artist,
            cover: card.dataset.cover,
            duration: parseInt(card.dataset.duration) || 0
        };
        
        const result = await addToCollection(collectionId, songData);
        if (result) successCount++;
    }
    
    alert(`成功添加 ${successCount}/${selectedCards.length} 首歌曲到收藏夹`);
};

// Selection toggle
window.toggleSongSelection = function(card) {
    card.classList.toggle('selected');
    
    const selectedCount = document.querySelectorAll('.song-card.selected').length;
    const batchActions = document.getElementById('batch-actions');
    
    if (batchActions) {
        batchActions.style.display = selectedCount > 0 ? 'flex' : 'none';
    }
};
