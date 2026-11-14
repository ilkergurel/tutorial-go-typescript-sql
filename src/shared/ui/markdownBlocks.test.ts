import { describe, expect, it } from 'vitest';
import { parseBlocks, verifyText } from './markdownBlocks';

describe('pipe tables', () => {
  it('parses a header, a separator and rows, keeping pipes inside code', () => {
    const blocks = parseBlocks('Before:\n| Command | Meaning |\n| --- | :---: |\n| `\\dt` | tables |\n| `a || b` | or \\| concat |');
    expect(blocks).toEqual([
      { kind: 'paragraph', text: 'Before:' },
      {
        kind: 'table',
        header: ['Command', 'Meaning'],
        rows: [
          ['`\\dt`', 'tables'],
          ['`a || b`', 'or | concat'],
        ],
      },
    ]);
  });

  it('leaves a lone pipe line without a separator as a paragraph', () => {
    expect(parseBlocks('| not a table |')).toEqual([{ kind: 'paragraph', text: '| not a table |' }]);
  });
});

describe('HTML comments in lesson prose', () => {
  it('turns a VERIFY comment on its own lines into an unverified block', () => {
    const blocks = parseBlocks('Intro.\n\n<!-- VERIFY: Excel writes\n   a BOM here -->\n\nOutro.');
    expect(blocks).toEqual([
      { kind: 'paragraph', text: 'Intro.' },
      { kind: 'unverified', text: 'Excel writes a BOM here' },
      { kind: 'paragraph', text: 'Outro.' },
    ]);
  });

  it('ends a paragraph at a comment that follows without a blank line', () => {
    const blocks = parseBlocks('Claim one.\n<!-- VERIFY: claim one -->');
    expect(blocks.map((b) => b.kind)).toEqual(['paragraph', 'unverified']);
  });

  it('hides other comments', () => {
    expect(parseBlocks('<!-- author note -->\nText.')).toEqual([{ kind: 'paragraph', text: 'Text.' }]);
  });

  it('keeps a comment inside a list item in the item text, for inline rendering', () => {
    const [list] = parseBlocks('- Item\n  <!-- VERIFY: nested -->\n- Next');
    expect(list).toEqual({ kind: 'list', ordered: false, items: ['Item <!-- VERIFY: nested -->', 'Next'] });
  });

  it('reads the claim out of a whole comment', () => {
    expect(verifyText('<!-- VERIFY: a `b` c -->')).toBe('a `b` c');
    expect(verifyText('<!--VERIFY a -->')).toBe('a');
    expect(verifyText('<!-- TODO: a -->')).toBeNull();
  });
});
