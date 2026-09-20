// SPA 真实渲染体检（低内存机器上替代 agent-browser）。
//
// 背景：本机未安装 agent-browser，且内存吃紧装不了 Chromium；但 Windows 自带 Edge。
// 用 `msedge --headless=new --remote-debugging-port` + Node 22 内置 WebSocket 走 CDP，
// 可以拿到「真实渲染后的 DOM + 控制台报错 + 截图」——这是唯一能抓到
// `ReferenceError: xxx is not defined` 这类「编译通过但运行时白屏」问题的手段。
//
// 为什么不用 `--dump-dom` / `--screenshot`：它们在 load 事件就动手，SPA 初次导航
// （惰性路由组件尚未解析）时 #app 仍是空的，会误判成「渲染失败」。必须 CDP 等待页面 settle。
//
// 用法：
//   node scripts/render-check.mjs <url> [--wait 12000] [--shot out.png] [--quiet]
//
// 需要登录态的页面：先在 frontend/public/ 放一个同源探针页注入 token 再 location.replace(目标路由)，
// 或者直接对本项目用 `--token <jwt>`（脚本会自动造一个探针页、跑完再删）。
import { spawn } from 'node:child_process';
import { writeFileSync, readFileSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { join, dirname, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';

const EDGE_CANDIDATES = [
  'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
  'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
];

const argv = process.argv.slice(2);
let url = argv.find((a) => !a.startsWith('--'));
const flag = (name, def) => {
  const i = argv.indexOf(name);
  return i >= 0 && argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[i + 1] : def;
};
const has = (name) => argv.includes(name);

if (!url) {
  console.error('用法: node scripts/render-check.mjs <url> [--wait ms] [--shot out.png]');
  process.exit(2);
}

const WAIT_MS = Number(flag('--wait', 12000));
const SHOT = flag('--shot', '');
const QUIET = has('--quiet');
const EDGE = EDGE_CANDIDATES.find((p) => existsSync(p));
if (!EDGE) {
  console.error('找不到 msedge.exe');
  process.exit(2);
}

const PORT = 9333;
const PROFILE = join(tmpdir(), 'edge-render-' + Date.now());
let pendingShot = null;

// --token <jwt>：自动在 frontend/public 下造同源探针页（跑完删除），用于验证登录态页面
let probePath = null;
const token = flag('--token', '');
if (token) {
  const here = dirname(fileURLToPath(import.meta.url));
  const publicDir = resolve(here, '..', 'public');
  if (!existsSync(publicDir)) mkdirSync(publicDir, { recursive: true });
  probePath = join(publicDir, '__render_check.html');
  const target = new URL(url).pathname + new URL(url).search;
  writeFileSync(
    probePath,
    `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>probe</title></head><body><script>
try { localStorage.setItem('token', ${JSON.stringify(token)}); localStorage.setItem('refreshToken', ${JSON.stringify(token)}); localStorage.setItem('sidebarCollapsed','false'); } catch(e){}
location.replace(${JSON.stringify(target)});
</script></body></html>`,
    'utf8'
  );
  url = new URL(url).origin + '/__render_check.html';
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let proc = null;

async function main() {
  proc = spawn(
    EDGE,
    [
      '--headless=new',
      '--disable-gpu',
      '--no-first-run',
      '--no-default-browser-check',
      '--disable-extensions',
      '--mute-audio',
      '--disable-features=Translate,OptimizationGuideModelDownloading',
      `--remote-debugging-port=${PORT}`,
      `--user-data-dir=${PROFILE}`,
      '--window-size=1600,1000',
      'about:blank',
    ],
    { stdio: 'ignore' }
  );

  for (let i = 0; i < 80; i++) {
    try {
      const r = await fetch(`http://127.0.0.1:${PORT}/json/version`);
      if (r.ok) break;
    } catch {}
    if (i === 79) throw new Error('devtools 未就绪');
    await sleep(400);
  }

  const created = await fetch(`http://127.0.0.1:${PORT}/json/new?${encodeURIComponent('about:blank')}`, {
    method: 'PUT',
  });
  const target = await created.json();
  const ws = new WebSocket(target.webSocketDebuggerUrl);

  let id = 0;
  const pending = new Map();
  const consoleMsgs = [];
  const exceptions = [];

  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.id && pending.has(msg.id)) {
      const { resolve, reject } = pending.get(msg.id);
      pending.delete(msg.id);
      msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
      return;
    }
    if (msg.method === 'Runtime.consoleAPICalled') {
      const text = (msg.params.args || []).map((a) => a.value ?? a.description ?? a.type).join(' ');
      consoleMsgs.push(`[${msg.params.type}] ${text}`);
    }
    if (msg.method === 'Runtime.exceptionThrown') {
      const d = msg.params.exceptionDetails || {};
      exceptions.push(`${d.text || ''} ${d.exception?.description || d.exception?.value || ''}`.trim());
    }
    if (msg.method === 'Log.entryAdded') {
      const e = msg.params.entry || {};
      if (e.level === 'error') consoleMsgs.push(`[log:error] ${e.text} ${e.url || ''}`);
    }
  };

  const send = (method, params = {}) =>
    new Promise((res, rej) => {
      const mid = ++id;
      pending.set(mid, { resolve: res, reject: rej });
      ws.send(JSON.stringify({ id: mid, method, params }));
    });

  await new Promise((res, rej) => {
    ws.onopen = res;
    ws.onerror = rej;
  });

  await send('Page.enable');
  await send('Runtime.enable');
  await send('Log.enable');
  await send('Page.navigate', { url });
  await sleep(WAIT_MS);

  const probe = `(() => {
    const txt = (el) => (el ? (el.innerText || '').replace(/\\s+/g, ' ').trim() : null);
    const app = document.querySelector('#app');
    const main = document.querySelector('.main-content');
    return JSON.stringify({
      title: document.title,
      href: location.href,
      appHtmlLen: app ? app.innerHTML.length : -1,
      mainHtmlLen: main ? main.innerHTML.length : -1,
      mainTextHead: txt(main) ? txt(main).slice(0, 500) : null,
      tableRows: document.querySelectorAll('.el-table__body tbody tr').length,
      cards: document.querySelectorAll('.mat-card').length,
      dialogOpen: !!document.querySelector('.el-dialog'),
      bodyTextLen: document.body.innerText.length
    });
  })()`;

  const res = await send('Runtime.evaluate', { expression: probe, returnByValue: true });

  // --eval-file：settle 之后再跑一段自定义 JS（用于驱动交互：打开弹窗、点选人员、读输入框等）。
  // 先 eval 再截图，这样截图反映的是交互后的真实状态。
  const evalFile = flag('--eval-file', '');
  if (evalFile) {
    const expr = readFileSync(evalFile, 'utf8');
    const r2 = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
    console.log(`=== EVAL (${evalFile}) ===`);
    console.log(JSON.stringify(r2.result?.value ?? r2.result, null, 2));
    const after = await send('Runtime.evaluate', { expression: probe, returnByValue: true });
    console.log('=== PROBE AFTER EVAL ===');
    console.log(after.result?.value || '');
  }

  if (SHOT) {
    const shot = await send('Page.captureScreenshot', { format: 'png' });
    writeFileSync(SHOT, Buffer.from(shot.data, 'base64'));
    pendingShot = SHOT;
  }

  console.log('=== PROBE ===');
  console.log(res.result?.value || JSON.stringify(res));
  console.log(`=== EXCEPTIONS (${exceptions.length}) ===`);
  exceptions.slice(0, 20).forEach((e) => console.log('  ' + e));
  if (!QUIET) {
    const noisy = consoleMsgs.filter((m) => !m.startsWith('[debug] [vite]'));
    console.log(`=== CONSOLE (${noisy.length}) ===`);
    noisy.slice(0, 30).forEach((m) => console.log('  ' + m));
  }
  ws.close();
}

main()
  .catch((e) => {
    console.log('FATAL ' + e.message);
    process.exitCode = 1;
  })
  .finally(async () => {
    try { proc?.kill('SIGKILL'); } catch {}
    if (probePath) {
      try { rmSync(probePath, { force: true }); } catch {}
    }
    if (pendingShot) console.log('\n截图已写入: ' + pendingShot);
    await sleep(800);
    process.exit(process.exitCode || 0);
  });
