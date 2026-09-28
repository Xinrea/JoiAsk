import assert from 'node:assert/strict';
import test from 'node:test';
import {
  CONTENT_EMOJI_MAP,
  parseContentMarkup,
} from './content-markup.ts';

test('parses nested effects and legacy hidden content', () => {
  assert.deepEqual(
    parseContentMarkup(
      'A{{秘密}}[bold]粗体[big=1.5][shake=.5]动[/shake][/big][/bold]'
    ),
    [
      { type: 'text', value: 'A' },
      {
        type: 'effect',
        name: 'hide',
        strength: undefined,
        children: [{ type: 'text', value: '秘密' }],
      },
      {
        type: 'effect',
        name: 'bold',
        strength: undefined,
        children: [
          { type: 'text', value: '粗体' },
          {
            type: 'effect',
            name: 'big',
            strength: 1.5,
            children: [
              {
                type: 'effect',
                name: 'shake',
                strength: 0.5,
                children: [{ type: 'text', value: '动' }],
              },
            ],
          },
        ],
      },
    ]
  );
});

test('uses defaults and clamps effect strengths', () => {
  const nodes = parseContentMarkup(
    '[big]默认[/big][big=99]最大[/big][shake=0]最小[/shake]'
  );
  assert.equal(nodes[0].type === 'effect' && nodes[0].strength, 1);
  assert.equal(nodes[1].type === 'effect' && nodes[1].strength, 3);
  assert.equal(nodes[2].type === 'effect' && nodes[2].strength, 0.1);
});

test('keeps malformed, unknown, and full-width square tags literal', () => {
  const value =
    '[big=fast]内容[/big][unknown]文字[/unknown]［bold］全角［/bold］';
  assert.deepEqual(parseContentMarkup(value), [{ type: 'text', value }]);
});

test('keeps an unclosed effect region literal', () => {
  const value = '之前[bold]未闭合[big=2]内容[/big]';
  assert.deepEqual(parseContentMarkup(value), [{ type: 'text', value }]);
});

test('recognizes safe links, BV codes, and content emojis', () => {
  const emojiLabel = '[轴伊Joi收藏集动态表情包_跑了]';
  assert.deepEqual(
    parseContentMarkup(
      `看 https://example.com 和 BV1xx411c7mD ${emojiLabel}`
    ),
    [
      { type: 'text', value: '看 ' },
      {
        type: 'link',
        href: 'https://example.com',
        label: 'https://example.com',
      },
      { type: 'text', value: ' 和 ' },
      {
        type: 'link',
        href: 'https://www.bilibili.com/video/BV1xx411c7mD',
        label: 'BV1xx411c7mD',
      },
      { type: 'text', value: ' ' },
      {
        type: 'emoji',
        label: emojiLabel,
        src: CONTENT_EMOJI_MAP[emojiLabel],
      },
    ]
  );
});

test('supports escaping formatting delimiters', () => {
  assert.deepEqual(parseContentMarkup(String.raw`\[bold\]文字\[/bold\]`), [
    { type: 'text', value: '[bold]文字[/bold]' },
  ]);
});
