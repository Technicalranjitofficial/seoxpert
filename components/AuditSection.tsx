"use client";
import { useState, useEffect, useCallback } from "react";
import { createClient } from "@/lib/supabase/client";
import {
  Play, RefreshCw, CheckCircle, Clock, XCircle,
  AlertTriangle, Info, ChevronDown, ChevronUp,
  Globe, Zap, ExternalLink, TrendingUp,
} from "lucide-react";

interface Project {
  id: string;
  name: string;
  domain: string;
}

interface AuditIssue {
  id: string;
  audit_id: string;
  check_type: string;
  severity: "critical" | "warning" | "info";
  title: string;
  description: string;
  suggestion: string;
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

function scoreColor(s: number) {
  if (s >= 80) return { text: "text-emerald-400", ring: "#34d399", bg: "bg-emerald-500/10 border-emerald-500/20" };
  if (s >= 50) return { text: "text-amber-400",   ring: "#fbbf24", bg: "bg-amber-500/10 border-amber-500/20"   };
  return            { text: "text-red-400",        ring: "#f87171", bg: "bg-red-500/10 border-red-500/20"       };
}

function ScoreRing({ score }: { score: number }) {
  const r = 36, circ = 2 * Math.PI * r;
  const offset = circ - (score / 100) * circ;
  const c = scoreColor(score);
  return (
    <div className="relative w-24 h-24 flex items-center justify-center shrink-0">
      <svg className="absolute inset-0 -rotate-90" width="96" height="96">
        <circle cx="48" cy="48" r={r} fill="none" stroke="rgba(255,255,255,0.06)" strokeWidth="8" />
        <circle cx="48" cy="48" r={r} fill="none" stroke={c.ring} strokeWidth="8"
          strokeDasharray={circ} strokeDashoffset={offset} strokeLinecap="round"
          style={{ transition: "stroke-dashoffset 1s ease" }} />
      </svg>
      <div className="text-center z-10">
        <p className={`text-2xl font-black leading-none ${c.text}`}>{score}</p>
        <p className="text-gray-500 text-[10px] font-medium">/100</p>
      </div>
    </div>
  );
}

const SEV = {
  critical: { color: "text-red-400",   bg: "bg-red-500/10 border-red-500/20",     icon: <XCircle size={13} />,      dot: "bg-red-400"   },
  warning:  { color: "text-amber-400", bg: "bg-amber-500/10 border-amber-500/20", icon: <AlertTriangle size={13} />, dot: "bg-amber-400" },
  info:     { color: "text-blue-400",  bg: "bg-blue-500/10 border-blue-500/20",   icon: <Info size={13} />,          dot: "bg-blue-400"  },
};

function IssueCard({ issue }: { issue: AuditIssue }) {
  const [open, setOpen] = useState(false);
  const m = SEV[issue.severity];
  return (
    <div className={`rounded-xl border ${m.bg} overflow-hidden`}>
      <button className="w-full text-left px-4 py-3 flex items-start gap-3" onClick={() => setOpen(v => !v)}>
        <span className={`mt-0.5 shrink-0 ${m.color}`}>{m.icon}</span>
        <div className="flex-1 min-w-0">
          <p className={`text-sm font-semibold ${m.color}`}>{issue.title}</p>
          <p className="text-gray-400 text-xs mt-0.5 line-clamp-1">{issue.description}</p>
        </div>
        <span className={`shrink-0 mt-0.5 ${m.color} opacity-60`}>
          {open ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </span>
      </button>
      {open && (
        <div className="px-4 pb-4 pt-2 border-t border-white/5 space-y-3">
          <p className="text-gray-300 text-sm leading-relaxed">{issue.description}</p>
          {issue.suggestion && (
            <div className="flex gap-2 items-start bg-indigo-500/10 border border-indigo-500/20 rounded-lg p-3">
              <Zap size={13} className="text-indigo-400 mt-0.5 shrink-0" />
              <p className="text-indigo-200 text-xs leading-relaxed">{issue.suggestion}</p>
            </div>
          )}
          <a href={issue.url} target="_blank" rel="noreferrer"
            className="inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-indigo-400 font-mono transition-colors">
            <ExternalLink size={11} />{issue.url}
          </a>
        </div>
      )}
    </div>
  );
}

export default function AuditSection({ project, initialAudits }: {
  project: Project;
  initialAudits: Audit[];
}) {
  const supabase = createClient();
  const [audits, setAudits]     = useState<Audit[]>(initialAudits);
  const [selected, setSelected] = useState<Audit | null>(null);
  const [triggering, setTriggering] = useState(false);
  const [error, setError]       = useState("");
  const [pollingId, setPollingId] = useState<string | null>(null);
  const [filter, setFilter]     = useState<"all"|"critical"|"warning"|"info">("all");

  const fetchAudit = useCallback(async (id: string): Promise<Audit | null> => {
    const session = (await supabase.auth.getSession()).data.session;
    if (!session) return null;
    const res = await fetch(`/api/audits/${id}`, {
      headers: { Authorization: `Bearer ${session.access_token}` }, cache: "no-store",
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
    const neu: Audit = { id: auditId, project_id: project.id, status: "pending", created_at: new Date().toISOString() };
    setAudits(prev => [neu, ...prev]);
    setSelected(neu);
    setPollingId(auditId);
    setTriggering(false);
  };

  const openAudit = async (audit: Audit) => {
    setSelected(audit); setFilter("all");
    if (audit.status === "completed" && !Array.isArray(audit.issues)) {
      const full = await fetchAudit(audit.id);
      if (full) { setSelected(full); setAudits(prev => prev.map(a => a.id === audit.id ? full : a)); }
    }
    if (audit.status === "running" || audit.status === "pending") setPollingId(audit.id);
  };

  const issueCount = (a: Audit) => typeof a.issues === "number" ? a.issues : (Array.isArray(a.issues) ? a.issues.length : 0);

  return (
    <div className="space-y-5">
      {/* Trigger */}
      <div className="bg-gradient-to-r from-indigo-600/10 to-violet-600/10 border border-indigo-500/20 rounded-2xl p-5 flex items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <div className="w-11 h-11 rounded-xl bg-indigo-500/20 border border-indigo-500/20 flex items-center justify-center shrink-0">
            <Globe size={19} className="text-indigo-400" />
          </div>
          <div>
            <p className="text-white font-semibold">Run SEO Audit</p>
            <p className="text-gray-400 text-sm">Up to 50 pages · 18 checks · Full detailed report</p>
          </div>
        </div>
        <button onClick={triggerAudit} disabled={triggering}
          className="flex items-center gap-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-semibold rounded-xl transition-all shadow-lg shadow-indigo-500/20 shrink-0">
          {triggering ? <RefreshCw size={15} className="animate-spin" /> : <Play size={15} />}
          {triggering ? "Starting…" : "Run Audit"}
        </button>
      </div>

      {error && (
        <div className="text-red-300 text-sm bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 flex items-center gap-2">
          <XCircle size={15} className="text-red-400 shrink-0" />{error}
        </div>
      )}

      <div className="grid grid-cols-5 gap-4 items-start">
        {/* History */}
        <div className="col-span-2 space-y-2">
          <p className="text-gray-500 text-xs font-semibold uppercase tracking-widest mb-3">Audit History</p>
          {audits.length === 0 ? (
            <div className="bg-white/[0.02] border border-white/8 rounded-2xl p-8 text-center">
              <TrendingUp size={28} className="text-gray-700 mx-auto mb-2" />
              <p className="text-gray-500 text-sm">No audits yet</p>
            </div>
          ) : audits.map(a => {
            const score = typeof a.score === "number" ? a.score : null;
            const sc = score != null ? scoreColor(score) : null;
            const active = selected?.id === a.id;
            return (
              <button key={a.id} onClick={() => openAudit(a)}
                className={`w-full text-left p-4 rounded-xl border transition-all ${active
                  ? "bg-indigo-500/10 border-indigo-500/40 shadow-lg shadow-indigo-500/5"
                  : "bg-white/[0.02] border-white/8 hover:border-white/20 hover:bg-white/[0.04]"}`}>
                <div className="flex items-center justify-between mb-1.5">
                  <div className="flex items-center gap-2">
                    {a.status === "pending"   && <Clock size={13} className="text-amber-400" />}
                    {a.status === "running"   && <RefreshCw size={13} className="text-indigo-400 animate-spin" />}
                    {a.status === "completed" && <CheckCircle size={13} className="text-emerald-400" />}
                    {a.status === "failed"    && <XCircle size={13} className="text-red-400" />}
                    <span className="text-white text-sm font-medium capitalize">{a.status}</span>
                  </div>
                  {score != null && <span className={`text-lg font-black ${sc?.text}`}>{score}</span>}
                </div>
                <div className="flex items-center justify-between">
                  <p className="text-gray-600 text-xs">{new Date(a.created_at).toLocaleString()}</p>
                  {issueCount(a) > 0 && <span className="text-gray-600 text-xs">{issueCount(a)} issues</span>}
                </div>
                {a.status === "running" && a.crawled_pages != null && (
                  <div className="mt-2.5">
                    <div className="h-1 bg-white/5 rounded-full overflow-hidden">
                      <div className="h-full bg-indigo-500 rounded-full transition-all"
                        style={{ width: `${Math.min(100, ((a.crawled_pages ?? 0) / (a.total_pages || 50)) * 100)}%` }} />
                    </div>
                    <p className="text-gray-600 text-xs mt-1">{a.crawled_pages}/{a.total_pages ?? "?"} pages</p>
                  </div>
                )}
              </button>
            );
          })}
        </div>

        {/* Detail */}
        <div className="col-span-3">
          {!selected
            ? <div className="bg-white/[0.02] border border-white/8 rounded-2xl p-12 text-center flex flex-col items-center gap-3">
                <TrendingUp size={36} className="text-gray-700" />
                <p className="text-gray-500">Select an audit to view the full report</p>
              </div>
            : <AuditDetail audit={selected} filter={filter} setFilter={setFilter} />
          }
        </div>
      </div>
    </div>
  );
}

function AuditDetail({ audit, filter, setFilter }: {
  audit: Audit;
  filter: "all"|"critical"|"warning"|"info";
  setFilter: (f: "all"|"critical"|"warning"|"info") => void;
}) {
  if (audit.status === "pending" || audit.status === "running") {
    return (
      <div className="bg-white/[0.02] border border-white/8 rounded-2xl p-12 text-center flex flex-col items-center gap-4">
        <div className="w-16 h-16 rounded-full bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center">
          <RefreshCw size={28} className="text-indigo-400 animate-spin" />
        </div>
        <div>
          <p className="text-white font-semibold text-lg">Audit in progress…</p>
          <p className="text-gray-400 text-sm mt-1">
            {audit.crawled_pages ? `Crawled ${audit.crawled_pages} of ${audit.total_pages ?? "?"} pages` : "Starting crawler…"}
          </p>
        </div>
        {audit.crawled_pages != null && (
          <div className="w-full max-w-xs">
            <div className="h-1.5 bg-white/5 rounded-full overflow-hidden">
              <div className="h-full bg-indigo-500 rounded-full transition-all"
                style={{ width: `${Math.min(100, ((audit.crawled_pages ?? 0) / (audit.total_pages || 50)) * 100)}%` }} />
            </div>
          </div>
        )}
      </div>
    );
  }

  if (audit.status === "failed") {
    return (
      <div className="bg-red-500/5 border border-red-500/20 rounded-2xl p-12 text-center flex flex-col items-center gap-3">
        <XCircle size={36} className="text-red-400" />
        <p className="text-white font-semibold">Audit Failed</p>
        <p className="text-gray-400 text-sm">The crawler could not complete. Try running again.</p>
      </div>
    );
  }

  const all: AuditIssue[] = Array.isArray(audit.issues) ? audit.issues : [];
  const critical = all.filter(i => i.severity === "critical");
  const warnings = all.filter(i => i.severity === "warning");
  const infos    = all.filter(i => i.severity === "info");
  const score    = audit.score ?? 0;
  const c        = scoreColor(score);

  const visible = filter === "all" ? all : filter === "critical" ? critical : filter === "warning" ? warnings : infos;

  const byPage = visible.reduce<Record<string, AuditIssue[]>>((acc, i) => {
    const k = i.url ?? "unknown";
    (acc[k] = acc[k] || []).push(i);
    return acc;
  }, {});

  return (
    <div className="space-y-4">
      {/* Score card */}
      <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-5">
        <div className="flex items-center gap-5">
          <ScoreRing score={score} />
          <div className="flex-1">
            <p className="text-white font-semibold mb-3">SEO Health Score</p>
            <div className="grid grid-cols-3 gap-2">
              {([["critical","Critical",critical.length],["warning","Warnings",warnings.length],["info","Info",infos.length]] as const).map(([sev, label, count]) => (
                <button key={sev} onClick={() => setFilter(filter === sev ? "all" : sev)}
                  className={`rounded-xl border px-3 py-2 text-center transition-all ${filter === sev ? SEV[sev].bg + " scale-[1.04]" : "bg-white/[0.03] border-white/10 hover:border-white/20"}`}>
                  <p className={`text-xl font-black ${SEV[sev].color}`}>{count}</p>
                  <p className="text-gray-500 text-xs mt-0.5">{label}</p>
                </button>
              ))}
            </div>
          </div>
          <div className="text-right shrink-0">
            <p className="text-gray-500 text-xs mb-1">Pages crawled</p>
            <p className="text-white font-bold text-2xl">{audit.crawled_pages ?? "—"}</p>
            {audit.completed_at && (
              <>
                <p className="text-gray-500 text-xs mt-3 mb-1">Completed</p>
                <p className="text-gray-400 text-xs">{new Date(audit.completed_at).toLocaleString()}</p>
              </>
            )}
          </div>
        </div>
        {score >= 80 && (
          <div className="mt-4 flex items-center gap-2 text-sm bg-emerald-500/10 border border-emerald-500/20 rounded-xl px-4 py-2.5">
            <CheckCircle size={14} className="text-emerald-400 shrink-0" />
            <span className="text-emerald-300">Great SEO health! A few minor improvements available below.</span>
          </div>
        )}
        {score < 50 && (
          <div className="mt-4 flex items-center gap-2 text-sm bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-2.5">
            <AlertTriangle size={14} className="text-red-400 shrink-0" />
            <span className="text-red-300">Significant issues found. Fix critical items first for the biggest score boost.</span>
          </div>
        )}
      </div>

      {/* Filter tabs */}
      {all.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          {(["all","critical","warning","info"] as const).map(f => (
            <button key={f} onClick={() => setFilter(f)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${filter === f ? "bg-indigo-600 text-white shadow-sm" : "bg-white/[0.03] border border-white/10 text-gray-400 hover:text-white"}`}>
              {f === "all" ? `All (${all.length})` : f === "critical" ? `⊗ Critical (${critical.length})` : f === "warning" ? `⚠ Warning (${warnings.length})` : `ℹ Info (${infos.length})`}
            </button>
          ))}
        </div>
      )}

      {/* Issues grouped by page */}
      {all.length === 0 ? (
        <div className="bg-emerald-500/5 border border-emerald-500/20 rounded-2xl p-10 text-center">
          <CheckCircle size={36} className="text-emerald-400 mx-auto mb-3" />
          <p className="text-white font-semibold text-lg">No issues found!</p>
          <p className="text-gray-400 text-sm mt-1">This site has excellent SEO health.</p>
        </div>
      ) : visible.length === 0 ? (
        <p className="text-gray-500 text-sm text-center py-6">No {filter} issues found.</p>
      ) : (
        <div className="space-y-3 max-h-[560px] overflow-y-auto pr-1">
          {Object.entries(byPage).map(([url, pageIssues]) => (
            <div key={url} className="bg-white/[0.02] border border-white/8 rounded-2xl overflow-hidden">
              <div className="flex items-center gap-2 px-4 py-2.5 bg-white/[0.02] border-b border-white/5">
                <Globe size={11} className="text-gray-600 shrink-0" />
                <a href={url} target="_blank" rel="noreferrer"
                  className="text-gray-500 text-xs font-mono truncate flex-1 hover:text-indigo-400 transition-colors">{url}</a>
                <span className="text-gray-600 text-xs shrink-0">{pageIssues.length} issue{pageIssues.length > 1 ? "s" : ""}</span>
              </div>
              <div className="p-3 space-y-2">
                {pageIssues.map(issue => <IssueCard key={issue.id} issue={issue} />)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
