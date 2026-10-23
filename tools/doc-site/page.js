(() => {
  // ---------- 图表全屏查看 ----------
  const box = document.querySelector('.lightbox');
  const stage = box.querySelector('.lb-stage');
  const title = box.querySelector('#lb-title');
  const buttons = [...box.querySelectorAll('[data-zoom]')];
  let svg = null;
  let natural = 0;
  let mode = 'fit';
  let opener = null;

  function naturalWidth(el) {
    const vb = el.viewBox && el.viewBox.baseVal;
    if (vb && vb.width) return vb.width;
    return el.getBoundingClientRect().width || 1200;
  }

  function apply() {
    if (!svg) return;
    const avail = stage.clientWidth - 32;
    const width = mode === 'fit' ? Math.max(avail, 320) : natural * Number(mode);
    svg.style.width = `${Math.round(width)}px`;
    buttons.forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.zoom === mode)));
  }

  function open(figure) {
    const source = figure.querySelector('.plate-scroll > svg');
    if (!source) return;
    opener = document.activeElement;
    svg = source.cloneNode(true);
    svg.removeAttribute('style');
    svg.removeAttribute('width');
    svg.removeAttribute('height');
    natural = naturalWidth(source);
    stage.replaceChildren(svg);
    title.textContent = figure.dataset.title || '';
    // 宽图默认 100% 原始尺寸，保证文字为设计字号；窄屏先适应宽度
    mode = stage.clientWidth >= natural * 0.9 || window.innerWidth < 720 ? 'fit' : '1';
    box.hidden = false;
    document.body.style.overflow = 'hidden';
    apply();
    stage.scrollTo(0, 0);
    box.querySelector('.lb-close').focus();
  }

  function close() {
    box.hidden = true;
    stage.replaceChildren();
    svg = null;
    document.body.style.overflow = '';
    if (opener) opener.focus();
  }

  document.querySelectorAll('.plate').forEach((figure) => {
    const scroll = figure.querySelector('.plate-scroll');
    scroll.addEventListener('click', () => open(figure));
    scroll.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(figure); }
    });
    figure.querySelector('.zoom-btn').addEventListener('click', () => open(figure));
  });
  buttons.forEach((b) => b.addEventListener('click', () => { mode = b.dataset.zoom; apply(); }));
  box.querySelector('.lb-close').addEventListener('click', close);
  box.addEventListener('click', (e) => { if (e.target === stage) close(); });
  document.addEventListener('keydown', (e) => {
    if (box.hidden) return;
    if (e.key === 'Escape') close();
    const steps = ['fit', '1', '1.5', '2', '3'];
    const i = steps.indexOf(mode);
    if (e.key === '+' || e.key === '=') { mode = steps[Math.min(i + 1, steps.length - 1)]; apply(); }
    if (e.key === '-') { mode = steps[Math.max(i - 1, 0)]; apply(); }
  });
  window.addEventListener('resize', () => { if (mode === 'fit') apply(); });

  // ---------- 收起 / 展开目录（记住个人偏好） ----------
  const shell = document.querySelector('.shell');
  const showBtn = document.querySelector('.toc-open');
  function setCollapsed(collapsed) {
    shell.classList.toggle('toc-collapsed', collapsed);
    showBtn.hidden = !collapsed;
    try { localStorage.setItem('acme-doc-toc', collapsed ? 'hidden' : 'shown'); } catch (e) { /* 忽略 */ }
  }
  document.querySelectorAll('[data-toc]').forEach((b) => b.addEventListener('click', () => setCollapsed(b.dataset.toc === 'hide')));
  try { if (localStorage.getItem('acme-doc-toc') === 'hidden') setCollapsed(true); } catch (e) { /* 忽略 */ }

  // ---------- 目录高亮当前章节 ----------
  const links = new Map();
  document.querySelectorAll('.sidebar a[href^="#"]').forEach((a) => links.set(a.getAttribute('href').slice(1), a));
  const targets = [...document.querySelectorAll('.chapter-head, .chapter h2')].filter((h) => links.has(h.id));
  if ('IntersectionObserver' in window && targets.length) {
    let current = null;
    const io = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          if (current) current.removeAttribute('aria-current');
          current = links.get(entry.target.id);
          current.setAttribute('aria-current', 'true');
          const r = current.getBoundingClientRect();
          if (r.top < 80 || r.bottom > window.innerHeight - 40) current.scrollIntoView({ block: 'nearest' });
        }
      });
    }, { rootMargin: '0px 0px -75% 0px' });
    targets.forEach((t) => io.observe(t));
  }
})();
