// Video generation render worker
import { FFT, processVisualizerBars, drawVisualizerRings } from './fft.js';

// Configuration objects
class RenderConfig {
    constructor(apiRoot, sessionId, data, setStatus) {
        this.apiRoot = apiRoot;
        this.sessionId = sessionId;
        this.data = data;
        this.setStatus = setStatus;
    }
}

class CanvasConfig {
    constructor(logicalW, logicalH) {
        this.logicalW = logicalW;
        this.logicalH = logicalH;
        this.scaleFactor = 1.5;
        this.width = logicalW * this.scaleFactor;
        this.height = logicalH * this.scaleFactor;
    }
}

class RenderContext {
    constructor(canvasConfig) {
        this.canvas = document.createElement("canvas");
        this.canvas.width = canvasConfig.width;
        this.canvas.height = canvasConfig.height;
        this.ctx = this.canvas.getContext("2d");
        this.config = canvasConfig;
    }
}

class AudioRenderData {
    constructor(audioBuffer) {
        this.buffer = audioBuffer;
        this.fps = 30;
        this.duration = audioBuffer.duration;
        this.totalFrames = Math.floor(this.duration * this.fps);
        this.rawData = audioBuffer.getChannelData(0);
        this.samplesPerFrame = Math.floor(audioBuffer.sampleRate / this.fps);
        this.fftSize = 2048;
    }
}

class LyricConfig {
    constructor(lyricRaw, logicalW, logicalH) {
        this.lyricRaw = lyricRaw;
        this.logicalW = logicalW;
        this.logicalH = logicalH;
        this.lx = 600;
        this.baseLy = logicalH / 2;
        this.maxWidth = logicalW - 640;
    }
}

async function initializeSession(apiRoot, data, setStatus) {
    setStatus("正在初始化...", "请求云端处理通道", 5);
    
    let initRes;
    if (data.customAudioFile) {
        setStatus("正在初始化...", "正在向服务器投递您的本地音乐...", 5);
        const fd = new FormData();
        fd.append("id", data.id);
        fd.append("source", data.source);
        fd.append("audio_file", data.customAudioFile);
        
        initRes = await fetch(`${apiRoot}/videogen/init`, {
            method: "POST",
            body: fd
        }).then(r => r.json());
    } else {
        initRes = await fetch(`${apiRoot}/videogen/init`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ id: data.id, source: data.source }),
        }).then((r) => r.json());
    }
    
    if (initRes.error) throw new Error(initRes.error);
    return initRes;
}

async function loadAudioBuffer(data, initRes, setStatus) {
    setStatus("解码音频...", "解析本地高清音频数据...", 15);
    
    const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
    let audioBuffer;
    
    if (data.customAudioFile) {
        const arr = await data.customAudioFile.arrayBuffer();
        audioBuffer = await audioCtx.decodeAudioData(arr);
    } else {
        setStatus("下载与解码音频...", "可能需要一些时间，请耐心等待", 15);
        const arr = await fetch(initRes.audio_url).then((r) => r.arrayBuffer());
        audioBuffer = await audioCtx.decodeAudioData(arr);
    }
    
    return audioBuffer;
}

async function loadBackgroundMedia(apiRoot, data) {
    let bgMedia = null;
    
    if (data.isVideoBg) {
        bgMedia = document.createElement("video");
        bgMedia.src = data.rawCover;
        bgMedia.muted = true;
        bgMedia.loop = true;
        bgMedia.setAttribute('playsinline', '');
        await bgMedia.play();
        bgMedia.pause();
    } else {
        bgMedia = new Image();
        bgMedia.crossOrigin = "Anonymous";
        let coverSrc = data.rawCover;
        
        if (!data.rawCover.startsWith("data:")) {
            coverSrc = `${apiRoot}/download_cover?url=${encodeURIComponent(data.rawCover)}&name=render&artist=render`;
        }
        
        await Promise.race([
            new Promise(r => {
                bgMedia.onload = r;
                bgMedia.onerror = () => {
                    bgMedia.src = "https://via.placeholder.com/600";
                    setTimeout(r, 1000);
                };
                bgMedia.src = coverSrc;
            }),
            new Promise((_, r) => setTimeout(() => r(new Error("资源加载超时")), 15000))
        ]);
    }
    
    return bgMedia;
}

async function seekVideo(bgMedia, time) {
    if (!bgMedia.duration) return;
    const tt = time % bgMedia.duration;
    bgMedia.currentTime = tt;
    
    if (Math.abs(bgMedia.currentTime - tt) < 0.1 && bgMedia.readyState >= 3) return;
    
    await new Promise(r => {
        const onSeeked = () => {
            bgMedia.removeEventListener('seeked', onSeeked);
            r();
        };
        setTimeout(() => {
            bgMedia.removeEventListener('seeked', onSeeked);
            r();
        }, 500);
        bgMedia.addEventListener('seeked', onSeeked);
    });
}

async function uploadBatch(renderConfig, frames, startIdx) {
    await fetch(`${renderConfig.apiRoot}/videogen/frame`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            session_id: renderConfig.sessionId,
            frames: frames,
            start_idx: startIdx
        })
    });
}

// Background render configuration
class BackgroundConfig {
    constructor(bgMedia, data, canvasConfig) {
        this.bgMedia = bgMedia;
        this.data = data;
        this.canvasConfig = canvasConfig;
    }
}

function drawBackground(ctx, bgConfig, time) {
    const { bgMedia, data, canvasConfig } = bgConfig;
    const { logicalW, logicalH } = canvasConfig;
    let mw = data.isVideoBg ? bgMedia.videoWidth : bgMedia.width;
    let mh = data.isVideoBg ? bgMedia.videoHeight : bgMedia.height;
    if (!mw) mw = logicalW;
    if (!mh) mh = logicalH;

    const baseRatio = Math.max(logicalW / mw, logicalH / mh);
    let imgScale = 1.0;

    if (!data.isVideoBg) {
        const cycle = 20;
        const progress = (time % (cycle * 2)) / cycle;
        const ease = progress < 1 ? progress : 2 - progress;
        imgScale = 1.0 + (ease * ease * (3 - 2 * ease) * 0.1);
    }

    const finalRatio = baseRatio * imgScale;
    const bgW = mw * finalRatio;
    const bgH = mh * finalRatio;
    const bgX = (logicalW - bgW) / 2;
    const bgY = (logicalH - bgH) / 2;

    ctx.drawImage(bgMedia, bgX, bgY, bgW, bgH);
}

// Disc render configuration
class DiscConfig {
    constructor(bgMedia, discRadius, mediaWidth, mediaHeight) {
        this.bgMedia = bgMedia;
        this.discRadius = discRadius;
        this.mediaWidth = mediaWidth;
        this.mediaHeight = mediaHeight;
    }
}

function drawCenterDisc(ctx, discConfig, time) {
    const { bgMedia, discRadius, mediaWidth: mw, mediaHeight: mh } = discConfig;
    ctx.save();
    ctx.translate(discRadius, discRadius);
    ctx.beginPath();
    ctx.arc(0, 0, discRadius, 0, Math.PI * 2);
    ctx.fillStyle = "#111";
    ctx.fill();
    ctx.strokeStyle = "rgba(255,255,255,0.1)";
    ctx.lineWidth = 4;
    ctx.stroke();
    
    const grad = ctx.createRadialGradient(0, 0, discRadius * 0.5, 0, 0, discRadius);
    grad.addColorStop(0, '#1a1a1a');
    grad.addColorStop(0.5, '#222');
    grad.addColorStop(1, '#111');
    ctx.fillStyle = grad;
    ctx.fill();
    
    const coverRadius = discRadius * 0.65;
    ctx.save();
    ctx.rotate(time * 0.4);
    ctx.beginPath();
    ctx.arc(0, 0, coverRadius, 0, Math.PI * 2);
    ctx.clip();
    ctx.drawImage(bgMedia, 0, 0, mw, mh, -coverRadius, -coverRadius, coverRadius * 2, coverRadius * 2);
    ctx.restore();
    ctx.restore();
}

// Frame render configuration
class FrameRenderConfig {
    constructor(renderConfig, bgMedia, audioRenderData, renderCtx) {
        this.renderConfig = renderConfig;
        this.bgMedia = bgMedia;
        this.audioRenderData = audioRenderData;
        this.renderCtx = renderCtx;
    }
}

async function renderFrame(frameConfig, frameIdx) {
    const { renderConfig, bgMedia, audioRenderData, renderCtx } = frameConfig;
    const { data } = renderConfig;
    const { ctx, config: canvasConfig } = renderCtx;
    const { fps, samplesPerFrame, rawData, fftSize } = audioRenderData;

    const time = frameIdx / fps;
    if (data.isVideoBg) await seekVideo(bgMedia, time);

    const startSample = Math.max(0, Math.floor((frameIdx * samplesPerFrame) - (fftSize / 4)));

    let pcmSlice = rawData.slice(startSample, startSample + fftSize);
    if (pcmSlice.length < fftSize) {
        const padded = new Float32Array(fftSize);
        padded.set(pcmSlice);
        pcmSlice = padded;
    }

    const freqData = FFT.getFrequencyData(pcmSlice, fftSize, 0.65);
    const visResult = processVisualizerBars(freqData);

    ctx.clearRect(0, 0, canvasConfig.width, canvasConfig.height);
    ctx.save();
    ctx.scale(canvasConfig.scaleFactor, canvasConfig.scaleFactor);

    const bgConfig = new BackgroundConfig(bgMedia, data, canvasConfig);
    drawBackground(ctx, bgConfig, time);

    const cx = 320, cy = canvasConfig.logicalH / 2, discRadius = 200, barBaseRadius = discRadius + 2;
    drawVisualizerRings(ctx, cx, cy, barBaseRadius, visResult.heights);

    let mw = data.isVideoBg ? bgMedia.videoWidth : bgMedia.width;
    let mh = data.isVideoBg ? bgMedia.videoHeight : bgMedia.height;
    if (!mw) mw = canvasConfig.logicalW;
    if (!mh) mh = canvasConfig.logicalH;

    const discConfig = new DiscConfig(bgMedia, discRadius, mw, mh);
    drawCenterDisc(ctx, discConfig, time);

    const lyricConfig = new LyricConfig(data.lyricRaw, canvasConfig.logicalW, canvasConfig.logicalH);
    drawLyrics(ctx, time, lyricConfig);

    ctx.restore();

    return ctx.canvas.toDataURL("image/jpeg", 0.92);
}

async function renderLoop(renderConfig, bgMedia, audioBuffer, renderCtx) {
    const { setStatus } = renderConfig;
    const audioRenderData = new AudioRenderData(audioBuffer);
    const batchSize = 30;

    FFT.reset();
    setStatus("超清渲染中", "0%", 30);

    const frameConfig = new FrameRenderConfig(renderConfig, bgMedia, audioRenderData, renderCtx);

    for (let i = 0; i < audioRenderData.totalFrames; i += batchSize) {
        const batchPromises = [];
        const currentBatchSize = Math.min(batchSize, audioRenderData.totalFrames - i);

        for (let j = 0; j < currentBatchSize; j++) {
            batchPromises.push(renderFrame(frameConfig, i + j));
        }

        const frames = await Promise.all(batchPromises);
        await uploadBatch(renderConfig, frames, i);

        const pct = 30 + ((i / audioRenderData.totalFrames) * 65);
        setStatus("超清渲染中", `${Math.round(pct)}%`, pct);
    }
}

async function finishRendering(renderConfig, name) {
    const { apiRoot, sessionId, setStatus } = renderConfig;
    setStatus("正在合成视频...", "最后一步啦，再等一等吧~", 95);
    
    const finishRes = await fetch(`${apiRoot}/videogen/finish`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ session_id: sessionId, name: name })
    }).then(r => r.json());
    
    if (finishRes.error) throw new Error(finishRes.error);
    
    setStatus("完成!", "视频已生成，点击下载按钮保存", 100);
    
    if (window.opener && !window.opener.closed) {
        window.opener.postMessage({ type: 'render-complete', url: finishRes.url }, '*');
    }
    
    return finishRes.url;
}

export async function runOfflineRender(data) {
    const apiRoot = data.apiRoot;
    const statusText = document.getElementById("status-text");
    const progressFill = document.getElementById("progress-fill");
    const titleEl = document.getElementById("title");
    const previewCanvas = document.getElementById("preview-canvas");
    
    const setStatus = (title, desc, pct) => {
        if (title) titleEl.textContent = title;
        if (desc) statusText.textContent = desc;
        if (pct !== undefined) progressFill.style.width = pct + "%";
    };

    try {
        const logicalW = 1280, logicalH = 720;
        const canvasConfig = new CanvasConfig(logicalW, logicalH);
        const renderCtx = new RenderContext(canvasConfig);
        
        previewCanvas.width = canvasConfig.width;
        previewCanvas.height = canvasConfig.height;
        const previewCtx = previewCanvas.getContext("2d");
        
        const initRes = await initializeSession(apiRoot, data, setStatus);
        const renderConfig = new RenderConfig(apiRoot, initRes.session_id, data, setStatus);
        
        const audioBuffer = await loadAudioBuffer(data, initRes, setStatus);
        
        setStatus("加载视觉资源...", "准备 1080P 超清渲染画板", 25);
        const bgMedia = await loadBackgroundMedia(apiRoot, data);
        
        await renderLoop(renderConfig, bgMedia, audioBuffer, renderCtx);
        await finishRendering(renderConfig, data.name);
        
    } catch (err) {
        console.error("Render error:", err);
        setStatus("渲染失败", err.message, 0);
        
        if (window.opener && !window.opener.closed) {
            window.opener.postMessage({ type: 'render-error', error: err.message }, '*');
        }
    }
}

function findActiveLyricIndex(time, lyricRaw) {
    let activeIdx = -1;
    for (let i = 0; i < lyricRaw.length; i++) {
        if (time >= lyricRaw[i].time) activeIdx = i;
        else break;
    }
    return activeIdx;
}

function wrapText(ctx, text, maxW) {
    const lines = [];
    let currentLine = '';
    const chars = Array.from(text);
    
    for (let i = 0; i < chars.length; i++) {
        let testLine = currentLine + chars[i];
        if (ctx.measureText(testLine).width > maxW && currentLine.length > 0) {
            if (/[a-zA-Z]/.test(chars[i]) && currentLine.includes(' ')) {
                let lastSpace = currentLine.lastIndexOf(' ');
                lines.push(currentLine.substring(0, lastSpace));
                currentLine = currentLine.substring(lastSpace + 1) + chars[i];
            } else {
                lines.push(currentLine);
                currentLine = chars[i];
            }
        } else {
            currentLine = testLine;
        }
    }
    if (currentLine) lines.push(currentLine);
    return lines;
}

function createLyricsBlocks(ctx, lyricConfig, activeIdx) {
    const { lyricRaw, maxWidth } = lyricConfig;
    const blocks = [];
    let activeBlockIndex = -1;
    
    for (let offset = -4; offset <= 4; offset++) {
        const idx = activeIdx + offset;
        if (idx >= 0 && idx < lyricRaw.length) {
            const isCurrent = offset === 0;
            ctx.font = isCurrent ? "bold 36px sans-serif" : "600 26px sans-serif";
            const lineHeight = isCurrent ? 48 : 34;
            const textLines = wrapText(ctx, lyricRaw[idx].text, maxWidth);
            const blockHeight = (textLines.length - 1) * lineHeight;
            
            blocks.push({
                offset, textLines, isCurrent, lineHeight, blockHeight,
                font: ctx.font,
                color: isCurrent ? "#fff" : "rgba(255,255,255,0.85)",
                shadowBlur: isCurrent ? 6 : 4,
                shadowOffset: isCurrent ? 2 : 1
            });
            if (isCurrent) activeBlockIndex = blocks.length - 1;
        }
    }
    
    return { blocks, activeBlockIndex };
}

function calculateBlockPositions(blocks, activeBlockIndex, baseLy, gap) {
    if (activeBlockIndex === -1) return;
    
    const activeBlock = blocks[activeBlockIndex];
    activeBlock.startY = baseLy - (activeBlock.blockHeight / 2);
    
    for (let i = activeBlockIndex + 1; i < blocks.length; i++) {
        const prev = blocks[i - 1];
        blocks[i].startY = prev.startY + prev.blockHeight + gap + (prev.lineHeight / 2) + (blocks[i].lineHeight / 2);
    }
    
    for (let i = activeBlockIndex - 1; i >= 0; i--) {
        const next = blocks[i + 1];
        blocks[i].startY = next.startY - blocks[i].blockHeight - gap - (next.lineHeight / 2) - (blocks[i].lineHeight / 2);
    }
}

function renderLyricsBlocks(ctx, lyricConfig, blocks) {
    const { lx, baseLy } = lyricConfig;
    for (let block of blocks) {
        ctx.font = block.font;
        ctx.fillStyle = block.color;
        ctx.shadowColor = "rgba(0,0,0,0.9)";
        ctx.shadowBlur = block.shadowBlur;
        ctx.shadowOffsetX = block.shadowOffset;
        ctx.shadowOffsetY = block.shadowOffset;
        
        let lineY = block.startY;
        for (let lineText of block.textLines) {
            let alpha = 1;
            const dist = Math.abs(lineY - baseLy);
            if (dist > 230) alpha = Math.max(0, 1 - (dist - 230) / 70);
            
            ctx.globalAlpha = alpha;
            ctx.fillText(lineText, lx, lineY);
            lineY += block.lineHeight;
        }
    }
    ctx.globalAlpha = 1;
}

function drawLyrics(ctx, time, lyricConfig) {
    ctx.textAlign = "left";
    ctx.textBaseline = "middle";
    
    const activeIdx = findActiveLyricIndex(time, lyricConfig.lyricRaw);
    const { blocks, activeBlockIndex } = createLyricsBlocks(ctx, lyricConfig, activeIdx);
    
    if (activeBlockIndex !== -1) {
        const gap = 20;
        calculateBlockPositions(blocks, activeBlockIndex, lyricConfig.baseLy, gap);
        renderLyricsBlocks(ctx, lyricConfig, blocks);
    }
}
