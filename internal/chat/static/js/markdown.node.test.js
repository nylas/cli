const test = require('node:test');
const assert = require('node:assert/strict');

const Markdown = require('./markdown.js');

test('escape encodes quotes so attribute context cannot be broken out of', () => {
    assert.equal(
        Markdown.escape(`<b>&"quoted"'single'</b>`),
        '&lt;b&gt;&amp;&quot;quoted&quot;&#39;single&#39;&lt;/b&gt;'
    );
});

test('render neutralises attribute-breakout payload in link URL', () => {
    // Reproduces the reported exploit: a double quote in the URL used to
    // close the href attribute and inject an onmouseover handler.
    const html = Markdown.render('[click](http://x" onmouseover="alert(1))');

    assert.ok(!html.includes('onmouseover="alert(1)"'), 'must not inject event handler attribute');
    assert.ok(html.includes('href="http://x&quot;'), 'quote must stay encoded inside href');
});

test('render neutralises attribute-breakout payload in link label', () => {
    const html = Markdown.render('[x" onmouseover="alert(1)](http://example.com)');

    assert.ok(!html.includes('onmouseover="alert(1)"'), 'must not inject event handler attribute');
});

test('render blocks javascript: URLs', () => {
    const html = Markdown.render('[click](javascript:alert(1))');

    assert.ok(!html.includes('javascript:'), 'dangerous scheme must be replaced');
    assert.ok(html.includes('href="#"'), 'href must fall back to #');
});

test('render keeps plain http links working', () => {
    const html = Markdown.render('[docs](https://developer.nylas.com/)');

    assert.ok(html.includes('<a href="https://developer.nylas.com/" target="_blank" rel="noopener noreferrer">docs</a>'));
});
