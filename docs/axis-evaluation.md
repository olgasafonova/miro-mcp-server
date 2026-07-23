# AXIS evaluation of miro-mcp-server

Bead: `miro-mcp-server-zo9`
Date: 23-07-2026
Method: static source inspection only. No AXIS run was executed. See "Gate 1" for why.

## What AXIS actually is

AXIS (Agent Experience Index Score) is a real, runnable tool, not a rubric. Verified facts:

| Fact | Evidence |
|---|---|
| Owner and licence | `netlify/axis`, MIT, created 28-04-2026, last push 17-07-2026 (`gh api repos/netlify/axis`) |
| Distribution | npm package `@netlify/axis`, CLI entry `src/cli.ts` |
| Language | TypeScript |
| Docs | axis.run |
| Model | Give it a scenario JSON (prompt plus weighted `judge` checks), an agent, and it runs the agent, captures a transcript, and scores it |

It is not a linter. It produces no score without executing a live agent against a live system.

### The scoring model

Composite AXIS Result is a weighted average of four dimensions (`src/docs-site/src/pages/scoring.astro`):

- **Goal Achievement, 40 percent.** LLM judge grades the transcript against the scenario's `judge[]` checks, 0 to 10 each, weighted.
- **Environment, 20 percent.** Shell, filesystem, git, build tools. Scored on Success 0.7 and Speed 0.3 only.
- **Service, 20 percent.** APIs, MCP tools, network. Also Success 0.7 and Speed 0.3 only. **This is where miro-mcp-server lands.**
- **Agent, 20 percent.** Agent decision quality across every interaction in the run. Success 0.1, Speed 0.1, Weight 0.2, Relevance 0.2, Necessity 0.4.

Critical structural point for interpreting any future run: **the Service dimension does not grade response shape at all.** `src/scoring/prompt-templates.ts` `CATEGORY_GUIDANCE.service` explicitly instructs the judge not to lower the Service score because a response was irrelevant or the call was unnecessary. Every response-design property this repo has been optimising (payload size, totalCount, truncation hints, tool granularity) shows up in the **Agent** dimension (Weight, Relevance, Necessity) or leaks into Goal Achievement, never in Service. A high Service score means "the Miro API answered without erroring", nothing more.

### Gate 1: can AXIS drive a Go stdio MCP server?

**Yes, structurally.** `src/adapters/utils/mcp.ts` writes MCP server config for both supported adapter families and handles the stdio transport explicitly:

- `writeClaudeMcpConfig` emits `{ mcpServers: { name: { command, args, env } } }` to a path passed to Claude Code via `--mcp-config`, deliberately outside the workspace so the agent cannot read its own config.
- `writeCodexMcpConfig` hand-generates `[mcp_servers.<name>]` TOML with `command`, `args`, and an `env` table into `CODEX_HOME`.

Both branches accept an arbitrary `command`, so a compiled Go binary is a first-class case. There is nothing Node-specific in the wiring. Adapters exist for 20-plus agents (`src/adapters/`), with e2e fixtures specifically for MCP under Claude and Codex (`test/e2e/adapters/mcp-claude/`, `test/e2e/adapters/mcp-codex/`).

**But it could not be run for this bead.** Three blockers, none of them about AXIS:

1. It requires an installed agent CLI and real LLM spend for both the agent and the judge.
2. It requires a live `MIRO_ACCESS_TOKEN`. There is no fixture or record/replay layer in AXIS.
3. Any realistic Miro scenario mutates a board. The bead's hard constraints forbid mutating API calls.

So Gate 1 passes and Gate 2 (comparative run against the official 13-tool Miro server) remains open work. What follows is a static assessment mapped onto the AXIS criteria, which is a different and weaker artifact.

### Skepticism carried over from `rules/agent-interface-design.md`

That rules file covers **AXI** (Kun Chen, axi.md), a different framework with a confusingly similar name. AXIS is Netlify's. The two are not the same project, but the same cautions apply and some are sharper here.

1. **Optimistic default.** `src/scoring/category-score.ts` `DEFAULT_AUDIT_SCORES` sets success, speed, weight, and contextRelevance to 1.0 with the comment "If nothing was evaluated, assume perfect". `aggregateDimension` returns that default whenever `audits.length === 0`. An AXIS score is a floor on badness, not an estimate of quality. Anything the judge failed to look at scores perfect.
2. **The judge is an LLM reading a compressed transcript.** `CATEGORY_EVAL_TEMPLATE` feeds a sparse index plus per-interaction content and tells the judge "Base your scores strictly on the evidence presented... do not hedge or reduce scores based on speculation". That instruction plus the optimistic default both push scores up. Compare the reading-log note of 14-07-2026 (Parlance Labs, "Do Automated Evals Work"): the best eval agent caught 87 percent and missed whole failure classes.
3. **Calibration constants are unjustified.** `DEFAULT_CALIBRATION` sets median 0.5 and sigma 0.4 for all three categories, mapped through a log-normal CDF (`logNormalScore`). The choice of 0.5 as the raw score that maps to 50/100 is a design decision with no stated derivation. Cross-run comparisons are only meaningful if this stays fixed.
4. **`severityWeightedAverage` is applied to speed only**, with weight `(1-v)^2 + 1`. It makes one slow call pull harder than several fast ones push. That is defensible but it means Service scores will be latency-dominated for a network-bound server like this one.
5. **Do not read a single composite number as a verdict.** Run stability across repeats is the gate before building any change on an AXIS number, exactly as the bead already says.

## Static assessment against the AXIS criteria

An AXIS number cannot be produced from source. What follows grades each AXIS signal by what the code can and cannot show. Grades are PASS, PARTIAL, GAP, or NOT ASSESSABLE. No 0-100 figure is given, because inventing one would be exactly the overclaim the rules file warns against.

Inventory basis: 97 `ToolSpec` entries in `tools/definitions.go`, all registered by default under `ProfileFull` (`tools/profile.go:12`); 15 registered under `ProfileEssentials` (`tools/profile.go:22`).

### Service dimension (Success 0.7, Speed 0.3)

| Signal | Grade | Evidence |
|---|---|---|
| Success: calls complete, errors surfaced clearly | PASS | `miro/errors.go:85` `APIError.Suggestion()` attaches actionable guidance per status code; `miro/errors.go:150` wraps with "Suggestion:". Typed `APIError` / `ValidationError` / `DiagramError` documented in `ERRORS.md`. Panic recovery at `tools/handlers.go:388` converts panics into a correlated error rather than a silent fake success, with the named-return trap documented in place. |
| Success: transparent fallbacks | PASS | `miro/items.go:396` retries a delete against `/mindmap_nodes/` before failing, so callers do not have to pre-classify item IDs. |
| Speed: latency controls | PARTIAL, mostly NOT ASSESSABLE | Cache (`miro/cache.go`), rate limiter (`miro/ratelimit.go`), and circuit breaker (`miro/circuitbreaker.go`) exist and would help. Actual latency is a property of the Miro API under real load and cannot be read from source. |
| Speed: measured percentiles | NOT ASSESSABLE | No latency benchmark in the repo. `PERFORMANCE.md` exists but AXIS measures wall-clock per interaction in a live run. |

Reading: the Service dimension is the part of AXIS this server is most likely to score well on, and it is also the part that says the least about the work already done here. Do not use a good Service score as evidence the response design is good.

### Agent dimension (Necessity 0.4, Weight 0.2, Relevance 0.2, Success 0.1, Speed 0.1)

This is the dimension that actually tests the granular-versus-coarse question in the bead.

| Signal | Grade | Evidence |
|---|---|---|
| Weight: right-sized responses, view structs | PASS | `miro/types_items.go:11` `ItemSummary` is a trimmed six-field projection with extended fields gated behind `detail_level=full` (`miro/items.go:39`). `miro/boards.go:63-73` projects `Board` down to `BoardSummary`. This is `hg2-hardening` idiom 1 and idiom 4, already applied. |
| Weight: default caps on lists | PASS | `miro/constants.go:13` `DefaultBoardLimit = 20`, `:16` `MaxBoardLimit = 50`, `:40` `DefaultListAllMaxItems = 500`, clamped in `effectiveListAllMax` (`miro/items.go:447`). |
| Weight: full-text char cap | GAP | `ItemSummary.Content` (`miro/types_items.go:14`) is uncapped. A board with long sticky or document text returns it in full, times the item count. `truncate` exists (`miro/search.go:153`) but is only used for confirmation strings and search context, never on list payloads. `hg2-hardening` idiom 3 is not applied to item content. |
| Necessity: pre-computed aggregates that avoid round trips | PASS on the composite tools | `miro_get_board_summary` returns `ItemCounts map[string]int` plus `TotalItems` and `RecentItems` (`miro/types_boards.go:135-144`). That is AXI principle 4 done properly and is the single biggest turn-saver in the surface. `miro_get_board_content` similarly exists as a one-call export. |
| Necessity: `totalCount` on list responses | GAP | `ListBoardsResult` (`miro/types_boards.go:29`) carries `Count`, `HasMore`, `Offset` but no total, even though the Miro response total is parsed and then discarded at `miro/boards.go:54`. `ListItemsResult` (`miro/types_items.go:288`) carries `Count`, `HasMore`, `Cursor` and no total. An agent cannot tell a full page from the last page without a follow-up call. |
| Relevance: explicit zero-result messages | PARTIAL | Present and good on `SearchBoardResult` (`miro/types_items.go:400`, message built at `miro/search.go:77` `searchBoardMessage`) and on `miro_tool_search` (`tools/search.go:238-243`, four distinct phrasings covering query-only, category-only, and both). Absent on the two highest-traffic list tools: neither `ListBoardsResult` nor `ListItemsResult` has a `Message` field at all, so an empty board and a silently degraded call look identical. |
| Relevance: truncation size hint | PARTIAL | `ListAllItemsResult.Truncated` exists (`miro/types_items.go:312`) and `formatListAllMessage` (`miro/items.go:490`) says "truncated at max_items limit". It does not say how many items remain, so the agent cannot decide whether a follow-up is worth a turn. AXI principle 3 wants the size, not just the flag. |
| Necessity: next-step `help[]` templates | GAP | No result type carries next-step hints. `Suggestion` exists only on the error path (`miro/errors.go:85`). On the success path the agent gets no guidance about what to call next, despite the descriptions containing that knowledge in prose (for example `tools/definitions.go:87` "Get from miro_list_boards or miro_find_board"). |
| Necessity: tool discovery cost | PARTIAL | `miro_tool_search` is registered first so agents meet it before scanning (`tools/definitions.go:59`), and `ProfileEssentials` cuts the surface to 15. But the default is `ProfileFull` with all 97 tools, and AXIS's agent judge counts tool discovery against necessity only when redundant, so the cost shows up mainly as context weight rather than as a necessity penalty. |
| Unknown parameters fail loud | NOT ASSESSABLE from source, and design tension | The desire-path layer deliberately does the opposite of AXI principle 6: `tools/normalize.go:69` `applyKeyRemapping` silently rewrites camelCase keys to snake_case, and `applyValueNormalizers` (`tools/normalize.go:93`) coerces strings to numbers and booleans, with URL-to-ID extraction on top (`main.go:207`). Whether a genuinely unknown key produces an error or is dropped depends on go-sdk v1.6.1 schema validation behaviour, which requires a runtime test to establish. Marked NOT ASSESSABLE rather than guessed. |
| Idempotent mutations | PARTIAL | `ToolSpec` declares `Idempotent` and `Destructive` (`tools/definitions.go:44-51`) and these become MCP annotations. The declaration is metadata; whether the handler is genuinely idempotent is per-tool and not verified here. |

### Goal Achievement dimension (40 percent)

**NOT ASSESSABLE.** This dimension is defined entirely by scenario-specific `judge[]` checks that do not exist yet. Nothing in the repo can predict it. Writing the scenario set is the first concrete deliverable of Gate 2.

The repo does have a related asset worth reusing: `evals/tool_selection.json`, `evals/confusion_pairs.json`, and `evals/argument_correctness.json` with a Go runner (`evals/runner.go`). Those already encode "which tool should the model pick for this prompt", which is close to what an AXIS scenario prompt needs. They are not AXIS scenarios and the formats differ, but they are the source material.

### Environment dimension (20 percent)

**NOT ASSESSABLE and largely irrelevant to this server.** Environment covers shell, filesystem, and build tooling in the agent's workspace. A Miro scenario touches it only incidentally. Its 20 percent of the composite is effectively noise for this comparison, which is a reason to record the four sub-scores separately rather than lean on the composite, as the bead already requires.

### Summary of grades

| Dimension | Signals graded | PASS | PARTIAL | GAP | NOT ASSESSABLE |
|---|---|---|---|---|---|
| Service | 4 | 2 | 1 | 0 | 1 |
| Agent | 10 | 3 | 3 | 3 | 1 |
| Goal Achievement | 1 | 0 | 0 | 0 | 1 |
| Environment | 1 | 0 | 0 | 0 | 1 |

No composite number. AXIS produces one only from a live run, and the optimistic-default behaviour above means a synthesised number would read higher than the evidence supports.

## Cross-check against prior work in this portfolio

### Where AXIS agrees with `hg2-hardening`

The three size idioms that skill names are already applied here and AXIS's Agent Weight signal rewards them directly:

- Idiom 1, view structs: applied (`ItemSummary`, `BoardSummary`).
- Idiom 2, default caps: applied (`miro/constants.go`).
- Idiom 4, detail opt-in: applied (`detail_level=full`).

The four AXI response-shape behaviours that `hg2-hardening` lists as "not yet applied" are exactly the gaps this evaluation found independently: totalCount, zero-result messages, unknown-params-fail-loud, and `help[]` lines. AXIS's necessity and relevance weighting confirms the priority ordering that skill already proposes. This is convergent evidence, not a new finding.

Idiom 3, char-limit with a Truncated flag, is the one this repo skipped for item content. AXIS would surface it as Weight on any board with long text.

### Where AXIS adds something new

1. **It separates execution quality from decision quality**, and this repo has no equivalent split. `tool-audit` grades the description string; `hg2-hardening` grades the payload. Neither asks whether the agent's *sequence* of calls was necessary. AXIS's Necessity signal at 0.4 of the Agent dimension is the only instrument in reach that measures the granular-versus-coarse argument empirically, which is exactly what the bead wants.
2. **It measures across a whole task, not per tool.** Every existing audit in this portfolio is per-tool. The 97-tool surface may score fine tool-by-tool and badly across a five-step workflow. That gap is invisible to `tool-audit` and `hg2-hardening` by construction.

### Where AXIS contradicts prior work

1. **Service is not where response quality is measured.** Someone reading "Service dimension" and expecting it to grade MCP response design will misread the result. The `CATEGORY_GUIDANCE.service` text is explicit that relevance and necessity are judged elsewhere. Any future report on this must say so.
2. **The desire-path normalizers cut against AXI principle 6.** `rules/agent-interface-design.md` principle 6 wants unknown flags to fail loud so an agent learns its mistake. This server deliberately absorbs agent mistakes silently and logs them for later analysis. Under AXIS the normalizers likely *raise* the Service score (fewer errors) while suppressing exactly the signal that would improve the agent's future calls. That is a real design disagreement, not an oversight, and it should be stated as such rather than "fixed" by reflex. The desire-path log is the mitigating design; the question is whether the agent should also be told in-band.
3. **`tool-audit` C6 (Code Mode eligibility) is satisfied here by construction**, which AXIS does not check. `registerTool` (`tools/handlers.go:350` (the `mcp.AddTool` call)) uses `mcp.AddTool[Args, Result]` from go-sdk v1.6.1, so typed output schemas are generated from the Result structs. AXIS has no equivalent structural check, so a good AXIS score would say nothing about Code Mode readiness. The two audits are complements, not substitutes.

## Prioritised fixes

Each is a candidate follow-up bead. None is implemented here.

### 1. Add `Total` to `ListBoardsResult` and `ListItemsResult`

- `miro/types_boards.go:29` and `miro/types_items.go:288`
- The board total is already parsed and thrown away at `miro/boards.go:54`. Plumbing it through is a two-line change.
- `ListItems` will need the Miro cursor-paged endpoint's `size` or an explicit count call; if the API does not return a total, return the field with a documented sentinel rather than omitting it.
- Impact: highest ratio of agent-turns saved to lines changed. AXI principle 4, `hg2-hardening` behaviour 1.

### 2. Add a `Message` field with an explicit zero-result string to both list results

- `miro/types_boards.go:29` and `miro/types_items.go:288`, populated in `miro/boards.go:82` and `miro/items.go:48`
- Reuse the pattern already proven at `miro/search.go` `searchBoardMessage` and `tools/search.go:238`. The idiom exists in the codebase; it is missing from the two tools most likely to be called first.
- Impact: an empty board currently reads identically to a degraded call.

### 3. Put the remaining-item count in the truncation message

- `miro/items.go:490` `formatListAllMessage`, called from `miro/items.go:441`
- Currently "truncated at max_items limit". AXI principle 3 wants the size so the agent can decide whether a follow-up is worth a turn. If the true remainder is unknown, say "at least N more" from the cursor state.
- Also applies to `miro/types_boards.go:251` (board-content `Truncated`) and `miro/boards_summary.go:229`.

### 4. Char-cap `ItemSummary.Content` with a per-item truncation flag

- `miro/types_items.go:14`
- Add a `CharLimit` constant beside `miro/constants.go:40` and apply `truncate` (already at `miro/search.go:153`) in `parseItemSummary` (`miro/items.go:127` `parseItemSummary`). Add `ContentTruncated bool`.
- This is `hg2-hardening` idiom 3, the one idiom skipped on the read path. Worst case today is a 500-item list of long documents returned in full.

### 5. Add next-step `help[]` lines to the highest-traffic results

- `miro/types_boards.go:29` (`ListBoardsResult`), `miro/types_boards.go:135` (`GetBoardSummaryResult`), `miro/types_items.go:288` (`ListItemsResult`)
- The knowledge is already written in prose in the tool descriptions (for example `tools/definitions.go:87`). Moving it into the response carries it to the agent at the moment it is actionable.
- Placeholders only. `miro_get_item(board_id="<id>", item_id="<id>")`, never a guessed concrete ID (AXI principle 9).

### Lower priority, recorded not scheduled

6. Decide deliberately whether desire-path normalizations should be reported in-band (`tools/normalize.go:69`). Currently silent-and-logged. Options: stay silent, add a `normalizations[]` field to results, or fail loud on truly unknown keys. This is a design call, not a defect.
7. Establish empirically whether go-sdk v1.6.1 rejects unknown argument keys (`tools/handlers.go:350` (the `mcp.AddTool` call)). One runtime test settles criterion "unknown parameters fail loud", currently NOT ASSESSABLE.
8. Consider whether `ProfileEssentials` should become the default (`tools/profile.go:12`). Gate 2 data should decide this, not reasoning.

## What this evaluation did not cover

- No AXIS run. Every number AXIS would produce is absent by design, not omission.
- No comparison against the official Miro MCP server. Gate 2 is untouched.
- No handler correctness review. Grades address response shape and error surfacing only.
- No description-quality grading. That is `tool-audit`'s rubric and it has its own report.
- Speed under real load, Goal Achievement, and Environment are structurally unreachable from source and are marked NOT ASSESSABLE rather than estimated.

## Next steps for Gate 2

1. Write an AXIS scenario set. Source material: `evals/tool_selection.json` and `evals/confusion_pairs.json`.
2. Point it at a throwaway Miro team, since scenarios will mutate boards.
3. Run both servers with identical scenarios and identical agent adapters. Record all four sub-scores plus the baseline, never the composite alone.
4. Repeat each scenario enough times to report variance. AXIS reports none by default and the optimistic default makes single runs unreliable.
5. Add "time to first agreed decision" as a second metric alongside the AXIS sub-scores, per the note appended to the bead on 22-07-2026. That is Miro's own published yardstick and it holds the comparison to the vendor's stated criterion rather than to cost and turns alone.
