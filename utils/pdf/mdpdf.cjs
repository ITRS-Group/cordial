#!/usr/bin/env node

const fs = require('node:fs');
const { randomUUID } = require('node:crypto');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const MarkdownIt = require('markdown-it');

const markdown = new MarkdownIt({ html: true });

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
  const contents = convertAlerts(fs.readFileSync(source, 'utf8'));
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

module.exports = { convertAlerts };
