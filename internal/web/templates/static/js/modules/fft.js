// FFT and audio processing utilities
export const FFT = {
    windowed: null,
    mags: null,
    previousMags: null,

    reset: function() {
        this.previousMags = null;
    },

    fft: function(data) {
        const n = data.length;
        if (n <= 1) return data;
        
        const half = n / 2;
        const even = new Float32Array(half);
        const odd = new Float32Array(half);
        
        for (let i = 0; i < half; i++) {
            even[i] = data[2 * i];
            odd[i] = data[2 * i + 1];
        }
        
        const q = this.fft(even);
        const r = this.fft(odd);
        const output = new Float32Array(n);
        
        for (let k = 0; k < half; k++) {
            const t = r[k];
            output[k] = q[k] + t;
            output[k + half] = q[k] - t;
        }
        
        return output;
    },

    getFrequencyData: function(pcmData, fftSize, smoothing) {
        const half = fftSize / 2;
        
        if (!this.windowed || this.windowed.length !== fftSize) {
            this.windowed = new Float32Array(fftSize);
            this.mags = new Uint8Array(half);
            this.previousMags = new Float32Array(half);
        }
        
        for (let i = 0; i < fftSize; i++) {
            const val = (i < pcmData.length) ? pcmData[i] : 0;
            this.windowed[i] = val * (0.5 * (1 - Math.cos(2 * Math.PI * i / (fftSize - 1))));
        }
        
        const rawFFT = this.fft(this.windowed);
        
        for (let i = 0; i < half; i++) {
            let mag = Math.abs(rawFFT[i]) / fftSize;
            mag = mag * 2.0;
            mag = smoothing * this.previousMags[i] + (1 - smoothing) * mag;
            this.previousMags[i] = mag;
            
            let db = 20 * Math.log10(mag + 1e-6);
            const minDb = -100, maxDb = -10;
            let val = (db - minDb) * (255 / (maxDb - minDb));
            
            if (val < 0) val = 0;
            if (val > 255) val = 255;
            
            this.mags[i] = val;
        }
        
        return this.mags;
    }
};

export function processVisualizerBars(freqData) {
    const barsCount = 180;
    const barHeights = [];
    const maxIdx = Math.floor(freqData.length * 0.8);
    const minIdx = 1;
    
    for (let i = 0; i < barsCount; i++) {
        const logRange = Math.log(maxIdx / minIdx);
        const idx = minIdx * Math.exp(logRange * (i / barsCount));
        const lower = Math.floor(idx);
        const upper = Math.ceil(idx);
        const frac = idx - lower;
        
        let val = (freqData[lower] || 0) * (1 - frac) + (freqData[upper] || 0) * frac;
        val *= 1 + (i / barsCount) * 0.8;
        
        if (val > 255) val = 255;
        
        let h = 2;
        if (val > 0) h += Math.pow(val / 255.0, 2.5) * 40;
        
        barHeights.push(h);
    }
    
    return { heights: barHeights };
}

export function drawVisualizerRings(ctx, cx, cy, radius, heights) {
    ctx.save();
    ctx.translate(cx, cy);
    
    const barsCount = heights.length;
    const barWidth = 1.5;
    const halfWidth = barWidth / 2;
    
    for (let i = 0; i < barsCount; i++) {
        ctx.save();
        ctx.rotate((Math.PI * 2 / barsCount) * i - Math.PI / 2);
        
        const h = heights[i] || 2;
        const hue = (i / barsCount) * 360;
        
        ctx.fillStyle = `hsla(${hue}, 100%, 65%, 0.9)`;
        ctx.beginPath();
        
        if (ctx.roundRect) {
            ctx.roundRect(-halfWidth, -radius - h, barWidth, h, 0.5);
        } else {
            ctx.rect(-halfWidth, -radius - h, barWidth, h);
        }
        
        ctx.fill();
        ctx.restore();
    }
    
    ctx.restore();
}
