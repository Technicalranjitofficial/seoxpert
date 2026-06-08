"use client";
import { useEffect } from "react";
import { AlertTriangle, RefreshCw, ChevronDown, ChevronUp } from "lucide-react";
import { useState } from "react";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    console.error("[Dashboard Error]", error);
  }, [error]);

  return (
    <div className="flex items-start justify-center min-h-[60vh] p-8">
      <div className="w-full max-w-2xl">
        <div className="bg-red-500/10 border border-red-500/30 rounded-2xl p-6">
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-xl bg-red-500/20 flex items-center justify-center shrink-0 mt-0.5">
              <AlertTriangle size={20} className="text-red-400" />
            </div>
            <div className="flex-1 min-w-0">
              <h2 className="text-white font-semibold text-lg mb-1">Something went wrong</h2>
              <p className="text-red-300 text-sm font-mono break-all">{error.message}</p>
              {error.digest && (
                <p className="text-gray-500 text-xs mt-1">Digest: {error.digest}</p>
              )}

              <button
                onClick={() => setExpanded(v => !v)}
                className="flex items-center gap-1 text-gray-400 hover:text-white text-xs mt-3 transition-colors"
              >
                {expanded ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
                {expanded ? "Hide" : "Show"} stack trace
              </button>

              {expanded && error.stack && (
                <pre className="mt-3 p-4 bg-black/40 rounded-xl text-red-300 text-xs overflow-x-auto whitespace-pre-wrap break-all">
                  {error.stack}
                </pre>
              )}

              <button
                onClick={reset}
                className="mt-4 flex items-center gap-2 px-4 py-2 bg-red-500/20 hover:bg-red-500/30 border border-red-500/30 text-red-300 text-sm rounded-xl transition-colors"
              >
                <RefreshCw size={14} />
                Try again
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
