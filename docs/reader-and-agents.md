# Reader, agent tools and local controls

Build from the repository root with `go build -mod=readonly -o bin/ladder cmd/main.go`.
All reader conversion happens locally. Ladder has no telemetry, account requirement,
external conversion service or browser extension requirement.

## Use the reader

```sh
bin/ladder serve --host 127.0.0.1 --port 8080
```

Open `http://localhost:8080`. Paste a URL and choose Read. The reader offers HTML,
Markdown and plain text, Copy Markdown, refresh and the original source. Metadata
shows the requested and final URLs, actual upstream status, retrieval time and
cache freshness. An upstream refusal or error remains visible. Extraction uses
HTML heuristics; it cannot render JavaScript or recover content the server did
not supply. Images become their descriptions instead of loading remote assets.

The bookmarklet opens the current page's URL in this reader. Drag it from the
reader's controls into Chrome's bookmarks bar. It navigates to Ladder without
reading the page's contents, cookies or browser credentials.

Legacy proxy and raw routes remain available. Raw preserves the fetched body,
content type and upstream status. Active documents receive a sandbox policy to
isolate them from reader controls and history; sites that depend on scripts,
forms, cookies or browser storage can behave differently. Use Open original to
visit a site directly.

## Fetch from a terminal or agent

```sh
bin/ladder fetch 'https://example.org/article' --format markdown
bin/ladder fetch 'https://example.org/article' --format text
bin/ladder fetch 'https://example.org/article' --format html
bin/ladder fetch 'https://example.org/article' --json --refresh
```

These commands call the same retrieval service directly. The web server does
not need to be running. JSON includes the source metadata, the selected format and its content.
Choose `--format` to retrieve another representation. Invalid input, acquisition failures and upstream HTTP failures
have nonzero exit codes. Quote URLs so a shell cannot interpret their query
characters. The HTML format contains sanitized reader content.

## Connect an MCP client

Run `bin/ladder mcp --stdio` as a local MCP server. Configure an MCP client with
the absolute binary path and arguments `["mcp", "--stdio"]`. Set `RULESET` and
`RULE_PACKS` in that client's environment if required. Stdio messages use stdout;
diagnostics use stderr. The tools use the shared retrieval service without
requiring the web server or saving web history.

The implementation supports current discovery with protocol version `2026-07-28`
and legacy `2025-11-25` initialization. This is a focused tools connector, not an
HTTP MCP server or a general plugin host. See the [MCP specification](https://modelcontextprotocol.io/specification/2026-07-28)
for client lifecycle details.

## Versioned site-rule packs

`RULESET` continues to load existing Ladder rules. Versioned packs are explicit,
local YAML files selected through `RULE_PACKS` (separate multiple paths with `;`)
or the command's `--pack` option. Packs carry a schema version, stable ID,
version, rules and expected content checks. Ladder displays the SHA-256 digest
of the exact file. Changing a pack changes its digest; keep a new version when
sharing an update. No pack is downloaded or updated automatically, and packs do
not execute commands.

For example, the included Wikipedia pack can be validated and tested with:

```sh
bin/ladder rules check pkg/sitepack/testdata/wikipedia.yaml
bin/ladder rules test --pack pkg/sitepack/testdata/wikipedia.yaml \
  https://en.wikipedia.org/wiki/HTTP
```

The [pack schema](../pkg/sitepack/README.md) documents matching, transformations,
validation and test coverage. Validate a pack before using it, then run its URL
checks. The web Rules controls
show the loaded versions and let you test a URL with the same service. A passing
schema check is separate from a successful live content check.

## History and caching

The web reader saves up to 100 URL/title/time records on this Ladder server.
It does not persist article bodies. History can be disabled, individual records
deleted, or cleared. Disabling history stops new records; Clear history removes
existing ones. Treat stored URLs as private, since they can contain query data.

`LADDER_STATE_DIR` overrides the default user configuration directory's `ladder`
folder. State directories use mode `0700` and history files `0600`. Updates are
atomic and locked across processes. CLI and MCP calls do not add history.

The retrieval cache is bounded, in memory, and defaults to five minutes. The UI
shows whether a response was reused and its expiry. Refresh skips a cached
response. Clear cache removes process-local entries; restarting also clears
them. Private or credential-bearing responses are excluded. The cache is an
optimization, not a guarantee that an upstream page has stayed unchanged.

## Startup is optional

Starting the server in a terminal keeps it running until that process is
stopped. It is not automatically installed as a login service.

```sh
bin/ladder startup status
bin/ladder startup enable --port 8080
bin/ladder startup disable
```

Enable is an explicit action. It installs a per-user launchd job on macOS or a
systemd user service on Linux and binds it to loopback. Status reports manager
state separately from the presence of a configuration file. Keep the installed
binary at its configured absolute path. The generated service uses its own
bind/port configuration; terminal-only rules and authentication environment
settings are not automatically persisted into it. Linux user-service startup depends on
that user's systemd session. Other platforms can run the server manually.

## Configuration

| Variable | Purpose |
| --- | --- |
| `BIND_ADDR` | Bind address; set `127.0.0.1` for local use |
| `PORT` | Server port; default `8080` |
| `RULESET` | Existing rules, separated with `;` |
| `RULE_PACKS` | Explicit local versioned pack paths, separated with `;` |
| `HTTP_TIMEOUT` | Upstream timeout in seconds |
| `CACHE_TTL` | Go duration such as `5m` |
| `CACHE_MAX_ENTRIES` | Maximum process-local cache entries |
| `LADDER_STATE_DIR` | Private local history/preferences directory |
| `USERPASS` | Existing optional HTTP Basic authentication |
| `TRUSTED_HOSTS` | Additional hostnames allowed to access local controls |

Control APIs allow `localhost`, `127.0.0.1` and `::1` by default. Add a custom
hostname such as `ladder.local` to `TRUSTED_HOSTS` when using it. This pins the
management host independently of a browser Origin header.

New control APIs require `X-Ladder-Client: reader` and reject cross-origin and
opaque-origin browser requests. Do not expose an unauthenticated reader server
to an untrusted network. The CLI does not import browser logins. Existing
optional FlareSolverr configuration remains separate; no renderer is installed
by the reader feature.
