# MCP Protocol Audit Guide

**Purpose.** This document lets an agent walk into any stdio-based MCP server (Go, Python, TypeScript, hand-rolled or SDK-based) and audit it for the protocol-compliance issues that cause recent Claude Code versions to silently drop the connection. It captures the research, the specific issues found in this repo (todo-mcp v2.4.1 → v2.4.2), and a verification recipe.

**Scope.** stdio transport only. HTTP/SSE transport, sampling, elicitation, and resources/prompts are out of scope — but the audit checklist still applies as a baseline.

**Audience.** An agent in a sibling MCP repo that has been told "audit and fix MCP connection issues." Read sections 1–2 for context, run the checklist in section 3, apply matching fixes from section 4, then run section 5 to verify.

---

## 1. Background: what changed and why connections fail

The Model Context Protocol publishes dated spec revisions. The relevant ones, in order:

| Version       | Status as of April 2026                                   |
| ------------- | --------------------------------------------------------- |
| `2024-11-05`  | Original GA — the safe lowest common denominator          |
| `2025-03-26`  | Streamable HTTP transport added                           |
| `2025-06-18`  | **Current widely-supported baseline for Claude Code**     |
| `2025-11-25`  | Latest — Tasks, parallel tool calls, server-side agents   |

Claude Code negotiates **downward**: it advertises its newest supported version in the `initialize` request; the server replies with a version *it* supports. If the server returns an unrecognized future version, or fails strict validation of the response, Claude Code drops the server silently.

**The 2025-06-18 changes most likely to break older servers:**

- **JSON-RPC batching removed.** Servers MUST process one JSON object per stdio line. SDKs that still emit batches will be rejected.
- **`outputSchema` on tools, `structuredContent` on tool results.** Optional, but if you include them, they must be valid.
- **Stricter `initialize` result validation.** Extra top-level fields beyond what the spec defines may be rejected by the client's Zod schemas (see Claude Code [issue #768](https://github.com/anthropics/claude-code/issues/768)).
- **`MCP-Protocol-Version` HTTP header required** on subsequent HTTP requests. Stdio is unaffected.
- **Several lifecycle SHOULDs upgraded to MUSTs.**

**The expected stdio handshake:**

1. Client → server (one JSON object per line, UTF-8, no batching):
   ```json
   {"jsonrpc":"2.0","id":1,"method":"initialize","params":{
     "protocolVersion":"2025-06-18",
     "capabilities":{"roots":{"listChanged":true},"sampling":{}},
     "clientInfo":{"name":"claude-code","version":"2.x"}}}
   ```
2. Server → client (echo same id, return a version the server actually supports):
   ```json
   {"jsonrpc":"2.0","id":1,"result":{
     "protocolVersion":"2025-06-18",
     "capabilities":{"tools":{"listChanged":true}},
     "serverInfo":{"name":"my-server","version":"1.0.0"},
     "instructions":"(optional system-prompt-style guidance)"}}
   ```
   The result object MUST contain `protocolVersion`, `capabilities`, `serverInfo`. It MAY contain `instructions`. **Any other top-level field is non-spec** and risks rejection.
3. Client → server (notification, no id, no response expected):
   ```json
   {"jsonrpc":"2.0","method":"notifications/initialized"}
   ```
4. Client then sends `tools/list`, `resources/list`, etc.

**The cardinal stdio rule:** stdout is JSON-RPC only, newline-delimited. Logs go to stderr. **Any** stray write to stdout (a `print`, `fmt.Printf`, `console.log`, `puts`) corrupts the stream and the client drops the server on the next read.

---

## 2. Symptoms of a failed handshake

If a user reports "MCP server won't connect to the latest Claude Code," check for these symptoms:

| Symptom                                                  | Likely cause                                                    |
| -------------------------------------------------------- | --------------------------------------------------------------- |
| Server "appears connected" then disappears mid-session   | Stdout pollution from a code path the user just exercised       |
| Server never appears in the connected list at all        | Initialize result fails Zod validation (extra/missing fields)   |
| `protocolVersion: undefined` errors in Claude Code logs  | Server didn't echo a `protocolVersion` string in the response   |
| Spurious `{"id":null}` responses in transport logs       | Server is replying to notifications (it must not)               |
| Tools missing from the tool list, no error               | A tool's `inputSchema` is invalid JSON Schema, or `outputSchema`/`title`/`annotations` triggered Claude Code [#25081](https://github.com/anthropics/claude-code/issues/25081) |
| Connection succeeds, first tool call kills the server    | A handler writes to stdout (most common: warnings, debug prints)|
| Server with 50+ tools never finishes initializing        | Many-tool startup timeout — Claude Code [#38462](https://github.com/anthropics/claude-code/issues/38462) |

---

## 3. Audit checklist

Run these checks on any stdio MCP server. They take ~10 minutes total.

### 3.1 Stdout pollution (highest priority)

In Go:
```bash
grep -rn 'fmt\.Print\|fmt\.Println\|println\b' --include='*.go' \
  | grep -v _test.go | grep -v 'Fprintf\|Fprintln\|Fprint(os'
```
In Python:
```bash
grep -rn 'print(' --include='*.py' | grep -v 'file=sys.stderr' | grep -v _test.py
```
In TypeScript/Node:
```bash
grep -rn 'console\.log\|process\.stdout\.write' --include='*.ts' --include='*.js' \
  | grep -v _test
```

Every match in a code path reachable from `tools/call`, `tools/list`, or `initialize` is a connection-killer. The CLI/help/version paths that run *before* stdio mode is entered are fine — distinguish carefully.

### 3.2 Initialize response shape

Find the initialize handler and verify the result object contains **exactly**:
- `protocolVersion` (string) — required
- `capabilities` (object) — required
- `serverInfo` (object with `name` + `version`) — required
- `instructions` (string) — optional
- `_meta` (object) — optional, since 2025-06-18

If the handler returns any other top-level field (especially a `tools` array embedded in the initialize result — a common mistake), remove it. Tools belong only in `tools/list`.

### 3.3 protocolVersion negotiation

Find where the server picks the protocolVersion to return. Anti-pattern: blindly echoing whatever the client sent. Correct pattern: respond with the highest version *the server* supports, optionally echoing the client's version if it matches one in the server's supported set.

A safe supported set for a basic tools-only server is:
```
{"2024-11-05", "2025-03-26", "2025-06-18"}
```
with `2025-06-18` as the default response.

### 3.4 Notification handling

Find every code path that processes a parsed JSON-RPC request. Verify each path filters out messages where `id` is missing/null **before** sending a response. Notifications MUST NOT receive a response per JSON-RPC 2.0; sending `{"id":null,"result":{}}` will be rejected by strict clients.

In particular, `notifications/initialized` arrives right after the handshake and is the most common case to break.

### 3.5 Tool definitions

For every tool advertised in `tools/list`:
- `name` (string) — required, programmatic identifier
- `description` (string) — required
- `inputSchema` (object) — required, **must be a valid JSON Schema object**, even for zero-arg tools. Use `{"type":"object","properties":{},"additionalProperties":false}` — never `null`, never absent.

Optional fields to be careful with:
- `title`, `annotations`, `outputSchema` — these are valid in 2025-06-18+ but caused tool-drop bugs in older Claude Code releases ([#25081](https://github.com/anthropics/claude-code/issues/25081)). If you don't need them, leave them out for the widest compatibility.

### 3.6 JSON-RPC plumbing

- Every response includes `"jsonrpc": "2.0"`.
- Errors use the standard codes: `-32700` parse error, `-32600` invalid request, `-32601` method not found, `-32602` invalid params, `-32603` internal error.
- The server does NOT batch responses (one JSON object per line on stdout).
- The reader handles both raw line-delimited JSON and `Content-Length:`-framed messages, OR documents which it requires. Claude Code uses raw line-delimited for stdio.

### 3.7 Tool result format

`tools/call` results should look like:
```json
{
  "content": [{"type": "text", "text": "..."}],
  "isError": false
}
```
`content` is an array (required). `isError` is optional and defaults to false. `structuredContent` (object) is optional and was added in 2025-06-18.

---

## 4. The four fixes from todo-mcp v2.4.1 → v2.4.2

Concrete patterns an agent can pattern-match against in another repo. File paths reference todo-mcp; equivalent locations in other servers will differ.

### Fix 1: Stdout pollution in a tool handler 🔴 Critical

**File:** `internal/server/handlers.go:2367` (was inside `handleTodoTagAdd`)

**Before:**
```go
if similarity > 0.85 && newTag != existing {
    fmt.Printf("Warning: Tag '%s' is %.0f%% similar to existing '%s'\n",
        newTag, similarity*100, existing)
}
```

**After:**
```go
if similarity > 0.85 && newTag != existing {
    fmt.Fprintf(os.Stderr, "Warning: Tag '%s' is %.0f%% similar to existing '%s'\n",
        newTag, similarity*100, existing)
}
```

**Why it broke things:** `fmt.Printf` writes to stdout, which is the JSON-RPC transport channel. Mid-`tools/call` response, that warning string would land between or inside JSON responses, causing the client to fail to parse and drop the server. Every other log site in the file already used `os.Stderr` — this one was missed.

**How to find it elsewhere:** the grep in §3.1. Audit every match; promote each to stderr.

### Fix 2: Non-spec `tools` field in initialize response 🟡 Likely needed

**File:** `internal/server/handlers.go:251-863` (was inside `handleInitialize`)

**Before** (abbreviated):
```go
return map[string]interface{}{
    "protocolVersion": clientVersion,
    "capabilities": map[string]interface{}{"tools": map[string]interface{}{}},
    "serverInfo": map[string]interface{}{
        "name":    "todo-mcp",
        "version": Version,
    },
    "tools": []map[string]interface{}{   // ← non-spec, ~600 lines of tool defs
        {"name": "todo_list", ...},
        // ...
    },
}, nil
```

**After:**
```go
return map[string]interface{}{
    "protocolVersion": negotiatedVersion,
    "capabilities": map[string]interface{}{"tools": map[string]interface{}{}},
    "serverInfo": map[string]interface{}{
        "name":    "todo-mcp",
        "version": Version,
    },
}, nil
```

**Why it broke things:** the MCP spec defines exactly four allowable top-level keys in the initialize result (`protocolVersion`, `capabilities`, `serverInfo`, `instructions`). Claude Code's Zod schema rejects unknown fields. The tool list belongs only in the response to `tools/list`. The duplicate tool definitions also created a maintenance hazard — they had drifted slightly between the two locations.

**How to find it elsewhere:** open the initialize handler, look for any keys outside the four allowed.

### Fix 3: protocolVersion echo without validation 🟡 Defensive

**File:** `internal/server/handlers.go:213` (top of `handleInitialize`)

**Before:**
```go
clientVersion := "2024-11-05" // Default to older version
if paramsMap, ok := params.(map[string]interface{}); ok {
    if pv, ok := paramsMap["protocolVersion"].(string); ok {
        clientVersion = pv
        // ...
    }
}
// ... later: "protocolVersion": clientVersion
```

**After:**
```go
const serverMaxVersion = "2025-06-18"
supportedVersions := map[string]bool{
    "2024-11-05": true,
    "2025-03-26": true,
    "2025-06-18": true,
}
negotiatedVersion := serverMaxVersion
if paramsMap, ok := params.(map[string]interface{}); ok {
    if pv, ok := paramsMap["protocolVersion"].(string); ok {
        if supportedVersions[pv] {
            negotiatedVersion = pv
        }
        // ...
    }
}
// ... later: "protocolVersion": negotiatedVersion
```

**Why it broke things:** when Claude Code sends `"2025-11-25"` (or whatever future version), the old code echoed it back, claiming the server supports a spec it doesn't actually implement. Strict clients may then send messages the server can't handle. Correct behavior per the spec is to respond with the highest version *the server* knows, falling back to the client's version only if it's in the server's supported set.

**How to find it elsewhere:** look for `clientVersion = pv` or equivalent — anywhere the response version is taken directly from the request without a membership check.

### Fix 4: Notifications receive a stray response ⚪ Subtle

**File:** `internal/server/server.go:285` (in the async request goroutine inside `readLoop`)

**Before:**
```go
go func(req map[string]interface{}) {
    defer s.wg.Done()
    id := req["id"]
    method, ok := req["method"].(string)
    if !ok { ... return }
    // No id==nil filter here!
    result, err := s.routeMethod(method, req["params"])
    if err != nil { s.handleError(id, err); return }
    s.sendResponse(id, result)
}(request)
```

**After:**
```go
go func(req map[string]interface{}) {
    defer s.wg.Done()
    id := req["id"]
    method, ok := req["method"].(string)
    if !ok { ... return }

    // Notifications (no id) MUST NOT receive a response per JSON-RPC 2.0
    if id == nil {
        if s.verbose {
            fmt.Fprintf(os.Stderr, "Received notification: %s\n", method)
        }
        return
    }

    result, err := s.routeMethod(method, req["params"])
    if err != nil { s.handleError(id, err); return }
    s.sendResponse(id, result)
}(request)
```

**Why it broke things:** the codebase had two request-handling paths (a synchronous `handleRequest` and an async goroutine in `readLoop`). The synchronous one filtered notifications correctly; the async one didn't, so `notifications/initialized` triggered a `{"jsonrpc":"2.0","id":null,"result":{}}` response. Strict JSON-RPC 2.0 clients reject responses with null ids that don't correspond to a request.

**How to find it elsewhere:** find every place the server calls "send response" / "write to stdout with a JSON-RPC envelope" and trace back — does each path check for missing/null id? In a clean implementation there's one chokepoint; in a server that's grown organically, there may be two or three.

---

## 5. Verification recipe

After applying fixes, prove the handshake is clean. This works for any stdio MCP server — adjust the binary path.

```bash
# 1. Build the server.
go build -o /tmp/mcp-smoke ./cmd/your-mcp/

# 2. Feed it a canonical 3-message handshake.
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude-code","version":"2.0"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  > /tmp/mcp-input.txt

# 3. Run with stderr separated. Use timeout so the server's idle wait doesn't hang us.
timeout 4 /tmp/mcp-smoke --mode stdio < /tmp/mcp-input.txt \
  > /tmp/mcp-stdout.txt 2> /tmp/mcp-stderr.txt

# 4. Validate.
python3 <<'EOF'
import json
with open('/tmp/mcp-stdout.txt') as f:
    lines = [l.strip() for l in f if l.strip()]

# Expect exactly 2 responses: id=1 (init) and id=2 (tools/list). Notifications get nothing.
assert len(lines) == 2, f"expected 2 responses, got {len(lines)}: {lines}"

init, tools = json.loads(lines[0]), json.loads(lines[1])

# Init response must contain only the spec-compliant top-level keys.
allowed = {"protocolVersion", "capabilities", "serverInfo", "instructions", "_meta"}
extra = set(init["result"].keys()) - allowed
assert not extra, f"non-spec fields in initialize result: {extra}"

# protocolVersion must be a known string.
known = {"2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25"}
assert init["result"]["protocolVersion"] in known, init["result"]["protocolVersion"]

# serverInfo must have name + version.
si = init["result"]["serverInfo"]
assert "name" in si and "version" in si, si

# tools/list returns a tools array.
assert "tools" in tools["result"]

print("✓ initialize result keys:", sorted(init["result"].keys()))
print("✓ protocolVersion:", init["result"]["protocolVersion"])
print("✓ serverInfo:", si)
print("✓ tools count:", len(tools["result"]["tools"]))
print("PASS")
EOF
```

If all assertions pass, the server is spec-compliant for the basic stdio handshake. To stress-test individual handlers for stdout pollution, append `tools/call` messages for each tool to `/tmp/mcp-input.txt` and re-run; any tool whose handler writes to stdout will produce a response that fails to parse as JSON.

---

## 6. What this guide does NOT cover

- **HTTP/SSE transport.** The 2025-06-18 `MCP-Protocol-Version` header requirement, session ID handling, and SSE event framing are out of scope.
- **Sampling, elicitation, resources, prompts.** These have their own request/response shapes and capability flags. If your server advertises them, audit them against the spec separately.
- **Tool annotations and outputSchema.** Both are valid in 2025-06-18+ but caused regressions in older Claude Code. If you need them, test specifically against the Claude Code version your users run.
- **Tasks (2025-11-25).** Long-running work tracking is opt-in; only relevant if your tools take longer than the client's request timeout.
- **Server-side authentication, roots, and the filesystem capability.** Out of scope.

---

## 7. Sources

- [MCP specification index (2025-11-25)](https://modelcontextprotocol.io/specification)
- [MCP lifecycle / initialize (2025-11-25)](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle)
- [MCP tools spec (2025-11-25)](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)
- [2025-06-18 changelog](https://modelcontextprotocol.io/specification/2025-06-18/changelog)
- [Claude Code MCP docs](https://docs.claude.com/en/docs/claude-code/mcp)
- [Claude Code issue #768 — protocolVersion validation](https://github.com/anthropics/claude-code/issues/768)
- [Claude Code issue #25081 — outputSchema/title drops tools](https://github.com/anthropics/claude-code/issues/25081)
- [Claude Code issue #38462 — many-tool startup timeout](https://github.com/anthropics/claude-code/issues/38462)
- [Claude Code issue #21341 — multiple stdio servers](https://github.com/anthropics/claude-code/issues/21341)

---

## 8. Quick-reference card for the auditing agent

If you only have ten minutes, do this:

1. `grep -rn 'fmt\.Print\|console\.log\|^\s*print(' --include='*.go' --include='*.ts' --include='*.py'` and verify no match is reachable from a tool handler.
2. Open the initialize handler. Confirm the response has only `protocolVersion`, `capabilities`, `serverInfo` (and optionally `instructions`/`_meta`).
3. Confirm `protocolVersion` is selected from a server-defined supported set, not echoed blindly.
4. Find every "send response" call site; confirm each is preceded by an `id != nil` check.
5. Run the verification recipe in §5. If it passes, ship it.
