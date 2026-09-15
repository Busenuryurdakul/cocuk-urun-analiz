import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
export const maxDuration = 30;

const DEFAULT_AGENT_URL = "https://miyuna-agent.onrender.com/health";
const DEFAULT_API_URL = "https://miyuna-api.onrender.com/health";

async function ping(url: string) {
  const res = await fetch(url, {
    method: "GET",
    cache: "no-store",
    signal: AbortSignal.timeout(25_000),
  });
  return { url, status: res.status };
}

export async function GET(req: NextRequest) {
  const cronSecret = process.env.CRON_SECRET?.trim();
  if (cronSecret) {
    const auth = req.headers.get("authorization") ?? "";
    if (auth !== `Bearer ${cronSecret}`) {
      return NextResponse.json({ error: "unauthorized" }, { status: 401 });
    }
  }

  const agentUrl = process.env.AGENT_WAKE_URL?.trim() || DEFAULT_AGENT_URL;
  const apiUrl = process.env.API_WAKE_URL?.trim() || DEFAULT_API_URL;

  const results = await Promise.allSettled([ping(agentUrl), ping(apiUrl)]);
  const summary = results.map((result, index) => {
    const label = index === 0 ? "agent" : "api";
    if (result.status === "fulfilled") {
      return { service: label, ...result.value, ok: result.value.status >= 200 && result.value.status < 300 };
    }
    return { service: label, ok: false, error: result.reason instanceof Error ? result.reason.message : "failed" };
  });

  return NextResponse.json({ ok: summary.every((item) => item.ok), summary });
}
