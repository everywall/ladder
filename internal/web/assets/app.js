"use strict";

(() => {
  const $ = (id) => document.getElementById(id);
  const base = document.body.dataset.basePath || "";
  let current = null;
  let currentView = "html";
  let readController = null;
  let activePanel = "reader";
  let historyLoaded = false;

  function status(id, message, kind = "") {
    const element = $(id);
    element.textContent = message;
    element.className = `status${kind ? ` ${kind}` : ""}`;
    element.hidden = !message;
  }

  async function api(path, { method = "GET", body, signal } = {}) {
    let response;
    try {
      response = await fetch(`${base}/api/v1${path}`, {
        method, signal, credentials: "same-origin", cache: "no-store",
        headers: { "X-Ladder-Client": "reader", ...(body !== undefined ? { "Content-Type": "application/json" } : {}) },
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      });
    } catch (error) {
      if (error.name === "AbortError") throw error;
      throw new Error("Ladder could not be reached. Check that this server is running, then try again.");
    }
    let result;
    try { result = await response.json(); } catch (_) {
      throw new Error(`Ladder returned an unreadable response (HTTP ${response.status}). Try reloading this desk.`);
    }
    if (!response.ok) throw new Error(`${result.error?.message || "The request failed."} (HTTP ${response.status}${result.error?.code ? ` · ${result.error.code}` : ""})`);
    return result;
  }

  function text(tag, value, className) {
    const element = document.createElement(tag);
    element.textContent = value;
    if (className) element.className = className;
    return element;
  }

  function date(value) {
    const parsed = new Date(value);
    return value && !Number.isNaN(parsed.valueOf()) && parsed.getUTCFullYear() > 1 ? parsed.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "Unavailable";
  }

  function safeWebURL(value) {
    try { const url = new URL(value); return ["http:", "https:"].includes(url.protocol) && !url.username && !url.password ? url.href : null; } catch (_) { return null; }
  }

  function cacheDescription(cache = {}) {
    if (!cache.ttlSeconds) return "Not cached";
    return `${cache.hit ? "Cache hit" : "Fresh retrieval"} · ${cache.ageSeconds || 0}s old · ${cache.ttlSeconds}s lifetime`;
  }

  function metadata(id, data = {}, extra = []) {
    const cache = data.cache || {};
    const matches = data.rules || [];
    const rows = [
      ["Source URL", data.sourceUrl || "Unavailable"], ["Final URL", data.finalUrl || "Unavailable"],
      ["Upstream HTTP", data.upstreamStatus ? String(data.upstreamStatus) : "Unavailable"],
      ["Content type", data.mediaType || "Unavailable"], ["Retrieved", date(data.fetchedAt)],
      ["Served", date(data.servedAt)], ["Cache", cacheDescription(cache)],
      ["Expires", cache.ttlSeconds ? date(cache.expiresAt) : "Not cached"],
      ["Matched rules", matches.length ? matches.map((match) => `${match.packId} @ ${match.packVersion} / ${match.ruleId}`).join("; ") : "No versioned pack rules matched"],
      ...extra,
    ];
    const list = $(id);
    list.replaceChildren();
    for (const [label, value] of rows) list.append(text("dt", label), text("dd", value));
  }

  function showPanel(name, focus = false) {
    if (!["reader", "history", "rules", "settings"].includes(name)) name = "reader";
    activePanel = name;
    document.querySelectorAll(".panel").forEach((element) => { element.hidden = element.id !== `panel-${name}`; });
    document.querySelectorAll("[data-panel]").forEach((button) => {
      const selected = button.dataset.panel === name;
      button.classList.toggle("active", selected);
      if (selected) button.setAttribute("aria-current", "page"); else button.removeAttribute("aria-current");
    });
    if (name === "history") loadHistory();
    if (name === "rules") loadRules();
    if (name === "settings") loadSettings();
    if (focus) { $("main").focus({ preventScroll: true }); window.scrollTo({ top: 0, behavior: "auto" }); }
  }

  document.querySelectorAll("[data-panel]").forEach((button) => button.addEventListener("click", () => showPanel(button.dataset.panel, true)));

  function escapeHTML(value) {
    return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char]));
  }

  function showView(view) {
    currentView = view;
    document.querySelectorAll("[data-view]").forEach((button) => button.setAttribute("aria-pressed", String(button.dataset.view === view)));
    $("reading-frame").hidden = view !== "html";
    $("document-code").hidden = view === "html";
    if (!current) return;
    if (view === "html") {
      // The only HTML sink is this opaque-origin, script-free sandbox. Metadata
      // and the source text never enter the trusted application's HTML tree.
      const article = current.content.html || `<pre>${escapeHTML(current.content.text || "No readable text was found.")}</pre>`;
      $("reading-frame").srcdoc = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'sha256-${READER_STYLE_HASH}'; base-uri 'none'; form-action 'none'"><style>${READER_STYLE}</style></head><body><article>${article}</article></body></html>`;
    } else $("document-code").textContent = current.content[view] || "No readable content was found in this format.";
  }

  document.querySelectorAll("[data-view]").forEach((button) => button.addEventListener("click", () => showView(button.dataset.view)));

  function renderDocument(doc) {
    current = doc;
    $("reader-empty").hidden = true;
    $("document").hidden = false;
    $("document-title").textContent = doc.title || "Untitled page";
    $("document-source").textContent = doc.metadata.sourceHost || "Retrieved page";
    $("document-byline").textContent = doc.byline || "";
    $("document-byline").hidden = !doc.byline;
    const original = safeWebURL(doc.metadata.sourceUrl);
    if (original) { $("open-original").href = original; $("open-original").hidden = false; } else $("open-original").hidden = true;
    const warnings = (doc.problems || []).map((problem) => problem.message);
    if (doc.metadata.upstreamStatus >= 400) warnings.unshift(`The source returned HTTP ${doc.metadata.upstreamStatus}. Any readable response content is shown below.`);
    if (doc.historyError) warnings.push(doc.historyError);
    status("document-problems", warnings.join(" "), "warning");
    $("retrieval-summary").textContent = `HTTP ${doc.metadata.upstreamStatus || "unknown"} · ${cacheDescription(doc.metadata.cache)} · Page details`;
    metadata("document-metadata", doc.metadata);
    $("copy-fallback").hidden = true;
    showView(currentView);
  }

  async function readPage(rawURL, cache = "default") {
    if (!safeWebURL(rawURL)) { status("read-status", "Enter a complete http:// or https:// page URL without a username or password.", "error"); return; }
    if (readController) readController.abort();
    const controller = new AbortController();
    readController = controller;
    $("read-url").value = rawURL;
    $("read-button").disabled = true;
    $("refresh-read").disabled = true;
    $("cancel-read").hidden = false;
    $("document").setAttribute("aria-busy", "true");
    status("read-status", cache === "reload" ? "Retrieving a fresh page, without using the cache…" : "Retrieving the page and clearing a space for the words…", "loading");
    try {
      const doc = await api("/read", { method: "POST", body: { url: rawURL, cache }, signal: controller.signal });
      if (readController !== controller) return;
      renderDocument(doc);
      status("read-status", `Page ready. ${doc.metadata.upstreamStatus >= 400 ? `The source returned HTTP ${doc.metadata.upstreamStatus}.` : ""}`.trim());
      const address = new URL(window.location.href);
      address.pathname = `${base}/reader`;
      address.search = "";
      address.searchParams.set("url", rawURL);
      address.hash = "";
      window.history.replaceState({}, "", address);
      if (historyLoaded) loadHistory(true);
    } catch (error) {
      if (readController !== controller) return;
      const message = error.name === "AbortError" ? "Retrieval canceled. Paste another URL or try this page again." : error.message;
      status("read-status", `${message}${current ? " The previous page remains below." : ""}`, error.name === "AbortError" ? "" : "error");
    } finally {
      if (readController === controller) {
        readController = null;
        $("read-button").disabled = false;
        $("refresh-read").disabled = false;
        $("cancel-read").hidden = true;
        $("document").removeAttribute("aria-busy");
      }
    }
  }

  $("read-form").addEventListener("submit", (event) => { event.preventDefault(); readPage($("read-url").value.trim()); });
  $("refresh-read").addEventListener("click", () => { if (current) readPage(current.metadata.sourceUrl, "reload"); });
  $("cancel-read").addEventListener("click", () => readController?.abort());
  document.addEventListener("keydown", (event) => {
    if (event.key === "/" && !event.metaKey && !event.ctrlKey && !event.altKey && !["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement.tagName) && !document.activeElement.isContentEditable) {
      event.preventDefault(); showPanel("reader"); $("read-url").focus(); $("read-url").select();
    }
    if (event.key === "Escape" && readController && activePanel === "reader") readController.abort();
  });

  $("copy-markdown").addEventListener("click", async () => {
    if (!current) return;
    try {
      if (!navigator.clipboard?.writeText) throw new Error("Clipboard unavailable");
      await navigator.clipboard.writeText(current.content.markdown || "");
      status("read-status", "Markdown copied.");
    } catch (_) {
      $("copy-fallback").hidden = false;
      $("copy-text").value = current.content.markdown || "";
      $("copy-text").focus(); $("copy-text").select();
      status("read-status", "Select the Markdown below and use your browser's copy shortcut.");
    }
  });
  $("close-copy").addEventListener("click", () => { $("copy-fallback").hidden = true; $("copy-markdown").focus(); });

  function setHistoryControls(enabled, available = true) {
    for (const id of ["history-enabled", "settings-history-enabled"]) { $(id).checked = enabled; $(id).disabled = !available; }
  }

  async function loadHistory(quiet = false) {
    if (!quiet) status("history-status", "Loading local history…", "loading");
    try {
      const snapshot = await api("/history");
      historyLoaded = true;
      renderHistory(snapshot);
      if (!quiet) status("history-status", snapshot.historyEnabled ? "" : "History is off. Existing entries remain until you delete or clear them.");
    } catch (error) { status("history-status", error.message, "error"); setHistoryControls(false, false); }
  }

  function renderHistory(snapshot) {
    setHistoryControls(snapshot.historyEnabled);
    const entries = snapshot.entries || [];
    $("history-count").textContent = entries.length ? String(entries.length) : "";
    $("clear-history").disabled = entries.length === 0;
    $("history-list").replaceChildren();
    if (!entries.length) { $("history-list").append(text("li", "Your next read starts the trail.", "list-empty")); return; }
    for (const entry of entries) {
      const row = text("li", "", "history-entry");
      const open = text("button", "", "history-open");
      open.type = "button";
      open.append(text("span", entry.title || "Untitled page", "history-title"), text("span", entry.url, "history-url"), text("span", date(entry.viewedAt), "history-time"));
      open.addEventListener("click", () => { showPanel("reader", true); readPage(entry.url); });
      const remove = text("button", "Delete", "history-delete");
      remove.type = "button";
      remove.setAttribute("aria-label", `Delete ${entry.title || entry.url} from history`);
      remove.addEventListener("click", async () => {
        remove.disabled = true;
        try { renderHistory(await api(`/history/${encodeURIComponent(entry.id)}`, { method: "DELETE" })); status("history-status", "History entry deleted."); }
        catch (error) { status("history-status", error.message, "error"); remove.disabled = false; }
      });
      row.append(open, remove); $("history-list").append(row);
    }
  }

  $("clear-history").addEventListener("click", async () => {
    if (!window.confirm("Clear all reading history on this server? This removes the saved URLs, titles, and read times.")) return;
    $("clear-history").disabled = true;
    try { renderHistory(await api("/history", { method: "DELETE" })); status("history-status", "Reading history cleared."); }
    catch (error) { status("history-status", error.message, "error"); $("clear-history").disabled = false; }
  });

  for (const id of ["history-enabled", "settings-history-enabled"]) $(id).addEventListener("change", async (event) => {
    const desired = event.target.checked;
    setHistoryControls(!desired, false);
    const target = activePanel === "settings" ? "settings-status" : "history-status";
    try { renderHistory(await api("/settings", { method: "PATCH", body: { historyEnabled: desired } })); status(target, desired ? "Reading history enabled." : "Reading history disabled. Existing entries remain until you clear them."); }
    catch (error) { setHistoryControls(!desired); status(target, error.message, "error"); }
  });

  async function loadRules() {
    status("rules-status", "Loading rule packs…", "loading");
    try {
      const result = await api("/rules");
      $("pack-list").replaceChildren();
      if (!result.packs?.length) $("pack-list").append(text("p", "No versioned rule packs are loaded. Pages use the server's default retrieval settings and any configured legacy rules.", "small"));
      for (const pack of result.packs || []) {
        const row = text("section", "", "pack");
        const heading = text("div", "", "pack-heading");
        heading.append(text("h2", pack.id), text("span", pack.version, "badge"), text("span", `${pack.rules} ${pack.rules === 1 ? "rule" : "rules"}`, "small"));
        row.append(heading, text("p", `SHA-256  ${pack.sha256}`, "pack-digest"));
        $("pack-list").append(row);
      }
      status("rules-status", "");
    } catch (error) { status("rules-status", error.message, "error"); }
  }

  $("test-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    $("test-button").disabled = true;
    $("test-result").hidden = true;
    status("test-status", "Retrieving the page and checking the matched rules…", "loading");
    try {
      const result = await api("/rules/test", { method: "POST", body: { url: $("test-url").value.trim(), cache: "bypass" } });
      status("test-status", result.passed ? "URL checks passed." : "Some URL checks did not pass. Inspect each result below.", result.passed ? "" : "warning");
      $("test-checks").replaceChildren();
      for (const check of result.checks || []) {
        const item = text("li", "", check.passed ? "passed" : "failed");
        item.append(text("strong", `${check.passed ? "✓" : "×"} ${check.name}`), text("p", check.detail));
        $("test-checks").append(item);
      }
      const matched = (result.matched || []).map((rule) => `${rule.packId} @ ${rule.packVersion} / ${rule.ruleId}`).join("; ") || "None";
      metadata("test-metadata", result.metadata, [["Test matches", matched]]);
      $("test-result").hidden = false;
    } catch (error) { status("test-status", error.message, "error"); }
    finally { $("test-button").disabled = false; }
  });

  async function loadSettings() {
    status("settings-status", "Loading settings…", "loading");
    try {
      const settings = await api("/settings");
      setHistoryControls(settings.historyEnabled, settings.historyAvailable);
      const state = settings.startup || {};
      $("startup-badge").textContent = settings.startupError ? "Unavailable" : state.enabled ? "Enabled" : "Off";
      $("startup-status").textContent = settings.startupError ? `Startup status could not be checked: ${settings.startupError}` : `${state.manager || "Service manager"}: ${state.enabled ? "enabled" : "disabled"}, ${state.running ? "running" : "not running"}.${state.installed && !state.enabled ? " A service file is installed." : ""}${state.detail ? ` ${state.detail}` : ""}`;
      status("settings-status", settings.historyError || (settings.historyAvailable ? "" : "Local history is unavailable on this server."), settings.historyAvailable ? "" : "warning");
    } catch (error) { status("settings-status", error.message, "error"); }
  }

  $("clear-cache").addEventListener("click", async () => {
    $("clear-cache").disabled = true;
    try { await api("/cache", { method: "DELETE" }); $("cache-status").textContent = "Page cache cleared. The next read will retrieve a fresh page."; }
    catch (error) { $("cache-status").textContent = error.message; }
    finally { $("clear-cache").disabled = false; }
  });

  const bookmark = `${window.location.origin}${base}/reader?url=`;
  $("bookmarklet-link").href = `javascript:(function(){location.href=${JSON.stringify(bookmark)}+encodeURIComponent(location.href)})()`;
  $("bookmarklet-link").addEventListener("click", (event) => { event.preventDefault(); status("read-status", "Drag “Read with Ladder” to Chrome's bookmarks bar. Then use the bookmark while viewing another page."); });
  function showBookmarklet(event) { event.preventDefault(); showPanel("reader"); $("bookmarklet").scrollIntoView({ behavior: "auto", block: "center" }); $("bookmarklet-link").focus({ preventScroll: true }); }
  $("bookmarklet-nav").addEventListener("click", showBookmarklet);
  $("show-bookmarklet").addEventListener("click", showBookmarklet);

  const initialURL = new URLSearchParams(window.location.search).get("url");
  if (initialURL) { $("read-url").value = initialURL; readPage(initialURL); }
})();
