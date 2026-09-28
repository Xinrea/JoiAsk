'use client';

import {
  Fragment,
  memo,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type MouseEvent,
  type ReactNode,
} from 'react';
import {
  parseContentMarkup,
  type ContentEffectName,
  type ContentNode,
} from '@joiask/content-markup';

interface FormattedContentProps {
  content: string;
}

interface ContentNodesProps {
  nodes: ContentNode[];
  activeEffects?: ReadonlySet<ContentEffectName>;
  dynamicEffects?: readonly DynamicEffect[];
  characterOffset?: number;
}

interface DynamicEffect {
  name: 'big' | 'shake';
  strength: number;
}

const graphemeSegmenter =
  typeof Intl.Segmenter === 'function'
    ? new Intl.Segmenter('zh-CN', { granularity: 'grapheme' })
    : null;
let contentAnimationObserver: IntersectionObserver | null = null;

function getContentAnimationObserver() {
  if (typeof IntersectionObserver === 'undefined') return null;
  if (!contentAnimationObserver) {
    contentAnimationObserver = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        entry.target.classList.toggle(
          'content-animations-paused',
          !entry.isIntersecting
        );
      }
    });
  }
  return contentAnimationObserver;
}

function splitGraphemes(value: string) {
  if (!graphemeSegmenter) return Array.from(value);
  return Array.from(graphemeSegmenter.segment(value), ({ segment }) => segment);
}

function countCharacters(node: ContentNode): number {
  if (node.type === 'text') return splitGraphemes(node.value).length;
  if (node.type === 'link') return splitGraphemes(node.label).length;
  if (node.type === 'emoji') return 1;
  return node.children.reduce(
    (count, child) => count + countCharacters(child),
    0
  );
}

function wrapAnimatedUnit(
  content: ReactNode,
  effects: readonly DynamicEffect[],
  characterIndex: number,
  key: string
) {
  return effects.reduceRight<ReactNode>((children, effect, effectIndex) => {
    if (effect.name === 'big') {
      const style = {
        '--content-effect-scale': String(1 + effect.strength * 0.15),
        animationDelay: `${-((characterIndex % 14) * 70)}ms`,
      } as CSSProperties;
      return (
        <span
          key={`${key}-big-${effectIndex}`}
          className="content-effect-big"
          style={style}
        >
          {children}
        </span>
      );
    }

    const shift = effect.strength * 1.5;
    const rotation = effect.strength * 0.4;
    const style = {
      '--content-effect-shift': `${shift}px`,
      '--content-effect-shift-negative': `${-shift}px`,
      '--content-effect-shift-half': `${shift * 0.5}px`,
      '--content-effect-shift-negative-half': `${shift * -0.5}px`,
      '--content-effect-rotation': `${rotation}deg`,
      '--content-effect-rotation-negative': `${-rotation}deg`,
      animationDelay: `${-((characterIndex % 8) * 45)}ms`,
    } as CSSProperties;
    return (
      <span
        key={`${key}-shake-${effectIndex}`}
        className="content-effect-shake"
        style={style}
      >
        {children}
      </span>
    );
  }, content);
}

function renderAnimatedText(
  value: string,
  effects: readonly DynamicEffect[],
  characterOffset: number,
  keyPrefix: string
) {
  if (effects.length === 0) return value;
  return splitGraphemes(value).map((character, index) => {
    if (/^\s$/u.test(character)) return character;
    return wrapAnimatedUnit(
      character,
      effects,
      characterOffset + index,
      `${keyPrefix}-${index}`
    );
  });
}

function HiddenContent({ children }: { children: ReactNode }) {
  const [revealed, setRevealed] = useState(false);

  const handleClick = (event: MouseEvent<HTMLSpanElement>) => {
    if (!revealed) {
      event.preventDefault();
      setRevealed(true);
      return;
    }
    if (event.target instanceof Element && event.target.closest('a')) {
      return;
    }
    setRevealed(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
    if (event.target !== event.currentTarget) return;
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      setRevealed((current) => !current);
    }
  };

  return (
    <span
      className="content-effect-hide"
      data-revealed={revealed}
      role="button"
      tabIndex={0}
      aria-label={revealed ? '隐藏内容已显示，点击重新遮挡' : '隐藏内容，点击显示'}
      aria-pressed={revealed}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
    >
      {children}
    </span>
  );
}

function ContentNodes({
  nodes,
  activeEffects = new Set<ContentEffectName>(),
  dynamicEffects = [],
  characterOffset = 0,
}: ContentNodesProps) {
  let nextCharacterOffset = characterOffset;

  return nodes.map((node, index) => {
    const key = `${node.type}-${index}`;
    const nodeCharacterOffset = nextCharacterOffset;
    nextCharacterOffset += countCharacters(node);

    if (node.type === 'text') {
      return renderAnimatedText(
        node.value,
        dynamicEffects,
        nodeCharacterOffset,
        key
      );
    }
    if (node.type === 'link') {
      return (
        <a
          key={key}
          href={node.href}
          target="_blank"
          rel="noopener noreferrer"
          className="content-markup-link"
        >
          {renderAnimatedText(
            node.label,
            dynamicEffects,
            nodeCharacterOffset,
            key
          )}
        </a>
      );
    }
    if (node.type === 'emoji') {
      const image = (
        <img
          src={node.src}
          alt={node.label}
          className="content-markup-emoji"
        />
      );
      return (
        <Fragment key={key}>
          {wrapAnimatedUnit(
            image,
            dynamicEffects,
            nodeCharacterOffset,
            key
          )}
        </Fragment>
      );
    }

    const isAlreadyActive = activeEffects.has(node.name);
    const nextActiveEffects = isAlreadyActive
      ? activeEffects
      : new Set<ContentEffectName>([...activeEffects, node.name]);
    const nextDynamicEffects =
      !isAlreadyActive && (node.name === 'big' || node.name === 'shake')
        ? [
            ...dynamicEffects,
            {
              name: node.name,
              strength: node.strength ?? 1,
            },
          ]
        : dynamicEffects;
    const children = (
      <ContentNodes
        nodes={node.children}
        activeEffects={nextActiveEffects}
        dynamicEffects={nextDynamicEffects}
        characterOffset={nodeCharacterOffset}
      />
    );

    if (isAlreadyActive) {
      return <Fragment key={key}>{children}</Fragment>;
    }
    if (node.name === 'hide') {
      return <HiddenContent key={key}>{children}</HiddenContent>;
    }
    if (node.name === 'bold') return <strong key={key}>{children}</strong>;
    return <Fragment key={key}>{children}</Fragment>;
  });
}

export const FormattedContent = memo(function FormattedContent({
  content,
}: FormattedContentProps) {
  const containerRef = useRef<HTMLSpanElement>(null);
  const nodes = useMemo(() => parseContentMarkup(content), [content]);

  useEffect(() => {
    const container = containerRef.current;
    const observer = getContentAnimationObserver();
    if (!container || !observer) return;

    observer.observe(container);
    return () => observer.unobserve(container);
  }, []);

  return (
    <span ref={containerRef}>
      <ContentNodes nodes={nodes} />
    </span>
  );
});
