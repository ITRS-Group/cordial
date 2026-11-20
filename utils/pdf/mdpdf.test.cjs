const test = require('node:test');
const assert = require('node:assert/strict');
const { convertAlerts } = require('./mdpdf.cjs');

test('converts each GitHub alert, retaining the rest of the blockquote', () => {
  for (const type of ['NOTE', 'TIP', 'IMPORTANT', 'WARNING', 'CAUTION']) {
    const source = `>[!${type}]\n> First **paragraph**.\n>\n> - list item\n>\n> \`\`\`sh\n> echo hello\n> \`\`\`\n`;
    const result = convertAlerts(source);
    assert.match(result, new RegExp(`^>\\s*<span class="markdown-alert-title markdown-alert-${type.toLowerCase()}">`));
    assert.equal(result.slice(result.indexOf('\n')), source.slice(source.indexOf('\n')));
  }
});

test('leaves examples and ordinary blockquotes unchanged', () => {
  const source = [
    '```md',
    '> [!NOTE]',
    '> Example',
    '```',
    '',
    '> Ordinary quote',
    '> [!WARNING]',
    '',
    '> [!UNKNOWN]',
    '> Unknown alert',
    '',
    '> ```md',
    '> > [!CAUTION]',
    '> ```',
    '',
    '    > [!TIP]',
    '',
  ].join('\n');
  assert.equal(convertAlerts(source), source);
});

test('converts alerts in list items and supports CRLF', () => {
  const source = '- > [!NOTE]\r\n  > Text\r\n';
  assert.equal(
    convertAlerts(source),
    '- > <span class="markdown-alert-title markdown-alert-note">Note</span>\r\n  > Text\r\n'
  );
});
