"""极简 SVG 绘图工具：为架构文档生成风格统一的图。

只依赖标准库。所有图都有白色背景，深色模式下也能正常阅读。
"""
from html import escape

FONT = ("'PingFang SC','Hiragino Sans GB','Microsoft YaHei','Noto Sans CJK SC',"
        "'Source Han Sans SC','WenQuanYi Zen Hei',system-ui,sans-serif")
MONO = "'JetBrains Mono','SFMono-Regular',Consolas,'Liberation Mono',monospace"

# 语义配色：(主色, 浅底色, 文字色)
C = {
    "access": ("#2563eb", "#eff6ff", "#1e3a8a"),     # 接入层
    "k8s": ("#0284c7", "#f0f9ff", "#0c4a6e"),        # K8s 集群
    "svc": ("#16a34a", "#f0fdf4", "#14532d"),        # 业务服务
    "cloud": ("#7c3aed", "#f5f3ff", "#4c1d95"),      # Spring Cloud 基础设施（网关/注册配置/认证）
    "data": ("#d97706", "#fffbeb", "#78350f"),       # 数据库
    "mq": ("#ea580c", "#fff7ed", "#7c2d12"),         # 消息 / 缓存
    "obs": ("#db2777", "#fdf2f8", "#831843"),        # 可观测
    "ops": ("#475569", "#f8fafc", "#1e293b"),        # DevOps / 平台
    "sec": ("#dc2626", "#fef2f2", "#7f1d1d"),        # 安全
    "user": ("#0f766e", "#f0fdfa", "#134e4a"),       # 用户端
    "gray": ("#64748b", "#f1f5f9", "#334155"),
    "aliyun": ("#ff6a00", "#fff4ec", "#7a3200"),
    "aws": ("#232f3e", "#fff7e6", "#232f3e"),
    "azure": ("#0078d4", "#eef6fd", "#003f73"),
}


def text_width(s, size):
    """粗略估算文本宽度：CJK 字符约 1em，ASCII 约 0.56em。"""
    w = 0.0
    for ch in s:
        w += size if ord(ch) > 0x2E80 else size * 0.56
    return w


def wrap(s, width, size):
    """按估算宽度折行（优先在空格、“·”“/”“（”处断开）。"""
    lines, cur = [], ""
    for ch in s:
        if text_width(cur + ch, size) > width and cur:
            cut = max(cur.rfind(" "), cur.rfind("·"), cur.rfind("/"), cur.rfind("（"))
            if cut > len(cur) * 0.5:
                lines.append(cur[:cut + (0 if cur[cut] == "（" else 1)].strip())
                cur = cur[cut + (0 if cur[cut] == "（" else 1):].lstrip()
            else:
                lines.append(cur)
                cur = ""
        cur += ch
    if cur:
        lines.append(cur.strip())
    return lines


class Svg:
    def __init__(self, width, height, title=""):
        self.w, self.h = width, height
        self.title = title
        self.parts = []
        self.defs = []
        self._markers = set()

    # ---------- 基础图元 ----------
    def raw(self, s):
        self.parts.append(s)

    def rect(self, x, y, w, h, fill="#fff", stroke="none", sw=1.5, rx=8, dash=None, opacity=None):
        d = f' stroke-dasharray="{dash}"' if dash else ""
        o = f' opacity="{opacity}"' if opacity is not None else ""
        self.parts.append(f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{rx}" fill="{fill}" '
                          f'stroke="{stroke}" stroke-width="{sw}"{d}{o}/>')

    def text(self, x, y, s, size=14, weight=400, fill="#1f2937", anchor="start", mono=False, italic=False):
        fam = MONO if mono else FONT
        st = ' font-style="italic"' if italic else ""
        self.parts.append(f'<text x="{x}" y="{y}" font-family="{fam}" font-size="{size}" font-weight="{weight}" '
                          f'fill="{fill}" text-anchor="{anchor}" dominant-baseline="middle"{st}>{escape(s)}</text>')

    def line(self, x1, y1, x2, y2, color="#64748b", sw=1.6, dash=None):
        d = f' stroke-dasharray="{dash}"' if dash else ""
        self.parts.append(f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" stroke-width="{sw}"{d}/>')

    def _marker(self, color):
        mid = "arr" + color.replace("#", "")
        if mid not in self._markers:
            self._markers.add(mid)
            self.defs.append(f'<marker id="{mid}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" '
                             f'markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" '
                             f'fill="{color}"/></marker>')
        return mid

    def arrow(self, pts, color="#475569", sw=1.8, dash=None, both=False, label=None, label_pos=None,
              label_size=12.5, label_color=None):
        """折线箭头。pts=[(x,y),...]；label 显示在 label_pos 或第一段中点。"""
        mid = self._marker(color)
        d = "M" + " L".join(f"{x},{y}" for x, y in pts)
        da = f' stroke-dasharray="{dash}"' if dash else ""
        start = f' marker-start="url(#{mid})"' if both else ""
        self.parts.append(f'<path d="{d}" fill="none" stroke="{color}" stroke-width="{sw}"{da} '
                          f'marker-end="url(#{mid})"{start} stroke-linejoin="round"/>')
        if label:
            if label_pos is None:
                (x1, y1), (x2, y2) = pts[0], pts[1]
                label_pos = ((x1 + x2) / 2, (y1 + y2) / 2)
            lx, ly = label_pos
            tw = text_width(label, label_size) + 10
            self.rect(lx - tw / 2, ly - 10, tw, 20, fill="#ffffff", rx=4, opacity=0.92)
            self.text(lx, ly + 1, label, size=label_size, fill=label_color or color, anchor="middle", weight=500)

    # ---------- 复合元素 ----------
    def group(self, x, y, w, h, label, kind="k8s", dash="6 4", fill=True, label_size=15, sub=None):
        """容器（集群 / 命名空间 / 网络区域），左上角带标签。"""
        main, light, dark = C[kind]
        self.rect(x, y, w, h, fill=light if fill else "none", stroke=main, sw=1.6, rx=12, dash=dash)
        tw = text_width(label, label_size) + 22
        self.rect(x + 12, y - 13, tw, 26, fill=main, rx=13)
        self.text(x + 12 + tw / 2, y, label, size=label_size, weight=600, fill="#fff", anchor="middle")
        if sub:
            sw_ = text_width(sub, 12.5) + 18
            self.rect(x + 12 + tw + 8, y - 11, sw_, 22, fill="#ffffff", stroke=main, sw=1, rx=11)
            self.text(x + 12 + tw + 8 + sw_ / 2, y + 0.5, sub, size=12.5, fill=dark, anchor="middle")

    def box(self, x, y, w, h, title, lines=(), kind="svc", icon=None, title_size=15, line_size=12.5,
            align="center", badge=None):
        """组件卡片：白底、彩色边框与左侧色条、标题 + 若干说明行。"""
        main, light, dark = C[kind]
        self.rect(x, y, w, h, fill="#ffffff", stroke=main, sw=1.5, rx=8)
        self.rect(x, y, 5, h, fill=main, rx=2)
        n = len(lines)
        block = title_size + 6 + n * (line_size + 5)
        top = y + (h - block) / 2 + title_size / 2 + 1
        cx = x + w / 2 + (8 if icon else 2)
        tx = cx if align == "center" else x + 16
        anchor = "middle" if align == "center" else "start"
        if icon:
            self.icon(icon, x + 22, y + h / 2, 20, main)
            tx = x + 40 + (w - 40) / 2 if align == "center" else x + 40
        self.text(tx, top, title, size=title_size, weight=600, fill=dark, anchor=anchor)
        for i, ln in enumerate(lines):
            self.text(tx, top + title_size / 2 + 6 + (i + 0.5) * (line_size + 5), ln, size=line_size,
                      fill="#475569", anchor=anchor)
        if badge:
            bw = text_width(badge, 11) + 12
            self.rect(x + w - bw - 6, y + 6, bw, 18, fill=light, stroke=main, sw=1, rx=9)
            self.text(x + w - bw / 2 - 6, y + 15.5, badge, size=11, fill=dark, anchor="middle", weight=600)

    def chip(self, x, y, label, kind="cloud", size=12, h=22, solid=False):
        main, light, dark = C[kind]
        w = text_width(label, size) + 16
        self.rect(x, y, w, h, fill=main if solid else light, stroke=main, sw=1, rx=h / 2)
        self.text(x + w / 2, y + h / 2 + 0.5, label, size=size, fill="#fff" if solid else dark, anchor="middle",
                  weight=600)
        return w

    def note(self, x, y, s, size=12.5, fill="#64748b", anchor="start", italic=False):
        self.text(x, y, s, size=size, fill=fill, anchor=anchor, italic=italic)

    # ---------- 图标（通用几何图形，非任何厂商商标） ----------
    def icon(self, kind, cx, cy, s, color):
        r = s / 2
        p = self.parts
        if kind == "db":
            p.append(f'<g fill="none" stroke="{color}" stroke-width="1.8">'
                     f'<ellipse cx="{cx}" cy="{cy - r * 0.6}" rx="{r * 0.8}" ry="{r * 0.3}"/>'
                     f'<path d="M{cx - r * 0.8},{cy - r * 0.6} v{r * 1.2} a{r * 0.8},{r * 0.3} 0 0 0 {r * 1.6},0 '
                     f'v{-r * 1.2}"/><path d="M{cx - r * 0.8},{cy} a{r * 0.8},{r * 0.3} 0 0 0 {r * 1.6},0"/></g>')
        elif kind == "mq":
            for i in range(3):
                p.append(f'<rect x="{cx - r * 0.85 + i * r * 0.6}" y="{cy - r * 0.45}" width="{r * 0.45}" '
                         f'height="{r * 0.9}" rx="2" fill="{color}" opacity="{0.45 + i * 0.25}"/>')
        elif kind == "k8s":
            import math
            pts = " ".join(f"{cx + r * 0.95 * math.cos(math.radians(-90 + i * 360 / 7)):.1f},"
                           f"{cy + r * 0.95 * math.sin(math.radians(-90 + i * 360 / 7)):.1f}" for i in range(7))
            p.append(f'<polygon points="{pts}" fill="{color}"/>')
            p.append(f'<circle cx="{cx}" cy="{cy}" r="{r * 0.38}" fill="none" stroke="#fff" stroke-width="1.6"/>')
            for i in range(7):
                a = math.radians(-90 + i * 360 / 7)
                p.append(f'<line x1="{cx}" y1="{cy}" x2="{cx + r * 0.62 * math.cos(a):.1f}" '
                         f'y2="{cy + r * 0.62 * math.sin(a):.1f}" stroke="#fff" stroke-width="1.3"/>')
        elif kind == "lb":
            p.append(f'<circle cx="{cx}" cy="{cy}" r="{r * 0.85}" fill="none" stroke="{color}" stroke-width="1.8"/>')
            p.append(f'<path d="M{cx - r * 0.5},{cy} h{r} M{cx + r * 0.2},{cy - r * 0.3} l{r * 0.3},{r * 0.3} '
                     f'l{-r * 0.3},{r * 0.3}" stroke="{color}" stroke-width="1.8" fill="none"/>')
        elif kind == "shield":
            p.append(f'<path d="M{cx},{cy - r * 0.9} l{r * 0.75},{r * 0.3} v{r * 0.45} c0,{r * 0.5} '
                     f'{-r * 0.35},{r * 0.8} {-r * 0.75},{r * 1.05} c{-r * 0.4},{-r * 0.25} {-r * 0.75},{-r * 0.55} '
                     f'{-r * 0.75},{-r * 1.05} v{-r * 0.45} z" fill="{color}" opacity="0.85"/>')
        elif kind == "globe":
            p.append(f'<g fill="none" stroke="{color}" stroke-width="1.6"><circle cx="{cx}" cy="{cy}" r="{r * 0.85}"/>'
                     f'<ellipse cx="{cx}" cy="{cy}" rx="{r * 0.38}" ry="{r * 0.85}"/>'
                     f'<line x1="{cx - r * 0.85}" y1="{cy}" x2="{cx + r * 0.85}" y2="{cy}"/></g>')
        elif kind == "lock":
            p.append(f'<g><rect x="{cx - r * 0.65}" y="{cy - r * 0.1}" width="{r * 1.3}" height="{r * 0.95}" rx="2" '
                     f'fill="{color}"/><path d="M{cx - r * 0.4},{cy - r * 0.1} v{-r * 0.3} a{r * 0.4},{r * 0.4} 0 0 1 '
                     f'{r * 0.8},0 v{r * 0.3}" fill="none" stroke="{color}" stroke-width="1.8"/></g>')
        elif kind == "chart":
            p.append(f'<g fill="{color}"><rect x="{cx - r * 0.8}" y="{cy}" width="{r * 0.35}" height="{r * 0.8}"/>'
                     f'<rect x="{cx - r * 0.2}" y="{cy - r * 0.45}" width="{r * 0.35}" height="{r * 1.25}"/>'
                     f'<rect x="{cx + r * 0.4}" y="{cy - r * 0.8}" width="{r * 0.35}" height="{r * 1.6}"/></g>')
        elif kind == "gear":
            p.append(f'<circle cx="{cx}" cy="{cy}" r="{r * 0.62}" fill="none" stroke="{color}" stroke-width="3.2" '
                     f'stroke-dasharray="2.6 2.2"/><circle cx="{cx}" cy="{cy}" r="{r * 0.3}" fill="{color}"/>')
        elif kind == "box":
            p.append(f'<g fill="none" stroke="{color}" stroke-width="1.7"><path d="M{cx},{cy - r * 0.85} '
                     f'l{r * 0.8},{r * 0.4} v{r * 0.9} l{-r * 0.8},{r * 0.4} l{-r * 0.8},{-r * 0.4} v{-r * 0.9} z"/>'
                     f'<path d="M{cx - r * 0.8},{cy - r * 0.45} l{r * 0.8},{r * 0.4} l{r * 0.8},{-r * 0.4} '
                     f'M{cx},{cy - r * 0.05} v{r * 1.3}"/></g>')
        elif kind == "user":
            p.append(f'<g fill="{color}"><circle cx="{cx}" cy="{cy - r * 0.4}" r="{r * 0.35}"/>'
                     f'<path d="M{cx - r * 0.75},{cy + r * 0.8} a{r * 0.75},{r * 0.7} 0 0 1 {r * 1.5},0 z"/></g>')
        elif kind == "git":
            p.append(f'<g fill="none" stroke="{color}" stroke-width="1.8"><circle cx="{cx - r * 0.4}" '
                     f'cy="{cy + r * 0.5}" r="{r * 0.22}"/><circle cx="{cx - r * 0.4}" cy="{cy - r * 0.55}" '
                     f'r="{r * 0.22}"/><circle cx="{cx + r * 0.45}" cy="{cy - r * 0.1}" r="{r * 0.22}"/>'
                     f'<path d="M{cx - r * 0.4},{cy - r * 0.33} v{r * 0.62} M{cx - r * 0.4},{cy + r * 0.2} '
                     f'q0,{-r * 0.3} {r * 0.65},{-r * 0.3}"/></g>')
        elif kind == "doc":
            p.append(f'<g fill="none" stroke="{color}" stroke-width="1.7"><path d="M{cx - r * 0.6},{cy - r * 0.85} '
                     f'h{r * 0.85} l{r * 0.35},{r * 0.35} v{r * 1.35} h{-r * 1.2} z"/>'
                     f'<path d="M{cx - r * 0.35},{cy - r * 0.15} h{r * 0.7} M{cx - r * 0.35},{cy + r * 0.25} '
                     f'h{r * 0.7}"/></g>')
        elif kind == "cloud":
            p.append(f'<path d="M{cx - r * 0.55},{cy + r * 0.45} a{r * 0.4},{r * 0.4} 0 0 1 0,{-r * 0.8} '
                     f'a{r * 0.55},{r * 0.55} 0 0 1 {r * 1.0},{-r * 0.15} a{r * 0.4},{r * 0.4} 0 0 1 {r * 0.1},'
                     f'{r * 0.95} z" fill="{color}"/>')
        elif kind == "bolt":
            p.append(f'<path d="M{cx + r * 0.15},{cy - r * 0.9} l{-r * 0.65},{r * 1.0} h{r * 0.5} l{-r * 0.2},'
                     f'{r * 0.8} l{r * 0.7},{-r * 1.05} h{-r * 0.5} z" fill="{color}"/>')
        elif kind == "compass":
            p.append(f'<circle cx="{cx}" cy="{cy}" r="{r * 0.85}" fill="none" stroke="{color}" stroke-width="1.8"/>'
                     f'<path d="M{cx},{cy - r * 0.6} l{r * 0.22},{r * 0.6} l{-r * 0.22},{r * 0.6} '
                     f'l{-r * 0.22},{-r * 0.6} z" fill="{color}"/>')
        elif kind == "fn":
            p.append(f'<text x="{cx}" y="{cy + 1}" font-family="{MONO}" font-size="{s * 0.8}" font-weight="700" '
                     f'fill="{color}" text-anchor="middle" dominant-baseline="middle">{{ }}</text>')

    # ---------- 输出 ----------
    def title_block(self, title, subtitle=None):
        self.text(28, 34, title, size=22, weight=700, fill="#0f172a")
        if subtitle:
            self.text(28, 62, subtitle, size=13.5, fill="#475569")

    def legend(self, x, y, items, size=12.5):
        cx = x
        for kind, label in items:
            main, light, _ = C[kind]
            self.rect(cx, y - 7, 14, 14, fill=light, stroke=main, sw=1.5, rx=3)
            self.text(cx + 20, y + 0.5, label, size=size, fill="#334155")
            cx += 20 + text_width(label, size) + 22

    def render(self):
        head = (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {self.w} {self.h}" width="{self.w}" '
                f'height="{self.h}" role="img" aria-label="{escape(self.title)}">')
        title = f"<title>{escape(self.title)}</title>" if self.title else ""
        defs = "<defs>" + "".join(self.defs) + "</defs>"
        bg = f'<rect width="{self.w}" height="{self.h}" fill="#ffffff"/>'
        return head + title + defs + bg + "\n" + "\n".join(self.parts) + "\n</svg>\n"

    def save(self, path):
        with open(path, "w", encoding="utf-8") as f:
            f.write(self.render())
