# Site packs

These versioned local packs are opt-in. Select one with `--pack` for the CLI
or `RULE_PACKS` for the server. Pack loading and checking stay local; fetching
or testing a URL makes a network request and is non-hermetic.

## Wikipedia 1.0.1

`wikipedia/1.0.1.yaml` removes English Wikipedia's language picker, page tools,
table of contents, navigation sidebars, category controls and edit controls
before reader extraction. In Vector 2022 the language picker sits inside the
main article container, so ordinary navigation removal leaves its language
list ahead of the article. The pack targets that picker explicitly.

Article headings, prose, infoboxes, data tables, links, footnotes and reference
lists remain available to the reader. Navigation tables with the `navbox` or
`sidebar` class are removed. The pack identifies Ladder through its User-Agent
and project URL and removes forwarded-address and referrer headers.

```sh
ladder rules check --pack sitepacks/wikipedia/1.0.1.yaml
ladder fetch https://en.wikipedia.org/wiki/HTTP --pack sitepacks/wikipedia/1.0.1.yaml --format text
ladder rules test https://en.wikipedia.org/wiki/HTTP --pack sitepacks/wikipedia/1.0.1.yaml
```

Declared URL tests cover HTTP, Proxy server and Encyclopedia with status and
article-content checks plus missing-menu checks. They run only when requested;
declaration or a successful local pack check does not verify a live page.
The synthetic regression test runs the pack and actual reader together to
check both menu removal and retained article content without a network request:

```sh
go test ./pkg/sitepack -run TestWikipediaReader
```

The existing sample at `pkg/sitepack/testdata/wikipedia.yaml` remains version
1.0.0. Select one Wikipedia version at a time because pack IDs must be unique.
Once a version has been used, retain its exact bytes and SHA-256 and create a
new versioned file for subsequent changes. See `pkg/sitepack/README.md` for the
pack schema, matching rules and transformation behavior.
