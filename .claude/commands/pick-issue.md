---
description: Take one open issue from TASKS.md through OpenSpec or direct implementation, verify it, and open one pull request
---

# Pick an Issue: One Issue Per Invocation

Choose exactly one open issue in `rshade/finfocus-plugin-opencost`, take it to
a verified pull request, and **stop**. Do not start a second one.

Adapted from the `pick-issue` command in `gojev` (OpenSpec) and in `finfocus`
(Spec Kit). What differs here:

- **One worker, one branch, no claims.** The claim protocol (labels and
  `claim:` comments) exists to arbitrate several machines. This repo has one
  worker, so claims are off. Coordination lives in `.superpowers/ledger.md`.
  Posting a claim or release comment is a public write; do it only if the
  owner turns claims on in the request.
- **GitHub writes are one pull request, plus an out-of-date close.** Read with
  `gh issue view`, `gh issue list`, and `gh api` GETs. Branch from `origin/main`
  (or from the previous open issue branch when stacking), push that branch, and open one pull request. Watch its checks and push fixes
  to the same branch. The owner merges. Put `Closes #N` in the pull request body
  when the pull request finishes the issue. Close an issue with `gh issue close`
  only after a command shows the current code already satisfies it or the issue
  is out of date, and cite that command and commit in the comment. #18 stays
  open for the owner. #63 is outside Phase 9; take it only when the launch
  prompt names it. The 2026-10-03 read-only rule applied to Phases 1 to 8.
- **The plan of record is `TASKS.md`**, not roadmap labels. Phase 9 is the
  queue. A row is eligible when its status is exactly `TODO` and every task id
  in Depends is `DONE` or `IN-PROGRESS` on the open stack. `TODO (owner)`, `BLOCKED`, and `BLOCKED-ON-INPUT` are
  not eligible.
- **OpenSpec, not Spec Kit.** The CLI is pinned in `mise.toml` as
  `npm:@fission-ai/openspec`; run it as `mise exec -- openspec ...`.
- **Ground truth is owner-owned.** `testdata/opencost-real/` (responses,
  `expected.json`, provenance) is read-only for you. Never regenerate or edit
  it to make a test pass.

## Phase 0: Preflight

```bash
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
git fetch origin
git rev-parse --abbrev-ref HEAD    # main equal to origin/main, or the previous issue branch when stacking
git status --short                 # tracked files clean
git rev-parse HEAD origin/main
mise exec -- openspec list --json  # .root.path must be $ROOT
```

The tree is clean before you branch. When the previous issue's pull request
is open, branch from that issue branch and open the new pull request against
it. Do not wait for the owner to merge. Otherwise start from `main` equal to
`origin/main` and create `issue-<n>-<slug>`. Continue when you are already on
that issue's branch and the only commits are yours for this issue. Any other
branch: stop and report. Stop when the tree has uncommitted tracked changes
you did not make. Never `git add -A` or `git add .`; stage named files only.
Push the issue branch. A push to `main` is outside this command.

## Phase 1: Choose

If an issue number was given, use it. Otherwise take the first Phase 9 row
whose status is exactly `TODO` and whose Depends task ids are `DONE` or
`IN-PROGRESS` on the open stack. Phase 0 rows marked `TODO (owner)` are the
owner's. When the issue has a Phase 9 row and that row is `BLOCKED` or
`BLOCKED-ON-INPUT`, report the row and stop.

Read the issue and its comments (`gh issue view N --comments`) and the
`TASKS.md` entry. Then reconcile before routing:

```bash
git log --oneline <checkpoint>..HEAD
mise exec -- openspec list --json
rg -n "<symbol or file the issue cites>"
```

A mismatch is the normal case. The work may already be done: report the
commit and mark it `DONE (already)` in the ledger. An OpenSpec change may
already exist: resume it. Every file path, function name or line number in an
old issue is evidence to re-check, not an instruction.

## Phase 2: Route

First match wins.

| # | Issue shape | Route |
| --- | --- | --- |
| 1 | Marked Kubecost-only, or needs a backend this repo cannot run | Do not implement. Record it in the "Not delivered" register with the reason |
| 2 | Changes plugin behaviour: an RPC, the config surface, an endpoint profile, the resource-type mapping, the cost arithmetic | OpenSpec change, then implement, then verify |
| 3 | Contained: tests only, docs only, CI, a bug fix with no surface change | Direct implementation with a test |
| 4 | Needs a finfocus-spec change | Apply the genericity rule in the run prompt first. Usually the answer is plugin, core, or docs |

Do not send a one-file fix through OpenSpec. Propose, apply, verify and
archive for a one-line change is ceremony without protection.

### The OpenSpec pipeline (row 2)

Use the skills under `.claude/skills/` (created by `openspec init`) and the
`/opsx:*` commands. Do not hand-roll the files.

```text
openspec-propose         proposal.md, design.md, specs/<capability>/spec.md, tasks.md
openspec-apply-change    work the tasks, ticking tasks.md as you go
openspec-verify-change   compare the code with the artifacts; fix every real finding
openspec-archive-change  fold the delta specs into openspec/specs/, move the change to archive/
```

Every task in a change's `tasks.md` carries a `Verify:` clause that is a
command with an expected result, plus a break check (a one-line change that
must make the command fail). Flag shapes for CLI 1.13.2:

```bash
mise exec -- openspec status --change "<slug>" --json
mise exec -- openspec validate "<slug>" --strict
mise exec -- openspec archive "<slug>" -y
```

Archive lands in the same commit as the code, after verify passes.

## Phase 3: Work it

One conventional commit per task on the issue branch, header ending in the task
id, for example `feat(client): add opencost endpoint profile (OC-3.2)`.
Stage named files only. Run the commit message through
`cat PR_MESSAGE.md | npx commitlint` before committing. Never add
`Co-Authored-By` or session-link trailers. Push the issue branch and open one
pull request with `gh pr create`. The body carries the verify output and, when
the pull request finishes the issue, `Closes #N`.

When the task says to stop and ask, stop. Do not invent the missing input.
For OC-9.3 the missing input is the owner-captured controller fixture.

Use TDD: a failing test first. Use testify. Tests that mutate environment or
process state are not parallel.

## Phase 4: Verify

```bash
make build
make test
make lint
npx markdownlint-cli2 "**/*.md"
mise exec -- openspec validate --all --strict
```

When the issue touches the client, the server or the cost arithmetic, also
run the real-backend suite:

```bash
make e2e-kind
```

That target creates the kind cluster, installs Prometheus and the OpenCost
chart with fixed pricing, waits for data, and runs the oracle comparison. A
result that never fails proves nothing: confirm the break check fails first.
Report real failures and unavailable tools. Do not edit a test, a fixture or
the oracle to make a gate pass.

## Pull request mode

Pull request mode is on (owner, 2026-10-04). Open one pull request, watch its
checks, and push fixes to that branch. The owner merges. Stack the next issue
on the previous open pull request. Do not wait for that merge. After a base merges, rebase the next
branch onto `main`. Phases 1 to 8 stayed
on the local run branch under the 2026-10-03 rule.

## Phase 5: Report and stop

Update the ledger and the task status. Report: the issue and why it was
chosen, the route, the OpenSpec change if any, gate results, the commits, the
pull request URL, and anything that goes in the "Not delivered" register. Then
stop.
