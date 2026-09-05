#!/usr/bin/env python3
"""Local-only HTTP server for Miyuna Gold Pilot review helper."""

from __future__ import annotations

import json
import socket
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from review_store import ReviewStore

HELPER_DIR = Path(__file__).resolve().parent
DEFAULT_HOST = "127.0.0.1"
DEFAULT_PORT = 8765


class ReviewHandler(BaseHTTPRequestHandler):
    store: ReviewStore

    def log_message(self, format: str, *args) -> None:  # noqa: A003
        sys.stderr.write("%s - %s\n" % (self.address_string(), format % args))

    def _full_state_payload(self) -> dict:
        payload = self.store.state_payload()
        payload["apiVersion"] = 2
        payload["aiSuggestionsLoaded"] = len(self.store.ai_by_id)
        return payload

    def _send_json(self, payload: dict, status: int = 200) -> None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_json(self) -> dict:
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length else b"{}"
        return json.loads(raw.decode("utf-8"))

    def do_GET(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)
        path = parsed.path

        if path in ("/", "/index.html"):
            index_path = HELPER_DIR / "index.html"
            content = index_path.read_bytes()
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(content)))
            self.end_headers()
            self.wfile.write(content)
            return

        if path == "/api/state":
            params = parse_qs(parsed.query)
            review_filter = params.get("filter", [None])[0]
            if review_filter:
                self.store.set_filter(review_filter)
            index_param = params.get("index", [None])[0]
            if index_param is not None:
                self.store.set_current_index(int(index_param))
            payload = self._full_state_payload()
            self._send_json(payload)
            return

        if path.startswith("/api/record/"):
            try:
                index = int(path.rsplit("/", 1)[-1])
                self.store.set_current_index(index)
                self._send_json(self._full_state_payload())
            except (ValueError, IndexError) as exc:
                self._send_json({"error": str(exc)}, status=400)
            return

        self._send_json({"error": "not found"}, status=404)

    def do_POST(self) -> None:  # noqa: N802
        parsed = urlparse(self.path)
        if parsed.path == "/api/filter":
            try:
                payload = self._read_json()
                self.store.set_filter(payload["filter"])
                self._send_json(self._full_state_payload())
            except (ValueError, KeyError, json.JSONDecodeError) as exc:
                self._send_json({"error": str(exc)}, status=400)
            return

        if parsed.path == "/api/bulk-approve-safe":
            try:
                payload = self._read_json()
                if not payload.get("confirmed"):
                    self._send_json({"error": "confirmation required"}, status=400)
                    return
                result = self.store.approve_safe_bulk_candidates(
                    expected_count=int(payload["expectedCount"]),
                    confirmed=True,
                )
                self._send_json(self._full_state_payload() | {"ok": True, "bulkResult": result})
            except (ValueError, KeyError, json.JSONDecodeError) as exc:
                self._send_json({"error": str(exc)}, status=400)
            return

        if parsed.path != "/api/action":
            self._send_json({"error": "not found"}, status=404)
            return

        try:
            payload = self._read_json()
            index = int(payload.get("index", self.store.current_index))
            action = payload.get("action")
            manual_labels = payload.get("manualLabels")
            reviewer_notes = payload.get("reviewerNotes")
            record = self.store.apply_action(
                index=index,
                action=action,
                manual_labels=manual_labels,
                reviewer_notes=reviewer_notes,
            )
            if action in ("approve", "approve_ai", "save", "reject"):
                if self.store.progress()["pending"] > 0:
                    self.store.set_current_index(
                        self.store.next_pending_index(index, self.store.active_filter)
                    )
                else:
                    self.store.set_current_index(index)
            self._send_json(self._full_state_payload() | {"ok": True})
        except (ValueError, IndexError, json.JSONDecodeError) as exc:
            self._send_json({"error": str(exc)}, status=400)


def main() -> int:
    host = DEFAULT_HOST
    port = DEFAULT_PORT
    probe = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        probe.bind((host, port))
    except OSError:
        print(f"ERROR: Port {port} is already in use.")
        print("Stop the old review helper (Ctrl+C in its terminal), then restart.")
        return 1
    finally:
        probe.close()

    store = ReviewStore()
    restored = store.restore_accidental_rejections_if_needed()
    ReviewHandler.store = store

    server = ThreadingHTTPServer((host, port), ReviewHandler)
    ai_count = len(store.ai_by_id)
    pending = store.progress()["pending"]
    print(f"Gold review helper running at http://{host}:{port}")
    print(f"Loaded {ai_count} AI suggestions for {pending} pending records.")
    if ai_count == 0 and pending > 0:
        print("WARNING: No AI suggestions found. Run: python ../run_ai_prereview.py")
    if restored:
        print(f"Restored {restored} accidental rejection(s) to PENDING_HUMAN_REVIEW.")
    safe_bulk = store.queue_counts()["SAFE_BULK_CANDIDATES"]
    print(f"Safe bulk candidates: {safe_bulk}")
    print("Press Ctrl+C to stop.")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
