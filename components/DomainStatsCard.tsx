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

