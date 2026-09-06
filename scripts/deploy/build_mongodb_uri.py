"""Build a MongoDB Atlas URI without shell-escaping issues."""
from __future__ import annotations

import os
import sys
from urllib.parse import quote_plus


def main() -> int:
    user = os.environ.get("MONGODB_USER", "yurdakulbusenur38_db_user")
    password = os.environ.get("MONGODB_PASSWORD", "")
    host = os.environ.get("MONGODB_HOST", "cluster0.wnbyhet.mongodb.net")
    database = os.environ.get("MONGODB_DATABASE", "miyuna")
    auth_source = os.environ.get("MONGODB_AUTH_SOURCE", "admin")

    if not password:
        print("MONGODB_PASSWORD is required.", file=sys.stderr)
        return 1

    query = "retryWrites=true&w=majority"
    if auth_source:
        query = f"authSource={quote_plus(auth_source)}&{query}"

    uri = (
        f"mongodb+srv://{quote_plus(user)}:{quote_plus(password)}"
        f"@{host}/{quote_plus(database)}?{query}"
    )
    sys.stdout.write(uri)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
