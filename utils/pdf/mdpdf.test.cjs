const test = require('node:test');
const assert = require('node:assert/strict');
const { convertAlerts, convertHtmlEntities } = require('./mdpdf.cjs');

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

test('decodes entities inside raw HTML without changing regular Markdown', () => {
  const source = [
    '| Type | Meaning |',
    '| --- | --- |',
    '| <code>Layer&nbsp;2&amp;More</code> | Text&nbsp;value |',
    '',
    '<span>A&nbsp;B &lt;example&gt; &#169;</span>',
  ].join('\n');
  assert.equal(convertHtmlEntities(source), [
    '| Type | Meaning |',
    '| --- | --- |',
    '| <code>Layer\u202f2&More</code> | Text&nbsp;value |',
    '',
    '<span>A\u202fB <example> ©</span>',
  ].join('\n'));
});

test('preserves entities in fenced, indented and inline code', () => {
  const source = [
    'Outside <code>A&nbsp;B</code>.',
    '',
    '`<code>A&nbsp;B</code>`',
    '',
    '```xml',
    '<code>A&nbsp;B</code>',
    '```',
    '',
    '    <code>A&nbsp;B</code>',
  ].join('\n');
  const expected = source.replace(
    'Outside <code>A&nbsp;B</code>.',
    'Outside <code>A\u202fB</code>.'
  );
  assert.equal(convertHtmlEntities(source), expected);
});

test('decodes entities in multiline raw HTML', () => {
  const source = '<span>\nA&nbsp;B &amp; C\n</span>';
  assert.equal(convertHtmlEntities(source), '<span>\nA\u202fB & C\n</span>');
});
