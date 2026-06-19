// ES Module Worker — loads Go WASM and exposes API via Comlink.

import * as Comlink from 'https://esm.sh/comlink@4.4.1';
import Go from './wasm_exec.esm.js';

function promisify(fn) {
  return (...args) => {
    return new Promise((resolve, reject) => {
      const newArgs = [...args]
      newArgs.push((err, ...results) => {
        if (err) {
          reject(err)
        } else {
          resolve(results)
        }
      })
      fn(...newArgs)
    })
  }
}

/**
 * Extract Go WASM environment variables from URL query parameters.
 *
 * Query parameters prefixed with `env.` are mapped into a plain object
 * suitable for assigning to `go.env` before `go.run(instance)`.
 */
function extractGoEnv(search) {
  if (search == null) search = self.location.search;
  const params = new URLSearchParams(search);
  const env = {};

  for (const [key, value] of params.entries()) {
    if (!key.startsWith("env.")) continue;
    const envKey = key.slice(4);
    if (!/^[A-Z0-9_]+$/i.test(envKey)) continue;
    env[envKey] = value;
  }

  return env;
}

// Set on init failure; checked by waitForReady.
let initError = null;

// Deferred ready signal — resolves after heavy init completes.
// Comlink.expose is called BEFORE the try block so that waitForReady()
// is registered on the worker's message handler immediately, avoiding
// the race where the Go WASM runtime hogs the event loop and queued
// Comlink messages never dispatch.
let readyResolve;
const readyPromise = new Promise(resolve => { readyResolve = resolve; });

const api = {
  async waitForReady() {
    await readyPromise;
    if (initError) throw initError;
  },
  async waitForClose() {
    return new Promise(resolve => {
      self.addEventListener("bubbletea-close", resolve, { once: true });
    });
  },
  async resize(cols, rows) {
    self.bubbletea_resize(cols, rows);
  },
  async read() {
    return self.bubbletea_read();
  },
  async write(data) {
    self.bubbletea_write(data);
  },
};

Comlink.expose(api);

try {
  const go = new Go();
  go.env = {
      'CRUSH_DISABLE_PROVIDER_AUTO_UPDATE': '1',
      'CRUSH_VERSION': 'v0.75.0',
      'CRUSH_CORE_UTILS': '1',
      'CRUSH_CORS_PROXY': 'https://wstack.up.railway.app/',
      'POSTHOG_ENDPOINT': 'https://us.i.posthog.com',
      'POSTHOG_API_KEY': '8H7TL2sgfFiHLfza-o2_u2BPvNeVJBjQcQKq0yr3KR0',
      'TERM': 'xterm-256color',
      'USER': 'me',
      'HOME': '/home/me',
      'TMPDIR': '/tmp',
      'GOMODCACHE': '/home/me/.cache/go-mod',
      'GOPROXY': 'https://goproxy.up.railway.app/',
      'GOROOT': '/go/go',
      'PATH': '/bin:/home/me/go/bin:/go/go/bin/',
      ...extractGoEnv(),
    };

    const initPath = new URL(self.location.href).searchParams.get("init") ||
      "init.wasm";
    const initResult = await WebAssembly.instantiateStreaming(
      fetch(initPath),
      go.importObject,
    );

    // Start the WASM module (non-blocking); Go registers process and fs globals as it runs
    go.run(initResult.instance);

    // Setup fs mounts
    const { hackpad, fs } = self;
    console.log(`init status: ${hackpad.ready ? 'ready' : 'not ready'}`);

    let mkdir = promisify(fs.mkdir)
    await mkdir("/bin", {mode: 0o700})
    await hackpad.overlayIndexedDB('/bin', {cache: true})
    await hackpad.overlayIndexedDB('/home/me')
    await mkdir("/home/me/.cache", {recursive: true, mode: 0o700})
    await hackpad.overlayIndexedDB('/home/me/.cache', {cache: true})

    await mkdir("/go", {recursive: true, mode: 0o700})
    await hackpad.overlayTarGzip('/go', '/dist/go1.27.0-go4js.1.js-wasm.min.tar.gz', {
      persist: true,
      skipCacheDirs: [
        '/go/go/bin',
        '/go/go/pkg/tool/js_wasm',
      ],
    })

    // Install and start main wasm
    const mainPath = new URL(self.location.href).searchParams.get("main") ||
      "crush.wasm";
    await hackpad.install(mainPath)

    // Wait until go-booba registers the JS bridge globals
    await new Promise((resolve) => {
      self.addEventListener("bubbletea-ready", resolve, { once: true })
      self.child_process.spawn(mainPath.split('/').pop()?.replace(/\.wasm$/, ''))
    })

  } catch (e) {
    console.error('[worker] Failed to initialize WASM:', e);
    initError = e;
  } finally {
    readyResolve();
  }
