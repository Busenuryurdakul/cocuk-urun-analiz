import { NextRequest, NextResponse } from "next/server";

const VERIFY_RE = /https?:\/\/[^\s"'<>]+\/auth\/verify-email\?token=[A-Za-z0-9_-]+/;

type MailhogItem = {
  Content?: { Body?: string; Headers?: Record<string, string[]> };
  Raw?: { Data?: string };
};

function isLocalDev() {
  return process.env.NODE_ENV !== "production";
}

export async function GET(req: NextRequest) {
  if (!isLocalDev()) {
    return NextResponse.json({ error: "not found" }, { status: 404 });
  }
  const email = (req.nextUrl.searchParams.get("email") ?? "").trim().toLowerCase();
  if (!email || !email.includes("@")) {
    return NextResponse.json({ url: null }, { status: 400 });
  }
  try {
    const res = await fetch("http://127.0.0.1:8025/api/v2/messages?limit=20", { cache: "no-store" });
    if (!res.ok) {
      return NextResponse.json({ url: null });
    }
    const payload = (await res.json()) as { items?: MailhogItem[] };
    for (const item of payload.items ?? []) {
      const headers = item.Content?.Headers ?? {};
      const to = (headers.To ?? []).join(" ").toLowerCase();
      const body = `${item.Content?.Body ?? ""}\n${item.Raw?.Data ?? ""}`;
      if (to && !to.includes(email)) {
        continue;
      }
      const match = body.match(VERIFY_RE);
      if (match) {
        return NextResponse.json({ url: match[0] });
      }
    }
    return NextResponse.json({ url: null });
  } catch {
    return NextResponse.json({ url: null });
  }
}
