"use client";
import { useState, useEffect, useCallback } from "react";
import { createClient } from "@/lib/supabase/client";
import {
  Play, RefreshCw, CheckCircle, Clock, XCircle,
  AlertTriangle, Info, ChevronDown,
  Globe, Zap, ExternalLink, TrendingUp,
  ArrowRight, Lightbulb, Shield,
} from "lucide-react";

interface Project { id: string; name: string; domain: string; }

interface AuditIssue {
  id: string;
  audit_id: string;
  check_type: string;
  severity: "critical" | "warning" | "info";
  title: string;
  description: string;
  suggestion: string;
  value: string;
  url: string;
}

interface Audit {
  id: string;
  project_id: string;
  status: "pending" | "running" | "completed" | "failed";
  score?: number;
  crawled_pages?: number;
  total_pages?: number;
  issues?: AuditIssue[] | number;
  created_at: string;
  completed_at?: string;
}

function timeAgo(date: string) {
  const s = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

function grade(score: number) {
  if (score >= 90) return { letter: "A", label: "Excellent", color: "text-emerald-400", bg: "bg-emerald-500/15 border-emerald-500/30" };
  if (score >= 75) return { letter: "B", label: "Good",      color: "text-green-400",   bg: "bg-green-500/15 border-green-500/30"   };
  if (score >= 60) return { letter: "C", label: "Fair",      color: "text-yellow-400",  bg: "bg-yellow-500/15 border-yellow-500/30"  };
  if (score >= 40) return { letter: "D", label: "Poor",      color: "text-orange-400",  bg: "bg-orange-500/15 border-orange-500/30"  };
  return                  { letter: "F", label: "Critical",  color: "text-red-400",     bg: "bg-red-500/15 border-red-500/30"        };
}

function ringStroke(score: number) {
  if (score >= 75) return "#34d399";
  if (score >= 60) return "#facc15";
  if (score >= 40) return "#fb923c";
  return "#f87171";
}

function ScoreGauge({ score }: { score: number }) {
  const r = 42, circ = 2 * Math.PI * r;
  const offset = circ - (score / 100) * circ;
  const g = grade(score);
  return (
    <div className="relative w-28 h-28 flex items-center justify-center shrink-0">
      <svg className="absolute inset-0 -rotate-90" width="112" height="112">
        <circle cx="56" cy="56" r={r} fill="none" stroke="rgba(255,255,255,0.05)" strokeWidth="10" />
        <circle cx="56" cy="56" r={r} fill="none" stroke={ringStroke(score)} strokeWidth="10"
          strokeDasharray={circ} strokeDashoffset={offset} strokeLinecap="round"
          style={{ transition: "stroke-dashoffset 1.2s cubic-bezier(.4,0,.2,1)" }} />
      </svg>
      <div className="text-center z-10">
        <p className={`text-3xl font-black leading-none ${g.color}`}>{score}</p>
        <p className={`text-[11px] font-bold mt-0.5 opacity-60 ${g.color}`}>{g.letter}</p>
      </div>
    </div>
  );
}

const SEV = {
  critical: {
    color:   "text-red-400",
    bg:      "bg-red-950/40 border-red-500/25",
    header:  "bg-red-500/8",
    badge:   "bg-red-500/15 text-red-300 border-red-500/25",
    fixBg:   "bg-red-500/8 border-red-500/20",
    fixText: "text-red-200",
    icon:    <XCircle size={14} />,
  },
  warning: {
    color:   "text-amber-400",
    bg:      "bg-amber-950/30 border-amber-500/20",
    header:  "bg-amber-500/6",
    badge:   "bg-amber-500/15 text-amber-300 border-amber-500/25",
    fixBg:   "bg-amber-500/8 border-amber-500/20",
    fixText: "text-amber-100",
    icon:    <AlertTriangle size={14} />,
  },
  info: {
    color:   "text-sky-400",
    bg:      "bg-sky-950/20 border-sky-500/15",
    header:  "bg-sky-500/5",
    badge:   "bg-sky-500/15 text-sky-300 border-sky-500/25",
    fixBg:   "bg-sky-500/8 border-sky-500/15",
    fixText: "text-sky-100",
    icon:    <Info size={14} />,
  },
} as const;

function IssueCard({ issue, defaultOpen = false }: { issue: AuditIssue; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  const m = SEV[issue.severity];
  return (
    <div className={`rounded-2xl border overflow-hidden ${m.bg}`}>
      {/* clickable header */}
      <button
        className={`w-full text-left px-5 py-4 flex items-start gap-3.5 transition-colors ${open ? m.header : "hover:bg-white/2"}`}
        onClick={() => setOpen(v => !v)}
      >
        <span className={`mt-0.5 shrink-0 ${m.color}`}>{m.icon}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap mb-1">
            <span className={`inline-flex text-[10px] font-bold px-2 py-0.5 rounded-full border uppercase tracking-wider ${m.badge}`}>
              {issue.severity}
            </span>
            <span className="text-gray-600 text-[10px] font-mono">{issue.check_type}</span>
          </div>
          <p className={`text-sm font-semibold leading-snug ${m.color}`}>{issue.title}</p>
          {!open && (
            <p className="text-gray-500 text-xs mt-0.5 line-clamp-1">{issue.description}</p>
          )}
        </div>
        <span className={`shrink-0 mt-1 ${m.color} opacity-50 transition-transform ${open ? "rotate-180" : ""}`}>
          <ChevronDown size={15} />
        </span>
      </button>

      {/* expanded body */}
      {open && (
        <div className="px-5 pb-5 pt-3 space-y-4 border-t border-white/5">

          {/* What's wrong */}
          <div className="flex gap-3">
            <div className="shrink-0 w-5 h-5 mt-0.5 rounded-md bg-white/5 flex items-center justify-center">
              <AlertTriangle size={11} className="text-gray-500" />
            </div>
            <div>
              <p className="text-gray-500 text-[10px] font-bold uppercase tracking-wider mb-1">What&apos;s Wrong</p>
              <p className="text-gray-200 text-sm leading-relaxed">{issue.description}</p>
            </div>
          </div>

          {/* Detected value — the actual content that triggered the issue */}
          {issue.value && (
            <div>
              <p className="text-gray-500 text-[10px] font-bold uppercase tracking-wider mb-1.5">Detected Value</p>
              <div className="bg-black/40 border border-white/8 rounded-lg px-3.5 py-3">
                <p className={`text-sm font-mono break-all leading-relaxed ${m.color}`}>{issue.value}</p>
              </div>
            </div>
          )}

          {/* How to fix */}
          {issue.suggestion && (
            <div className={`rounded-xl border p-4 ${m.fixBg}`}>
              <div className="flex items-center gap-2 mb-2.5">
                <Lightbulb size={13} className={m.color} />
                <p className={`text-[11px] font-bold uppercase tracking-wider ${m.color}`}>How to Fix</p>
              </div>
              <p className={`text-sm leading-relaxed ${m.fixText}`}>{issue.suggestion}</p>
            </div>
          )}

          {/* Affected URL */}
          <a
            href={issue.url} target="_blank" rel="noreferrer"
            className="flex items-center gap-2 text-xs font-mono text-gray-500 hover:text-indigo-400 transition-colors"
          >
            <Globe size={11} className="shrink-0" />
            <span className="truncate">{issue.url}</span>
            <ExternalLink size={10} className="shrink-0" />
          </a>
        </div>
      )}
    </div>
  );
}

/* ─── MAIN EXPORT ─────────────────────────────────────────── */
export default function AuditSection({
  project,
  initialAudits,
}: {
  project: Project;
  initialAudits: Audit[];
}) {
  const supabase = createClient();
  const [audits, setAudits]     = useState<Audit[]>(initialAudits);
  const [selected, setSelected] = useState<Audit | null>(null);
  const [triggering, setTriggering] = useState(false);
  const [error, setError]       = useState("");
  const [pollingId, setPollingId] = useState<string | null>(null);
  const [filter, setFilter]     = useState<"all" | "critical" | "warning" | "info">("all");

  const fetchAudit = useCallback(async (id: string): Promise<Audit | null> => {
    const session = (await supabase.auth.getSession()).data.session;
    if (!session) return null;
    const res = await fetch(`/api/audits/${id}`, {
      headers: { Authorization: `Bearer ${session.access_token}` },
      cache: "no-store",
    });
    if (!res.ok) return null;
    const json = await res.json().catch(() => null);
    if (!json) return null;
    return { ...(json.data ?? json), issues: Array.isArray(json.issues) ? json.issues : [] };
  }, [supabase]);

  useEffect(() => {
    if (!pollingId) return;
    const t = setInterval(async () => {
      const upd = await fetchAudit(pollingId);
      if (!upd) return;
      setAudits(prev => prev.map(a => a.id === pollingId ? upd : a));
      if (selected?.id === pollingId) setSelected(upd);
      if (upd.status === "completed" || upd.status === "failed") setPollingId(null);
    }, 3000);
    return () => clearInterval(t);
  }, [pollingId, fetchAudit, selected?.id]);

  const triggerAudit = async () => {
    setError(""); setTriggering(true);
    const session = (await supabase.auth.getSession()).data.session;
    if (!session) { setTriggering(false); return; }
    const res = await fetch("/api/audits", {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${session.access_token}` },
      body: JSON.stringify({ project_id: project.id }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) { setError(data.error ?? "Failed to trigger audit."); setTriggering(false); return; }
    const auditId = data.data?.id ?? data.id;
    const neu: Audit = {
      id: auditId, project_id: project.id, status: "pending",
      created_at: new Date().toISOString(),
    };
    setAudits(prev => [neu, ...prev]);
    setSelected(neu); setPollingId(auditId); setTriggering(false);
  };

  const openAudit = async (audit: Audit) => {
    setSelected(audit); setFilter("all");
    if (audit.status === "completed" && !Array.isArray(audit.issues)) {
      const full = await fetchAudit(audit.id);
      if (full) { setSelected(full); setAudits(prev => prev.map(a => a.id === audit.id ? full : a)); }
    }
    if (audit.status === "running" || audit.status === "pending") setPollingId(audit.id);
  };

  const issueCount = (a: Audit) =>
    typeof a.issues === "number" ? a.issues : Array.isArray(a.issues) ? a.issues.length : 0;

  return (
    <div className="space-y-6">

      {/* ── Run Audit Banner ── */}
      <div className="relative overflow-hidden bg-linear-to-br from-indigo-600/15 via-violet-600/8 to-transparent border border-indigo-500/25 rounded-2xl p-6">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgba(99,102,241,0.18),transparent_65%)] pointer-events-none" />
        <div className="relative flex items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-2xl bg-linear-to-br from-indigo-500/30 to-violet-500/20 border border-indigo-500/30 flex items-center justify-center shrink-0">
              <Shield size={20} className="text-indigo-400" />
            </div>
            <div>
              <p className="text-white font-bold text-lg leading-tight">Run SEO Audit</p>
              <p className="text-gray-400 text-sm mt-0.5">Scans up to 50 pages · 18+ checks · Actionable fix guide</p>
            </div>
          </div>
          <button
            onClick={triggerAudit} disabled={triggering}
            className="flex items-center gap-2.5 px-6 py-3 bg-linear-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 disabled:opacity-50 text-white font-bold rounded-xl transition-all shadow-xl shadow-indigo-500/25 shrink-0 text-sm"
          >
            {triggering ? <RefreshCw size={15} className="animate-spin" /> : <Play size={15} />}
            {triggering ? "Starting…" : "Run Audit"}
          </button>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2.5 text-red-300 text-sm bg-red-500/10 border border-red-500/25 rounded-xl px-4 py-3">
          <XCircle size={15} className="text-red-400 shrink-0" />{error}
        </div>
      )}

      {audits.length === 0 ? (
        <div className="bg-white/2 border border-white/8 rounded-2xl p-16 text-center flex flex-col items-center gap-3">
          <div className="w-16 h-16 rounded-2xl bg-white/3 border border-white/8 flex items-center justify-center">
            <TrendingUp size={28} className="text-gray-600" />
          </div>
          <p className="text-white font-semibold">No audits yet</p>
          <p className="text-gray-500 text-sm">Run your first audit to see a full SEO report with actionable fixes</p>
        </div>
      ) : (
        <div className="grid grid-cols-5 gap-5 items-start">

          {/* ── Audit History ── */}
          <div className="col-span-2 space-y-2">
            <p className="text-gray-500 text-[11px] font-bold uppercase tracking-widest mb-3">Audit History</p>
            {audits.map(a => {
              const s = typeof a.score === "number" ? a.score : null;
              const g = s != null ? grade(s) : null;
              const active = selected?.id === a.id;
              const pct = a.crawled_pages != null
                ? Math.min(100, ((a.crawled_pages ?? 0) / (a.total_pages || 50)) * 100) : 0;

              return (
                <button key={a.id} onClick={() => openAudit(a)}
                  className={`w-full text-left p-4 rounded-2xl border transition-all ${
                    active
                      ? "bg-indigo-500/12 border-indigo-500/40 ring-1 ring-indigo-500/20 shadow-lg shadow-indigo-500/5"
                      : "bg-white/2 border-white/8 hover:border-white/18 hover:bg-white/4"
                  }`}
                >
                  <div className="flex items-center gap-3">
                    {/* Score square or status icon */}
                    <div className="shrink-0">
                      {s != null ? (
                        <div className={`w-12 h-12 rounded-xl border flex flex-col items-center justify-center ${g?.bg}`}>
                          <p className={`text-lg font-black leading-none ${g?.color}`}>{s}</p>
                          <p className={`text-[9px] font-bold opacity-60 ${g?.color}`}>{g?.letter}</p>
                        </div>
                      ) : (
                        <div className="w-12 h-12 rounded-xl border bg-white/3 border-white/10 flex items-center justify-center">
                          {a.status === "running" && <RefreshCw size={18} className="text-indigo-400 animate-spin" />}
                          {a.status === "pending" && <Clock size={18} className="text-amber-400" />}
                          {a.status === "failed"  && <XCircle size={18} className="text-red-400" />}
                        </div>
                      )}
                    </div>

                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-1.5 mb-0.5">
                        {a.status === "completed" && <CheckCircle size={11} className="text-emerald-400" />}
                        {a.status === "running"   && <RefreshCw   size={11} className="text-indigo-400 animate-spin" />}
                        {a.status === "pending"   && <Clock        size={11} className="text-amber-400" />}
                        {a.status === "failed"    && <XCircle      size={11} className="text-red-400" />}
                        <span className="text-white text-sm font-semibold capitalize">{a.status}</span>
                      </div>
                      <p className="text-gray-600 text-xs">{timeAgo(a.created_at)}</p>
                      {issueCount(a) > 0 && (
                        <p className="text-gray-500 text-xs mt-0.5">{issueCount(a)} issues found</p>
                      )}
                    </div>

                    {active && <ArrowRight size={14} className="text-indigo-400 shrink-0" />}
                  </div>

                  {a.status === "running" && (
                    <div className="mt-3">
                      <div className="flex justify-between mb-1.5">
                        <span className="text-gray-600 text-[10px]">Crawling pages…</span>
                        <span className="text-gray-600 text-[10px]">{a.crawled_pages ?? 0}/{a.total_pages ?? "?"}</span>
                      </div>
                      <div className="h-1.5 bg-white/5 rounded-full overflow-hidden">
                        <div className="h-full bg-linear-to-r from-indigo-500 to-violet-500 rounded-full transition-all"
                          style={{ width: `${pct}%` }} />
                      </div>
                    </div>
                  )}
                </button>
              );
            })}
          </div>

          {/* ── Detail Panel ── */}
          <div className="col-span-3">
            {!selected ? (
              <div className="bg-white/2 border border-white/8 rounded-2xl p-14 text-center flex flex-col items-center gap-4">
                <div className="w-20 h-20 rounded-3xl bg-white/3 border border-white/8 flex items-center justify-center">
                  <TrendingUp size={32} className="text-gray-600" />
                </div>
                <div>
                  <p className="text-white font-semibold">Select an audit</p>
                  <p className="text-gray-500 text-sm mt-1">Click any audit from the history to see the full report with fixes</p>
                </div>
              </div>
            ) : (
              <AuditDetail audit={selected} filter={filter} setFilter={setFilter} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

/* ─── AUDIT DETAIL PANEL ──────────────────────────────────── */
function AuditDetail({
  audit, filter, setFilter,
}: {
  audit: Audit;
  filter: "all" | "critical" | "warning" | "info";
  setFilter: (f: "all" | "critical" | "warning" | "info") => void;
}) {
  /* ── Running / pending state ── */
  if (audit.status === "pending" || audit.status === "running") {
    const pct = audit.crawled_pages != null
      ? Math.min(100, ((audit.crawled_pages ?? 0) / (audit.total_pages || 50)) * 100) : 0;
    return (
      <div className="bg-white/2 border border-white/8 rounded-2xl p-12 text-center flex flex-col items-center gap-5">
        <div className="relative">
          <div className="w-20 h-20 rounded-full border-2 border-indigo-500/30 flex items-center justify-center">
            <RefreshCw size={30} className="text-indigo-400 animate-spin" />
          </div>
          <div className="absolute inset-0 rounded-full bg-indigo-500/5 animate-pulse" />
        </div>
        <div>
          <p className="text-white font-bold text-xl">Audit in Progress</p>
          <p className="text-gray-400 text-sm mt-1.5">
            {audit.crawled_pages
              ? `Crawled ${audit.crawled_pages} of ${audit.total_pages ?? "?"} pages`
              : "Starting crawler…"}
          </p>
        </div>
        {audit.crawled_pages != null && (
          <div className="w-full max-w-xs">
            <div className="flex justify-between text-xs text-gray-600 mb-2">
              <span>{Math.round(pct)}% complete</span>
              <span>{audit.crawled_pages}/{audit.total_pages ?? "?"} pages</span>
            </div>
            <div className="h-2 bg-white/5 rounded-full overflow-hidden">
              <div className="h-full bg-linear-to-r from-indigo-500 to-violet-500 rounded-full transition-all duration-500"
                style={{ width: `${pct}%` }} />
            </div>
          </div>
        )}
        <p className="text-gray-600 text-xs">This may take 1–3 minutes depending on site size</p>
      </div>
    );
  }

  /* ── Failed state ── */
  if (audit.status === "failed") {
    return (
      <div className="bg-red-500/5 border border-red-500/20 rounded-2xl p-12 text-center flex flex-col items-center gap-4">
        <div className="w-16 h-16 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center">
          <XCircle size={28} className="text-red-400" />
        </div>
        <div>
          <p className="text-white font-bold text-lg">Audit Failed</p>
          <p className="text-gray-400 text-sm mt-1">The crawler could not complete. Please try again.</p>
        </div>
      </div>
    );
  }

  /* ── Completed state ── */
  const all: AuditIssue[] = Array.isArray(audit.issues) ? audit.issues : [];
  const critical = all.filter(i => i.severity === "critical");
  const warnings = all.filter(i => i.severity === "warning");
  const infos    = all.filter(i => i.severity === "info");
  const score    = audit.score ?? 0;
  const g        = grade(score);

  const visible =
    filter === "all"      ? all :
    filter === "critical" ? critical :
    filter === "warning"  ? warnings : infos;

  const byPage = visible.reduce<Record<string, AuditIssue[]>>((acc, i) => {
    const k = i.url ?? "unknown";
    (acc[k] = acc[k] || []).push(i);
    return acc;
  }, {});

  return (
    <div className="space-y-5">

      {/* ── Score Hero ── */}
      <div className="bg-white/3 border border-white/10 rounded-2xl p-6">
        <div className="flex items-start gap-6">
          <ScoreGauge score={score} />

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-1.5">
              <p className="text-white font-bold text-xl">SEO Score</p>
              <span className={`text-xs font-bold px-3 py-0.5 rounded-full border ${g.bg} ${g.color}`}>
                Grade {g.letter} — {g.label}
              </span>
            </div>
            <p className="text-gray-400 text-sm mb-4 leading-relaxed">
              {score >= 80 ? "Your site has strong SEO fundamentals. Focus on the remaining warnings." :
               score >= 60 ? "Good foundation, but several issues are limiting your rankings." :
               score >= 40 ? "Multiple problems are hurting your visibility. Prioritise the criticals." :
               "Severe SEO problems detected. Immediate action needed to rank."}
            </p>

            {/* Issue count chips */}
            <div className="grid grid-cols-3 gap-2.5">
              {([
                ["critical", "Critical",    critical.length, "Fix immediately — directly harm rankings"],
                ["warning",  "Warnings",    warnings.length, "Should fix — they affect SEO performance"],
                ["info",     "Suggestions", infos.length,    "Nice-to-have — best practice improvements"],
              ] as const).map(([sev, label, count, tip]) => (
                <button key={sev} onClick={() => setFilter(filter === sev ? "all" : sev)}
                  title={tip}
                  className={`rounded-xl border p-3 text-left transition-all ${
                    filter === sev
                      ? SEV[sev].bg + " scale-[1.04] shadow-lg"
                      : "bg-white/3 border-white/8 hover:border-white/18"
                  }`}
                >
                  <p className={`text-2xl font-black ${SEV[sev].color}`}>{count}</p>
                  <p className="text-gray-500 text-xs mt-0.5 font-medium">{label}</p>
                </button>
              ))}
            </div>
          </div>

          {/* Right meta */}
          <div className="text-right shrink-0 space-y-3">
            {audit.crawled_pages != null && (
              <div>
                <p className="text-gray-600 text-[10px] uppercase tracking-wider font-bold">Pages</p>
                <p className="text-white font-black text-3xl">{audit.crawled_pages}</p>
                <p className="text-gray-600 text-xs">crawled</p>
              </div>
            )}
            {audit.completed_at && (
              <div>
                <p className="text-gray-600 text-[10px] uppercase tracking-wider font-bold mt-2">Completed</p>
                <p className="text-gray-400 text-xs mt-0.5">{new Date(audit.completed_at).toLocaleString()}</p>
              </div>
            )}
          </div>
        </div>

        {/* Status banner */}
        {critical.length > 0 && (
          <div className="mt-5 flex items-start gap-3 bg-red-500/8 border border-red-500/20 rounded-xl px-4 py-3.5">
            <AlertTriangle size={15} className="text-red-400 mt-0.5 shrink-0" />
            <div>
              <p className="text-red-200 text-sm font-semibold">
                {critical.length} critical issue{critical.length > 1 ? "s" : ""} need{critical.length === 1 ? "s" : ""} immediate attention
              </p>
              <p className="text-red-400/70 text-xs mt-0.5">
                Fixing these will have the biggest positive impact on your search rankings.
              </p>
            </div>
          </div>
        )}
        {critical.length === 0 && score >= 75 && (
          <div className="mt-5 flex items-center gap-3 bg-emerald-500/8 border border-emerald-500/20 rounded-xl px-4 py-3">
            <CheckCircle size={15} className="text-emerald-400 shrink-0" />
            <p className="text-emerald-200 text-sm">
              No critical issues — great work! Address the warnings and suggestions below to push higher.
            </p>
          </div>
        )}
      </div>

      {/* ── Priority Fixes: top criticals callout ── */}
      {critical.length > 0 && filter === "all" && (
        <div className="bg-white/2 border border-red-500/20 rounded-2xl overflow-hidden">
          <div className="flex items-center gap-2.5 px-5 py-3.5 border-b border-red-500/15 bg-red-500/5">
            <Zap size={14} className="text-red-400" />
            <p className="text-red-200 text-sm font-bold">Fix These First</p>
            <span className="text-red-400/50 text-xs ml-1">— highest impact on rankings</span>
            <span className="ml-auto text-red-400/60 text-xs">{critical.length} critical issue{critical.length !== 1 ? "s" : ""}</span>
          </div>
          <div className="p-4 space-y-3">
            {critical.slice(0, 3).map(i => (
              <IssueCard key={i.id} issue={i} defaultOpen={critical.length <= 2} />
            ))}
            {critical.length > 3 && (
              <button
                onClick={() => setFilter("critical")}
                className="w-full text-center text-xs text-gray-500 hover:text-red-400 transition-colors py-2.5 border border-white/5 hover:border-red-500/20 rounded-xl"
              >
                View all {critical.length} critical issues →
              </button>
            )}
          </div>
        </div>
      )}

      {/* ── Filter Bar ── */}
      {all.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-gray-600 text-xs font-medium mr-1">Show:</span>
          {([
            ["all",      `All (${all.length})`],
            ["critical", `⊗ Critical (${critical.length})`],
            ["warning",  `⚠ Warnings (${warnings.length})`],
            ["info",     `ℹ Suggestions (${infos.length})`],
          ] as const).map(([f, label]) => (
            <button key={f} onClick={() => setFilter(f)}
              className={`px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                filter === f
                  ? "bg-indigo-600 text-white shadow-sm shadow-indigo-500/20"
                  : "bg-white/3 border border-white/10 text-gray-400 hover:text-white hover:border-white/20"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      )}

      {/* ── Issue List ── */}
      {all.length === 0 ? (
        <div className="bg-emerald-500/5 border border-emerald-500/20 rounded-2xl p-14 text-center flex flex-col items-center gap-4">
          <div className="w-16 h-16 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center">
            <CheckCircle size={28} className="text-emerald-400" />
          </div>
          <div>
            <p className="text-white font-bold text-lg">No issues found!</p>
            <p className="text-gray-400 text-sm mt-1">This site has excellent SEO health. Keep it up!</p>
          </div>
        </div>
      ) : visible.length === 0 ? (
        <p className="text-gray-500 text-sm text-center py-8">No {filter} issues found.</p>
      ) : (
        <div className="space-y-4 max-h-150 overflow-y-auto pr-1 -mr-1">
          {Object.entries(byPage).map(([url, pageIssues]) => {
            const pageCrit = pageIssues.filter(i => i.severity === "critical").length;
            return (
              <div key={url} className="bg-white/2 border border-white/8 rounded-2xl overflow-hidden">
                {/* Page URL header */}
                <div className="flex items-center gap-2.5 px-4 py-3 bg-white/2 border-b border-white/5">
                  <Globe size={12} className="text-gray-600 shrink-0" />
                  <a href={url} target="_blank" rel="noreferrer"
                    className="text-gray-400 text-xs font-mono truncate flex-1 hover:text-indigo-400 transition-colors">
                    {url}
                  </a>
                  <div className="flex items-center gap-2 shrink-0">
                    {pageCrit > 0 && (
                      <span className="flex items-center gap-1 text-[10px] text-red-400 bg-red-500/10 border border-red-500/20 rounded-full px-2 py-0.5 font-semibold">
                        <XCircle size={9} />{pageCrit} critical
                      </span>
                    )}
                    <span className="text-gray-600 text-[10px]">{pageIssues.length} issue{pageIssues.length > 1 ? "s" : ""}</span>
                  </div>
                </div>
                {/* Issues */}
                <div className="p-3 space-y-2.5">
                  {pageIssues.map(issue => (
                    <IssueCard key={issue.id} issue={issue} />
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
