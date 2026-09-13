#!/usr/bin/env python3
"""Predict Amazon Baby review rating via Miyuna LLM gateway (rating_prediction task)."""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request

INSTRUCTION = """Predict the original rating assigned to this Amazon baby product review.
The rating must be exactly one integer from: 1, 2, 3, 4, 5
Carefully read the complete product name and review.
Return only the rating number."""


def build_prompt(product: str, review: str) -> str:
    return f"""### Task
{INSTRUCTION}

### Product
{product}

### Review
{review}

### Rating
"""


def parse_rating(text: str) -> int | None:
    match = re.search(r"[1-5]", text.strip())
    if not match:
        return None
    return int(match.group(0))


def main() -> int:
    parser = argparse.ArgumentParser(description="Amazon Baby rating prediction")
    parser.add_argument("--product", required=True)
    parser.add_argument("--review", required=True)
    parser.add_argument("--org-id", default=os.environ.get("MIYUNA_ORG_ID", "6a9b348fc82d85a213d2d6f2"))
    parser.add_argument("--api-url", default=os.environ.get("GO_API_URL", "http://127.0.0.1:8080"))
    parser.add_argument(
        "--token",
        default=os.environ.get("AGENT_INTERNAL_TOKEN", os.environ.get("INTERNAL_TOKEN", "dev-internal-token-change-me")),
    )
    args = parser.parse_args()

    payload = {
        "organizationId": args.org_id,
        "taskType": "rating_prediction",
        "personaKey": "careful_analyst",
        "userPrompt": build_prompt(args.product, args.review),
        "requireEvidence": False,
    }
    req = urllib.request.Request(
        f"{args.api_url.rstrip('/')}/internal/llm/v1/complete",
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Content-Type": "application/json",
            "X-Miyuna-Internal-Token": args.token,
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            body = json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        print(exc.read().decode("utf-8", errors="replace"), file=sys.stderr)
        return 1

    content = body.get("content") or body.get("text") or ""
    rating = parse_rating(content)
    print(json.dumps({"raw": content.strip(), "rating": rating, "modelKey": body.get("modelKey")}, ensure_ascii=False))
    return 0 if rating is not None else 2


if __name__ == "__main__":
    raise SystemExit(main())
