export const MIN_EFFECT_STRENGTH = 0.1;
export const MAX_EFFECT_STRENGTH = 3;
export const DEFAULT_EFFECT_STRENGTH = 1;

export const CONTENT_EMOJI_MAP: Record<string, string> = {
  '[轴伊Joi收藏集动态表情包_跑了]': '/joi-emojis/paole.webp',
  '[轴伊Joi收藏集动态表情包_鞠躬]': '/joi-emojis/jugong.webp',
  '[轴伊Joi收藏集动态表情包_摇你]': '/joi-emojis/yaoni.webp',
  '[轴伊Joi收藏集动态表情包_愤怒]': '/joi-emojis/fennu.webp',
  '[轴伊Joi收藏集动态表情包_猴]': '/joi-emojis/hou.webp',
  '[轴伊Joi收藏集动态表情包_NO]': '/joi-emojis/no.webp',
  '[轴伊Joi收藏集动态表情包_贴贴]': '/joi-emojis/tietie.webp',
  '[轴伊Joi收藏集动态表情包_呆]': '/joi-emojis/dai.webp',
  '[轴伊Joi收藏集动态表情包_唔唔]': '/joi-emojis/wuwu.webp',
  '[轴伊Joi收藏集动态表情包_啊这]': '/joi-emojis/azhe.webp',
  '[轴伊Joi收藏集动态表情包_失落]': '/joi-emojis/shiluo.webp',
  '[轴伊Joi收藏集动态表情包_神气]': '/joi-emojis/shenqi.webp',
  '[轴伊Joi收藏集动态表情包_怎么这样]': '/joi-emojis/zenmezhyang.webp',
  '[轴伊Joi收藏集动态表情包_睡觉]': '/joi-emojis/shuijiao.webp',
  '[轴伊Joi收藏集动态表情包_爆]': '/joi-emojis/bao.webp',
};

export type ContentEffectName = 'hide' | 'bold' | 'big' | 'shake';

export type ContentNode =
  | { type: 'text'; value: string }
  | { type: 'link'; href: string; label: string }
  | { type: 'emoji'; label: string; src: string }
  | {
      type: 'effect';
      name: ContentEffectName;
      strength?: number;
      children: ContentNode[];
    };

interface EffectFrame {
  name: ContentEffectName;
  strength?: number;
  children: ContentNode[];
  closeToken: string;
  start: number;
  valid: boolean;
}

const EFFECT_OPEN_RE = /^\[(hide|bold|big|shake)(?:=([^\]]*))?\]/;
const EFFECT_CLOSE_RE = /^\[\/(hide|bold|big|shake)\]/;
const STRENGTH_RE = /^(?:\d+(?:\.\d*)?|\.\d+)$/;
const URL_RE = /https?:\/\/[^\s<\[]+/;
const BV_RE = /\bBV[a-zA-Z0-9]{10}\b/;
const MAX_EFFECT_DEPTH = 8;
const MAX_EFFECT_COUNT = 50;

export function clampEffectStrength(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_EFFECT_STRENGTH;
  return Math.min(MAX_EFFECT_STRENGTH, Math.max(MIN_EFFECT_STRENGTH, value));
}

function appendNode(nodes: ContentNode[], node: ContentNode) {
  const previous = nodes[nodes.length - 1];
  if (node.type === 'text' && previous?.type === 'text') {
    previous.value += node.value;
    return;
  }
  nodes.push(node);
}

function tokenizeText(value: string): ContentNode[] {
  const nodes: ContentNode[] = [];
  let cursor = 0;

  while (cursor < value.length) {
    const remaining = value.slice(cursor);
    let next:
      | { index: number; length: number; node: ContentNode; priority: number }
      | undefined;

    const urlMatch = URL_RE.exec(remaining);
    if (urlMatch?.index !== undefined) {
      next = {
        index: urlMatch.index,
        length: urlMatch[0].length,
        node: { type: 'link', href: urlMatch[0], label: urlMatch[0] },
        priority: 0,
      };
    }

    const bvMatch = BV_RE.exec(remaining);
    if (
      bvMatch?.index !== undefined &&
      (!next ||
        bvMatch.index < next.index ||
        (bvMatch.index === next.index && next.priority > 1))
    ) {
      const code = bvMatch[0];
      next = {
        index: bvMatch.index,
        length: code.length,
        node: {
          type: 'link',
          href: `https://www.bilibili.com/video/${code}`,
          label: code,
        },
        priority: 1,
      };
    }

    for (const [label, src] of Object.entries(CONTENT_EMOJI_MAP)) {
      const index = remaining.indexOf(label);
      if (
        index !== -1 &&
        (!next || index < next.index || (index === next.index && next.priority > 2))
      ) {
        next = {
          index,
          length: label.length,
          node: { type: 'emoji', label, src },
          priority: 2,
        };
      }
    }

    if (!next) {
      appendNode(nodes, { type: 'text', value: remaining });
      break;
    }

    if (next.index > 0) {
      appendNode(nodes, {
        type: 'text',
        value: remaining.slice(0, next.index),
      });
    }
    appendNode(nodes, next.node);
    cursor += next.index + next.length;
  }

  return nodes;
}

function tokenizeTextNodes(nodes: ContentNode[]): ContentNode[] {
  return nodes.flatMap((node) => {
    if (node.type === 'text') return tokenizeText(node.value);
    if (node.type !== 'effect') return node;
    return {
      ...node,
      children: tokenizeTextNodes(node.children),
    };
  });
}

function parseEffectOpening(
  source: string,
  index: number,
  validDepth: number,
  effectCount: number
): { frame: EffectFrame; length: number; countsAsEffect: boolean } | undefined {
  if (source.startsWith('{{', index)) {
    return {
      frame: {
        name: 'hide',
        children: [],
        closeToken: '}}',
        start: index,
        valid: validDepth < MAX_EFFECT_DEPTH && effectCount < MAX_EFFECT_COUNT,
      },
      length: 2,
      countsAsEffect: validDepth < MAX_EFFECT_DEPTH && effectCount < MAX_EFFECT_COUNT,
    };
  }

  const match = EFFECT_OPEN_RE.exec(source.slice(index));
  if (!match) return undefined;

  const name = match[1] as ContentEffectName;
  const rawStrength = match[2];
  let syntaxValid = true;
  let strength: number | undefined;

  if (name === 'big' || name === 'shake') {
    if (rawStrength === undefined) {
      strength = DEFAULT_EFFECT_STRENGTH;
    } else if (STRENGTH_RE.test(rawStrength)) {
      strength = clampEffectStrength(Number(rawStrength));
    } else {
      syntaxValid = false;
    }
  } else if (rawStrength !== undefined) {
    syntaxValid = false;
  }

  const withinLimits =
    validDepth < MAX_EFFECT_DEPTH && effectCount < MAX_EFFECT_COUNT;
  const valid = syntaxValid && withinLimits;

  return {
    frame: {
      name,
      strength,
      children: [],
      closeToken: `[/${name}]`,
      start: index,
      valid,
    },
    length: match[0].length,
    countsAsEffect: valid,
  };
}

export function parseContentMarkup(content: string): ContentNode[] {
  const source = content.replace(/｛/g, '{').replace(/｝/g, '}');
  const root: ContentNode[] = [];
  const stack: EffectFrame[] = [];
  let effectCount = 0;
  let index = 0;

  const currentNodes = () =>
    stack.length > 0 ? stack[stack.length - 1].children : root;

  while (index < source.length) {
    if (
      source[index] === '\\' &&
      index + 1 < source.length &&
      ['\\', '[', ']', '{', '}'].includes(source[index + 1])
    ) {
      appendNode(currentNodes(), {
        type: 'text',
        value: source[index + 1],
      });
      index += 2;
      continue;
    }

    const top = stack[stack.length - 1];
    if (top && source.startsWith(top.closeToken, index)) {
      stack.pop();
      const target = currentNodes();
      if (top.valid) {
        appendNode(target, {
          type: 'effect',
          name: top.name,
          strength: top.strength,
          children: top.children,
        });
      } else {
        appendNode(target, {
          type: 'text',
          value: source.slice(top.start, index + top.closeToken.length),
        });
      }
      index += top.closeToken.length;
      continue;
    }

    const closeMatch = EFFECT_CLOSE_RE.exec(source.slice(index));
    if (closeMatch) {
      appendNode(currentNodes(), { type: 'text', value: closeMatch[0] });
      index += closeMatch[0].length;
      continue;
    }
    if (source.startsWith('}}', index)) {
      appendNode(currentNodes(), { type: 'text', value: '}}' });
      index += 2;
      continue;
    }

    const validDepth = stack.reduce(
      (depth, frame) => depth + (frame.valid ? 1 : 0),
      0
    );
    const opening = parseEffectOpening(
      source,
      index,
      validDepth,
      effectCount
    );
    if (opening) {
      stack.push(opening.frame);
      if (opening.countsAsEffect) effectCount += 1;
      index += opening.length;
      continue;
    }

    appendNode(currentNodes(), { type: 'text', value: source[index] });
    index += 1;
  }

  if (stack.length > 0) {
    appendNode(root, {
      type: 'text',
      value: source.slice(stack[0].start),
    });
  }

  return tokenizeTextNodes(root);
}
