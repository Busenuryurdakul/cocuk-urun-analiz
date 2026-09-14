import { NextRequest, NextResponse } from "next/server";
import { upstreamGraphqlUrl } from "@/lib/api-base";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";
// startAgentRun used to block on Render agent cold-start; keep headroom for slow upstream paths.
export const maxDuration = 120;

function forwardSetCookies(upstream: Response, response: NextResponse) {
  const setCookies =
    typeof upstream.headers.getSetCookie === "function" ? upstream.headers.getSetCookie() : [];
  if (setCookies.length > 0) {
    for (const cookie of setCookies) {
      response.headers.append("Set-Cookie", cookie);
    }
    return;
  }
  const legacy = upstream.headers.get("set-cookie");
  if (legacy) {
    response.headers.set("Set-Cookie", legacy);
  }
}

async function proxyGraphql(req: NextRequest) {
  const contentType = req.headers.get("content-type") ?? "application/json";
  const cookie = req.headers.get("cookie");
  const body = await req.text();

  const upstream = await fetch(upstreamGraphqlUrl(), {
    method: "POST",
    headers: {
      "Content-Type": contentType,
      ...(cookie ? { cookie } : {}),
    },
    body,
    cache: "no-store",
  });

  const responseBody = await upstream.text();
  const response = new NextResponse(responseBody, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") ?? "application/json",
    },
  });
  forwardSetCookies(upstream, response);
  return response;
}

export async function POST(req: NextRequest) {
  try {
    return await proxyGraphql(req);
  } catch {
    return NextResponse.json(
      { errors: [{ message: "upstream unavailable", extensions: { code: "UPSTREAM_UNAVAILABLE" } }] },
      { status: 502 },
    );
  }
}
