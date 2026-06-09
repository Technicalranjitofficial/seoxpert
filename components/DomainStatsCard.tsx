"use client";
import { useEffect, useState } from "react";
import { Calendar, Globe, Clock, Shield, CheckCircle, AlertTriangle, RefreshCw, ExternalLink, Lock, Unlock } from "lucide-react";

interface DomainInfo {
  domain: string;
  registrar: string | null;
  created_at: string | null;
  expires_at: string | null;
  domain_age_years: number | null;
  days_to_expiry: number | null;
  first_indexed: string | null;
  https: boolean | null;
  status_code: number | null;
  response_ms: number | null;
}

function Stat({
  label, value, icon, warn = false, good = false,
}: {
  label: string; value: string | null; icon: React.ReactNode; warn?: boolean; good?: boolean;
}) {
  const color = warn ? "text-amber-400" : good ? "text-emerald-400" : "text-white";
  return (
    <div className="flex items-center gap-3 px-4 py-3 bg-white/2 border border-white/6 rounded-xl min-w-0">
      <div className="text-gray-500 shrink-0">{icon}</div>
      <div className="min-w-0">
        <p className="text-gray-500 text-[10px] font-semibold uppercase tracking-wider">{label}</p>
        <p className={`text-sm font-semibold mt-0.5 truncate ${color}`}>{value ?? "—"}</p>
      </div>
    </div>
  );
}

function fmt_date(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });
}

export default function DomainStatsCard({ domain }: { domain: string }) {
  const [data, setData]       = useState<DomainInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setError(null);
    fetch(`/api/da?domain=${encodeURIComponent(domain)}`)
      .then(r => r.json())
      .then(j => { if (j.error) setError(j.error); else setData(j); })
      .catch(() => setError("Failed to load domain info"))
      .finally(() => setLoading(false));
  }, [domain]);

  if (loading) return (
    <div className="bg-white/2 border border-white/8 rounded-2xl p-5 flex items-center gap-3">
      <RefreshCw size={15} className="text-indigo-400 animate-spin shrink-0" />
      <p className="text-gray-500 text-sm">Loading domain information…</p>
    </div>
  );

  if (error || !data) return (
    <div className="bg-amber-500/5 border border-amber-500/20 rounded-2xl p-4 flex items-center gap-3">
      <AlertTriangle size={14} className="text-amber-400 shrink-0" />
      <p className="text-amber-300 text-sm">{error ?? "No domain data available"}</p>
    </div>
  );

  const expiryWarn = data.days_to_expiry !== null && data.days_to_expiry < 60;
  const httpsOk    = data.https === true;
  const speedLabel = data.response_ms === null ? null
    : data.response_ms < 500 ? "Fast" : data.response_ms < 1500 ? "Moderate" : "Slow";
  const speedGood  = data.response_ms !== null && data.response_ms < 500;
  const speedWarn  = data.response_ms !== null && data.response_ms >= 1500;

  return (
    <div className="bg-white/2 border border-white/8 rounded-2xl overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 border-b border-white/6 bg-white/1">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-xl bg-violet-500/15 border border-violet-500/20 flex items-center justify-center">
            <Globe size={14} className="text-violet-400" />
          </div>
          <div>
            <p className="text-white font-semibold text-sm">Domain Overview</p>
            <p className="text-gray-500 text-xs">RDAP · Wayback Machine · Live check — no API key required</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {httpsOk
            ? <span className="flex items-center gap-1 text-xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 rounded-full px-2.5 py-0.5"><Lock size={10} /> HTTPS</span>
            : <span className="flex items-center gap-1 text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded-full px-2.5 py-0.5"><Unlock size={10} /> No HTTPS</span>
          }
          {data.status_code && (
            <span className={`text-xs rounded-full px-2.5 py-0.5 border font-mono ${data.status_code < 400 ? "text-emerald-400 bg-emerald-500/10 border-emerald-500/20" : "text-red-400 bg-red-500/10 border-red-500/20"}`}>
              {data.status_code}
            </span>
          )}
        </div>
      </div>

      <div className="p-5 space-y-4">
        {/* Age + expiry banner */}
        {data.domain_age_years !== null && (
          <div className="flex items-center gap-4 bg-white/2 border border-white/6 rounded-xl px-5 py-4">
            <div className="text-center shrink-0">
              <p className="text-3xl font-black text-white leading-none">{data.domain_age_years}</p>
              <p className="text-gray-500 text-xs mt-0.5">years old</p>
            </div>
            <div className="w-px h-10 bg-white/8 shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-white text-sm font-semibold">
                {data.domain_age_years >= 5 ? "Established domain" : data.domain_age_years >= 2 ? "Maturing domain" : "New domain"}
              </p>
              <p className="text-gray-500 text-xs mt-0.5">
                {data.domain_age_years >= 5
                  ? "Older domains generally rank better — Google trusts age."
                  : data.domain_age_years >= 2
                  ? "Good foundation. Domain trust builds over time."
                  : "New domains may take 6–12 months to build authority."}
              </p>
            </div>
            {expiryWarn && (
              <div className="shrink-0 flex items-center gap-1.5 text-xs text-amber-300 bg-amber-500/10 border border-amber-500/20 rounded-xl px-3 py-2">
                <AlertTriangle size={12} />
                Expires in {data.days_to_expiry}d
              </div>
            )}
          </div>
        )}

        {/* Stats grid */}
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          <Stat label="Registered"    value={fmt_date(data.created_at)}  icon={<Calendar size={13} />} />
          <Stat label="Expires"       value={fmt_date(data.expires_at)}  icon={<Calendar size={13} />} warn={expiryWarn} />
          <Stat label="First Indexed" value={data.first_indexed ? fmt_date(data.first_indexed) : "—"} icon={<ExternalLink size={13} />} />
          <Stat label="Registrar"     value={data.registrar}              icon={<Shield size={13} />} />
          <Stat label="Response Time" value={data.response_ms ? `${data.response_ms}ms${speedLabel ? ` · ${speedLabel}` : ""}` : null} icon={<Clock size={13} />} good={speedGood} warn={speedWarn} />
          <Stat label="Live Status"   value={data.status_code ? `HTTP ${data.status_code}` : "Unreachable"} icon={<CheckCircle size={13} />} good={!!data.status_code && data.status_code < 400} warn={!!data.status_code && data.status_code >= 400} />
        </div>
      </div>
    </div>
  );
}


interface DomainStats {
  domain: string;
  trust_flow: number | null;
  citation_flow: number | null;
  ref_domains: number | null;
  ext_backlinks: number | null;
  indexed_urls: number | null;
  tf_cf_ratio: number | null;
}

function ScoreBadge({ value, max = 100, label }: { value: number | null; max?: number; label: string }) {
  if (value === null) return (
    <div className="flex flex-col items-center gap-1">
      <div className="w-14 h-14 rounded-2xl bg-white/3 border border-white/8 flex items-center justify-center">
        <span className="text-gray-600 text-xs">—</span>
      </div>
      <span className="text-gray-600 text-[10px] font-medium">{label}</span>
    </div>
  );

  const pct = (value / max) * 100;
  const color = pct >= 60 ? "text-emerald-400" : pct >= 30 ? "text-amber-400" : "text-red-400";
  const bg    = pct >= 60 ? "bg-emerald-500/10 border-emerald-500/25" : pct >= 30 ? "bg-amber-500/10 border-amber-500/25" : "bg-red-500/10 border-red-500/25";
  const ring  = pct >= 60 ? "#34d399" : pct >= 30 ? "#fbbf24" : "#f87171";
  const r = 22, circ = 2 * Math.PI * r;
  const offset = circ - (pct / 100) * circ;

  return (
    <div className="flex flex-col items-center gap-1.5">
      <div className={`relative w-16 h-16 rounded-2xl border flex items-center justify-center ${bg}`}>
        <svg className="absolute inset-0 -rotate-90 w-full h-full" viewBox="0 0 64 64">
          <circle cx="32" cy="32" r={r} fill="none" stroke="rgba(255,255,255,0.06)" strokeWidth="5" />
          <circle cx="32" cy="32" r={r} fill="none" stroke={ring} strokeWidth="5"
            strokeDasharray={circ} strokeDashoffset={offset} strokeLinecap="round"
            style={{ transition: "stroke-dashoffset 1s ease" }} />
        </svg>
        <span className={`z-10 text-lg font-black ${color}`}>{value}</span>
      </div>
      <span className="text-gray-500 text-[10px] font-semibold uppercase tracking-wider">{label}</span>
    </div>
  );
}

function Stat({ label, value, icon }: { label: string; value: string | null; icon: React.ReactNode }) {
  return (
    <div className="flex items-center gap-3 px-4 py-3 bg-white/2 border border-white/6 rounded-xl">
      <div className="text-gray-500 shrink-0">{icon}</div>
      <div className="min-w-0">
        <p className="text-gray-500 text-[10px] font-semibold uppercase tracking-wider">{label}</p>
        <p className="text-white text-sm font-semibold mt-0.5">{value ?? "—"}</p>
      </div>
    </div>
  );
}

function fmt(n: number | null): string {
  if (n === null) return "—";
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toString();
}

export default function DomainStatsCard({ domain }: { domain: string }) {
  const [data, setData] = useState<DomainStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setError(null);
    fetch(`/api/da?domain=${encodeURIComponent(domain)}`)
      .then(r => r.json())
      .then(j => {
        if (j.error) setError(j.error);
        else setData(j);
      })
      .catch(() => setError("Failed to load domain metrics"))
      .finally(() => setLoading(false));
  }, [domain]);

  if (loading) {
    return (
      <div className="bg-white/2 border border-white/8 rounded-2xl p-5 flex items-center gap-3">
        <RefreshCw size={15} className="text-indigo-400 animate-spin shrink-0" />
        <p className="text-gray-500 text-sm">Loading domain authority metrics…</p>
      </div>
    );
  }

  if (error) {
    // If API key not set, show a soft "configure" state instead of an error
    const notConfigured = error.includes("not configured");
    return (
      <div className={`border rounded-2xl p-5 flex items-center gap-3 ${notConfigured ? "bg-white/2 border-white/8" : "bg-amber-500/5 border-amber-500/20"}`}>
        <AlertTriangle size={15} className={notConfigured ? "text-gray-600" : "text-amber-400"} />
        <p className={`text-sm ${notConfigured ? "text-gray-500" : "text-amber-300"}`}>
          {notConfigured
            ? "Domain authority metrics not configured — add MAJESTIC_API_KEY to .env"
            : error}
        </p>
        {notConfigured && (
          <a href="https://developer.majestic.com/" target="_blank" rel="noreferrer"
            className="ml-auto flex items-center gap-1 text-xs text-indigo-400 hover:text-indigo-300 shrink-0">
            Get free key <ExternalLink size={10} />
          </a>
        )}
      </div>
    );
  }

  if (!data) return null;

  const tfLabel = data.trust_flow === null ? "—"
    : data.trust_flow >= 50 ? "High authority"
    : data.trust_flow >= 20 ? "Moderate authority"
    : "Low authority";

  const ratioLabel = data.tf_cf_ratio === null ? null
    : data.tf_cf_ratio >= 0.5 ? "High-quality links" : "Mixed link quality";

  return (
    <div className="bg-white/2 border border-white/8 rounded-2xl overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 border-b border-white/6 bg-white/1">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-xl bg-violet-500/15 border border-violet-500/20 flex items-center justify-center">
            <Shield size={14} className="text-violet-400" />
          </div>
          <div>
            <p className="text-white font-semibold text-sm">Domain Authority</p>
            <p className="text-gray-500 text-xs">Powered by Majestic · Updated daily</p>
          </div>
        </div>
        <a href={`https://majestic.com/reports/site-explorer?IndexDataSource=F&oq=${data.domain}&q=${data.domain}`}
          target="_blank" rel="noreferrer"
          className="flex items-center gap-1 text-xs text-gray-600 hover:text-indigo-400 transition-colors">
          Full report <ExternalLink size={10} />
        </a>
      </div>

      <div className="p-5 space-y-5">
        {/* Score badges */}
        <div className="flex items-start gap-6">
          <div className="flex gap-5">
            <ScoreBadge value={data.trust_flow}    label="Trust Flow" />
            <ScoreBadge value={data.citation_flow} label="Citation Flow" />
          </div>

          <div className="flex-1 space-y-2.5 min-w-0">
            {/* Authority interpretation */}
            <div className="flex items-center gap-2">
              <div className={`h-2 flex-1 rounded-full overflow-hidden bg-white/5`}>
                <div className="h-full rounded-full transition-all duration-1000"
                  style={{
                    width: `${data.trust_flow ?? 0}%`,
                    background: (data.trust_flow ?? 0) >= 50 ? "#34d399" : (data.trust_flow ?? 0) >= 20 ? "#fbbf24" : "#f87171",
                  }} />
              </div>
              <span className="text-gray-400 text-xs font-medium shrink-0">{data.trust_flow ?? 0}/100</span>
            </div>
            <p className="text-white text-sm font-semibold">{tfLabel}</p>
            {ratioLabel && (
              <p className="text-gray-500 text-xs">{ratioLabel} (TF/CF ratio: {data.tf_cf_ratio})</p>
            )}
            <p className="text-gray-600 text-xs leading-relaxed">
              Trust Flow measures link quality from trusted seed sites. Citation Flow measures raw link volume. Higher TF/CF ratio = better link quality.
            </p>
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 gap-2">
          <Stat label="Ref. Domains"  value={fmt(data.ref_domains)}   icon={<Globe size={13} />} />
          <Stat label="Backlinks"     value={fmt(data.ext_backlinks)} icon={<Link2 size={13} />} />
          <Stat label="Indexed URLs"  value={fmt(data.indexed_urls)}  icon={<TrendingUp size={13} />} />
        </div>
      </div>
    </div>
  );
}
