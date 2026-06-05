import * as Comlink from 'comlink';
import { Terminal } from '@xterm/xterm';
import { ClipboardAddon } from '@xterm/addon-clipboard';
import { FitAddon } from '@xterm/addon-fit';
import { ImageAddon } from '@xterm/addon-image';
import { WebglAddon } from '@xterm/addon-webgl';

const loading = document.getElementById("loading");

const workerUrl = new URL('./worker.js', import.meta.url);
workerUrl.search = location.search;
const worker = new Worker(workerUrl, { type: 'module' });
const wasm = Comlink.wrap(worker);

function initTerminal() {
  const term = new Terminal({
    convertEol: true,
    cursorBlink: true,
    allowTransparency: true,
    scrollbar: { showScrollbar: false },
  });

  const clipboardAddon = new ClipboardAddon();
  term.loadAddon(clipboardAddon);

  const imageAddon = new ImageAddon();
  term.loadAddon(imageAddon);

  const fitAddon = new FitAddon();
  if (new URLSearchParams(location.search).get("webgl") !== null) {
    const webglAddon = new WebglAddon();
    try {
      term.loadAddon(webglAddon);
    } catch (e) {
      console.warn(
        "WebGL addon failed to load, falling back to canvas renderer",
        e,
      );
    }
  }
  term.loadAddon(fitAddon);

  term.open(document.getElementById("terminal-container"));

  fitAddon.fit();
  window.addEventListener("resize", () => fitAddon.fit());

  term.focus();

  // Send initial size to Go via worker
  wasm.resize(term.cols, term.rows).catch(() => {});

  /** Whether the Go program has exited; gate all input after this point. */
  let exited = false;

  // Poll Go output and write to terminal
  // Guard flag prevents overlapping read() calls when round-trip > interval.
  let reading = false;
  const pollInterval = setInterval(() => {
    if (exited || reading) return;
    reading = true;
    wasm.read()
      .then(data => data?.length && term.write(data))
      .catch(err => {
        console.error("read error:", err);
        setExited(true);
        clearInterval(pollInterval);
        term.write("\r\n\r\n[Worker error — reload page to restart]");
      })
      .finally(() => { reading = false; });
  }, 16);

  // Forward resize events to Go (debounced — skip intermediate states during live resize)
  let resizeTimer;
  term.onResize((size) => {
    if (exited) return;
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
      wasm.resize(size.cols, size.rows).catch(() => {});
    }, 80);
  });

  // Forward key/paste input to Go; reload after exit
  term.onData((data) => {
    if (exited) {
      location.reload();
      return;
    }
    wasm.write(data).catch(() => {});
  });

  return {
    term,
    pollInterval,
    setExited: (v) => {
      exited = v;
    },
  };
}

async function main() {
  // Wait for the worker to finish initialising the WASM environment
  try {
    await wasm.waitForReady();
  } catch (e) {
    loading.textContent = "Failed to load crush.wasm — check console for details";
    throw e;
  }

  // Hide the loading overlay
  loading.classList.add("hidden");

  const { term, pollInterval, setExited } = initTerminal();

  // When the Go program exits, show a restart prompt
  // Note: bubbletea-close may not reliably reach the main worker from the
  // sub-worker, so the promise often rejects. Handle both paths identically.
  wasm.waitForClose()
    .catch(() => {})
    .finally(() => {
      console.log("session ended");
      setExited(true);
      clearInterval(pollInterval);
      term.write("\r\n\r\nPress any key to continue...");
    });
}

main().catch(console.error);
