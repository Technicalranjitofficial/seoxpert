"use client";
import { useState, useEffect, useCallback } from "react";
import { createClient } from "@/lib/supabase/client";
import { Play, RefreshCw, CheckCircle, Clock, XCircle, AlertTriangle, Info } from "lucide-react";

interface Project {
  id: string;
  name: string;
  domain: string;
}

interface AuditIssue {
  id: string;
  check_name: string;
  severity: "critical" | "warning" | "info";
  message: string;
  element?: string;
  url: string;
}

interface Audit {
  id: string;
  project_id: string;
  status: "pending" | "running" | "completed" | "failed";
  score?: number;
  pages_crawled?: number;
  total_issues?: number;
  created_at: string;
  completed_at?: string;
  issues?: AuditIssue[];
}

const statusIcon = {
  pending: <Clock size={15} className="text-amber-400" />,
  running: <RefreshCw size={15} className="text-indigo-400 animate-spin" />,
  completed: <CheckCircle size={15} className="text-emerald-400" />,
  failed: <XCircle size={15} className="text-red-400" />,
};

const severityColor = {
  critical: "text-red-400 bg-red-500/10 border-red-500/20",
  warning: "text-amber-400 bg-amber-500/10 border-amber-500/20",
  info: "text-blue-400 bg-blue-500/10 border-blue-500/20",
};

const severityIcon = {
  critical: <XCircle size={13} />,
  warning: <AlertTriangle size={13} />,
  info: <Info size={13} />,
};

export default function AuditSection({
  project,
  initialAudits,
}: {
  project: Project;
  initialAudits: Audit[];
}) {
  const supabase = createClient();
  const [audits, setAudits] = useState<Audit[]>(initialAudits);
  const [selectedAudit, setSelectedAudit] = useState<Audit | null>(null);
  const [triggering, setTriggering] = useState(false);
  const [error, setError] = useState("");
  const [pollingId, setPollingId] = useState<string | null>(null);

  const fetchAudit = useCallback(async (auditId: string): Promise<Audit | null> => {
    const session = (await supabase.auth.getSession()).data.session;
    if (!session) return null;
    const res = await fetch(`/api/audits/${auditId}`, {
      headers: { Authorization: `Bearer ${session.access_token}` },
      cache: "no-store",
    });
    if (!res.ok) return null;
    const json = await res.json().catch(() => null);
    if (!json) return null;
    // API returns { data: audit, issues: [] } — merge into flat object
    const audit: Audit = { ...(json.data ?? json), issues: Array.isArray(json.issues) ? json.issues : (Array.isArray(json.data?.issues) ? json.data.issues : []) };
    return audit;
  }, [supabase]);

  // Poll running audits every 3s
  useEffect(() => {
    if (!pollingId) return;
    const interval = setInterval(async () => {
      const updated = await fetchAudit(pollingId);
      if (!updated) return;
      setAudits(prev => prev.map(a => (a.id === pollingId ? updated : a)));
      if (selectedAudit?.id === pollingId) setSelectedAudit(updated);
      if (updated.status === "completed" || updated.status === "failed") {
        setPollingId(null);
      }
    }, 3000);
    return () => clearInterval(interval);
  }, [pollingId, fetchAudit, selectedAudit?.id]);

  const triggerAudit = async () => {
    setError("");
    setTriggering(true);
    const session = (await supabase.auth.getSession()).data.session;
    if (!session) return;

    const res = await fetch("/api/audits", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${session.access_token}`,
      },
      body: JSON.stringify({ project_id: project.id }),
    });

    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      setError(data.error ?? "Failed to trigger audit.");
      setTriggering(false);
      return;
    }

    // API returns { data: { id, ... }, message }
    const auditId = data.data?.id ?? data.audit_id ?? data.id;
    const newAudit: Audit = {
      id: auditId,
      project_id: project.id,
      status: "pending",
      created_at: new Date().toISOString(),
    };
    setAudits(prev => [newAudit, ...prev]);
    setSelectedAudit(newAudit);
    setPollingId(auditId);
    setTriggering(false);
  };

  const openAudit = async (audit: Audit) => {
    setSelectedAudit(audit);
    if (audit.status === "completed" && !audit.issues) {
      const full = await fetchAudit(audit.id);
      if (full) {
        setSelectedAudit(full);
        setAudits(prev => prev.map(a => (a.id === audit.id ? full : a)));
      }
    }
    if (audit.status === "running" || audit.status === "pending") {
      setPollingId(audit.id);
    }
  };

  return (
    <div className="space-y-6">
      {/* Trigger */}
      <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-6 flex items-center justify-between">
        <div>
          <h2 className="text-white font-semibold">Run SEO audit</h2>
          <p className="text-gray-400 text-sm mt-0.5">
            Crawls {project.domain} and checks for SEO issues
          </p>
        </div>
        <button
          onClick={triggerAudit}
          disabled={triggering}
          className="flex items-center gap-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-medium rounded-xl transition-colors"
        >
          <Play size={15} />
          {triggering ? "Starting..." : "Run audit"}
        </button>
      </div>

      {error && (
        <p className="text-red-400 text-sm bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3">
          {error}
        </p>
      )}

      {/* Audit history + detail side-by-side */}
      <div className="grid grid-cols-5 gap-4">
        {/* History list */}
        <div className="col-span-2 space-y-2">
          <h3 className="text-gray-400 text-xs font-medium uppercase tracking-wider mb-3">
            Audit history
          </h3>
          {audits.length === 0 ? (
            <p className="text-gray-500 text-sm">No audits yet.</p>
          ) : (
            audits.map(a => (
              <button
                key={a.id}
                onClick={() => openAudit(a)}
                className={`w-full text-left p-3.5 rounded-xl border transition-all ${
                  selectedAudit?.id === a.id
                    ? "bg-indigo-500/10 border-indigo-500/40"
                    : "bg-white/[0.03] border-white/10 hover:border-white/20"
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {statusIcon[a.status]}
                    <span className="text-white text-sm capitalize">
                      {a.status}
                    </span>
                  </div>
                  {a.score != null && (
                    <span
                      className={`text-sm font-bold ${
                        a.score >= 80
                          ? "text-emerald-400"
                          : a.score >= 50
                          ? "text-amber-400"
                          : "text-red-400"
                      }`}
                    >
                      {a.score}
                    </span>
                  )}
                </div>
                <p className="text-gray-500 text-xs mt-1">
                  {new Date(a.created_at).toLocaleString()}
                </p>
              </button>
            ))
          )}
        </div>

        {/* Detail panel */}
        <div className="col-span-3">
          {selectedAudit ? (
            <AuditDetail audit={selectedAudit} />
          ) : (
            <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-8 text-center h-full flex items-center justify-center">
              <p className="text-gray-500 text-sm">
                Select an audit to view results
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function AuditDetail({ audit }: { audit: Audit }) {
  if (audit.status === "pending" || audit.status === "running") {
    return (
      <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-8 text-center h-full flex flex-col items-center justify-center gap-3">
        <RefreshCw size={32} className="text-indigo-400 animate-spin" />
        <p className="text-white font-medium">Audit in progress…</p>
        <p className="text-gray-400 text-sm">
          {audit.pages_crawled
            ? `${audit.pages_crawled} pages crawled`
            : "Crawling started"}
        </p>
      </div>
    );
  }

  if (audit.status === "failed") {
    return (
      <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-8 text-center h-full flex flex-col items-center justify-center gap-3">
        <XCircle size={32} className="text-red-400" />
        <p className="text-white font-medium">Audit failed</p>
        <p className="text-gray-400 text-sm">
          The crawler could not complete this audit.
        </p>
      </div>
    );
  }

  const issues = Array.isArray(audit.issues) ? audit.issues : [];
  const critical = issues.filter(i => i.severity === "critical");
  const warnings = issues.filter(i => i.severity === "warning");
  const infos = issues.filter(i => i.severity === "info");

  return (
    <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-6 space-y-5">
      {/* Score */}
      <div className="flex items-center justify-between">
        <h3 className="text-white font-semibold">Results</h3>
        {audit.score != null && (
          <div className="flex items-center gap-2">
            <span className="text-gray-400 text-sm">Score</span>
            <span
              className={`text-2xl font-bold ${
                audit.score >= 80
                  ? "text-emerald-400"
                  : audit.score >= 50
                  ? "text-amber-400"
                  : "text-red-400"
              }`}
            >
              {audit.score}
            </span>
            <span className="text-gray-500 text-sm">/100</span>
          </div>
        )}
      </div>

      {/* Summary chips */}
      <div className="flex gap-3">
        <SeverityChip count={critical.length} severity="critical" />
        <SeverityChip count={warnings.length} severity="warning" />
        <SeverityChip count={infos.length} severity="info" />
      </div>

      {/* Issues */}
      {issues.length === 0 ? (
        <div className="text-center py-6">
          <CheckCircle size={28} className="text-emerald-400 mx-auto mb-2" />
          <p className="text-white font-medium">No issues found!</p>
        </div>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto pr-1">
          {[...critical, ...warnings, ...infos].map(issue => (
            <div
              key={issue.id}
              className={`flex gap-3 p-3 rounded-xl border text-sm ${severityColor[issue.severity]}`}
            >
              <span className="mt-0.5 shrink-0">{severityIcon[issue.severity]}</span>
              <div>
                <p className="font-medium">{issue.check_name}</p>
                <p className="opacity-80 text-xs mt-0.5">{issue.message}</p>
                {issue.url && (
                  <p className="text-xs opacity-60 mt-0.5 font-mono truncate">
                    {issue.url}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SeverityChip({
  count,
  severity,
}: {
  count: number;
  severity: "critical" | "warning" | "info";
}) {
  const colors = {
    critical: "text-red-400 bg-red-500/10 border-red-500/20",
    warning: "text-amber-400 bg-amber-500/10 border-amber-500/20",
    info: "text-blue-400 bg-blue-500/10 border-blue-500/20",
  };
  return (
    <div
      className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-sm font-medium ${colors[severity]}`}
    >
      {severityIcon[severity]}
      {count} {severity}
    </div>
  );
}
