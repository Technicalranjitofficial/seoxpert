"use client";
import { useState, useEffect, useCallback, useRef } from "react";
import { createClient } from "@/lib/supabase/client";
import { Bug, X, ChevronDown, ChevronRight, RefreshCw, Copy, Check } from "lucide-react";

interface ApiLog {
  id: number;
  ts: string;
  method: string;
  url: string;
  status?: number;
  ok?: boolean;
  durationMs?: number;
  response?: unknown;
  error?: string;
}

let logCounter = 0;
const listeners = new Set<(log: ApiLog) => void>();

// Monkey-patch fetch to capture all API calls automatically
if (typeof window !== "undefined" && !(window as unknown as Record<string, unknown>).__debugPatched) {
  (window as unknown as Record<string, unknown>).__debugPatched = true;
  const origFetch = window.fetch.bind(window);
  window.fetch = async (input, init) => {
    const url = typeof input === "string" ? input : (input as Request).url;
    const method = (init?.method ?? (typeof input !== "string" ? (input as Request).method : "GET")).toUpperCase();
    const id = ++logCounter;
    const ts = new Date().toISOString().slice(11, 23);
    const t0 = performance.now();

    try {
      const res = await origFetch(input, init);
      const durationMs = Math.round(performance.now() - t0);
      let response: unknown;
      try {
        const clone = res.clone();
        const ct = res.headers.get("content-type") ?? "";
        response = ct.includes("json") ? await clone.json() : await clone.text();
      } catch { /* ignore */ }

      const log: ApiLog = { id, ts, method, url, status: res.status, ok: res.ok, durationMs, response };
      listeners.forEach(fn => fn(log));
      return res;
    } catch (err) {
      const log: ApiLog = { id, ts, method, url, error: String(err) };
      listeners.forEach(fn => fn(log));
      throw err;
    }
  };
}

export default function DebugPanel() {
  const [open, setOpen] = useState(false);
  const [logs, setLogs] = useState<ApiLog[]>([]);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [tab, setTab] = useState<"logs" | "auth">("logs");
  const [session, setSession] = useState<Record<string, unknown> | null>(null);
  const [copied, setCopied] = useState<number | null>(null);
  const supabase = createClient();
  const bottomRef = useRef<HTMLDivElement>(null);

  // Keyboard shortcut: Shift+D to toggle
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.shiftKey && e.key === "D") setOpen(v => !v);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  // Subscribe to fetch logs
  useEffect(() => {
    const fn = (log: ApiLog) => setLogs(prev => [log, ...prev].slice(0, 50));
    listeners.add(fn);
    return () => { listeners.delete(fn); };
  }, []);

  const loadSession = useCallback(async () => {
    const { data } = await supabase.auth.getSession();
    const s = data.session;
    if (!s) { setSession(null); return; }
    setSession({
      user_id: s.user.id,
      email: s.user.email,
      role: s.user.role,
      expires_at: new Date(s.expires_at! * 1000).toISOString(),
      access_token_preview: s.access_token.slice(0, 40) + "...",
    });
  }, [supabase]);

  useEffect(() => { if (tab === "auth") loadSession(); }, [tab, loadSession]);

  const toggleExpand = (id: number) =>
    setExpanded(prev => {
      const n = new Set(prev);
      n.has(id) ? n.delete(id) : n.add(id);
      return n;
    });

  const copyToClipboard = (id: number, text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 1500);
  };

  const statusColor = (status?: number) => {
    if (!status) return "text-gray-500";
    if (status < 300) return "text-emerald-400";
    if (status < 400) return "text-amber-400";
    return "text-red-400";
  };

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        title="Debug Panel (Shift+D)"
        className="fixed bottom-4 right-4 z-50 w-10 h-10 bg-indigo-600 hover:bg-indigo-500 rounded-full flex items-center justify-center shadow-lg transition-colors"
      >
        <Bug size={18} className="text-white" />
      </button>
    );
  }

  return (
    <div className="fixed bottom-0 right-0 z-50 w-full max-w-2xl h-[420px] bg-[#0d0d18] border-t border-l border-white/10 flex flex-col shadow-2xl">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-white/10 bg-[#0a0a14]">
        <div className="flex items-center gap-3">
          <Bug size={15} className="text-indigo-400" />
          <span className="text-white text-sm font-semibold">Debug Panel</span>
          <span className="text-gray-500 text-xs">(Shift+D to toggle)</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setTab("logs")}
            className={`text-xs px-3 py-1 rounded-lg transition-colors ${tab === "logs" ? "bg-indigo-600 text-white" : "text-gray-400 hover:text-white"}`}
          >
            API Logs ({logs.length})
          </button>
          <button
            onClick={() => setTab("auth")}
            className={`text-xs px-3 py-1 rounded-lg transition-colors ${tab === "auth" ? "bg-indigo-600 text-white" : "text-gray-400 hover:text-white"}`}
          >
            Auth
          </button>
          <button
            onClick={() => setLogs([])}
            className="text-xs text-gray-500 hover:text-red-400 px-2 py-1 rounded transition-colors"
          >
            Clear
          </button>
          <button
            onClick={() => setOpen(false)}
            className="text-gray-400 hover:text-white transition-colors ml-1"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto font-mono text-xs">
        {tab === "logs" && (
          <div>
            {logs.length === 0 && (
              <p className="text-gray-500 text-center py-8">No API calls yet. Interact with the app to see logs.</p>
            )}
            {logs.map(log => (
              <div
                key={log.id}
                className="border-b border-white/5 hover:bg-white/[0.02] transition-colors"
              >
                <div
                  className="flex items-center gap-2 px-3 py-2 cursor-pointer"
                  onClick={() => toggleExpand(log.id)}
                >
                  {expanded.has(log.id) ? <ChevronDown size={11} className="text-gray-500 shrink-0" /> : <ChevronRight size={11} className="text-gray-500 shrink-0" />}
                  <span className="text-gray-500">{log.ts}</span>
                  <span className={`font-bold w-12 shrink-0 ${log.method === "GET" ? "text-blue-400" : log.method === "POST" ? "text-emerald-400" : log.method === "DELETE" ? "text-red-400" : "text-amber-400"}`}>
                    {log.method}
                  </span>
                  <span className="text-gray-300 truncate flex-1">{log.url}</span>
                  {log.status && (
                    <span className={`shrink-0 font-bold ${statusColor(log.status)}`}>{log.status}</span>
                  )}
                  {log.durationMs !== undefined && (
                    <span className="text-gray-500 shrink-0">{log.durationMs}ms</span>
                  )}
                  {log.error && <span className="text-red-400 shrink-0">ERR</span>}
                </div>
                {expanded.has(log.id) && (
                  <div className="px-8 pb-3">
                    {log.error && (
                      <div className="text-red-300 bg-red-500/10 rounded p-2 mb-2">{log.error}</div>
                    )}
                    {log.response !== undefined && (
                      <div className="relative">
                        <button
                          onClick={() => copyToClipboard(log.id, JSON.stringify(log.response, null, 2))}
                          className="absolute top-2 right-2 text-gray-500 hover:text-white transition-colors"
                        >
                          {copied === log.id ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
                        </button>
                        <pre className="text-gray-300 bg-black/40 rounded p-3 overflow-x-auto whitespace-pre-wrap break-all max-h-48">
                          {JSON.stringify(log.response, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}

        {tab === "auth" && (
          <div className="p-4">
            <div className="flex items-center justify-between mb-3">
              <span className="text-gray-400">Session State</span>
              <button onClick={loadSession} className="flex items-center gap-1 text-gray-400 hover:text-white transition-colors">
                <RefreshCw size={12} /> Refresh
              </button>
            </div>
            {session ? (
              <div className="space-y-2">
                {Object.entries(session).map(([k, v]) => (
                  <div key={k} className="flex gap-3 items-start">
                    <span className="text-indigo-400 w-36 shrink-0">{k}</span>
                    <span className="text-gray-300 break-all">{String(v)}</span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-red-400">No active session — not logged in</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
