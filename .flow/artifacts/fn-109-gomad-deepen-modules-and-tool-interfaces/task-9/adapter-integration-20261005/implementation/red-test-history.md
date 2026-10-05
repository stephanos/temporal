# Initial regression source recovery

`adapter_source_set_test.red.txt` retains the exact initial public-boundary and child test source. Its observed SHA-256 `4398a439879a34b3d75958ffc67264e23f827a8053302a580d7d6914130c3e1b` matches both `public-bounds-red/before.json` and `after.json`. The current final test remains unchanged at `39613c9c1510def7aef69b191792f4d6590249e4ddb486f0918c07dc506a946c`.

The first intervening edit added hostexec/build imports, request/projection, decode/source-failure and execution-precedence tests, and their two fixture helpers. Gofmt expanded the initial public-boundary and child functions' inline blocks and struct fields and aligned their fields. The later edit added the os/exec import, the real exit-7 fixture and two execution-precedence table entries. Neither edit changed the initial two functions' behavior or regression assertions.

`red-test-fidelity.json` records equality of both function bodies after ignoring whitespace and explicit struct-field separators outside string/rune literals. The comparison preserves literal contents and every identifier, operator, delimiter and assertion. All five boundary cases, 4 MiB limit, child argv, valid-JSON payload and space padding, marker, 10-second context, reaping/GOPATH checks and overflow type/stream/limit/empty-digest assertions remain identical.

Recovery copied the original apply_patch payload into a new evidence file and matched its recorded hash. It executed no Go, cache, test, lint, Git or Flow command and changed no current source or earlier capture. The source/Go/cache lane remains released to the conductor.
