#!/usr/bin/env python3
"""Probe suspected-but-undocumented Miro API paths and report status drift.

Usage:
    MIRO_ACCESS_TOKEN=... python3 live-probe.py <probe-watchlist.json>

Why this exists: the spec baseline (synced from miroapp/api-clients) omits
endpoints Miro actually ships. Comments answered a plain PAT for weeks while
carrying zero operations in the spec, and work was wrongly deferred on the
spec's word alone (bead 7rh). So 'absent from spec' must never be recorded as
'absent from API' without a live probe — this script is that probe.

Each watchlist entry records the last verified status (known_status). The
script re-issues the request and reports any deviation: a 400→200 flip means a
gated route went live, a 404→anything flip means a hosted-only surface reached
public REST. Exits 1 when any probe drifted so the api-tracking workflow can
file a GitHub issue; exits 0 when all statuses match; exits 2 on configuration
errors (missing token, no board to probe against).

Path placeholders resolved at run time:
    {board_id}  MIRO_PROBE_BOARD_ID, or the first board the token can see
    {team_id}   the token's team, from GET /v1/oauth-token
    {table_id}  the first data table on the probe board (entry skipped if none)
"""

import json
import os
import sys
import urllib.error
import urllib.request

API_ROOT = "https://api.miro.com"
TIMEOUT_SECONDS = 30


def http_status(url, token):
    """Issue an authenticated GET and return the status code alone.

    Bodies are deliberately ignored: the watchlist tracks route existence and
    gating, and the status code carries that entire signal.
    """
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}"})
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
            return resp.status
    except urllib.error.HTTPError as e:
        return e.code


def fetch_json(url, token):
    """GET a JSON document, or None on any HTTP error."""
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}"})
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
            return json.load(resp)
    except (urllib.error.HTTPError, urllib.error.URLError, json.JSONDecodeError):
        return None


def resolve_board_id(token):
    """Return the board to probe against: env override, else first visible."""
    board_id = os.environ.get("MIRO_PROBE_BOARD_ID", "").strip()
    if board_id:
        return board_id
    boards = fetch_json(f"{API_ROOT}/v2/boards?limit=1", token)
    if boards and boards.get("data"):
        return boards["data"][0]["id"]
    return None


def resolve_team_id(token):
    """Return the token's team id via GET /v1/oauth-token, or None."""
    info = fetch_json(f"{API_ROOT}/v1/oauth-token", token)
    if info and isinstance(info.get("team"), dict):
        return info["team"].get("id")
    return None


def resolve_table_id(token, board_id):
    """Return the first data table id on the probe board, or None."""
    tables = fetch_json(
        f"{API_ROOT}/v2/boards/{board_id}/data_table_formats", token
    )
    if tables and tables.get("data"):
        return tables["data"][0].get("id")
    return None


def substitute(path, values):
    """Fill path placeholders; return None if a needed value is missing."""
    for key, value in values.items():
        placeholder = "{" + key + "}"
        if placeholder in path:
            if not value:
                return None
            path = path.replace(placeholder, str(value))
    return path


def run_probes(probes, token, values):
    """Probe every resolvable entry; return (rows, drifted, skipped)."""
    rows, drifted, skipped = [], [], []
    for probe in probes:
        path = substitute(probe["path"], values)
        if path is None:
            skipped.append(probe)
            continue
        status = http_status(f"{API_ROOT}{path}", token)
        row = {**probe, "resolved_path": path, "live_status": status}
        rows.append(row)
        if status != probe["known_status"]:
            drifted.append(row)
    return rows, drifted, skipped


def print_report(rows, drifted, skipped):
    """Emit the Markdown report the workflow pastes into the issue."""
    print("## Live probe report")
    print()
    if drifted:
        print("### Drift detected — the API moved where the spec is silent")
        print()
        for row in drifted:
            print(
                f"- **{row['name']}**: `{row['method']} {row['path']}` answered "
                f"**{row['live_status']}** (was {row['known_status']}). {row['signal']}"
            )
        print()
    print("### All probes")
    print()
    print("| Probe | Path | Known | Live | Drift |")
    print("|-------|------|-------|------|-------|")
    for row in rows:
        drift = "**YES**" if row["live_status"] != row["known_status"] else "no"
        print(
            f"| {row['name']} | `{row['path']}` | {row['known_status']} "
            f"| {row['live_status']} | {drift} |"
        )
    for probe in skipped:
        print(f"| {probe['name']} | `{probe['path']}` | {probe['known_status']} | skipped | — |")
    print()
    print(
        "_A probe that drifted means a route the OpenAPI spec does not document "
        "changed behavior. Verify by hand, then update `known_status` in "
        "`api-tracking/probe-watchlist.json` (and file a tool bead if it went live)._"
    )


def main():
    if len(sys.argv) != 2:
        print("Usage: live-probe.py <probe-watchlist.json>", file=sys.stderr)
        sys.exit(2)

    token = os.environ.get("MIRO_ACCESS_TOKEN", "").strip()
    if not token:
        print("MIRO_ACCESS_TOKEN is not set; live probes need a token.", file=sys.stderr)
        sys.exit(2)

    with open(sys.argv[1], encoding="utf-8") as f:
        watchlist = json.load(f)

    board_id = resolve_board_id(token)
    if not board_id:
        print("No board to probe against (token sees no boards and "
              "MIRO_PROBE_BOARD_ID is unset).", file=sys.stderr)
        sys.exit(2)

    values = {
        "board_id": board_id,
        "team_id": resolve_team_id(token),
        "table_id": resolve_table_id(token, board_id),
    }

    rows, drifted, skipped = run_probes(watchlist["probes"], token, values)
    print_report(rows, drifted, skipped)
    sys.exit(1 if drifted else 0)


if __name__ == "__main__":
    main()
