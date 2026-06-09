import { NextRequest, NextResponse } from "next/server";
import { createClient } from "@/lib/supabase/server";

// All sources are 100% free with zero API keys:
//   • RDAP  (rdap.org  — ICANN standard)          → domain age, registrar, expiry
//   • Wayback Machine CDX API (archive.org)        → first indexed date on web
//   • HTTP HEAD request                            → HTTPS status, response time

export async function GET(req: NextRequest) {
  const supabase = await createClient();
  const { data: { user } } = await supabase.auth.getUser();
  if (!user) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const domain = req.nextUrl.searchParams.get("domain");
  if (!domain) return NextResponse.json({ error: "domain parameter required" }, { status: 400 });

  const clean = domain.replace(/^https?:\/\//, "").replace(/\/.*$/, "").toLowerCase();

  // Run all three lookups in parallel
  const [rdap, wayback, httpCheck] = await Promise.allSettled([
    fetchRDAP(clean),
    fetchWayback(clean),
    fetchHttpStatus(clean),
  ]);

  const r = rdap.status === "fulfilled" ? rdap.value : null;
  const w = wayback.status === "fulfilled" ? wayback.value : null;
  const h = httpCheck.status === "fulfilled" ? httpCheck.value : null;

  return NextResponse.json({
    domain: clean,
    // RDAP
    registrar:    r?.registrar   ?? null,
    created_at:   r?.created_at  ?? null,   // ISO date string
    expires_at:   r?.expires_at  ?? null,
    domain_age_years: r?.created_at
      ? Math.floor((Date.now() - new Date(r.created_at).getTime()) / (1000 * 60 * 60 * 24 * 365))
      : null,
    days_to_expiry: r?.expires_at
      ? Math.floor((new Date(r.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
      : null,
    // Wayback
    first_indexed: w?.first_indexed ?? null,  // ISO date string
    // HTTP check
    https:         h?.https        ?? null,
    status_code:   h?.status_code  ?? null,
    response_ms:   h?.response_ms  ?? null,
  }, {
    headers: { "Cache-Control": "public, s-maxage=86400" }, // cache 24 h
  });
}

async function fetchRDAP(domain: string) {
  // Use rdap.org which routes to the correct TLD registry automatically
  const res = await fetch(`https://rdap.org/domain/${domain}`, {
    headers: { Accept: "application/rdap+json" },
    signal: AbortSignal.timeout(8000),
  });
  if (!res.ok) return null;
  const j = await res.json();

  let registrar: string | null = null;
  let created_at: string | null = null;
  let expires_at: string | null = null;

  // Extract registrar from entities
  for (const entity of j.entities ?? []) {
    if (entity.roles?.includes("registrar")) {
      registrar = entity.vcardArray?.[1]
        ?.find((v: string[]) => v[0] === "fn")?.[3]
        ?? entity.handle ?? null;
      break;
    }
  }

  // Extract dates from events
  for (const ev of j.events ?? []) {
    if (ev.eventAction === "registration")  created_at = ev.eventDate ?? null;
    if (ev.eventAction === "expiration")    expires_at = ev.eventDate ?? null;
  }

  return { registrar, created_at, expires_at };
}

async function fetchWayback(domain: string) {
  // CDX API: get the very first snapshot ever recorded
  const url = `https://web.archive.org/cdx/search/cdx?url=${domain}&limit=1&output=json&fl=timestamp&from=19960101&to=20260101&fastLatest=false`;
  const res = await fetch(url, { signal: AbortSignal.timeout(6000) });
  if (!res.ok) return null;
  const rows: string[][] = await res.json();
  if (!rows || rows.length < 2) return null; // rows[0] is header
  const ts = rows[1][0]; // YYYYMMDDHHMMSS
  if (!ts || ts.length < 8) return null;
  const first_indexed = `${ts.slice(0,4)}-${ts.slice(4,6)}-${ts.slice(6,8)}`;
  return { first_indexed };
}

async function fetchHttpStatus(domain: string) {
  const start = Date.now();
  try {
    const res = await fetch(`https://${domain}`, {
      method: "HEAD",
      redirect: "follow",
      signal: AbortSignal.timeout(6000),
    });
    return {
      https: true,
      status_code: res.status,
      response_ms: Date.now() - start,
    };
  } catch {
    // Try HTTP fallback
    try {
      const res2 = await fetch(`http://${domain}`, {
        method: "HEAD",
        redirect: "follow",
        signal: AbortSignal.timeout(5000),
      });
      return {
        https: false,
        status_code: res2.status,
        response_ms: Date.now() - start,
      };
    } catch {
      return { https: false, status_code: null, response_ms: null };
    }
  }
}

