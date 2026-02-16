#!/usr/bin/env python3
"""Compare two Miro OpenAPI specs and report differences.

Usage:
    python3 diff-spec.py <baseline.json> <current.json>

Outputs a Markdown report of added, removed, and changed endpoints.
"""

import json
import sys
from collections import defaultdict

HTTP_METHODS = {"get", "post", "put", "patch", "delete"}


def extract_endpoints(spec):
    """Extract a dict of (method, path) -> endpoint info from an OpenAPI spec."""
    endpoints = {}
    for path, methods in spec.get("paths", {}).items():
        for method, details in methods.items():
            if method not in HTTP_METHODS:
                continue
            key = (method.upper(), path)
            endpoints[key] = {
                "operationId": details.get("operationId", ""),
                "summary": details.get("summary", ""),
                "tags": details.get("tags", []),
                "deprecated": details.get("deprecated", False),
                "parameters": [
                    {
                        "name": p.get("name"),
                        "in": p.get("in"),
                        "required": p.get("required", False),
                    }
                    for p in details.get("parameters", [])
                ],
            }
    return endpoints


def extract_schemas(spec):
    """Extract schema names and their property lists."""
    schemas = {}
    for name, schema in spec.get("components", {}).get("schemas", {}).items():
        props = sorted(schema.get("properties", {}).keys())
        required = sorted(schema.get("required", []))
        schemas[name] = {"properties": props, "required": required}
    return schemas


def diff_endpoints(baseline, current):
    """Compare two endpoint dicts and return added, removed, changed."""
    baseline_keys = set(baseline.keys())
    current_keys = set(current.keys())

    added = sorted(current_keys - baseline_keys)
    removed = sorted(baseline_keys - current_keys)

    changed = []
    for key in sorted(baseline_keys & current_keys):
        old = baseline[key]
        new = current[key]
        diffs = []

        if old["summary"] != new["summary"]:
            diffs.append(f"summary: '{old['summary']}' -> '{new['summary']}'")
        if old["deprecated"] != new["deprecated"]:
            diffs.append(f"deprecated: {old['deprecated']} -> {new['deprecated']}")
        if old["tags"] != new["tags"]:
            diffs.append(f"tags: {old['tags']} -> {new['tags']}")

        old_params = {p["name"] for p in old["parameters"]}
        new_params = {p["name"] for p in new["parameters"]}
        added_params = new_params - old_params
        removed_params = old_params - new_params
        if added_params:
            diffs.append(f"new params: {sorted(added_params)}")
        if removed_params:
            diffs.append(f"removed params: {sorted(removed_params)}")

        if diffs:
            changed.append((key, diffs))

    return added, removed, changed


def diff_schemas(baseline, current):
    """Compare schemas for added, removed, and changed."""
    baseline_keys = set(baseline.keys())
    current_keys = set(current.keys())

    added = sorted(current_keys - baseline_keys)
    removed = sorted(baseline_keys - current_keys)

    changed = []
    for name in sorted(baseline_keys & current_keys):
        old = baseline[name]
        new = current[name]
        diffs = []

        old_props = set(old["properties"])
        new_props = set(new["properties"])
        if new_props - old_props:
            diffs.append(f"new properties: {sorted(new_props - old_props)}")
        if old_props - new_props:
            diffs.append(f"removed properties: {sorted(old_props - new_props)}")

        if old["required"] != new["required"]:
            diffs.append(f"required changed: {old['required']} -> {new['required']}")

        if diffs:
            changed.append((name, diffs))

    return added, removed, changed


def categorize_endpoint(method, path):
    """Categorize an endpoint as standard, experimental, or enterprise."""
    if "/v2-experimental/" in path:
        return "experimental"
    if "/orgs/" in path or "enterprise" in path.lower():
        return "enterprise"
    return "standard"


def format_report(
    endpoint_added,
    endpoint_removed,
    endpoint_changed,
    schema_added,
    schema_removed,
    schema_changed,
    current_endpoints,
    baseline_endpoints,
):
    """Format a Markdown report."""
    lines = ["# Miro API Spec Diff Report\n"]

    total_changes = (
        len(endpoint_added)
        + len(endpoint_removed)
        + len(endpoint_changed)
        + len(schema_added)
        + len(schema_removed)
        + len(schema_changed)
    )

    if total_changes == 0:
        lines.append("No changes detected.\n")
        return "\n".join(lines)

    lines.append(f"**{total_changes} changes detected.**\n")

    # Endpoint changes
    if endpoint_added:
        lines.append(f"## New Endpoints ({len(endpoint_added)})\n")
        by_category = defaultdict(list)
        for method, path in endpoint_added:
            cat = categorize_endpoint(method, path)
            info = current_endpoints[(method, path)]
            by_category[cat].append((method, path, info))

        for cat in ["standard", "experimental", "enterprise"]:
            if cat in by_category:
                lines.append(f"### {cat.title()}\n")
                lines.append("| Method | Path | Summary |")
                lines.append("|--------|------|---------|")
                for method, path, info in by_category[cat]:
                    lines.append(f"| {method} | `{path}` | {info['summary']} |")
                lines.append("")

    if endpoint_removed:
        lines.append(f"## Removed Endpoints ({len(endpoint_removed)})\n")
        lines.append("| Method | Path | Summary |")
        lines.append("|--------|------|---------|")
        for method, path in endpoint_removed:
            info = baseline_endpoints[(method, path)]
            lines.append(f"| {method} | `{path}` | {info['summary']} |")
        lines.append("")

    if endpoint_changed:
        lines.append(f"## Changed Endpoints ({len(endpoint_changed)})\n")
        for (method, path), diffs in endpoint_changed:
            lines.append(f"### `{method} {path}`\n")
            for d in diffs:
                lines.append(f"- {d}")
            lines.append("")

    # Schema changes
    if schema_added:
        lines.append(f"## New Schemas ({len(schema_added)})\n")
        for name in schema_added:
            lines.append(f"- `{name}`")
        lines.append("")

    if schema_removed:
        lines.append(f"## Removed Schemas ({len(schema_removed)})\n")
        for name in schema_removed:
            lines.append(f"- `{name}`")
        lines.append("")

    if schema_changed:
        lines.append(f"## Changed Schemas ({len(schema_changed)})\n")
        for name, diffs in schema_changed:
            lines.append(f"### `{name}`\n")
            for d in diffs:
                lines.append(f"- {d}")
            lines.append("")

    return "\n".join(lines)


def main():
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <baseline.json> <current.json>", file=sys.stderr)
        sys.exit(1)

    with open(sys.argv[1]) as f:
        baseline_spec = json.load(f)
    with open(sys.argv[2]) as f:
        current_spec = json.load(f)

    baseline_endpoints = extract_endpoints(baseline_spec)
    current_endpoints = extract_endpoints(current_spec)
    baseline_schemas = extract_schemas(baseline_spec)
    current_schemas = extract_schemas(current_spec)

    ep_added, ep_removed, ep_changed = diff_endpoints(
        baseline_endpoints, current_endpoints
    )
    sc_added, sc_removed, sc_changed = diff_schemas(baseline_schemas, current_schemas)

    report = format_report(
        ep_added,
        ep_removed,
        ep_changed,
        sc_added,
        sc_removed,
        sc_changed,
        current_endpoints,
        baseline_endpoints,
    )

    print(report)

    # Exit with 1 if changes found (useful for CI)
    total = (
        len(ep_added)
        + len(ep_removed)
        + len(ep_changed)
        + len(sc_added)
        + len(sc_removed)
        + len(sc_changed)
    )
    sys.exit(1 if total > 0 else 0)


if __name__ == "__main__":
    main()
