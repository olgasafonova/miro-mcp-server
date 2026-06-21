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
from dataclasses import dataclass, field

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


def extract_parameters(details):
    """Extract inlined parameter metadata for one endpoint operation."""
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
    return params


def extract_endpoints(spec):
    """Extract per-endpoint metadata sufficient for behavioral diffing."""
    endpoints = {}
    for path, methods in spec.get("paths", {}).items():
        for method, details in methods.items():
            if method not in HTTP_METHODS:
                continue
            endpoints[(method.upper(), path)] = {
                "operationId": details.get("operationId", ""),
                "summary": details.get("summary", ""),
                "tags": details.get("tags", []),
                "deprecated": details.get("deprecated", False),
                "parameters": extract_parameters(details),
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


def enum_delta(old_enum, new_enum):
    """Return (added, removed) sorted lists of enum values between two sets."""
    old_set = old_enum or set()
    new_set = new_enum or set()
    return sorted(new_set - old_set), sorted(old_set - new_set)


def _diff_endpoint_metadata(old, new, breaking, additive, cosmetic):
    """Diff deprecated/request-body/response/summary/tags into the buckets."""
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

    if old.get("summary") != new.get("summary"):
        cosmetic.append(f"summary: {old.get('summary')!r} -> {new.get('summary')!r}")
    if old.get("tags") != new.get("tags"):
        cosmetic.append(f"tags: {old.get('tags')} -> {new.get('tags')}")


def _diff_one_param(name, op, np, breaking, additive):
    """Diff a single parameter (old op, new np) into the breaking/additive buckets."""
    if op is None:
        (breaking if np.get("required") else additive).append(
            f"new param `{name}` ({np.get('in')}, {np.get('type')}, "
            f"required={np.get('required')})"
        )
        return
    if np is None:
        breaking.append(f"removed param `{name}`")
        return

    if op.get("type") != np.get("type"):
        breaking.append(f"param `{name}` type: {op.get('type')} -> {np.get('type')}")
    if op.get("required") != np.get("required"):
        msg = f"param `{name}` required: {op.get('required')} -> {np.get('required')}"
        (breaking if np.get("required") else additive).append(msg)
    if op.get("enum") != np.get("enum"):
        added_e, removed_e = enum_delta(op.get("enum"), np.get("enum"))
        if removed_e:
            breaking.append(f"param `{name}` removed enum values: {removed_e}")
        if added_e:
            additive.append(f"param `{name}` new enum values: {added_e}")


def diff_endpoint_pair(old, new):
    """Compare one endpoint's old/new dict; return (breaking, additive, cosmetic)."""
    breaking, additive, cosmetic = [], [], []

    _diff_endpoint_metadata(old, new, breaking, additive, cosmetic)

    old_p = {p["name"]: p for p in old.get("parameters", [])}
    new_p = {p["name"]: p for p in new.get("parameters", [])}
    for name in sorted(set(old_p) | set(new_p)):
        _diff_one_param(name, old_p.get(name), new_p.get(name), breaking, additive)

    return breaking, additive, cosmetic


def _diff_one_property(prop, op, np, breaking, additive):
    """Diff a single schema property (old op, new np) into the buckets."""
    if op is None:
        additive.append(f"new property `{prop}`: {np.get('type')}")
        return
    if np is None:
        breaking.append(f"removed property `{prop}`: was {op.get('type')}")
        return

    if op.get("type") != np.get("type"):
        breaking.append(f"property `{prop}` type: {op.get('type')} -> {np.get('type')}")
    if op.get("enum") != np.get("enum"):
        added_e, removed_e = enum_delta(op.get("enum"), np.get("enum"))
        if removed_e:
            breaking.append(f"property `{prop}` removed enum values: {removed_e}")
        if added_e:
            additive.append(f"property `{prop}` new enum values: {added_e}")


def _diff_required(old, new, breaking, additive):
    """Diff the required-property sets into the buckets."""
    old_req = old.get("required", frozenset())
    new_req = new.get("required", frozenset())
    if old_req == new_req:
        return
    added_r = new_req - old_req
    removed_r = old_req - new_req
    if added_r:
        breaking.append(f"newly required properties: {sorted(added_r)}")
    if removed_r:
        additive.append(f"no-longer-required properties: {sorted(removed_r)}")


def _diff_top_enum(old, new, breaking, additive):
    """Diff the schema's top-level enum into the buckets."""
    if old.get("top_enum") == new.get("top_enum"):
        return
    added_e, removed_e = enum_delta(old.get("top_enum"), new.get("top_enum"))
    if removed_e:
        breaking.append(f"removed top-level enum values: {removed_e}")
    if added_e:
        additive.append(f"new top-level enum values: {added_e}")


def diff_schema_pair(old, new):
    """Compare one schema's old/new dict; return (breaking, additive, cosmetic)."""
    breaking, additive, cosmetic = [], [], []

    old_props = old.get("properties", {})
    new_props = new.get("properties", {})
    for prop in sorted(set(old_props) | set(new_props)):
        _diff_one_property(prop, old_props.get(prop), new_props.get(prop), breaking, additive)

    _diff_required(old, new, breaking, additive)
    _diff_top_enum(old, new, breaking, additive)

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


@dataclass
class DiffResult:
    """All computed diffs between two specs, ready for report formatting."""

    ep_added: list = field(default_factory=list)
    ep_removed: list = field(default_factory=list)
    ep_changes: dict = field(default_factory=dict)
    sc_added: list = field(default_factory=list)
    sc_removed: list = field(default_factory=list)
    sc_changes: dict = field(default_factory=dict)
    current_endpoints: dict = field(default_factory=dict)
    baseline_endpoints: dict = field(default_factory=dict)

    def breaking_eps(self):
        return [(k, b) for k, (b, _, _) in self.ep_changes.items() if b]

    def additive_eps(self):
        return [(k, a) for k, (_, a, _) in self.ep_changes.items() if a]

    def cosmetic_eps(self):
        return [(k, c) for k, (_, _, c) in self.ep_changes.items() if c]

    def breaking_sc(self):
        return [(n, b) for n, (b, _, _) in self.sc_changes.items() if b]

    def additive_sc(self):
        return [(n, a) for n, (_, a, _) in self.sc_changes.items() if a]


def _render_diff_blocks(lines, header, entries, label_fn):
    """Render a '### header' followed by per-entry '#### label / - diff' blocks."""
    if not entries:
        return
    lines.append(f"### {header} ({len(entries)})\n")
    for key, diffs in entries:
        lines.append(f"#### `{label_fn(key)}`\n")
        for d in diffs:
            lines.append(f"- {d}")
        lines.append("")


def _render_named_list(lines, header, names):
    """Render a '### header' followed by a flat bullet list of names."""
    if not names:
        return
    lines.append(f"### {header} ({len(names)})\n")
    for name in names:
        lines.append(f"- `{name}`")
    lines.append("")


def _ep_label(key):
    method, path = key
    return f"{method} {path}"


def _render_breaking_section(lines, diff):
    """Render the breaking-changes section."""
    lines.append("## :rotating_light: Breaking changes\n")
    lines.append(
        "_Investigate these before the next release; they may affect existing tools._\n"
    )
    if diff.ep_removed:
        lines.append(f"### Removed endpoints ({len(diff.ep_removed)})\n")
        lines.extend(format_endpoint_table(diff.ep_removed, diff.baseline_endpoints))
    _render_named_list(lines, "Removed schemas", diff.sc_removed)
    _render_diff_blocks(lines, "Endpoints with breaking changes", diff.breaking_eps(), _ep_label)
    _render_diff_blocks(lines, "Schemas with breaking changes", diff.breaking_sc(), str)


def _render_additive_section(lines, diff):
    """Render the additive-changes section."""
    lines.append("## :sparkles: Additive changes\n")
    lines.append("_New surface; safe for existing callers but may unlock new tools._\n")
    if diff.ep_added:
        lines.append(f"### New endpoints ({len(diff.ep_added)})\n")
        lines.extend(format_endpoint_table(diff.ep_added, diff.current_endpoints))
    _render_named_list(lines, "New schemas", diff.sc_added)
    _render_diff_blocks(lines, "Endpoints with additive changes", diff.additive_eps(), _ep_label)
    _render_diff_blocks(lines, "Schemas with additive changes", diff.additive_sc(), str)


def _render_cosmetic_section(lines, cosmetic_eps):
    """Render the collapsed cosmetic-changes section."""
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


def format_report(diff):
    """Format a Markdown report grouped by impact (breaking, additive, cosmetic)."""
    lines = ["# Miro API Spec Diff Report\n"]

    breaking_eps = diff.breaking_eps()
    breaking_sc = diff.breaking_sc()
    additive_eps = diff.additive_eps()
    additive_sc = diff.additive_sc()
    cosmetic_eps = diff.cosmetic_eps()

    breaking_count = (
        len(diff.ep_removed) + len(diff.sc_removed) + len(breaking_eps) + len(breaking_sc)
    )
    additive_count = (
        len(diff.ep_added) + len(diff.sc_added) + len(additive_eps) + len(additive_sc)
    )
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
        _render_breaking_section(lines, diff)
    if additive_count:
        _render_additive_section(lines, diff)
    if cosmetic_eps:
        _render_cosmetic_section(lines, cosmetic_eps)

    return "\n".join(lines)


def _diff_collection(baseline, current, diff_pair):
    """Diff two name->dict collections; return (added, removed, changes)."""
    base_keys = set(baseline)
    cur_keys = set(current)
    added = sorted(cur_keys - base_keys)
    removed = sorted(base_keys - cur_keys)
    changes = {}
    for key in sorted(base_keys & cur_keys):
        b, a, c = diff_pair(baseline[key], current[key])
        if b or a or c:
            changes[key] = (b, a, c)
    return added, removed, changes


def compute_diff(baseline_spec, current_spec):
    """Compute the full DiffResult between two parsed specs."""
    baseline_endpoints = extract_endpoints(baseline_spec)
    current_endpoints = extract_endpoints(current_spec)
    baseline_schemas = extract_schemas(baseline_spec)
    current_schemas = extract_schemas(current_spec)

    ep_added, ep_removed, ep_changes = _diff_collection(
        baseline_endpoints, current_endpoints, diff_endpoint_pair
    )
    sc_added, sc_removed, sc_changes = _diff_collection(
        baseline_schemas, current_schemas, diff_schema_pair
    )

    return DiffResult(
        ep_added=ep_added,
        ep_removed=ep_removed,
        ep_changes=ep_changes,
        sc_added=sc_added,
        sc_removed=sc_removed,
        sc_changes=sc_changes,
        current_endpoints=current_endpoints,
        baseline_endpoints=baseline_endpoints,
    )


def diff_total(diff):
    """Return the total number of detected differences."""
    return (
        len(diff.ep_added)
        + len(diff.ep_removed)
        + len(diff.ep_changes)
        + len(diff.sc_added)
        + len(diff.sc_removed)
        + len(diff.sc_changes)
    )


def main():
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <baseline.json> <current.json>", file=sys.stderr)
        sys.exit(1)

    with open(sys.argv[1]) as f:
        baseline_spec = json.load(f)
    with open(sys.argv[2]) as f:
        current_spec = json.load(f)

    diff = compute_diff(baseline_spec, current_spec)
    print(format_report(diff))
    sys.exit(1 if diff_total(diff) > 0 else 0)


if __name__ == "__main__":
    main()
