# Versioned site packs

Packs are explicit local inputs selected with `RULE_PACKS=file1.yaml;file2.yaml`
or CLI `--pack`. They are separate from the recursive legacy `RULESET` loader.
No pack is enabled by default. The current [Wikipedia reader pack](../../sitepacks/wikipedia/1.0.1.yaml) removes
Vector navigation and tools before article extraction. The original
`testdata/wikipedia.yaml` sample remains version 1.0.0 for reproducible tests.
Both are intended for public encyclopedia reading. It identifies Ladder honestly and does not
set personal cookies, addresses, or a forged referrer.

```yaml
schemaVersion: 1
id: example
version: "1.0.0"
rules:
  - id: article
    domains: [example.org]
    paths: [/article]
    headers:
      set: {Accept: text/html}
      remove: [Referer]
    url:
      query: [{key: tracking, value: ""}]
      domain: [{match: '^www\.', replace: ''}]
      path: [{match: '^/old/', replace: '/article/'}]
    regex: [{match: 'old word', replace: 'new word'}]
    remove: ['.advertisement']
    tests:
      - name: article-fixture
        url: https://example.org/article/fixture
        status: 200
        contains: [new word]
        notContains: [advertisement]
```

`schemaVersion` is exactly 1. Pack and rule IDs are stable lowercase letters,
digits, dots, underscores and dashes, at most 64 characters. `version` is an
explicit semantic version. Unknown fields, duplicate IDs, malformed regexes,
selectors, URLs and header values are errors. Each rule declares at least one
named test with a status or nonempty text assertion. Load/check is read-only,
local validation. Tests retrieve only URLs explicitly requested by the caller.

Selection uses the original requested HTTP(S) URL, before URL changes or
redirects. The first matching rule in each pack wins, and selected packs apply
in configured file order. A domain matches itself and real subdomains, with
case and trailing-dot normalization. IP addresses match exactly. Paths match
an exact segment prefix: `/article` matches `/article/one`, not `/articles`.
There is no suffix wildcard syntax. Domains do not contain schemes or ports.

URL modifications and request headers affect the shared fetch. Query edits
preserve unrelated duplicate values. Changed URLs and every HTTP redirect are
validated again. Origin changes strip credential headers, including configured
cookies. Document regexes and selector removals apply to HTML in reader/proxy
views; raw fetch bytes remain unchanged. Legacy injection/regex fields retain
their previous inactive runtime behavior. Binary bodies are never transformed.

Each loaded pack reports the SHA-256 of its exact file bytes, including comments
and whitespace. Use a new file/version when changing a pack used by an
experiment; keep its prior digest in the experiment record. The service is
immutable after construction, so a new pack selection uses a new cache.
`TestURL` reports selected IDs, final status and declared assertions whose URL
equals the requested or final URL. No matching declared test is a failed test,
not a claim that the site has been verified.
