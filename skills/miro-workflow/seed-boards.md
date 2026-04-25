# Seed Boards (Optional Power-User Path)

Seed boards are Miroverse templates the user has imported into their own Miro account via the web UI's "Use template" button. Once imported, the MCP can copy them via `miro_copy_board` (because the user owns them) and personalize the copy.

This is an **optional** path. The skill works without seed boards; it falls back to from-scratch construction. Seed boards trade portability for output quality.

## When to use seed boards vs from-scratch

| Situation | Recommended path |
|-----------|------------------|
| User installs the MCP fresh, no setup | from-scratch (default 5 workflows) |
| User wants polished, designer-quality output | seed boards |
| Workflow has no good Miroverse template | from-scratch |
| User asks for "the Design Sprint" or named template | seed boards |
| Skill ships in Anthropic directory cohort | from-scratch primary, seed boards documented as power-user |

## How seed boards work at runtime

1. The skill calls `miro_find_board(name="<canonical seed name>")` to discover whether the user has imported the template.
2. If found, the skill calls `miro_copy_board(board_id=<found_id>, name="<personalized name>")` to make a fresh copy.
3. The skill then personalizes the copy: find specific stickies/text by content and update them via `miro_update_sticky` or `miro_update_text`.
4. If not found, the skill falls back to the from-scratch workflow file in `workflows/`.

The fallback is silent. The user gets a working board either way; the only difference is visual polish.

## Setup for users (one-time per template)

To enable a seed board, the user does:

1. Open the Miroverse template page (URL provided below).
2. Click **"Use template"**. This copies the template into their team account.
3. Rename the copy to the canonical seed name listed in the registry.
4. The skill now finds it by name on subsequent runs.

Renaming is critical. If the user keeps the default name ("Copy of The Design Sprint by Jake Knapp"), the skill won't match.

## Registry

The current registry has 1 entry. Expand as templates are validated.

### Design Sprint (Jake Knapp)

- **Miroverse:** https://miro.com/templates/the-2024-design-sprint/
- **Canonical name:** `Design Sprint Template`
- **Maps to:** none of the 5 canonical workflows. This is a **standalone seed**, invoked when the user explicitly asks for "the Design Sprint" or a 5-day product validation workshop.
- **Reference board ID:** supplied by the user after they import the template. Use `miro_find_board(name="Design Sprint Template")` to resolve, or read the ID from the URL when the user copies it.
- **Personalization points:**
  - Day 1 / 2 / 3 / 4 / 5 frame titles (typically left as-is)
  - Team name in the title text
  - Date range in the calendar frame
- **Note:** This is NOT a sprint planning board. Don't confuse with `sprint_board` workflow (which is a backlog tracker).

## Future entries (not yet validated)

These would benefit from seed boards if curated. Each requires (a) finding a high-quality Miroverse template, (b) the user importing + renaming it, (c) testing the personalization flow.

| Workflow | Candidate templates to consider |
|----------|---------------------------------|
| sprint_board | "AI Sprint Planning Template"; but verify it matches our 4-column shape |
| retrospective | "Sailboat Retrospective", "Start Stop Continue", "Mad Sad Glad" |
| brainstorm | "Crazy 8s", "Lotus Blossom", "SCAMPER" |
| story_map | "User Story Mapping" by Jeff Patton |
| kanban | "Simple Kanban Board" |

Don't add an entry to the registry above until the template has been imported, renamed, and tested with a copy + personalize cycle.

## Maintenance

- **Template author updates:** Miroverse template authors can publish new versions. Seed boards age out. When you notice drift (the template looks different from when imported), the user re-imports.
- **Renaming drift:** If the user manually renames their seed board, the skill stops finding it. Document canonical names clearly.
- **Cross-account portability:** Seed boards are per-account. Sharing a board ID across users doesn't work; each user must import their own copy.

## Why this stays optional

For the cohort skill (Anthropic directory alongside Canva/Notion/Sentry), requiring per-user setup breaks the "drop-in skill" promise. The from-scratch workflows in `workflows/` are the primary path because they need zero setup and work for any user who installs the miro-mcp-server.

Seed boards are a power-user optimization. Document them; don't gate the skill on them.
