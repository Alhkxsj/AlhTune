// Download utilities module
import { webSettings } from './settings.js';

export function buildDownloadURL(id, source, name, artist, cover = '', extra = '') {
    const params = new URLSearchParams({
        id: String(id || ''),
        source: String(source || ''),
        name: String(name || ''),
        artist: String(artist || '')
    });

    const coverValue = String(cover || '');
    if (coverValue !== '') {
        params.set('cover', coverValue);
    }
    
    const extraValue = String(extra || '');
    if (extraValue !== '' && extraValue !== '{}' && extraValue !== 'null') {
        params.set('extra', extraValue);
    }
    
    if (webSettings.embedDownload) {
        params.set('embed', '1');
    }

    return `${window.API_ROOT}/download?${params.toString()}`;
}

export function refreshDownloadLinks() {
    document.querySelectorAll('.song-card').forEach(card => {
        const dl = card.querySelector('.btn-download');
        if (!dl) return;

        const ds = card.dataset;
        dl.href = buildDownloadURL(
            ds.id,
            ds.source,
            ds.name,
            ds.artist,
            ds.cover || '',
            ds.extra || ''
        );
    });
}
