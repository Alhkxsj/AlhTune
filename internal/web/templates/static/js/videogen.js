// Main VideoGen module - UI and player controls
import { FFT, processVisualizerBars, drawVisualizerRings } from './modules/fft.js';

(function() {
    // Check if running in render worker mode
    if (window.isRenderWorker) {
        import('./modules/render-worker.js').then(({ runOfflineRender }) => {
            runOfflineRender(window.renderData);
        });
        return;
    }

    // Main VideoGen UI controller
    window.VideoGen = {
        data: null,
        customVisual: null,
        lyricTimes: [],
        lyricRaw: [],
        lastActiveIndex: -1,
        
        audioCtx: null,
        analyser: null,
        sourceNode: null,
        localSourceNode: null,
        
        isPlaying: false,
        rtCanvas: null,
        rtCtx: null,
        animationId: null,
        isVideoBg: false,
        resizeObserver: null,
        isDraggingProgress: false,
        
        // Local audio support
        isLocalAudio: false,
        localAudio: null,
        _currentAudioEl: null,
        _currentLocalAudioFile: null,
        
        apTimeHandler: null,
        apPlayHandler: null,
        apPauseHandler: null,
        apEndHandler: null,

        // Format time display
        formatTime: function(s) {
            if (isNaN(s) || !isFinite(s)) return "00:00";
            const m = Math.floor(s / 60);
            const sec = Math.floor(s % 60);
            return `${m < 10 ? '0' : ''}${m}:${sec < 10 ? '0' : ''}${sec}`;
        },

        // Volume control
        setVolume: function(vol) {
            if (this.isLocalAudio && this.localAudio) {
                this.localAudio.volume = vol;
            } else if (window.ap) {
                window.ap.volume && window.ap.volume(vol, true);
            }
            const vt = document.getElementById('vg-vol-text');
            if (vt) vt.textContent = Math.round(vol * 100) + '%';
            this.updateVolIcon(vol);
        },

        updateVolIcon: function(vol) {
            const icon = document.getElementById('vg-vol-icon');
            if (!icon) return;
            
            if (vol === 0) {
                icon.className = "fa-solid fa-volume-xmark";
            } else if (vol < 0.5) {
                icon.className = "fa-solid fa-volume-low";
            } else {
                icon.className = "fa-solid fa-volume-high";
            }
        },

        toggleMute: function() {
            const vb = document.getElementById('vg-volume-bar');
            if (!vb) return;
            
            let currentVol = vb.value / 100;
            if (currentVol > 0) {
                this._lastVol = currentVol;
                vb.value = 0;
                this.setVolume(0);
            } else {
                let targetVol = this._lastVol || 0.7;
                vb.value = targetVol * 100;
                this.setVolume(targetVol);
            }
        },

        // File selection handlers
        handleFileSelect: function(input) {
            if (input.files && input.files[0]) {
                const file = input.files[0];
                const reader = new FileReader();
                reader.onload = (e) => {
                    this.customVisual = e.target.result;
                    this.updateVisuals(this.customVisual, file.type.startsWith("video/"));
                };
                reader.readAsDataURL(file);
            }
            input.value = "";
        },

        handleAudioSelect: function(input) {
            if (input.files && input.files[0]) {
                const file = input.files[0];
                this._currentLocalAudioFile = file;
                
                if (!this.localAudio) {
                    this.localAudio = document.createElement("audio");
                    this.localAudio.crossOrigin = "anonymous";
                }
                
                this.localAudio.src = URL.createObjectURL(file);
                this.isLocalAudio = true;
                
                const fileName = file.name.replace(/\.[^/.]+$/, "");
                document.getElementById("vg-title").textContent = fileName;
                this.data.name = fileName;
                document.getElementById("vg-artist").textContent = "";
                this.data.artist = "";

                if (window.ap && !window.ap.audio.paused) window.ap.pause();
                
                this.attachEvents(this.localAudio);
                this.localAudio.play();
            }
            input.value = "";
        },

        handleLyricSelect: function(input) {
            if (input.files && input.files[0]) {
                const reader = new FileReader();
                reader.onload = (e) => {
                    this.parseAndSetLyrics(e.target.result);
                };
                reader.readAsText(input.files[0]);
            }
            input.value = "";
        },

        // Update background visuals
        updateVisuals: function(src, isVideo) {
            this.isVideoBg = isVideo;
            const bgImg = document.getElementById("vg-bg-img");
            const bgVid = document.getElementById("vg-bg-video");
            const cvImg = document.getElementById("vg-cover-img");
            const cvVid = document.getElementById("vg-cover-video");
            
            bgImg.style.display = "none";
            bgVid.style.display = "none";
            cvImg.style.display = "none";
            cvVid.style.display = "none";
            bgVid.pause();
            cvVid.pause();
            
            if (isVideo) {
                bgVid.src = src;
                bgVid.style.display = "block";
                bgVid.play().catch(() => {});
                cvVid.src = src;
                cvVid.style.display = "block";
                cvVid.play().catch(() => {});
            } else {
                bgImg.src = src;
                bgImg.style.display = "block";
                cvImg.src = src;
                cvImg.style.display = "block";
            }
        },

        // Attach audio event listeners
        attachEvents: function(audioEl) {
            if (!audioEl) return;
            this.detachEvents(this._currentAudioEl);
            this._currentAudioEl = audioEl;

            if (!this.apTimeHandler) {
                this.apTimeHandler = () => this.syncLyrics();
                this.apPlayHandler = () => {
                    if (!this.isLocalAudio && window.currentPlayingId !== this.data.id) return;
                    this.isPlaying = true;
                    this.updatePlayUI();
                    this.initAudioContext();
                    this.startRealtimeVisualizer();
                    
                    const b = document.getElementById("vg-bg-video");
                    const c = document.getElementById("vg-cover-video");
                    if (b?.style.display !== 'none') b.play().catch(() => {});
                    if (c?.style.display !== 'none') c.play().catch(() => {});
                    document.getElementById("vg-bg-img")?.classList.add("playing");
                };
                this.apPauseHandler = () => {
                    this.isPlaying = false;
                    this.updatePlayUI();
                    this.stopRealtimeVisualizer();
                    
                    const b = document.getElementById("vg-bg-video");
                    const c = document.getElementById("vg-cover-video");
                    if (b?.style.display !== 'none') b.pause();
                    if (c?.style.display !== 'none') c.pause();
                    document.getElementById("vg-bg-img")?.classList.remove("playing");
                };
                this.apEndHandler = () => {
                    this.isPlaying = false;
                    this.updatePlayUI();
                    this.stopRealtimeVisualizer();
                };
            }

            audioEl.addEventListener('timeupdate', this.apTimeHandler);
            audioEl.addEventListener('play', this.apPlayHandler);
            audioEl.addEventListener('pause', this.apPauseHandler);
            audioEl.addEventListener('ended', this.apEndHandler);
        },

        // Detach audio event listeners
        detachEvents: function(audioEl) {
            if (!audioEl) return;
            if (this.apTimeHandler) audioEl.removeEventListener('timeupdate', this.apTimeHandler);
            if (this.apPlayHandler) audioEl.removeEventListener('play', this.apPlayHandler);
            if (this.apPauseHandler) audioEl.removeEventListener('pause', this.apPauseHandler);
            if (this.apEndHandler) audioEl.removeEventListener('ended', this.apEndHandler);
        },

        // Sync lyrics with current time
        syncLyrics: function() {
            const audio = this.isLocalAudio ? this.localAudio : (window.ap ? window.ap.audio : null);
            if (!audio) return;
            
            const currentTime = audio.currentTime;
            let activeIdx = -1;
            
            for (let i = 0; i < this.lyricRaw.length; i++) {
                if (currentTime >= this.lyricRaw[i].time) activeIdx = i;
                else break;
            }
            
            if (activeIdx !== this.lastActiveIndex) {
                this.lastActiveIndex = activeIdx;
                this.renderLyrics(activeIdx);
            }
        },

        // Render lyrics at position
        renderLyrics: function(activeIdx) {
            const container = document.getElementById("vg-lyrics");
            if (!container) return;
            
            container.innerHTML = "";
            const startIdx = Math.max(0, activeIdx - 2);
            const endIdx = Math.min(this.lyricRaw.length, activeIdx + 3);
            
            for (let i = startIdx; i < endIdx; i++) {
                const line = document.createElement("div");
                line.textContent = this.lyricRaw[i].text;
                line.className = i === activeIdx ? "active" : "";
                container.appendChild(line);
            }
            
            container.scrollTop = (activeIdx - startIdx) * 40 - 80;
        },

        // Parse LRC lyrics
        parseAndSetLyrics: function(lrcText) {
            const lines = lrcText.split("\n");
            const timeRegex = /\[(\d{2}):(\d{2})\.(\d{2,3})\]/;
            
            this.lyricRaw = [];
            this.lyricTimes = [];
            
            for (const line of lines) {
                const match = timeRegex.exec(line);
                if (match) {
                    const minutes = parseInt(match[1]);
                    const seconds = parseInt(match[2]);
                    const milliseconds = parseInt(match[3]) / (match[3].length > 2 ? 100 : 10);
                    const time = minutes * 60 + seconds + milliseconds;
                    const text = line.replace(timeRegex, "").trim();
                    
                    if (text) {
                        this.lyricRaw.push({ time, text });
                        this.lyricTimes.push(time);
                    }
                }
            }
            
            this.renderLyrics(-1);
        },

        // Update play/pause UI
        updatePlayUI: function() {
            const btn = document.getElementById("vg-play-btn");
            if (!btn) return;
            
            if (this.isPlaying) {
                btn.innerHTML = '<i class="fa-solid fa-pause"></i>';
            } else {
                btn.innerHTML = '<i class="fa-solid fa-play"></i>';
            }
        },

        // Initialize audio context for visualizer
        initAudioContext: function() {
            if (this.audioCtx) return;
            
            const audio = this.isLocalAudio ? this.localAudio : (window.ap ? window.ap.audio : null);
            if (!audio) return;
            
            this.audioCtx = new (window.AudioContext || window.webkitAudioContext)();
            this.analyser = this.audioCtx.createAnalyser();
            this.analyser.fftSize = 256;
            
            try {
                this.sourceNode = this.audioCtx.createMediaElementSource(audio);
                this.sourceNode.connect(this.analyser);
                this.analyser.connect(this.audioCtx.destination);
            } catch (e) {
                console.warn("Audio context already initialized");
            }
        },

        // Start realtime visualizer
        startRealtimeVisualizer: function() {
            if (!this.analyser || !this.rtCanvas) return;
            
            this.rtCtx = this.rtCanvas.getContext("2d");
            const bufferLength = this.analyser.frequencyBinCount;
            const dataArray = new Uint8Array(bufferLength);
            
            const draw = () => {
                if (!this.isPlaying) return;
                
                this.animationId = requestAnimationFrame(draw);
                this.analyser.getByteFrequencyData(dataArray);
                
                this.rtCtx.fillStyle = "rgba(0, 0, 0, 0.2)";
                this.rtCtx.fillRect(0, 0, this.rtCanvas.width, this.rtCanvas.height);
                
                const barWidth = (this.rtCanvas.width / bufferLength) * 2.5;
                let x = 0;
                
                for (let i = 0; i < bufferLength; i++) {
                    const barHeight = (dataArray[i] / 255) * this.rtCanvas.height;
                    this.rtCtx.fillStyle = `hsl(${i * 2}, 100%, 50%)`;
                    this.rtCtx.fillRect(x, this.rtCanvas.height - barHeight, barWidth, barHeight);
                    x += barWidth + 1;
                }
            };
            
            draw();
        },

        // Stop realtime visualizer
        stopRealtimeVisualizer: function() {
            if (this.animationId) {
                cancelAnimationFrame(this.animationId);
                this.animationId = null;
            }
        },

        // Open render window
        openRenderWindow: function() {
            const width = 1280, height = 720;
            const left = (screen.width - width) / 2;
            const top = (screen.height - height) / 2;
            
            const win = window.open(
                "",
                "VideoGen",
                `width=${width},height=${height},left=${left},top=${top}`
            );
            
            if (win) {
                win.isRenderWorker = true;
                win.renderData = {
                    apiRoot: window.apiRoot || "/api",
                    id: this.data.id,
                    source: this.data.source,
                    name: document.getElementById("vg-title")?.textContent || this.data.name,
                    artist: document.getElementById("vg-artist")?.textContent || this.data.artist,
                    rawCover: this.customVisual || this.data.cover,
                    isVideoBg: this.isVideoBg,
                    lyricRaw: this.lyricRaw,
                    customAudioFile: this._currentLocalAudioFile
                };
                
                // Copy required modules
                win.FFT = FFT;
                win.processVisualizerBars = processVisualizerBars;
                win.drawVisualizerRings = drawVisualizerRings;
            }
        }
    };

    // Initialize on DOM ready
    document.addEventListener("DOMContentLoaded", function() {
        const songDataEl = document.getElementById("song-data");
        if (!songDataEl) return;
        
        try {
            const data = JSON.parse(songDataEl.textContent);
            window.VideoGen.data = data;
            
            // Setup canvas
            window.VideoGen.rtCanvas = document.getElementById("vg-visualizer");
            
            // Initialize lyrics
            if (data.lyric) {
                window.VideoGen.parseAndSetLyrics(data.lyric);
            }
            
            // Load cover
            if (data.cover) {
                window.VideoGen.updateVisuals(data.cover, false);
            }
        } catch (e) {
            console.error("Failed to initialize VideoGen:", e);
        }
    });
})();
