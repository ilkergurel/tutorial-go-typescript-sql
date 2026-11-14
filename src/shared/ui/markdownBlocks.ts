/**
 * Block-level parsing for <Markdown>: the subset lesson prose uses (see
 * Markdown.tsx). Separate from the component so tests can import it and the
 * component file exports components only (React Fast Refresh).
 */
export type Block =
  | { kind: 'heading'; level: 3 | 4; text: string }
  | { kind: 'paragraph'; text: string }
  | { kind: 'list'; ordered: boolean; items: string[] }
  | { kind: 'code'; lang: string; code: string }
  | { kind: 'table'; header: string[]; rows: string[][] }
  | { kind: 'unverified'; text: string };

const LIST_ITEM = /^([-*]|\d+\.)\s+(.*)$/;
const FENCE = /^```(\S*)\s*$/;
const HEADING = /^(#{3,4})\s+(.*)$/;
const COMMENT_START = /^<!--/;
const TABLE_ROW = /^\|.*\|$/;
const TABLE_SEPARATOR = /^\|(\s*:?-{3,}:?\s*\|)+$/;

/**
 * Cells of one `| a | b |` row. Pipes inside `code` spans and escaped `\|`
 * pipes belong to the cell: lessons put things like `a || b` in tables.
 */
export function tableCells(row: string): string[] {
  const cells: string[] = [];
  let cell = '';
  let inCode = false;
  const inner = row.trim().slice(1, -1);
  for (let i = 0; i < inner.length; i++) {
    const ch = inner[i];
    if (ch === '\\' && inner[i + 1] === '|') {
      cell += '|';
      i++;
    } else if (ch === '`') {
      inCode = !inCode;
      cell += ch;
    } else if (ch === '|' && !inCode) {
      cells.push(cell.trim());
      cell = '';
    } else {
      cell += ch;
    }
  }
  cells.push(cell.trim());
  return cells;
}

/**
 * `<!-- VERIFY: claim -->` marks a claim the author couldn't confirm in
 * official docs (CLAUDE.md rule 7). The app shows it as a warning instead of
 * hiding it like other HTML comments, so the owner can check it.
 */
export const VERIFY_COMMENT = /<!--\s*VERIFY:?\s*([\s\S]*?)\s*-->/g;

/** The claim inside a whole HTML comment, or null for any other comment. */
export function verifyText(comment: string): string | null {
  const m = /^<!--\s*VERIFY:?\s*([\s\S]*?)\s*-->$/.exec(comment.trim());
  return m ? m[1].replace(/\s+/g, ' ') : null;
}

/** Splits Markdown into blocks; exported for tests. */
export function parseBlocks(source: string): Block[] {
  const lines = source.replace(/\r\n/g, '\n').split('\n');
  const blocks: Block[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (line.trim() === '') {
      i++;
      continue;
    }
    if (COMMENT_START.test(line.trim())) {
      // A comment runs to the line holding "-->", possibly several lines on.
      const comment: string[] = [];
      while (i < lines.length) {
        comment.push(lines[i]);
        if (lines[i++].includes('-->')) break;
      }
      const claim = verifyText(comment.join('\n'));
      if (claim !== null) blocks.push({ kind: 'unverified', text: claim });
      continue;
    }
    const fence = FENCE.exec(line.trim());
    if (fence) {
      const code: string[] = [];
      i++;
      while (i < lines.length && !/^```\s*$/.test(lines[i].trim())) code.push(lines[i++]);
      i++; // closing fence
      blocks.push({ kind: 'code', lang: fence[1], code: code.join('\n') });
      continue;
    }
    if (TABLE_ROW.test(line.trim()) && TABLE_SEPARATOR.test((lines[i + 1] ?? '').trim().replace(/\s+/g, ' '))) {
      const header = tableCells(line);
      const rows: string[][] = [];
      i += 2;
      while (i < lines.length && TABLE_ROW.test(lines[i].trim())) rows.push(tableCells(lines[i++]));
      blocks.push({ kind: 'table', header, rows });
      continue;
    }
    const heading = HEADING.exec(line);
    if (heading) {
      blocks.push({ kind: 'heading', level: heading[1].length === 3 ? 3 : 4, text: heading[2] });
      i++;
      continue;
    }
    const item = LIST_ITEM.exec(line);
    if (item) {
      const ordered = /\d/.test(item[1]);
      const items: string[] = [];
      while (i < lines.length) {
        const next = LIST_ITEM.exec(lines[i]);
        if (next && /\d/.test(next[1]) === ordered) {
          items.push(next[2]);
        } else if (items.length > 0 && /^\s+\S/.test(lines[i])) {
          // An indented line continues the previous item (wrapped text or a
          // nested item, which this subset flattens).
          items[items.length - 1] += ' ' + lines[i].trim();
        } else {
          break;
        }
        i++;
      }
      blocks.push({ kind: 'list', ordered, items });
      continue;
    }
    // The first line always belongs to the paragraph, whatever it looks like:
    // every other block start was tried above, so stopping here would loop.
    const para: string[] = [lines[i++].trim()];
    while (
      i < lines.length &&
      lines[i].trim() !== '' &&
      !FENCE.test(lines[i].trim()) &&
      !HEADING.test(lines[i]) &&
      !LIST_ITEM.test(lines[i]) &&
      !COMMENT_START.test(lines[i].trim()) &&
      !TABLE_ROW.test(lines[i].trim())
    ) {
      para.push(lines[i++].trim());
    }
    blocks.push({ kind: 'paragraph', text: para.join(' ') });
  }
  return blocks;
}
