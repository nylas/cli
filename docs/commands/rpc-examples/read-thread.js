// Find a multi-message thread and expand every message in it.
// Run: node read-thread.js   (token comes from NYLAS_WS_TOKEN or `nylas rpc token`)
const { execFileSync } = require("node:child_process");

const token = process.env.NYLAS_WS_TOKEN || execFileSync("nylas", ["rpc", "token"]).toString().trim();
const ws = new WebSocket(`ws://127.0.0.1:7369/ws?token=${token}`);

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
  if (typeof resolve === "function") resolve(msg);
});

ws.addEventListener("open", async () => {
  const list = await call("thread.list", { limit: 50 });
  const threads = list.result?.threads || [];
  console.log(`scanned ${threads.length} threads`);

  const multi = threads
    .map((t) => ({ t, n: (t.message_ids || []).length }))
    .filter((x) => x.n > 1)
    .sort((a, b) => b.n - a.n);

  if (multi.length === 0) {
    console.log("no multi-message threads in the first 50");
    ws.close();
    return;
  }

  const { t, n } = multi[0];
  console.log(`\n=== thread ${t.id} (${n} messages) ===`);
  console.log("subject     :", t.subject);
  console.log("participants:", (t.participants || []).map((p) => p.email || p.name).join(", "));
  console.log("unread      :", t.unread, "| starred:", t.starred);

  const got = await call("thread.get", { thread_id: t.id });
  const mids = got.result?.message_ids || t.message_ids || [];
  console.log(`\nexpanding ${mids.length} messages:\n`);

  let idx = 0;
  for (const mid of mids) {
    idx++;
    const m = await call("email.get", { message_id: mid });
    if (m.error) {
      console.log(`  [${idx}] ${mid} ERROR ${m.error.message}`);
      continue;
    }
    const r = m.result;
    console.log(`  [${idx}/${mids.length}] ${r.date}`);
    console.log(`       from   : ${(r.from || []).map((f) => f.email).join(", ")}`);
    console.log(`       subject: ${r.subject}`);
    console.log(`       snippet: ${(r.snippet || "").replace(/\s+/g, " ").slice(0, 130)}`);
    console.log("");
  }
  ws.close();
});

ws.addEventListener("close", () => console.log("closed"));
ws.addEventListener("error", (e) => { console.log("ws error:", e.message || e); process.exit(1); });
