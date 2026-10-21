package handler

import (
	_ "embed"
	"net/http"
)

const homeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>ShortLink — Cloud URL Shortener</title>
  <meta name="description" content="High performance production-ready URL Shortener powered by Go and Neon PostgreSQL">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: rgba(18, 24, 38, 0.85);
      --border: rgba(255, 255, 255, 0.08);
      --primary: #6366f1;
      --primary-hover: #4f46e5;
      --primary-glow: rgba(99, 102, 241, 0.25);
      --accent: #06b6d4;
      --text-main: #f8fafc;
      --text-muted: #94a3b8;
      --success: #10b981;
      --error: #ef4444;
      --radius: 16px;
    }

    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }

    body {
      font-family: 'Plus Jakarta Sans', -apple-system, sans-serif;
      background-color: var(--bg);
      background-image: 
        radial-gradient(circle at 15% 20%, rgba(99, 102, 241, 0.15) 0%, transparent 40%),
        radial-gradient(circle at 85% 80%, rgba(6, 182, 212, 0.12) 0%, transparent 40%);
      color: var(--text-main);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 40px 20px;
    }

    .container {
      width: 100%;
      max-width: 820px;
      display: flex;
      flex-direction: column;
      gap: 32px;
    }

    header {
      text-align: center;
    }

    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 6px 14px;
      border-radius: 9999px;
      background: rgba(16, 185, 129, 0.1);
      border: 1px solid rgba(16, 185, 129, 0.3);
      color: #34d399;
      font-size: 0.85rem;
      font-weight: 600;
      margin-bottom: 16px;
    }

    .badge-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #10b981;
      box-shadow: 0 0 10px #10b981;
      animation: pulse 2s infinite;
    }

    @keyframes pulse {
      0%, 100% { opacity: 1; transform: scale(1); }
      50% { opacity: 0.5; transform: scale(0.85); }
    }

    h1 {
      font-size: 2.8rem;
      font-weight: 800;
      letter-spacing: -0.03em;
      background: linear-gradient(135deg, #ffffff 40%, var(--text-muted) 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      margin-bottom: 12px;
    }

    p.subtitle {
      color: var(--text-muted);
      font-size: 1.1rem;
      max-width: 580px;
      margin: 0 auto;
      line-height: 1.6;
    }

    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 32px;
      backdrop-filter: blur(20px);
      box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
    }

    .input-group {
      display: flex;
      flex-direction: column;
      gap: 16px;
    }

    .row {
      display: flex;
      gap: 12px;
    }

    @media (max-width: 640px) {
      .row { flex-direction: column; }
      h1 { font-size: 2.2rem; }
    }

    label {
      font-size: 0.9rem;
      font-weight: 600;
      color: var(--text-muted);
      margin-bottom: 6px;
      display: block;
    }

    input {
      width: 100%;
      background: rgba(11, 15, 25, 0.7);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px 18px;
      color: var(--text-main);
      font-size: 1rem;
      font-family: inherit;
      transition: all 0.2s ease;
      outline: none;
    }

    input:focus {
      border-color: var(--primary);
      box-shadow: 0 0 0 3px var(--primary-glow);
    }

    .btn {
      background: linear-gradient(135deg, var(--primary) 0%, var(--primary-hover) 100%);
      color: #ffffff;
      border: none;
      border-radius: 12px;
      padding: 15px 28px;
      font-size: 1rem;
      font-weight: 700;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      transition: all 0.2s ease;
      box-shadow: 0 4px 15px var(--primary-glow);
    }

    .btn:hover {
      transform: translateY(-1px);
      box-shadow: 0 6px 20px rgba(99, 102, 241, 0.4);
    }

    .btn:active {
      transform: translateY(0);
    }

    .result-box {
      margin-top: 24px;
      padding: 20px;
      border-radius: 12px;
      background: rgba(99, 102, 241, 0.08);
      border: 1px solid rgba(99, 102, 241, 0.2);
      display: none;
      animation: fadeIn 0.3s ease;
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(6px); }
      to { opacity: 1; transform: translateY(0); }
    }

    .result-content {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
    }

    .short-link {
      font-family: 'JetBrains Mono', monospace;
      color: var(--accent);
      font-size: 1.1rem;
      font-weight: 600;
      word-break: break-all;
      text-decoration: none;
    }

    .copy-btn {
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid var(--border);
      color: var(--text-main);
      padding: 8px 16px;
      border-radius: 8px;
      font-size: 0.85rem;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s ease;
      white-space: nowrap;
    }

    .copy-btn:hover {
      background: rgba(255, 255, 255, 0.2);
    }

    .status-panel {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 16px;
    }

    .stat-item {
      background: rgba(18, 24, 38, 0.5);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 16px 20px;
    }

    .stat-title {
      font-size: 0.85rem;
      color: var(--text-muted);
      margin-bottom: 4px;
    }

    .stat-value {
      font-size: 1.25rem;
      font-weight: 700;
      color: var(--text-main);
    }

    .api-docs-btn {
      color: var(--text-muted);
      text-decoration: none;
      font-size: 0.9rem;
      display: inline-flex;
      align-items: center;
      gap: 6px;
      transition: color 0.2s;
    }

    .api-docs-btn:hover {
      color: var(--text-main);
    }

    footer {
      text-align: center;
      font-size: 0.85rem;
      color: var(--text-muted);
      margin-top: 20px;
    }

    footer a {
      color: var(--primary);
      text-decoration: none;
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div class="badge">
        <span class="badge-dot"></span>
        <span id="system-status">System Operational &bull; Neon PostgreSQL</span>
      </div>
      <h1>Cloud URL Shortener</h1>
      <p class="subtitle">High-speed, collision-resistant URL shortener running on Go and Neon Serverless PostgreSQL with atomic click analytics.</p>
    </header>

    <div class="card">
      <form id="shorten-form" onsubmit="shortenURL(event)">
        <div class="input-group">
          <div>
            <label for="url-input">Destination URL</label>
            <input type="url" id="url-input" placeholder="https://example.com/very/long/destination/url" required autocomplete="off">
          </div>
          <div class="row">
            <div style="flex: 1;">
              <label for="code-input">Custom Alias (Optional)</label>
              <input type="text" id="code-input" placeholder="e.g. my-promo" pattern="^[a-zA-Z0-9_-]{4,32}$" title="4-32 alphanumeric characters, dashes, or underscores">
            </div>
            <div style="align-self: flex-end;">
              <button type="submit" class="btn" id="submit-btn">Shorten URL &rarr;</button>
            </div>
          </div>
        </div>
      </form>

      <div class="result-box" id="result-box">
        <div class="result-content">
          <div>
            <span style="font-size: 0.8rem; color: var(--text-muted); display: block; margin-bottom: 4px;">Your Shortened Link:</span>
            <a href="#" target="_blank" class="short-link" id="short-link-href"></a>
          </div>
          <button class="copy-btn" id="copy-btn" onclick="copyLink()">Copy Link</button>
        </div>
      </div>
    </div>

    <div class="card">
      <h3 style="font-size: 1.1rem; margin-bottom: 16px;">Quick API Reference</h3>
      <div class="status-panel">
        <div class="stat-item">
          <div class="stat-title">Create URL</div>
          <code style="color: var(--accent); font-size: 0.85rem;">POST /api/v1/urls</code>
        </div>
        <div class="stat-item">
          <div class="stat-title">Get URL & Analytics</div>
          <code style="color: var(--accent); font-size: 0.85rem;">GET /api/v1/urls/{code}</code>
        </div>
        <div class="stat-item">
          <div class="stat-title">Temporary Redirect</div>
          <code style="color: var(--accent); font-size: 0.85rem;">GET /{code} (HTTP 307)</code>
        </div>
        <div class="stat-item">
          <div class="stat-title">Health Probe</div>
          <code style="color: var(--accent); font-size: 0.85rem;">GET /healthz</code>
        </div>
      </div>
    </div>

    <footer>
      Powered by Go 1.25 &bull; Neon Serverless PostgreSQL &bull; Hosted on Render &bull; <a href="https://github.com/srinivasan3012/Short_URL_Generator" target="_blank">View GitHub Repository</a>
    </footer>
  </div>

  <script>
    async function shortenURL(e) {
      e.preventDefault();
      const urlInput = document.getElementById('url-input');
      const codeInput = document.getElementById('code-input');
      const submitBtn = document.getElementById('submit-btn');
      const resultBox = document.getElementById('result-box');
      const linkHref = document.getElementById('short-link-href');

      submitBtn.disabled = true;
      submitBtn.innerText = 'Shortening...';

      const payload = {
        url: urlInput.value.trim()
      };
      if (codeInput.value.trim()) {
        payload.custom_code = codeInput.value.trim();
      }

      try {
        const res = await fetch('/api/v1/urls', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (!res.ok) {
          alert('Error: ' + (data.error?.message || 'Failed to shorten URL'));
          return;
        }

        const fullURL = window.location.origin + '/' + data.code;
        linkHref.href = fullURL;
        linkHref.innerText = fullURL;
        resultBox.style.display = 'block';
      } catch (err) {
        alert('Request failed: ' + err.message);
      } finally {
        submitBtn.disabled = false;
        submitBtn.innerText = 'Shorten URL \u2192';
      }
    }

    function copyLink() {
      const link = document.getElementById('short-link-href').innerText;
      navigator.clipboard.writeText(link).then(() => {
        const btn = document.getElementById('copy-btn');
        btn.innerText = 'Copied!';
        btn.style.color = '#10b981';
        setTimeout(() => {
          btn.innerText = 'Copy Link';
          btn.style.color = '';
        }, 2000);
      });
    }

    // Check readiness on load
    fetch('/readyz')
      .then(r => r.json())
      .then(d => {
        if (d.status === 'ready') {
          document.getElementById('system-status').innerText = 'System Operational \u2022 Neon PostgreSQL Connected';
        }
      })
      .catch(() => {});
  </script>
</body>
</html>`

// Home serves the interactive web interface at the root URL.
func (h *URLHandler) Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(homeHTML))
}
