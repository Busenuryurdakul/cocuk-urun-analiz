#!/usr/bin/env python3
"""One-shot Amazon Baby real import E2E runner (local dev)."""

from __future__ import annotations

import base64
import json
import re
import sys
import time
import uuid
from pathlib import Path

import pyotp
import requests

API = "http://localhost:8080/graphql"
MAILHOG = "http://localhost:8025/api/v2/messages"
ARTIFACT = Path(__file__).resolve().parent / "output" / "miyuna_amazon_baby_import.jsonl"
E2E_SUBSET = Path(__file__).resolve().parent / "output" / "_e2e_import_subset.jsonl"
IMPORT_LINES = 1000
POLL_SECONDS = 180


def gql(session: requests.Session, query: str, variables: dict | None = None) -> dict:
    resp = session.post(API, json={"query": query, "variables": variables or {}})
    resp.raise_for_status()
    payload = resp.json()
    if payload.get("errors"):
        raise RuntimeError(json.dumps(payload["errors"], ensure_ascii=False))
    return payload["data"]


def _mail_recipients(item: dict) -> list[str]:
    recipients: list[str] = []
    for to in item.get("To") or []:
        if isinstance(to, dict):
            recipients.append(f"{to.get('Mailbox', '')}@{to.get('Domain', '')}".lower())
        else:
            recipients.append(str(to).lower())
    header_to = item.get("Content", {}).get("Headers", {}).get("To") or []
    recipients.extend(str(x).lower() for x in header_to)
    return recipients


def wait_mail(session: requests.Session, email: str, timeout: int = 30) -> str:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = session.get(MAILHOG, params={"limit": 10})
        resp.raise_for_status()
        for item in resp.json().get("items", []):
            if email.lower() not in _mail_recipients(item):
                continue
            body = item.get("Content", {}).get("Body", "")
            m = re.search(r"token=([A-Za-z0-9_-]+)", body)
            if m:
                return m.group(1)
        time.sleep(1)
    raise TimeoutError(f"verification mail not found for {email}")


def wait_device_code(session: requests.Session, email: str, timeout: int = 30) -> str:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = session.get(MAILHOG, params={"limit": 10})
        resp.raise_for_status()
        for item in resp.json().get("items", []):
            if email.lower() not in _mail_recipients(item):
                continue
            subject = (item.get("Content", {}).get("Headers", {}).get("Subject") or [""])[0]
            if "cihaz" not in subject.lower():
                continue
            body = item.get("Content", {}).get("Body", "")
            m = re.search(r"(\d{6})", body)
            if m:
                return m.group(1)
        time.sleep(1)
    raise TimeoutError(f"device verification mail not found for {email}")


def make_subset() -> tuple[Path, int]:
    lines = [line for line in ARTIFACT.read_text(encoding="utf-8").splitlines() if line.strip()]
    subset = lines[:IMPORT_LINES]
    E2E_SUBSET.write_text("\n".join(subset) + "\n", encoding="utf-8")
    return E2E_SUBSET, len(lines)


def authenticate(session: requests.Session, email: str, password: str, fp: str) -> str:
    token = wait_mail(session, email)
    setup = gql(
        session,
        "mutation($token:String!){ verifyEmail(token:$token){ secret otpauthUrl } }",
        {"token": token},
    )["verifyEmail"]
    secret = setup["secret"]
    gql(
        session,
        "mutation($input:ConfirmMFAInput!){ confirmMFA(input:$input) }",
        {"input": {"code": pyotp.TOTP(secret).now()}},
    )

    login = gql(
        session,
        "mutation($input:LoginInput!){ login(input:$input){ status user { id email } } }",
        {"input": {"email": email, "password": password, "deviceFingerprint": fp}},
    )["login"]
    print(f"LOGIN_STATUS={login['status']}")

    if login["status"] == "MFA_REQUIRED":
        login = gql(
            session,
            "mutation($input:VerifyLoginMFAInput!){ verifyLoginMFA(input:$input){ status user { id } } }",
            {"input": {"code": pyotp.TOTP(secret).now(), "deviceFingerprint": fp}},
        )["verifyLoginMFA"]
        print(f"LOGIN_AFTER_MFA={login['status']}")

    if login["status"] == "DEVICE_VERIFICATION_REQUIRED":
        device_code = wait_device_code(session, email)
        login = gql(
            session,
            "mutation($input:VerifyDeviceInput!){ verifyDevice(input:$input){ status user { id } } }",
            {"input": {"code": device_code, "deviceFingerprint": fp}},
        )["verifyDevice"]
        print(f"LOGIN_AFTER_DEVICE={login['status']}")

    if login["status"] != "AUTHENTICATED":
        raise RuntimeError(f"login not authenticated: {login['status']}")

    me = gql(session, "query { me { id personalOrgId email } }")["me"]
    return me["personalOrgId"]


def main() -> int:
    if not ARTIFACT.exists():
        print("INPUT_ARTIFACT_MISSING")
        return 1

    subset_path, total_records = make_subset()
    email = f"e2e-{uuid.uuid4().hex[:8]}@miyuna.local"
    password = "Password123!"
    fp = f"e2e-device-{uuid.uuid4().hex[:8]}"

    s = requests.Session()
    print(f"ARTIFACT_TOTAL={total_records}")
    print(f"IMPORT_SUBSET={IMPORT_LINES}")

    gql(
        s,
        "mutation($input:RegisterInput!){ register(input:$input){ message } }",
        {"input": {"email": email, "password": password}},
    )

    org_id = authenticate(s, email, password, fp)
    print(f"ORG_ID={org_id}")

    gql(
        s,
        "mutation($input:GrantConsentInput!){ grantConsent(input:$input){ id purpose } }",
        {"input": {"purpose": "DATA_PROCESSING", "organizationId": org_id}},
    )

    content_b64 = base64.b64encode(subset_path.read_bytes()).decode("ascii")
    run = gql(
        s,
        """
        mutation($input:StartMarketplaceFileImportInput!){
          startMarketplaceFileImport(input:$input){
            id status recordsSeen recordsAccepted recordsRejected
          }
        }
        """,
        {
            "input": {
                "organizationId": org_id,
                "source": "OTHER",
                "accessMode": "JSON_IMPORT",
                "filename": "miyuna_amazon_baby_import.jsonl",
                "contentType": "application/jsonl",
                "contentBase64": content_b64,
            }
        },
    )["startMarketplaceFileImport"]
    run_id = run["id"]
    print(f"IMPORT_RUN_ID={run_id}")
    print(f"IMPORT_INITIAL_STATUS={run['status']}")

    final = None
    deadline = time.time() + POLL_SECONDS
    while time.time() < deadline:
        status = gql(
            s,
            """
            query($org:ID!, $run:ID!){
              marketplaceImportStatus(organizationId:$org, importRunId:$run){
                status recordsSeen recordsAccepted recordsRejected errorCode errorMessage
              }
            }
            """,
            {"org": org_id, "run": run_id},
        )["marketplaceImportStatus"]
        if status["status"] in {"SUCCEEDED", "PARTIAL", "FAILED", "REJECTED_BY_POLICY"}:
            final = status
            break
        time.sleep(2)

    if not final:
        print("IMPORT_TIMEOUT")
        return 1

    print("IMPORT_FINAL", json.dumps(final, ensure_ascii=False))

    products = gql(
        s,
        "query($org:ID!){ products(organizationId:$org, limit:5){ id name { value missing } } }",
        {"org": org_id},
    )["products"]
    product_id = products[0]["id"] if products else None
    print(f"PRODUCT_SAMPLE={product_id}")

    reviews = []
    if product_id:
        reviews = gql(
            s,
            """
            query($org:ID!, $pid:ID!){
              productMarketplaceReviews(organizationId:$org, productId:$pid){
                id reviewText rating datasetEligibility source
              }
            }
            """,
            {"org": org_id, "pid": product_id},
        )["productMarketplaceReviews"]

    summary = gql(
        s,
        """
        query($org:ID!){
          datasetEligibilitySummary(organizationId:$org){
            totalRecords marketplaceCount eligibilityDistribution { eligibility count }
          }
        }
        """,
        {"org": org_id},
    )["datasetEligibilitySummary"]

    ux = None
    if product_id:
        ux = gql(
            s,
            """
            mutation($input:CreateUserExperienceInput!){
              createUserExperience(input:$input){
                id moderationStatus datasetEligibility narrative
              }
            }
            """,
            {
                "input": {
                    "organizationId": org_id,
                    "productId": product_id,
                    "usageStatus": "USING",
                    "satisfactionLevel": "NEUTRAL",
                    "rating": 3,
                    "issueType": "USABILITY",
                    "narrative": "LOCAL E2E TEST: Miyuna portal deneyimi kaydi.",
                }
            },
        )["createUserExperience"]

    report = {
        "artifact_total": total_records,
        "import_subset": IMPORT_LINES,
        "org_id": org_id,
        "import_run": final,
        "products_sample_count": len(products),
        "reviews_sample_count": len(reviews),
        "review_sample_eligibility": reviews[0]["datasetEligibility"] if reviews else None,
        "review_sample_source": reviews[0]["source"] if reviews else None,
        "eligibility_summary": summary,
        "ugc": ux,
    }
    out = Path(__file__).resolve().parent / "output" / "_e2e_report.json"
    out.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    print("E2E_REPORT", out)
    return 0 if final["status"] in {"SUCCEEDED", "PARTIAL"} else 1


if __name__ == "__main__":
    sys.exit(main())
