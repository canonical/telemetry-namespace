# Telemetry Namespace

`github.com/canonical/telemetry-namespace` is the reference implementation of the Telemetry MQTT
namespace. It is shared by the Connect Agent, the Telemetry Management Service (TMS), the connector
service and the Telemetry CLI, so that every component builds, parses and validates topics in exactly the
same way.

It implements the specs TM102 (Topic Namespace & User Properties) and TM103 (Topic Selectors).

## Packages

| Package | Provides |
|---|---|
| `topic` | Wire topic build and parse (`<tenant>/devices\|groups\|_system/...`), identifier, publisher and local-topic validation, platform topics |
| `match` | MQTT topic-name and filter validation, filter matching and containment, shared subscriptions |
| `fga` | Encoding of topic filters as OpenFGA object IDs |
| `selector` | Topic selector parser (`[devices: d1]/t`, `[groups: g1]/t`, ...) and resolution to wire topics |
| `props` | `cc-*` MQTT 5 user property names and helpers |
| `system` | JSON payloads published under `<tenant>/_system/...` |

Packages land incrementally; see the [issues](https://github.com/canonical/telemetry-namespace/issues).

## Goals

- **One implementation.** Components must use this module rather than re-implement the namespace rules.
- **No runtime dependencies.** Only the Go standard library is imported by non-test code.
- **Small, stable API.** Packages expose plain functions and value types, with no I/O and no global state.

## Versioning

The module follows [semantic versioning](https://semver.org/), and releases are cut automatically from
[conventional commits](https://www.conventionalcommits.org/) on `main`.

- `v0.x` until the namespace v2 release: the API may change between minor versions.
- `v1.0.0` with the namespace v2 release, after which breaking changes need a new major version.

## Development

Requirements: Go (version from `go.mod`) and [golangci-lint](https://golangci-lint.run/) v2.

```bash
go test -race ./...       # tests, as run in CI
go test -cover ./...      # coverage
golangci-lint run ./...   # lint, as run in CI
```

CI runs lint and test, a code scan (gosec, Trivy) and a repo scan (Gitleaks, zizmor) on every pull
request, and semantic-release on `main`. The workflows mirror the team's shared SSDLC workflows, inlined
because a public repository can't call reusable workflows from an internal one.

## Security

See [SECURITY.md](./SECURITY.md).

## License

Copyright 2026 Canonical Ltd.

Licensed under the [Apache License, Version 2.0](./LICENSE).
