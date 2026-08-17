# DeepSeek Harness integration

This directory is the deployment-owned plugin project for the learning
assistant. It uses the official DeepSeek Harness JSON-RPC runtime, but mounts
only the model, agent loop, session persistence, compaction, and
`learning-library` tool plugin.

It deliberately excludes the official filesystem, shell, subprocess, and
generic editor plugins. The only available model tools call the local Go
application with a per-turn capability token. They can create a new note or
save an existing note only with the exact version it was read from; moves,
deletes, and force-overwrites are not exposed to the model.

Install the pinned official runtime dependencies with a compatible Node.js
version (22.19+):

```powershell
cd harness
npm install --registry=https://registry.npmmirror.com --no-audit
```

The Go server launches `dsh-jsonrpc-agent` from this directory for every AI
chat. The server passes the API key, local bridge URL, session root, and
short-lived tool token directly to the child process; none of them are written
to this project or sent to the browser.

The restricted tool implementation lives in
`../packages/dsh-learning-library`, which is shaped as a reusable npm package.
This deployment temporarily uses the small local adapter in `src/` so its
DeepSeek Harness peer dependency resolves from this directory. After the package
is published, replace the plugin name in `learning-agent.cordis.yml` with
`@zilvren/dsh-learning-library`; its bridge contract and configuration are
unchanged.

For a local launch, make sure the server can find the same Node.js 22.19+
runtime used for installation. If it is not the default `node` on your PATH,
set `STUDY_HARNESS_NODE` to the absolute `node` executable path before starting
the Go service:

```powershell
$env:STUDY_HARNESS_NODE = 'C:\absolute\path\to\node.exe' # only when needed
go run . --no-browser
```

Harness is the only AI path. If its runtime is unavailable, the AI page keeps
chat disabled and shows the reason returned by `GET /api/ai/harness`.
