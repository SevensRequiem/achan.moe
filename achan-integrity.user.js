// ==UserScript==
// @name         achan.moe Asset Integrity Checker
// @namespace    https://achan.moe
// @version      1.1.0
// @description  Verifies JS asset integrity against out-of-band SHA-256 hashes
// @match        https://achan.moe/*
// @match        https://*.achan.moe/*
// @grant        GM_xmlhttpRequest
// @connect      raw.githubusercontent.com
// @connect      gitlab.com
// @run-at       document-idle
// ==/UserScript==

(function() {
  'use strict';

  var GITHUB_URL = 'https://raw.githubusercontent.com/SevensRequiem/achan.moe/main/e2e-signing-key.json';
  var GITLAB_URL = 'https://gitlab.com/sevensrequiem/achan-moe/-/raw/main/e2e-signing-key.json';

  var ASSET_PATTERNS = {
    'achan.js': /\/assets\/js\/dist\/achan-[a-f0-9]{8}\.js$/,
    'admin.js': /\/assets\/js\/dist\/admin-[a-f0-9]{8}\.js$/,
    'pqc.min.js': /\/assets\/js\/pqc\.min\.js$/
  };

  function findLoadedAssets() {
    var found = {};
    var scripts = document.querySelectorAll('script[src]');
    for (var i = 0; i < scripts.length; i++) {
      var src = scripts[i].getAttribute('src');
      for (var logical in ASSET_PATTERNS) {
        if (ASSET_PATTERNS[logical].test(src)) {
          found[logical] = src;
        }
      }
    }
    return found;
  }

  function sha256hex(text) {
    var buf = new TextEncoder().encode(text);
    return crypto.subtle.digest('SHA-256', buf).then(function(hash) {
      return Array.from(new Uint8Array(hash)).map(function(b) {
        return b.toString(16).padStart(2, '0');
      }).join('');
    });
  }

  function fetchJSON(url) {
    return new Promise(function(resolve) {
      GM_xmlhttpRequest({
        method: 'GET',
        url: url,
        onload: function(resp) {
          try { resolve(JSON.parse(resp.responseText)); }
          catch (_) { resolve(null); }
        },
        onerror: function() { resolve(null); }
      });
    });
  }

  function showResult(ok, details) {
    var el = document.createElement('div');
    el.style.cssText = 'position:fixed;bottom:8px;right:8px;z-index:99999;padding:6px 12px;font:12px monospace;border-radius:4px;';
    if (ok) {
      el.style.background = '#1a472a';
      el.style.color = '#4ade80';
      el.textContent = 'Assets: VERIFIED';
    } else {
      el.style.background = '#7f1d1d';
      el.style.color = '#fca5a5';
      el.textContent = 'Assets: TAMPERED - ' + details;
    }
    document.body.appendChild(el);
  }

  function verify() {
    var loaded = findLoadedAssets();

    Promise.all([fetchJSON(GITHUB_URL), fetchJSON(GITLAB_URL)]).then(function(results) {
      var gh = results[0];
      var gl = results[1];

      if (!gh && !gl) {
        showResult(false, 'could not reach verification sources');
        return;
      }

      var trusted = (gh && gl && JSON.stringify(gh.assets) === JSON.stringify(gl.assets)) ? gh : null;
      if (!trusted) {
        showResult(false, 'GitHub and GitLab hashes disagree');
        return;
      }

      var checks = [];
      for (var logical in loaded) {
        checks.push((function(name, src) {
          var expectedHash = trusted.assets[name];
          if (!expectedHash) return Promise.resolve(null);

          return fetch(src, { cache: 'no-store' }).then(function(resp) {
            if (!resp.ok) return name;
            return resp.text();
          }).then(function(text) {
            if (typeof text !== 'string') return text;
            return sha256hex(text).then(function(actualHash) {
              return actualHash !== expectedHash ? name : null;
            });
          }).catch(function() { return null; });
        })(logical, loaded[logical]));
      }

      Promise.all(checks).then(function(results) {
        var mismatches = results.filter(function(r) { return r !== null; });
        if (mismatches.length > 0) {
          showResult(false, mismatches.join(', '));
        } else {
          showResult(true, '');
        }
      });
    });
  }

  verify();
})();
