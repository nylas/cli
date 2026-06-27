// Sweep every list method over one connection and print a pass/fail summary.
// Run: node list-sweep.js   (token comes from NYLAS_WS_TOKEN or `nylas rpc token`)
const { execFileSync } = require("node:child_process");

const token = process.env.NYLAS_WS_TOKEN || execFileSync("nylas", ["rpc", "token"]).toString().trim();
const ws = new WebSocket(`ws://127.0.0.1:7369/ws?token=${token}`);

// [method, params, result-field-to-count]
const calls = [
  ["config.read", null, null],
  ["grant.list", null, "grants"],
  ["calendar.list", null, "calendars"],
  ["email.list", { limit: 3 }, "messages"],
  ["thread.list", { limit: 3 }, "threads"],
  ["contact.list", { limit: 3 }, "contacts"],
  ["contact.group.list", null, "groups"],
  ["email.folder.list", null, "folders"],
  ["draft.list", { limit: 3 }, "drafts"],
  ["notetaker.list", null, "notetakers"],
  ["template.list", null, "templates"],
  ["audit.list", { limit: 3 }, "entries"],
  ["agentAccount.list", null, "accounts"],
  ["email.signature.list", null, "signatures"],
  ["email.scheduled.list", null, "scheduled"],
];

let id = 0;
const pending = new Map();
function call(method, params) {
  return new Promise((resolve) => {
    const myId = ++id;
    pending.set(myId, resolve);
    const req = { jsonrpc: "2.0", id: myId, method };
    if (params) req.params = params;
    ws.send(JSON.stringify(req));
  });
}

ws.addEventListener("message", (ev) => {
  const msg = JSON.parse(ev.data.toString());
  const resolve = pending.get(msg.id);
  pending.delete(msg.id);
  if (resolve) resolve(msg);
});

ws.addEventListener("open", async () => {
  console.log("connected\n");
  let firstCalendarId = null;
  let firstMessageId = null;

  for (const [method, params, field] of calls) {
    const msg = await call(method, params);
    if (msg.error) {
      console.log(`✗ ${method}  ERROR ${msg.error.code}: ${msg.error.message}`);
      continue;
    }
    if (field && Array.isArray(msg.result?.[field])) {
      const arr = msg.result[field];
      console.log(`✓ ${method}  → ${arr.length} ${field}`);
      if (method === "calendar.list" && arr[0]) firstCalendarId = arr[0].id;
      if (method === "email.list" && arr[0]) firstMessageId = arr[0].id;
    } else {
      console.log(`✓ ${method}  → { ${Object.keys(msg.result || {}).join(", ")} }`);
    }
  }

  // follow-up get-by-id calls using ids captured above
  for (const [method, params] of [
    ["calendar.get", firstCalendarId && { calendar_id: firstCalendarId }],
    ["email.get", firstMessageId && { message_id: firstMessageId }],
  ]) {
    if (!params) continue;
    const msg = await call(method, params);
    if (msg.error) console.log(`✗ ${method}  ERROR ${msg.error.message}`);
    else console.log(`✓ ${method}  → { ${Object.keys(msg.result || {}).join(", ")} }`);
  }

  ws.close();
});

ws.addEventListener("close", () => console.log("\nclosed"));
ws.addEventListener("error", (e) => { console.log("ws error:", e.message || e); process.exit(1); });
