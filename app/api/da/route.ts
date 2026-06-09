import { NextRequest, NextResponse } from "next/server";
import { createClient } from "@/lib/supabase/server";

// Majestic API — free developer tier: 1000 checks/month
// Register at: https://developer.majestic.com/
// Env var: MAJESTIC_API_KEY

export async function GET(req: NextRequest) {
  const supabase = await createClient();
  const { data: { user } } = await supabase.auth.getUser();
  if (!user) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const domain = req.nextUrl.searchParams.get("domain");
  if (!domain) return NextResponse.json({ error: "domain parameter required" }, { status: 400 });

  const apiKey = process.env.MAJESTIC_API_KEY;
  if (!apiKey) {
    return NextResponse.json({ error: "Majestic API key not configured" }, { status: 503 });
  }

  // Strip protocol + path, keep bare domain
  const clean = domain.replace(/^https?:\/\//, "").replace(/\/.*$/, "").toLowerCase();

  const url = new URL("https://api.majestic.com/api/json");
  url.searchParams.set("app_api_key", apiKey);
  url.searchParams.set("cmd", "GetIndexItemInfo");
  url.searchParams.set("items", "1");
  url.searchParams.set("item0", clean);
  url.searchParams.set("datasource", "fresh"); // latest crawl data

  const res = await fetch(url.toString(), {
    next: { revalidate: 86400 }, // cache 24 h
  });

  if (!res.ok) {
    return NextResponse.json({ error: "Majestic API error", status: res.status }, { status: 502 });
  }

  const json = await res.json().catch(() => null);
  if (!json || json.Code !== "OK") {
    return NextResponse.json({ error: json?.ErrorMessage ?? "Majestic API failed" }, { status: 502 });
  }

  const item = json.DataTables?.Results?.Data?.[0];
  if (!item) {
    return NextResponse.json({ error: "No data returned for domain" }, { status: 404 });
  }

  return NextResponse.json({
    domain: clean,
    trust_flow:      item.TrustFlow      ?? null,   // 0–100, like DA
    citation_flow:   item.CitationFlow   ?? null,   // 0–100, raw link power
    ref_domains:     item.RefDomains     ?? null,   // unique linking domains
    ext_backlinks:   item.ExtBackLinks   ?? null,   // total backlinks
    indexed_urls:    item.IndexedURLs    ?? null,
    // TF/CF ratio — closer to 1.0 = higher quality links
    tf_cf_ratio: item.TrustFlow && item.CitationFlow
      ? Math.round((item.TrustFlow / Math.max(item.CitationFlow, 1)) * 100) / 100
      : null,
  });
}
