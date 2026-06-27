# RPC server — Node examples

Zero-dependency Node clients for the `nylas rpc serve` JSON-RPC WebSocket server
(see [`../../RPC.md`](../../RPC.md) for the full protocol and method reference).

These use Node's built-in global `WebSocket` (Node 21+), so there's nothing to install.

## Run

Start the server in one terminal:

```bash
nylas rpc serve            # binds 127.0.0.1:7369
```

Then run an example — each script fetches the token itself via `nylas rpc token`
(or honors `NYLAS_WS_TOKEN` if you set it):

```bash
node list-sweep.js
node read-thread.js
```

| Script | What it does |
|--------|--------------|
| `list-sweep.js` | Calls every `*.list` method + a follow-up `get`, prints a pass/fail summary |
| `read-thread.js` | Finds a multi-message thread and expands each message |
