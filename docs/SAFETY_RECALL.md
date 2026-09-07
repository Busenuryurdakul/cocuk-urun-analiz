# Miyuna — Safety Findings and Official Recall Sources

> Implementation state for **P0 PR-A2 + PR-A3 + PR-C validation**. GÜBİS live remains environment-dependent; real LLM live verify documented in [REAL_LLM_FINAL_VERIFICATION.md](./REAL_LLM_FINAL_VERIFICATION.md).

## 1. Scope

PR-A2 implemented:

- Safety finding domain + Mongo persistence
- Deterministic `safety_analyzer` via Executor
- Official CPSC adapter + GÜBİS adapter boundary
- Recall matching, false-positive guard
- Confirmed recall → Evidence via PR-A1 `EvidenceService`
- GraphQL read: `analysisSafetyFindings`, `analysisRecallMatches`, `AnalysisRun.safetyFindings`, `AnalysisRun.recalls`

PR-A3 added:

- planner / orchestrator wiring for `safety_analyzer`
- `review_analyzer` implementation (AVAILABLE)
- structured `finalResult` on `analysis_runs` (schema `1.0.0`)
- hallucination guard, confidence engine, ALLOW/BLOCK policy engine
- minimum Product/Analysis UI

## 2. Safety Analyzer

Input: `analysisRunId`, `productId`, `evidenceIds`, optional `recallIds`.

Output: `{ findings, issues }`.

Rules:

- CRITICAL requires strong evidence
- official recall + strong product match may be high confidence
- weak fuzzy match cannot become a confirmed CRITICAL finding
- no supporting evidence → `INSUFFICIENT_EVIDENCE` / `NO_EVIDENCE`
- cross-org evidence → `WRONG_TENANT`

`safety_analyzer` is AVAILABLE and planner-included when the registry marks it AVAILABLE.

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

## 5. Decision policy (PR-A3)

LLM output cannot set the product decision. Deterministic rules:

- confirmed critical official recall → `BLOCK`
- high severity + strong evidence → `BLOCK`
- weak/fuzzy recall or unsupported claim → `REVIEW_REQUIRED` (never `BLOCK`)
- contradictory evidence → `REVIEW_REQUIRED`
- medium supported warning → `ALLOW_WITH_WARNING`
- no significant supported risk → `ALLOW`

## 6. Security

- tenant-scoped finding, match, and finalResult reads
- adapters never fetch arbitrary user URLs
- host allowlist, redirect cap, timeout, bounded retries, response size limit
- private/internal targets are not accepted as official sources
