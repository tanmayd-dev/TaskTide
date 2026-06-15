// perlin.js - Premium Ambient Perlin Noise Background

class PerlinNoise {
    constructor() {
        this.p = new Uint8Array(512);
        const permutation = new Uint8Array(256);
        for (let i = 0; i < 256; i++) permutation[i] = i;
        
        // Shuffle permutation table
        for (let i = 255; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            const tmp = permutation[i];
            permutation[i] = permutation[j];
            permutation[j] = tmp;
        }
        
        for (let i = 0; i < 512; i++) {
            this.p[i] = permutation[i & 255];
        }
    }
    
    fade(t) { return t * t * t * (t * (t * 6 - 15) + 10); }
    lerp(t, a, b) { return a + t * (b - a); }
    grad(hash, x, y) {
        const h = hash & 7;
        const u = h < 4 ? x : y;
        const v = h < 4 ? y : x;
        return ((h & 1) ? -u : u) + ((h & 2) ? -2.0 * v : 2.0 * v);
    }
    
    noise(x, y) {
        const X = Math.floor(x) & 255;
        const Y = Math.floor(y) & 255;
        x -= Math.floor(x);
        y -= Math.floor(y);
        const u = this.fade(x);
        const v = this.fade(y);
        const A = this.p[X] + Y;
        const B = this.p[X + 1] + Y;
        
        return this.lerp(v,
            this.lerp(u, this.grad(this.p[A], x, y), this.grad(this.p[B], x - 1, y)),
            this.lerp(u, this.grad(this.p[A + 1], x, y - 1), this.grad(this.p[B + 1], x - 1, y - 1))
        );
    }
}

// Initialize and animate background canvas
window.addEventListener('DOMContentLoaded', () => {
    const canvas = document.createElement('canvas');
    canvas.id = 'perlin-bg';
    canvas.style.position = 'fixed';
    canvas.style.top = '0';
    canvas.style.left = '0';
    canvas.style.width = '100vw';
    canvas.style.height = '100vh';
    canvas.style.zIndex = '-1';
    canvas.style.pointerEvents = 'none';
    canvas.style.filter = 'blur(80px)';
    canvas.style.opacity = '0.55';
    document.body.appendChild(canvas);

    const ctx = canvas.getContext('2d');
    const perlin = new PerlinNoise();
    
    // Set small dimensions for high performance (the CSS blur + stretch handles the smoothness)
    const width = 48;
    const height = 48;
    canvas.width = width;
    canvas.height = height;

    let time = 0;

    function animate() {
        time += 0.003;
        const imageData = ctx.createImageData(width, height);
        const data = imageData.data;

        for (let y = 0; y < height; y++) {
            for (let x = 0; x < width; x++) {
                // Generate layered noise for rich textures
                const nx = x * 0.07;
                const ny = y * 0.07;
                
                const n1 = perlin.noise(nx + time, ny);
                const n2 = perlin.noise(ny - time * 0.5, nx + time * 0.7);
                const val = (n1 + n2) * 0.5; // range [-1, 1]

                // Map noise value to deep purple, blue, and teal gradients
                const hue = 220 + val * 60; // range 160 (teal) to 280 (purple)
                const lightness = Math.floor(12 + (val + 1) * 6); // range 6% to 18% (dark mode ambient)
                
                // Convert HSL to RGB to paint pixels
                const rgb = hslToRgb(hue / 360, 0.75, lightness / 100);

                const idx = (y * width + x) * 4;
                data[idx] = rgb[0];     // R
                data[idx + 1] = rgb[1]; // G
                data[idx + 2] = rgb[2]; // B
                data[idx + 3] = 255;    // A (fully opaque)
            }
        }

        ctx.putImageData(imageData, 0, 0);
        requestAnimationFrame(animate);
    }

    // Helper: HSL to RGB conversion
    function hslToRgb(h, s, l) {
        let r, g, b;
        if (s === 0) {
            r = g = b = l; // achromatic
        } else {
            const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
            const p = 2 * l - q;
            r = hue2rgb(p, q, h + 1/3);
            g = hue2rgb(p, q, h);
            b = hue2rgb(p, q, h - 1/3);
        }
        return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255)];
    }

    function hue2rgb(p, q, t) {
        if (t < 0) t += 1;
        if (t > 1) t -= 1;
        if (t < 1/6) return p + (q - p) * 6 * t;
        if (t < 1/2) return q;
        if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
        return p;
    }

    animate();
});
