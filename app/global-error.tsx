"use client";
import { AlertTriangle, RefreshCw } from "lucide-react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html>
      <body className="min-h-screen bg-[#0a0a0f] flex items-center justify-center p-8">
        <div className="w-full max-w-lg bg-red-500/10 border border-red-500/30 rounded-2xl p-8 text-center">
          <div className="w-14 h-14 rounded-2xl bg-red-500/20 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle size={28} className="text-red-400" />
          </div>
          <h1 className="text-white font-bold text-xl mb-2">Application Error</h1>
          <p className="text-red-300 font-mono text-sm break-all mb-2">{error.message}</p>
          {error.digest && (
            <p className="text-gray-500 text-xs mb-4">Digest: {error.digest}</p>
          )}
          {error.stack && (
            <pre className="text-left mb-4 p-4 bg-black/40 rounded-xl text-red-300 text-xs overflow-x-auto whitespace-pre-wrap break-all">
              {error.stack}
            </pre>
          )}
          <button
            onClick={reset}
            className="inline-flex items-center gap-2 px-5 py-2.5 bg-red-500/20 hover:bg-red-500/30 border border-red-500/30 text-red-300 rounded-xl transition-colors"
          >
            <RefreshCw size={15} />
            Reload
          </button>
        </div>
      </body>
    </html>
  );
}
