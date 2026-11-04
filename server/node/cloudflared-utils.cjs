'use strict';

const fs = require('fs');
const path = require('path');

// Android/Termux must use the package installed under $PREFIX, not a
// downloaded Linux binary left in the project's bin/ directory.
function findTermuxCloudflared(env = process.env, exists = fs.existsSync) {
    if (!env.PREFIX) return null;
    const candidate = path.join(env.PREFIX, 'bin', 'cloudflared');
    return exists(candidate) ? candidate : null;
}

// cloudflared's URL may be split across multiple stderr data events.
function createQuickTunnelUrlReader() {
    let pending = '';
    return (chunk) => {
        pending = (pending + String(chunk)).slice(-4096);
        const match = pending.match(/https:\/\/[a-zA-Z0-9-]+\.trycloudflare\.com/);
        return match ? match[0] : null;
    };
}

module.exports = { findTermuxCloudflared, createQuickTunnelUrlReader };
