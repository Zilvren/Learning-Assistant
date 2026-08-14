# @zilvren/dsh-learning-library

Restricted learning-library tools for [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness).

The plugin exposes only five bridge-backed tools:

- `list_library_paths`
- `search_library`
- `read_library_note`
- `create_library_note`
- `update_library_note`

It deliberately contains no filesystem, shell, move, delete, or force-overwrite
tool. Your host application owns authorization, data scope, and persistence.
`create_library_note` creates only a new path; `update_library_note` requires
the exact version returned by `read_library_note`, so it refuses to overwrite a
newer user change.

## Install

After this package is published, install it alongside a compatible DeepSeek
Harness runtime:

```sh
npm install @zilvren/dsh-learning-library @deepseek-ai/dsh-tools
```

Configure the Harness plugin with environment-variable *names*, never with a
literal bridge URL or capability token:

```yaml
- id: learning-library
  name: '@zilvren/dsh-learning-library'
  config:
    bridgeUrlEnv: DSH_LEARNING_LIBRARY_BRIDGE_URL
    tokenEnv: DSH_LEARNING_LIBRARY_CAPABILITY_TOKEN
    toolPathPrefix: /v1/learning-library/tools
    requestTimeoutMs: 45000
```

Set both variables in the process that starts Harness. The bridge URL must use
HTTPS except for a loopback (`127.0.0.1`, `localhost`, or `::1`) HTTP address.
The plugin refuses URLs containing credentials, queries, or fragments.

## Bridge contract

For each tool, the plugin sends:

```text
POST <bridge URL><toolPathPrefix>/<bridge tool name>
Authorization: Bearer <capability token>
Content-Type: application/json
```

The request body is the tool arguments. A successful response is JSON with a
`result` field. Non-2xx responses may return a `detail`, `error.message`, or
`message` for a safe user-facing error.

| Harness tool | Bridge tool name |
| --- | --- |
| `list_library_paths` | `list_paths` |
| `search_library` | `search` |
| `read_library_note` | `read_note` |
| `create_library_note` | `create_note` |
| `update_library_note` | `update_note` |

## Development and release

```sh
npm test
npm pack --dry-run
```

`npm test` includes a Harness-entrypoint check in addition to bridge-client
tests. The package declares `@deepseek-ai/dsh-tools` as a peer dependency and
uses the same version as a development dependency for that check.

This package is currently marked `UNLICENSED`, so do not publish it publicly
until you select and add an open-source license. Once published in a public GitHub
repository, add the `dsh-plugin` topic for DeepSeek Harness discoverability.
