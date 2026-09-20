// 低内存机器上的 Vue SFC 体检脚本（替代会 OOM 的 vue-tsc）。
//
// 两类检查：
//  1) SFC 编译检查：parse / compileScript / compileTemplate（语法、模板表达式）
//  2) Vue API 导入完整性检查 —— 本项目 vite.config.ts 里的
//     `AutoImport({ resolvers: [ElementPlusResolver()] })` **只自动导入 Element Plus 的 API**，
//     并没有配 `imports: ['vue']`，所以所有 Vue API（ref/onMounted/onUnmounted…）都必须手写 import。
//     漏掉一个不会编译报错，只会在浏览器里抛
//     `Uncaught (in promise) ReferenceError: xxx is not defined`，
//     表现是「整个页面内容区空白」（setup 抛错 → 组件树挂载失败），极难排查。
//     compiler-sfc 不做标识符解析，抓不到这类问题，所以这里单独静态兜底。
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { resolve, dirname, relative, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const { parse, compileScript, compileTemplate } = require('@vue/compiler-sfc');

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dirname, '..');
const srcDir = join(root, 'src');

// 必须显式 import 的 Vue API（不含编译器宏）
const VUE_APIS = [
  'ref', 'shallowRef', 'reactive', 'shallowReactive', 'readonly', 'computed',
  'watch', 'watchEffect', 'watchPostEffect', 'watchSyncEffect',
  'onMounted', 'onUnmounted', 'onBeforeMount', 'onBeforeUnmount',
  'onActivated', 'onDeactivated', 'onUpdated', 'onBeforeUpdate',
  'onErrorCaptured', 'onRenderTracked', 'onRenderTriggered', 'onScopeDispose',
  'nextTick', 'toRef', 'toRefs', 'toValue', 'provide', 'inject',
  'markRaw', 'effectScope', 'getCurrentInstance', 'useSlots', 'useAttrs', 'h',
];

function walkVue(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walkVue(p, out);
    else if (name.endsWith('.vue')) out.push(p);
  }
  return out;
}

let failed = 0;
const files = walkVue(srcDir);
console.log(`扫描 ${files.length} 个 .vue 文件\n`);

for (const abs of files) {
  const rel = relative(root, abs).replace(/\\/g, '/');
  const source = readFileSync(abs, 'utf8');

  // ---- 1) 编译检查 ----
  let descriptor;
  try {
    const parsed = parse(source, { filename: rel });
    if (parsed.errors && parsed.errors.length) {
      console.log(`FAIL  ${rel}  -> parse: ${parsed.errors.map((e) => e.message).join('; ')}`);
      failed++;
      continue;
    }
    descriptor = parsed.descriptor;
    if (descriptor.script || descriptor.scriptSetup) compileScript(descriptor, { id: rel });
    if (descriptor.template) {
      const r = compileTemplate({ source: descriptor.template.content, filename: rel, id: rel });
      if (r.errors && r.errors.length) {
        console.log(`FAIL  ${rel}  -> template: ${r.errors.map((e) => (typeof e === 'string' ? e : e.message)).join('; ')}`);
        failed++;
        continue;
      }
    }
  } catch (e) {
    console.log(`FAIL  ${rel}  -> ${e.message}`);
    failed++;
    continue;
  }

  // ---- 2) Vue API 导入完整性 ----
  const script = [descriptor.script?.content, descriptor.scriptSetup?.content].filter(Boolean).join('\n');
  if (!script) {
    console.log(`OK    ${rel}`);
    continue;
  }
  const imported = new Set();
  const importRe = /import\s*\{([^}]+)\}\s*from\s*['"]vue['"]/g;
  let m;
  while ((m = importRe.exec(script))) {
    m[1].split(',').forEach((s) => {
      const name = s.trim().split(/\s+as\s+/).pop().trim();
      if (name) imported.add(name);
    });
  }
  // 先剥注释再判断，避免注释里的名字误报
  const code = script.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '');
  const missing = VUE_APIS.filter((api) => {
    if (imported.has(api)) return false;
    // `api(` / `api<...>(`，且不是 obj.api( 这类成员访问
    return new RegExp(`(^|[^\\w.$])${api}\\s*[<(]`, 'm').test(code);
  });

  if (missing.length) {
    console.log(`FAIL  ${rel}  -> 使用了但未从 'vue' 导入: ${missing.join(', ')}`);
    console.log(`        修复：在 import { ... } from 'vue' 补上（本项目未开启 vue API 自动导入）`);
    failed++;
  } else {
    console.log(`OK    ${rel}`);
  }
}

console.log('\n---');
console.log(failed === 0 ? 'ALL CHECKS PASSED' : `${failed} file(s) FAILED`);
process.exit(failed === 0 ? 0 : 1);
