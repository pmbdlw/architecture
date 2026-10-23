// 生成 HTML 版文档：node build.mjs
//   输出 html/index.html（完整、可离线打开的单文件）。
//   --fragment <path> 另外输出不含 <html>/<head>/<body> 外壳的版本（用于发布为 claude.ai Artifact）。
// 处理内容：Markdown → HTML、内联 SVG 架构图、Chromium 预渲染 Mermaid、highlight.js 预高亮代码。
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Marked } from 'marked';
import hljs from 'highlight.js';
import { chromium } from 'playwright';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '../..');
const docsDir = path.join(root, 'docs');
const outFile = path.join(root, 'html', 'index.html');
// 文档中指向工程代码的链接统一转到 GitHub
const REPO_URL = process.env.REPO_URL ?? 'https://github.com/pmbdlw/architecture';
const REPO_REF = process.env.REPO_REF ?? 'claude/brave-bohr-ncvazk';

const CHAPTERS = [
  '01-runtime-architecture.md',
  '02-spring-cloud-in-practice.md',
  '03-team-conventions.md',
  '04-code-structure.md',
];

const FONT_STACK = "'Noto Sans SC','PingFang SC','Hiragino Sans GB','Microsoft YaHei','WenQuanYi Zen Hei',sans-serif";

// ---------- GitHub 风格的标题锚点 ----------
function slugger() {
  const seen = new Map();
  return (text) => {
    let slug = text.toLowerCase().trim()
      .replace(/<[^>]+>/g, '')
      .replace(/[^\p{L}\p{M}\p{N}\p{Pc} -]/gu, '')
      .replace(/ /g, '-');
    const count = seen.get(slug) ?? 0;
    seen.set(slug, count + 1);
    return count ? `${slug}-${count}` : slug;
  };
}

const escapeHtml = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

// ---------- Markdown 渲染 ----------
const mermaidBlocks = [];
const toc = [];
const slug = slugger();
let chapterNo = 0;
let figureNo = 0;

function figure(inner, caption, kind) {
  figureNo += 1;
  const label = `图 ${chapterNo}-${figureNo}`;
  return `<figure class="plate plate-${kind}" data-title="${escapeHtml(label + '  ' + caption)}">`
    + `<div class="plate-scroll" tabindex="0" role="button" aria-label="全屏查看：${escapeHtml(caption)}">${inner}</div>`
    + `<figcaption><span class="fig-no">${label}</span><span class="fig-text">${escapeHtml(caption)}</span>`
    + `<button type="button" class="zoom-btn">全屏查看</button></figcaption></figure>`;
}

function resolveLink(href) {
  if (/^https?:/.test(href) || href.startsWith('#')) return href;
  const [file, anchor] = href.split('#');
  if (/^0\d-.*\.md$/.test(path.basename(file))) {
    return anchor ? `#${anchor}` : `#ch${Number(path.basename(file).slice(0, 2))}`;
  }
  if (file.startsWith('../')) {
    const target = path.normalize(file.slice(3)).replace(/\\/g, '/');
    const isDir = !path.extname(target);
    return `${REPO_URL}/${isDir ? 'tree' : 'blob'}/${REPO_REF}/${target}${anchor ? '#' + anchor : ''}`;
  }
  return href;
}

const marked = new Marked({
  gfm: true,
  renderer: {
    heading({ tokens, depth, text }) {
      const html = this.parser.parseInline(tokens);
      if (depth === 1) {
        const [, num, title] = text.match(/^第\s*(\d+)\s*章\s*(.*)$/) ?? [null, String(chapterNo), text];
        toc.push({ depth, id: `ch${chapterNo}`, text: `${num}. ${title}` });
        return `<header class="chapter-head" id="ch${chapterNo}"><p class="chapter-no">第 ${num} 章</p>`
          + `<h1>${escapeHtml(title)}</h1></header>\n`;
      }
      const id = slug(text);
      if (depth === 2) toc.push({ depth, id, text });
      return `<h${depth} id="${id}"><a class="anchor" href="#${id}" aria-hidden="true">#</a>${html}</h${depth}>\n`;
    },
    code({ text, lang }) {
      if (lang === 'mermaid') {
        mermaidBlocks.push(text);
        const n = mermaidBlocks.length - 1;
        return figure(`<!--MERMAID:${n}-->`, mermaidCaption(text), 'mermaid');
      }
      const language = lang && hljs.getLanguage(lang) ? lang : null;
      const body = language ? hljs.highlight(text, { language }).value : escapeHtml(text);
      const label = lang && lang !== 'text' ? `<span class="code-lang">${escapeHtml(lang)}</span>` : '';
      return `<div class="code">${label}<pre><code>${body}</code></pre></div>\n`;
    },
    image({ href, text }) {
      if (href.endsWith('.svg')) {
        const svg = fs.readFileSync(path.join(docsDir, href), 'utf8');
        return figure(svg, text, 'svg');
      }
      return `<img src="${href}" alt="${escapeHtml(text)}">`;
    },
    link({ href, tokens }) {
      const target = resolveLink(href);
      const external = /^https?:/.test(target);
      return `<a href="${escapeHtml(target)}"${external ? ' target="_blank" rel="noopener"' : ''}>`
        + `${this.parser.parseInline(tokens)}</a>`;
    },
  },
});

// Mermaid 图没有 alt 文本：按图的首个节点 / 参与者生成说明
function mermaidCaption(code) {
  const kind = code.trim().split(/\s|\n/)[0];
  const names = { flowchart: '流程图', sequenceDiagram: '时序图', gitGraph: '分支模型' };
  const prev = currentHeading;
  return `${prev}（${names[kind] ?? '示意图'}）`;
}
let currentHeading = '';

function renderChapter(file) {
  chapterNo += 1;
  figureNo = 0;
  const md = fs.readFileSync(path.join(docsDir, file), 'utf8');
  // 记录每个代码块之前最近的标题，用作 Mermaid 图的说明
  const tokens = marked.lexer(md);
  let html = '';
  for (const token of tokens) {
    if (token.type === 'heading') currentHeading = token.text.replace(/^[\d.]+\s*/, '');
    html += marked.parser([token]);
  }
  html = html
    .replace(/<p>(<figure[\s\S]*?<\/figure>)<\/p>/g, '$1')
    .replace(/<table>/g, '<div class="table-wrap"><table>')
    .replace(/<\/table>/g, '</table></div>')
    .replace(/<blockquote>/g, '<blockquote class="callout">');
  return `<section class="chapter" aria-labelledby="ch${chapterNo}">${html}</section>`;
}

// ---------- Mermaid 预渲染 ----------
async function renderMermaid(blocks) {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 1000 } });
  await page.setContent(`<html><body style="font-family:${FONT_STACK}"></body></html>`);
  await page.addScriptTag({ path: path.join(here, 'node_modules/mermaid/dist/mermaid.min.js') });
  const svgs = await page.evaluate(async ({ blocks, font }) => {
    window.mermaid.initialize({
      startOnLoad: false,
      securityLevel: 'strict',
      theme: 'base',
      fontFamily: font,
      themeVariables: {
        fontFamily: font,
        fontSize: '15px',
        primaryColor: '#eef7ea',
        primaryBorderColor: '#4d8f3a',
        primaryTextColor: '#1b2a1c',
        secondaryColor: '#eef3fb',
        tertiaryColor: '#f6f7f9',
        lineColor: '#4b5563',
        textColor: '#1f2937',
        noteBkgColor: '#fff8e1',
        noteBorderColor: '#d9a400',
        actorBkg: '#eef7ea',
        actorBorder: '#4d8f3a',
        signalColor: '#374151',
        signalTextColor: '#111827',
        git0: '#4d8f3a', git1: '#2563eb', git2: '#d97706', git3: '#db2777',
      },
      flowchart: { htmlLabels: true, curve: 'basis', padding: 14 },
      sequence: { mirrorActors: false, actorMargin: 60, messageFontSize: 15, noteFontSize: 14 },
    });
    const out = [];
    for (let i = 0; i < blocks.length; i++) {
      const { svg } = await window.mermaid.render(`mmd-${i}`, blocks[i]);
      out.push(svg);
    }
    return out;
  }, { blocks, font: FONT_STACK });
  await browser.close();
  return svgs;
}

// ---------- 页面模板 ----------
function tocHtml() {
  let html = '<ol class="toc-list">';
  let open = false;
  for (const item of toc) {
    if (item.depth === 1) {
      if (open) html += '</ol></li>';
      html += `<li class="toc-ch"><a href="#${item.id}">${escapeHtml(item.text)}</a><ol>`;
      open = true;
    } else {
      html += `<li><a href="#${item.id}">${escapeHtml(item.text)}</a></li>`;
    }
  }
  if (open) html += '</ol></li>';
  return html + '</ol>';
}

const coverHtml = `
<header class="cover" id="top">
  <p class="eyebrow">架构说明 · acme-cloud · 2026 年 10 月</p>
  <h1 class="cover-title">Spring Cloud 中大型微服务系统架构说明</h1>
  <p class="lead">运行在 Kubernetes 上的微服务参考架构：数据库 MySQL / PostgreSQL 二选一，消息 Kafka / RabbitMQ 二选一，
  覆盖测试、预发布、生产三套环境，给出全自建与阿里云两种落地方案，并附可直接编译运行的初始工程。</p>
  <ul class="stack" aria-label="版本基线">
    <li><span>Java</span>21</li>
    <li><span>Spring Boot</span>4.0.9</li>
    <li><span>Spring Cloud</span>2025.1.4</li>
    <li><span>Spring Cloud Alibaba</span>2025.1.0.0</li>
    <li><span>Nacos</span>3.1</li>
    <li><span>K8s</span>1.32+</li>
  </ul>
  <ol class="chapter-cards">
    <li><a href="#ch1"><b>运行架构</b><span>环境规划 · 全自建 · 阿里云 · AWS / Azure · 发布与灰度</span></a></li>
    <li><a href="#ch2"><b>Spring Cloud 技术落地</b><span>Nacos · Gateway · Feign · 熔断 · Stream · 认证 · 可观测</span></a></li>
    <li><a href="#ch3"><b>团队开发规范</b><span>命名 · 分层 · API · 数据库 · 消息 · 安全 · 测试 · 评审</span></a></li>
    <li><a href="#ch4"><b>初始代码结构</b><span>模块全景 · 目录树 · 关键代码 · 本地开发 · 新增服务</span></a></li>
  </ol>
  <p class="hint">图表均为矢量图：点击任意图表或“全屏查看”可放大至 100%~300% 阅读。</p>
</header>`;

const css = fs.readFileSync(path.join(here, 'page.css'), 'utf8');
const js = fs.readFileSync(path.join(here, 'page.js'), 'utf8');

function pageBody(content) {
  return `<a class="skip" href="#content">跳到正文</a>
<div class="shell">
  <aside class="sidebar" aria-label="目录">
    <a class="brand" href="#top">acme-cloud<small>架构说明</small></a>
    <button type="button" class="toc-toggle" data-toc="hide">收起目录 · 图表更大</button>
    <nav>${tocHtml()}</nav>
  </aside>
  <main id="content">
    ${coverHtml}
    <details class="toc-mobile"><summary>目录</summary><nav>${tocHtml()}</nav></details>
    ${content}
    <footer class="page-foot">
      <p>源文件：<a href="${REPO_URL}/tree/${REPO_REF}/docs" target="_blank" rel="noopener">docs/*.md</a> ·
      初始工程：<a href="${REPO_URL}/tree/${REPO_REF}/acme-cloud" target="_blank" rel="noopener">acme-cloud/</a> ·
      本页由 tools/doc-site 生成</p>
    </footer>
  </main>
</div>
<button type="button" class="toc-toggle toc-open" data-toc="show" hidden>☰ 显示目录</button>
<div class="lightbox" hidden role="dialog" aria-modal="true" aria-labelledby="lb-title">
  <div class="lb-bar">
    <p id="lb-title"></p>
    <div class="lb-tools" role="group" aria-label="缩放">
      <button type="button" data-zoom="fit">适应宽度</button>
      <button type="button" data-zoom="1">100%</button>
      <button type="button" data-zoom="1.5">150%</button>
      <button type="button" data-zoom="2">200%</button>
      <button type="button" data-zoom="3">300%</button>
      <button type="button" class="lb-close" aria-label="关闭">关闭 ✕</button>
    </div>
  </div>
  <div class="lb-stage" tabindex="0"></div>
</div>`;
}

const HEAD = `<title>Spring Cloud 微服务架构说明</title>
<meta name="claude:viewport-fit-sides" content="cover">
<meta name="description" content="Spring Cloud + Kubernetes 中大型微服务系统架构说明：环境规划、全自建与阿里云方案、Spring Cloud 技术落地、团队规范与初始代码结构">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&family=Noto+Sans+SC:wght@400;500;700&family=Noto+Serif+SC:wght@600;800&display=swap">
<style>${css}</style>`;

// ---------- 主流程 ----------
let content = CHAPTERS.map(renderChapter).join('\n');
const svgs = await renderMermaid(mermaidBlocks);
content = content.replace(/<!--MERMAID:(\d+)-->/g, (_, n) => svgs[Number(n)]);
const body = pageBody(content);

fs.mkdirSync(path.dirname(outFile), { recursive: true });
fs.writeFileSync(outFile, `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
${HEAD}
</head>
<body>
${body}
<script>${js}</script>
</body>
</html>
`);
console.log(`generated ${path.relative(root, outFile)} (${(fs.statSync(outFile).size / 1024).toFixed(0)} KB, `
  + `${mermaidBlocks.length} mermaid, ${toc.length} toc entries)`);

const fragIdx = process.argv.indexOf('--fragment');
if (fragIdx > 0) {
  const fragFile = path.resolve(process.argv[fragIdx + 1]);
  fs.writeFileSync(fragFile, `${HEAD}\n${body}\n<script>${js}</script>\n`);
  console.log(`generated fragment ${fragFile}`);
}
