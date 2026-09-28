# 前台样式分类与自定义 CSS

前台样式大致分为以下几层。后台“自定义”页面中的 CSS 是前台主题层的完整内容，保存后会替换默认主题。

## 1. 设计令牌

设计令牌是全局 CSS 变量，决定颜色、圆角、边框和阴影等基础视觉参数：

- `--page-background-color`：页面背景色
- `--page-background-image`：页面背景图层，可以是图片、渐变或 `none`
- `--page-background-size`：背景图层尺寸
- `--page-background-attachment`：背景滚动行为
- `--color-page-foreground`：页面文字颜色
- `--color-brand`、`--color-brand-foreground`：品牌色及其前景色
- `--color-nav-foreground`：导航栏文字和图标颜色，默认跟随品牌前景色
- `--color-control-secondary`、`--color-control-hover`：控件的次要和悬浮颜色
- `--color-surface`、`--color-overlay`：卡片表面和浮层表面
- `--color-control-border`、`--color-input-border`、`--color-focus-ring`
- `--radius-base`
- `--color-fabric-stitch`、`--color-fabric-shadow`、`--color-fabric-grain`

推荐优先修改变量，而不是给大量组件逐个写规则。例如：

```css
:root {
  --color-brand: #5b65c8;
  --color-brand-foreground: #ffffff;
  --page-background-color: #f4f6ff;
  --color-fabric-stitch: #8490e0;
  --page-background-image: none;
}
```

例如设置页面渐变背景：

```css
:root {
  --page-background-image: linear-gradient(135deg, #f4f6ff, #fff7ed);
  --page-background-size: auto;
  --page-background-attachment: scroll;
}
```

## 2. 全局背景与导航栏

- `body`：页面基础背景和前景色
- `.fabric-linen`：织物纹理页面背景
- `.navbar`、`.fabric-nav`：前台导航栏

## 3. 通用表面和表单

- `.fabric-card`：提问卡片等内容表面
- `.fabric-form`：提问表单表面
- `.fabric-pill`：标签、筛选器等胶囊元素
- `.fabric-input`：输入框
- `.fabric-label`：主要操作按钮或标签
- `.fabric-shadow`、`.fabric-shadow-hover`：阴影和悬浮效果

## 4. 提问内容和内容特效

- `.card`、`.card-content`：提问卡片及正文区域
- `.content-markup-link`：正文链接
- `.content-markup-emoji`：正文中的表情图片
- `.content-effect-hide`：隐藏内容
- `.content-effect-big`、`.content-effect-shake`：正文动画
- `.content-animations-paused`：暂停正文动画
- `.rainbow-bg`：彩虹提问背景
- `.watermark`、`.stamp`：归档水印和印章

## 5. 浮层和交互控件

这部分目前主要使用 Tailwind 工具类实现，必要时可通过稳定的语义选择器或现有组件结构覆盖：

- 登录注册对话框
- 表情选择器
- 图片预览器
- 回到顶部按钮
- 文件上传预览区域

自定义 CSS 不建议依赖 Tailwind 生成的长工具类组合，因为这些类属于实现细节，后续调整组件时可能变化。

## 6. 响应式、动画和无障碍

- 使用 `@media` 处理移动端布局；
- 使用 `@keyframes` 定义动画；
- 使用 `@media (prefers-reduced-motion: reduce)` 为动画提供降级；
- 覆盖焦点样式时应保留 `:focus-visible`，避免键盘用户失去焦点提示。

## 恢复默认主题

“自定义”页面提供“恢复默认”按钮。恢复后会清空数据库中的自定义 CSS，前台会重新使用内置的默认主题。

页面还提供九种颜色预设：红、橙、黄、绿、青、蓝、紫、黑、白。预设只修改主题变量，不会覆盖编辑器中其他布局或组件规则；选择后仍需点击“保存”才会应用到前台。

## 使用限制

- CSS 由管理员保存并直接注入前台页面；
- 不要填写 `<style>` 标签，只填写 CSS 规则；
- 单条配置建议控制在 128 KiB 以内；
- 可以使用 `url()` 引用外部资源，但应确认资源来源可信；
- 配置会在 SSR 阶段写入 HTML，保存后新请求即可看到效果。
