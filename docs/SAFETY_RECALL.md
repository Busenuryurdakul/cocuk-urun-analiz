# Miyuna — Safety Findings and Official Recall Sources

> Implementation state for **P0 PR-A2**. This is not Phase 7 complete.

## 1. Scope

PR-A2 implements:

- Safety finding domain + Mongo persistence
- Deterministic `safety_analyzer` via Executor
- Official CPSC adapter + GÜBİS adapter boundary
- Recall matching, false-positive guard
- Confirmed recall → Evidence via PR-A1 `EvidenceService`
- GraphQL read: `analysisSafetyFindings`, `analysisRecallMatches`, `AnalysisRun.safetyFindings`, `AnalysisRun.recalls`

Deferred to PR-A3:

- planner / orchestrator wiring
- `review_analyzer`
- final structured analysis, confidence engine, ALLOW/BLOCK
- Safety/Evidence UI

## 2. Safety Analyzer

Input: `analysisRunId`, `productId`, `evidenceIds`, optional `recallIds`.

Output: `{ findings, issues }`.

Rules:

- CRITICAL requires strong evidence
- official recall + strong product match may be high confidence
- weak fuzzy match cannot become a confirmed CRITICAL finding
- no supporting evidence → `INSUFFICIENT_EVIDENCE` / `NO_EVIDENCE`
- cross-org evidence → `WRONG_TENANT`

`safety_analyzer` is AVAILABLE for `Executor.Execute`. Python planner still excludes it.

## 3. Official sources

### CPSC

Canonical source: SaferProducts REST API

`https://www.saferproducts.gov/RestWebServices/Recall?format=json`

Adapter search uses UPC, product name, or manufacturer. Empty identity does not query the unbounded catalog.

### GÜBİS

No stable public machine-readable API is available (`https://gubis.ticaret.gov.tr/`).

The adapter interface, normalization, and contract fixtures exist. Live search returns `NO_STABLE_PUBLIC_MACHINE_READABLE_API`.

## 4. Matching

Priority: source id → GTIN → UPC/EAN → manufacturer+model → brand+model → brand+normalized name → controlled fuzzy.

Confirmed match = matched + not `requiresReview` + confidence ≥ 0.90.

Low-confidence fuzzy matches stay review-required signals only.

## 5. Security

- tenant-scoped finding and match reads
- adapters never fetch arbitrary user URLs
- host allowlist, redirect cap, timeout, bounded retries, response size limit
- private/internal targets are not accepted as official sources
