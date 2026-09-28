export const DEFAULT_CUSTOM_CSS = String.raw`
:root {
  /* 基础尺寸 */
  --radius-base: 0.625rem;
  /* 页面背景 */
  --page-background-color: #f5f0e8;
  --page-background-image: url(/fabric-texture.svg), repeating-linear-gradient(0deg, transparent, transparent 3px, var(--color-fabric-grain) 3px, var(--color-fabric-grain) 4px), repeating-linear-gradient(90deg, transparent, transparent 3px, var(--color-fabric-grain) 3px, var(--color-fabric-grain) 4px);
  --page-background-size: 100px 100px, 4px 4px, 4px 4px;
  --page-background-attachment: fixed;
  /* 页面文字与表面 */
  --color-page-foreground: #3d352b;
  --color-surface: #faf7f2;
  --color-surface-foreground: #3d352b;
  --color-overlay: #faf7f2;
  --color-overlay-foreground: #3d352b;
  /* 品牌与控件 */
  --color-brand: #8b6f47;
  --color-brand-foreground: #faf7f2;
  --color-nav-foreground: var(--color-brand-foreground);
  --color-control-secondary: #ece6db;
  --color-control-secondary-foreground: #5a4e3f;
  --color-surface-muted: #ece6db;
  --color-text-muted: #8c7e6e;
  --color-control-hover: #e8dfd2;
  --color-control-hover-foreground: #5a4e3f;
  --color-danger: #b85450;
  --color-control-border: #d6ccbd;
  --color-input-border: #d6ccbd;
  --color-focus-ring: #a08b72;
  /* 图表颜色 */
  --chart-1: oklch(0.646 0.222 41.116);
  --chart-2: oklch(0.6 0.118 184.704);
  --chart-3: oklch(0.398 0.07 227.392);
  --chart-4: oklch(0.828 0.189 84.429);
  --chart-5: oklch(0.769 0.188 70.08);
  /* 兼容组件的侧栏颜色 */
  --color-sidebar: #faf7f2;
  --color-sidebar-foreground: #3d352b;
  --color-sidebar-primary: #8b6f47;
  --color-sidebar-primary-foreground: #faf7f2;
  --color-sidebar-accent: #ece6db;
  --color-sidebar-accent-foreground: #5a4e3f;
  --color-sidebar-border: #d6ccbd;
  --color-sidebar-ring: #a08b72;
  /* Fabric 主题细节 */
  --color-fabric-stitch: #b8a68e;
  --color-fabric-shadow: rgba(139, 111, 71, 0.08);
  --color-fabric-grain: #e8e0d4;
}

.fabric-linen {
  background-color: var(--page-background-color);
  background-image: var(--page-background-image);
  background-size: var(--page-background-size);
  background-attachment: var(--page-background-attachment);
}

.fabric-stitch {
  border: 2px dashed var(--color-fabric-stitch);
  box-shadow: 0 0 0 4px var(--color-surface), 0 0 0 5px var(--color-fabric-stitch);
}

.fabric-shadow {
  box-shadow: 0 1px 3px var(--color-fabric-shadow), 0 4px 12px var(--color-fabric-shadow);
}

.fabric-shadow-hover {
  transition: box-shadow 220ms ease, transform 220ms ease;
}

.fabric-shadow-hover:hover {
  box-shadow: 0 2px 6px var(--color-fabric-shadow), 0 8px 24px rgba(139, 111, 71, 0.12);
  transform: translateY(-1px);
}

.fabric-label {
  background: var(--color-brand);
  color: var(--color-brand-foreground);
  border: 1.5px dashed rgba(255, 255, 255, 0.35);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.15), 0 2px 6px var(--color-fabric-shadow);
  transition: box-shadow 220ms ease, transform 200ms ease;
}

.fabric-label:hover {
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.2), 0 3px 10px rgba(139, 111, 71, 0.18);
}

.fabric-label:active {
  transform: translateY(1px);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.1), 0 1px 3px var(--color-fabric-shadow);
}

.fabric-card {
  background-color: var(--color-surface);
  background-image:
    repeating-linear-gradient(0deg, transparent, transparent 5px, rgba(139, 111, 71, 0.03) 5px, rgba(139, 111, 71, 0.03) 6px),
    repeating-linear-gradient(90deg, transparent, transparent 5px, rgba(139, 111, 71, 0.03) 5px, rgba(139, 111, 71, 0.03) 6px);
}

.fabric-pill {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: 9999px;
  background: var(--color-control-secondary);
  border: 1.5px dashed var(--color-fabric-stitch);
  color: var(--color-brand);
  font-size: 0.75rem;
  font-weight: 500;
  transition: background 220ms ease, box-shadow 220ms ease;
}

.fabric-pill:hover {
  background: var(--color-control-hover);
  box-shadow: 0 2px 8px var(--color-fabric-shadow);
}

.fabric-nav {
  background-color: var(--color-brand);
  background-image:
    repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0, 0, 0, 0.06) 2px, rgba(0, 0, 0, 0.06) 3px),
    repeating-linear-gradient(90deg, transparent, transparent 2px, rgba(0, 0, 0, 0.06) 2px, rgba(0, 0, 0, 0.06) 3px);
  box-shadow: 0 2px 8px rgba(139, 111, 71, 0.15);
}

.fabric-input {
  background: var(--color-surface);
  border: 1.5px dashed var(--color-fabric-stitch);
  transition: border-color 220ms ease, box-shadow 220ms ease;
}

.fabric-input:focus {
  border-color: var(--color-brand);
  box-shadow: 0 0 0 3px rgba(139, 111, 71, 0.1);
  outline: none;
}

.fabric-form {
  background-color: var(--color-surface);
  background-image:
    repeating-linear-gradient(45deg, transparent, transparent 10px, rgba(139, 111, 71, 0.02) 10px, rgba(139, 111, 71, 0.02) 11px),
    repeating-linear-gradient(-45deg, transparent, transparent 10px, rgba(139, 111, 71, 0.02) 10px, rgba(139, 111, 71, 0.02) 11px);
  border: 2px dashed var(--color-fabric-stitch);
  box-shadow: 0 0 0 5px var(--color-surface), 0 0 0 6px var(--color-fabric-stitch), 0 4px 16px var(--color-fabric-shadow);
}

* {
  scrollbar-width: thin;
  scrollbar-color: var(--color-fabric-stitch) var(--color-control-secondary);
}

*::-webkit-scrollbar {
  width: 12px;
  height: 12px;
}

*::-webkit-scrollbar-track {
  background: var(--color-control-secondary);
  border-left: 1px dashed var(--color-control-border);
}

*::-webkit-scrollbar-thumb {
  background: var(--color-fabric-stitch);
  border: 2px solid var(--color-control-secondary);
  border-radius: 6px;
}

*::-webkit-scrollbar-thumb:hover {
  background: var(--color-brand);
}

.content-markup-link {
  color: #00a1d6;
  text-decoration: underline;
}

.content-markup-emoji {
  display: inline-block;
  width: 64px;
  height: 64px;
  margin: 0 2px;
  object-fit: contain;
  vertical-align: middle;
}

.content-effect-hide {
  color: #5a4e3f;
  background: #5a4e3f;
  border-radius: 2px;
  box-decoration-break: clone;
  cursor: pointer;
  transition: color 260ms ease, background-color 260ms ease;
  -webkit-box-decoration-break: clone;
}

.content-effect-hide:not([data-revealed="true"]) * {
  color: inherit !important;
  opacity: 0;
}

.content-effect-hide:hover,
.content-effect-hide[data-revealed="true"] {
  color: #faf7f2;
}

.content-effect-hide:hover *,
.content-effect-hide[data-revealed="true"] * {
  color: inherit !important;
  opacity: 1;
}

.content-effect-hide:focus-visible {
  outline: 2px solid var(--color-focus-ring);
  outline-offset: 2px;
}

.content-effect-big,
.content-effect-shake {
  display: inline-block;
  transform-origin: center;
}

.content-effect-big {
  animation: content-effect-big 1.4s ease-in-out infinite;
}

.content-effect-shake {
  animation: content-effect-shake 360ms linear infinite;
}

.content-animations-paused .content-effect-big,
.content-animations-paused .content-effect-shake {
  animation-play-state: paused;
}

@keyframes content-effect-big {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(var(--content-effect-scale)); }
}

@keyframes content-effect-shake {
  0%, 100% { transform: translate(0, 0) rotate(0); }
  20% { transform: translate(var(--content-effect-shift-negative), var(--content-effect-shift)) rotate(var(--content-effect-rotation-negative)); }
  40% { transform: translate(var(--content-effect-shift), 0) rotate(var(--content-effect-rotation)); }
  60% { transform: translate(var(--content-effect-shift-negative-half), var(--content-effect-shift-negative)) rotate(var(--content-effect-rotation-negative)); }
  80% { transform: translate(var(--content-effect-shift-half), var(--content-effect-shift-half)) rotate(var(--content-effect-rotation)); }
}

@media (prefers-reduced-motion: reduce) {
  .content-effect-big,
  .content-effect-shake {
    animation: none;
  }
}
`;
