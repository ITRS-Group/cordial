#!/usr/bin/env node

const fs = require('node:fs');
const { randomUUID } = require('node:crypto');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const MarkdownIt = require('markdown-it');

const markdown = new MarkdownIt({ html: true });

function decodeEntity(match) {
  const token = markdown.parseInline(match, {})[0];
  if (token.children?.length === 1 && token.children[0].type === 'text') {
    const decoded = token.children[0].content;
    return decoded === '\u00a0' ? '\u202f' : decoded;
  }
  return match;
}

function decodeHtmlContentEntities(chunk) {
  const pattern = /&(?:#\d+|#x[\da-f]+|[a-z][\da-z]+);/gi;
  return chunk.replace(/<([a-z][\w:-]*)\b[^>]*>([\s\S]*?)<\/\1>/gi, (match, tag, content) => {
    return match
      .slice(0, match.indexOf('>') + 1) +
      decodeHtmlContentEntities(content).replace(pattern, decodeEntity) +
      `</${tag}>`;
  });
}

function convertHtmlEntities(source) {
  const tokens = markdown.parse(source, {});
  const codeLines = new Set();
  for (const token of tokens) {
    if (!token.map || (token.type !== 'fence' && token.type !== 'code_block')) continue;
    for (let line = token.map[0]; line < token.map[1]; line++) codeLines.add(line);
  }

  const lines = source.split('\n');
  const inlineCode = [];
  let converted = [];
  let prose = [];

  function flushProse() {
    if (prose.length === 0) return;
    let text = prose.join('\n');
    text = text.replace(/(`+)[\s\S]*?\1/g, (match) => {
      const marker = `\uE000${inlineCode.length}\uE001`;
      inlineCode.push(match);
      return marker;
    });
    converted.push(decodeHtmlContentEntities(text));
    prose = [];
  }

  lines.forEach((line, index) => {
    if (codeLines.has(index)) {
      flushProse();
      converted.push(line);
      return;
    }
    prose.push(line);
  });
  flushProse();

  let result = converted.join('\n');
  for (let index = 0; index < inlineCode.length; index++) {
    result = result.replace(`\uE000${index}\uE001`, inlineCode[index]);
  }
  return result;
}

function convertAlerts(source) {
  const lines = source.split('\n');
  const tokens = markdown.parse(source, {});

  for (let i = 0; i < tokens.length - 2; i++) {
    if (tokens[i].type !== 'blockquote_open' ||
        tokens[i + 1].type !== 'paragraph_open' ||
        tokens[i + 2].type !== 'inline') {
      continue;
    }

    const firstLine = tokens[i + 2].content.split('\n', 1)[0].trim();
    const match = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$/.exec(firstLine);
    if (!match) continue;

    const line = tokens[i + 1].map[0];
    const type = match[1].toLowerCase();
    const title = type[0].toUpperCase() + type.slice(1);
    lines[line] = lines[line].replace(
      /\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\](?=[ \t\r]*$)/,
      `<span class="markdown-alert-title markdown-alert-${type}">${title}</span>`
    );
  }

  return lines.join('\n');
}

function main(args) {
  if (args.length < 2) {
    throw new Error('Usage: mdpdf.cjs <source.md> <destination.pdf> [mdpdf options]');
  }

  const source = path.resolve(args[0]);
  // Keep the temporary Markdown beside the source so relative images still resolve.
  const temporarySource = path.join(path.dirname(source), `.pdf-${randomUUID()}.md`);
  const contents = convertHtmlEntities(convertAlerts(fs.readFileSync(source, 'utf8')));
  fs.writeFileSync(temporarySource, contents, { flag: 'wx' });
  try {
    const result = spawnSync('mdpdf', [temporarySource, ...args.slice(1)], { stdio: 'inherit' });
    if (result.error) throw result.error;
    if (result.status !== 0) throw new Error(`mdpdf failed (status ${result.status}, signal ${result.signal})`);
  } finally {
    fs.unlinkSync(temporarySource);
  }
}

if (require.main === module) main(process.argv.slice(2));

module.exports = { convertAlerts, convertHtmlEntities };
