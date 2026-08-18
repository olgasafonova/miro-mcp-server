# API Tracking

Weekly detection of Miro API surface changes, in two independent layers:

| Layer | Source of truth | Script | Catches |
|-------|-----------------|--------|---------|
| Spec diff | `spec.json` from [miroapp/api-clients](https://github.com/miroapp/api-clients) | `diff-spec.py` | Documented changes: new/removed endpoints, schema changes, narrowed types |
| Live probe | The API itself | `live-probe.py` | Undocumented changes: routes the spec has never carried |

Both run in [`api-tracking.yml`](../.github/workflows/api-tracking.yml) every Monday; either one detecting a change files a GitHub issue.

## The rule: absent from spec is NOT absent from API

The spec baseline omits endpoints Miro actually ships. The proof is comments:
the spec carries **zero** comment operations, yet `GET` and `POST`
`/v2-experimental/boards/{id}/comments` answered a plain personal access token
when probed on 13-08-2026 — and had been doing so for weeks while a comment-tools
bead sat wrongly deferred on the spec's word alone.

So: **never record "absent from spec" as "absent from API" without a live
probe.** A `curl` with a PAT settles it in seconds. Read the status precisely:

| Status | Meaning |
|--------|---------|
| `404` `"No endpoint"` | The route does not exist in the gateway — genuinely absent today |
| `400` `"Validation failure"` | The route EXISTS but is feature-gated or needs a parameter you did not guess |
| `200` / `4xx` with domain error | The endpoint is live — spec silence notwithstanding |

## The probe watchlist

`probe-watchlist.json` holds every suspected-but-undocumented path with its
last verified status. `live-probe.py` re-issues each request and exits 1 on
any deviation — a `400 → 200` flip means a gated route went live, a `404 →`
anything flip means a hosted-connector-only surface reached public REST.

Run it locally:

```bash
MIRO_ACCESS_TOKEN=... python3 api-tracking/live-probe.py api-tracking/probe-watchlist.json
```

`MIRO_PROBE_BOARD_ID` pins the board used for `{board_id}` paths; without it
the first board the token can see is used. `{team_id}` comes from
`GET /v1/oauth-token`, `{table_id}` from the probe board's first data table.

When a probe drifts: verify by hand, update the entry's `known_status`, and
file a bead if a new surface went live. When you *suspect* a new path (a
hosted-connector tool with no REST twin, a URL seen in item payloads), add a
watchlist entry rather than a mental note — the workflow then watches it every
Monday.

In CI the probe step needs a `MIRO_PROBE_TOKEN` repository secret; without it
the step is skipped and only the spec diff runs.

## The official-surface baseline

`official-mcp-surface.json` records the tool names the **official hosted Miro
MCP server** registers, with an enumeration history. The README's
Official-vs-Community comparison cites its count. Unlike the spec diff and the
live probes, this one cannot run in CI: enumeration needs an authenticated
claude.ai (or MCP OAuth) session against `mcp.miro.com`. Re-enumerate monthly
or when Miro announces — diff against the file, update it and the README
comparison on drift (bead 6kn is the recurring tracker).
