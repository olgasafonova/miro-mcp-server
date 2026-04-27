#!/usr/bin/env python3
"""Compare two Miro OpenAPI specs and report behavior-affecting differences.

Usage:
    python3 diff-spec.py <baseline.json> <current.json>

Classifies every change as Breaking, Additive, or Cosmetic and groups the
Markdown report so the highest-impact items appear first. Exits 1 when any
change is detected so the api-tracking workflow can file a GitHub issue.

What "Breaking" means here: a change that may make our existing tools
malfunction without code changes. Examples: removed endpoint or schema,
removed property, narrowed property type, removed enum value, newly required
field/parameter, type/$ref change.

What "Additive" means: new endpoint, new schema, new optional property/
parameter, new enum value. Safe for existing callers; may unlock new tools.

What "Cosmetic" means: tag, summary, or description churn. Filtered out of
the main report (collapsed at the bottom) to keep noise out of the issue.
"""

import json
import sys
from collections import defaultdict

HTTP_METHODS = {"get", "post", "put", "patch", "delete"}


def prop_signature(p):
    """Return a stable comparable string describing a property/parameter type.

    Captures: type, format, $ref target, array element type, oneOf/anyOf/allOf
    composition, and nullable flag. Two signatures compare equal iff a value
    valid for one is necessarily valid for the other.
    """
    if not isinstance(p, dict):
        return repr(p)
    if "$ref" in p:
        return f"ref:{p['$ref'].split('/')[-1]}"
    for combinator in ("oneOf", "anyOf", "allOf"):
        if combinator in p:
            parts = ",".join(prop_signature(s) for s in p[combinator])
            return f"{combinator}<{parts}>"
    t = p.get("type")
    if t == "array":
        return f"array<{prop_signature(p.get('items', {}))}>"
    if t == "object" and "properties" in p:
        names = ",".join(sorted(p["properties"].keys()))
        return f"object<{names}>"
    fmt = p.get("format")
    nullable = "?" if p.get("nullable") else ""
    return f"{t or '?'}{f'({fmt})' if fmt else ''}{nullable}"


def prop_enum(p):
    """Return a frozenset of enum values, or None if not enumerated."""
    if isinstance(p, dict) and isinstance(p.get("enum"), list):
        return frozenset(p["enum"])
    return None


def request_body_ref(details):
    """Return the JSON request body schema $ref, if any."""
    rb = details.get("requestBody") or {}
    schema = rb.get("content", {}).get("application/json", {}).get("schema", {})
    return prop_signature(schema) if schema else None


def response_refs(details):
    """Return dict of status_code -> JSON response schema signature."""
    out = {}
    for code, resp in (details.get("responses") or {}).items():
        if not code.startswith("2"):
            continue
        schema = resp.get("content", {}).get("application/json", {}).get("schema", {})
        if schema:
            out[code] = prop_signature(schema)
    return out


def extract_endpoints(spec):
    """Extract per-endpoint metadata sufficient for behavioral diffing."""
    endpoints = {}
    for path, methods in spec.get("paths", {}).items():
        for method, details in methods.items():
            if method not in HTTP_METHODS:
                continue
            params = []
            for p in details.get("parameters", []):
                # Skip $ref-only parameters that aren't inlined; they have no name.
                if not p.get("name"):
                    continue
                schema = p.get("schema", {})
                params.append(
                    {
                        "name": p.get("name"),
                        "in": p.get("in"),
                        "required": p.get("required", False),
                        "type": prop_signature(schema),
                        "enum": prop_enum(schema),
                    }
                )
            endpoints[(method.upper(), path)] = {
                "operationId": details.get("operationId", ""),
                "summary": details.get("summary", ""),
                "tags": details.get("tags", []),
                "deprecated": details.get("deprecated", False),
                "parameters": params,
                "request_body": request_body_ref(details),
                "responses": response_refs(details),
            }
    return endpoints


def extract_schemas(spec):
    """Extract per-schema property type, enum, and required info."""
    out = {}
    for name, schema in spec.get("components", {}).get("schemas", {}).items():
        props = {
            pname: {"type": prop_signature(pschema), "enum": prop_enum(pschema)}
            for pname, pschema in schema.get("properties", {}).items()
        }
        out[name] = {
            "properties": props,
            "required": frozenset(schema.get("required", [])),
            "top_enum": prop_enum(schema),
        }
    return out


def categorize_endpoint(method, path):
    """Categorize an endpoint as standard, experimental, or enterprise."""
    if "/v2-experimental/" in path:
        return "experimental"
    if "/orgs/" in path or "enterprise" in path.lower():
        return "enterprise"
    return "standard"


def diff_endpoint_pair(old, new):
    """Compare one endpoint's old/new dict; return (breaking, additive, cosmetic)."""
    breaking, additive, cosmetic = [], [], []

    if old.get("deprecated") != new.get("deprecated"):
        msg = f"deprecated: {old.get('deprecated')} -> {new.get('deprecated')}"
        (additive if new.get("deprecated") else breaking).append(msg)

    if old.get("request_body") != new.get("request_body"):
        breaking.append(
            f"request body schema: {old.get('request_body')} -> {new.get('request_body')}"
        )

    old_resp = old.get("responses", {})
    new_resp = new.get("responses", {})
    for code in sorted(set(old_resp) | set(new_resp)):
        ov, nv = old_resp.get(code), new_resp.get(code)
        if ov != nv:
            breaking.append(f"response {code} schema: {ov} -> {nv}")

    old_p = {p["name"]: p for p in old.get("parameters", [])}
    new_p = {p["name"]: p for p in new.get("parameters", [])}
    for name in sorted(set(old_p) | set(new_p)):
        op, np = old_p.get(name), new_p.get(name)
        if op is None:
            (breaking if np.get("required") else additive).append(
                f"new param `{name}` ({np.get('in')}, {np.get('type')}, "
                f"required={np.get('required')})"
            )
        elif np is None:
            breaking.append(f"removed param `{name}`")
        else:
            if op.get("type") != np.get("type"):
                breaking.append(
                    f"param `{name}` type: {op.get('type')} -> {np.get('type')}"
                )
            if op.get("required") != np.get("required"):
                msg = (
                    f"param `{name}` required: {op.get('required')} -> "
                    f"{np.get('required')}"
                )
                (breaking if np.get("required") else additive).append(msg)
            if op.get("enum") != np.get("enum"):
                added_e = (np.get("enum") or set()) - (op.get("enum") or set())
                removed_e = (op.get("enum") or set()) - (np.get("enum") or set())
                if removed_e:
                    breaking.append(
                        f"param `{name}` removed enum values: {sorted(removed_e)}"
                    )
                if added_e:
                    additive.append(
                        f"param `{name}` new enum values: {sorted(added_e)}"
                    )

    if old.get("summary") != new.get("summary"):
        cosmetic.append(f"summary: {old.get('summary')!r} -> {new.get('summary')!r}")
    if old.get("tags") != new.get("tags"):
        cosmetic.append(f"tags: {old.get('tags')} -> {new.get('tags')}")

    return breaking, additive, cosmetic


def diff_schema_pair(old, new):
    """Compare one schema's old/new dict; return (breaking, additive, cosmetic)."""
    breaking, additive, cosmetic = [], [], []

    old_props = old.get("properties", {})
    new_props = new.get("properties", {})
    for prop in sorted(set(old_props) | set(new_props)):
        op, np = old_props.get(prop), new_props.get(prop)
        if op is None:
            additive.append(f"new property `{prop}`: {np.get('type')}")
        elif np is None:
            breaking.append(f"removed property `{prop}`: was {op.get('type')}")
        else:
            if op.get("type") != np.get("type"):
                breaking.append(
                    f"property `{prop}` type: {op.get('type')} -> {np.get('type')}"
                )
            if op.get("enum") != np.get("enum"):
                added_e = (np.get("enum") or set()) - (op.get("enum") or set())
                removed_e = (op.get("enum") or set()) - (np.get("enum") or set())
                if removed_e:
                    breaking.append(
                        f"property `{prop}` removed enum values: {sorted(removed_e)}"
                    )
                if added_e:
                    additive.append(
                        f"property `{prop}` new enum values: {sorted(added_e)}"
                    )

    old_req = old.get("required", frozenset())
    new_req = new.get("required", frozenset())
    if old_req != new_req:
        added_r = new_req - old_req
        removed_r = old_req - new_req
        if added_r:
            breaking.append(f"newly required properties: {sorted(added_r)}")
        if removed_r:
            additive.append(f"no-longer-required properties: {sorted(removed_r)}")

    if old.get("top_enum") != new.get("top_enum"):
        added_e = (new.get("top_enum") or set()) - (old.get("top_enum") or set())
        removed_e = (old.get("top_enum") or set()) - (new.get("top_enum") or set())
        if removed_e:
            breaking.append(f"removed top-level enum values: {sorted(removed_e)}")
        if added_e:
            additive.append(f"new top-level enum values: {sorted(added_e)}")

    return breaking, additive, cosmetic


def format_endpoint_table(rows, source_endpoints):
    """Render a Markdown table grouped by category for new/removed endpoints."""
    out = []
    by_category = defaultdict(list)
    for method, path in rows:
        info = source_endpoints[(method, path)]
        by_category[categorize_endpoint(method, path)].append((method, path, info))
    for cat in ("standard", "experimental", "enterprise"):
        if cat not in by_category:
            continue
        out.append(f"### {cat.title()}\n")
        out.append("| Method | Path | Summary |")
        out.append("|--------|------|---------|")
        for method, path, info in by_category[cat]:
            out.append(f"| {method} | `{path}` | {info['summary']} |")
        out.append("")
    return out


def format_report(
    ep_added,
    ep_removed,
    ep_changes,
    sc_added,
    sc_removed,
    sc_changes,
    current_endpoints,
    baseline_endpoints,
):
    """Format a Markdown report grouped by impact (breaking, additive, cosmetic)."""
    lines = ["# Miro API Spec Diff Report\n"]

    breaking_eps = [(k, b) for k, (b, _, _) in ep_changes.items() if b]
    additive_eps = [(k, a) for k, (_, a, _) in ep_changes.items() if a]
    cosmetic_eps = [(k, c) for k, (_, _, c) in ep_changes.items() if c]
    breaking_sc = [(n, b) for n, (b, _, _) in sc_changes.items() if b]
    additive_sc = [(n, a) for n, (_, a, _) in sc_changes.items() if a]

    breaking_count = len(ep_removed) + len(sc_removed) + len(breaking_eps) + len(breaking_sc)
    additive_count = len(ep_added) + len(sc_added) + len(additive_eps) + len(additive_sc)
    cosmetic_count = len(cosmetic_eps)
    total = breaking_count + additive_count + cosmetic_count

    if total == 0:
        lines.append("No changes detected.\n")
        return "\n".join(lines)

    lines.append(
        f"**Summary:** {breaking_count} breaking, {additive_count} additive, "
        f"{cosmetic_count} cosmetic.\n"
    )

    if breaking_count:
        lines.append("## :rotating_light: Breaking changes\n")
        lines.append(
            "_Investigate these before the next release; they may affect existing tools._\n"
        )

        if ep_removed:
            lines.append(f"### Removed endpoints ({len(ep_removed)})\n")
            lines.extend(format_endpoint_table(ep_removed, baseline_endpoints))

        if sc_removed:
            lines.append(f"### Removed schemas ({len(sc_removed)})\n")
            for name in sc_removed:
                lines.append(f"- `{name}`")
            lines.append("")

        if breaking_eps:
            lines.append(f"### Endpoints with breaking changes ({len(breaking_eps)})\n")
            for (method, path), diffs in breaking_eps:
                lines.append(f"#### `{method} {path}`\n")
                for d in diffs:
                    lines.append(f"- {d}")
                lines.append("")

        if breaking_sc:
            lines.append(f"### Schemas with breaking changes ({len(breaking_sc)})\n")
            for name, diffs in breaking_sc:
                lines.append(f"#### `{name}`\n")
                for d in diffs:
                    lines.append(f"- {d}")
                lines.append("")

    if additive_count:
        lines.append("## :sparkles: Additive changes\n")
        lines.append(
            "_New surface; safe for existing callers but may unlock new tools._\n"
        )

        if ep_added:
            lines.append(f"### New endpoints ({len(ep_added)})\n")
            lines.extend(format_endpoint_table(ep_added, current_endpoints))

        if sc_added:
            lines.append(f"### New schemas ({len(sc_added)})\n")
            for name in sc_added:
                lines.append(f"- `{name}`")
            lines.append("")

        if additive_eps:
            lines.append(f"### Endpoints with additive changes ({len(additive_eps)})\n")
            for (method, path), diffs in additive_eps:
                lines.append(f"#### `{method} {path}`\n")
                for d in diffs:
                    lines.append(f"- {d}")
                lines.append("")

        if additive_sc:
            lines.append(f"### Schemas with additive changes ({len(additive_sc)})\n")
            for name, diffs in additive_sc:
                lines.append(f"#### `{name}`\n")
                for d in diffs:
                    lines.append(f"- {d}")
                lines.append("")

    if cosmetic_eps:
        lines.append("<details>")
        lines.append(
            f"<summary>Cosmetic changes ({len(cosmetic_eps)} endpoints) - "
            "tag/summary churn, no behavior impact</summary>\n"
        )
        for (method, path), diffs in cosmetic_eps:
            lines.append(f"- `{method} {path}`")
            for d in diffs:
                lines.append(f"  - {d}")
        lines.append("\n</details>")

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

    base_keys = set(baseline_endpoints)
    cur_keys = set(current_endpoints)
    ep_added = sorted(cur_keys - base_keys)
    ep_removed = sorted(base_keys - cur_keys)
    ep_changes = {}
    for key in sorted(base_keys & cur_keys):
        b, a, c = diff_endpoint_pair(baseline_endpoints[key], current_endpoints[key])
        if b or a or c:
            ep_changes[key] = (b, a, c)

    base_schemas_keys = set(baseline_schemas)
    cur_schemas_keys = set(current_schemas)
    sc_added = sorted(cur_schemas_keys - base_schemas_keys)
    sc_removed = sorted(base_schemas_keys - cur_schemas_keys)
    sc_changes = {}
    for name in sorted(base_schemas_keys & cur_schemas_keys):
        b, a, c = diff_schema_pair(baseline_schemas[name], current_schemas[name])
        if b or a or c:
            sc_changes[name] = (b, a, c)

    report = format_report(
        ep_added,
        ep_removed,
        ep_changes,
        sc_added,
        sc_removed,
        sc_changes,
        current_endpoints,
        baseline_endpoints,
    )
    print(report)

    total = (
        len(ep_added)
        + len(ep_removed)
        + len(ep_changes)
        + len(sc_added)
        + len(sc_removed)
        + len(sc_changes)
    )
    sys.exit(1 if total > 0 else 0)


if __name__ == "__main__":
    main()
