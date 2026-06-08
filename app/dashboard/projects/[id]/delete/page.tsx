"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createClient } from "@/lib/supabase/client";
import { ArrowLeft, Trash2, AlertTriangle } from "lucide-react";

export default function DeleteProjectPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const router = useRouter();
  const supabase = createClient();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleDelete = async () => {
    setLoading(true);
    setError("");
    const session = (await supabase.auth.getSession()).data.session;
    if (!session) { router.push("/login"); return; }

    // Unwrap params in the handler
    const { id } = await params;

    const res = await fetch(`/api/projects/${id}`, {
      method: "DELETE",
      headers: { Authorization: `Bearer ${session.access_token}` },
    });

    if (!res.ok) {
      const d = await res.json().catch(() => ({}));
      setError(d.error ?? "Failed to delete project.");
      setLoading(false);
      return;
    }
    router.push("/dashboard/projects");
    router.refresh();
  };

  return (
    <div className="max-w-lg">
      <Link
        href={".."}
        className="inline-flex items-center gap-2 text-gray-400 hover:text-white text-sm mb-6 transition-colors"
      >
        <ArrowLeft size={15} />
        Back to project
      </Link>

      <div className="bg-red-500/5 border border-red-500/20 rounded-2xl p-8">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-xl bg-red-500/10 flex items-center justify-center">
            <AlertTriangle size={20} className="text-red-400" />
          </div>
          <h1 className="text-xl font-bold text-white">Delete project</h1>
        </div>

        <p className="text-gray-400 text-sm mb-6">
          This will permanently delete the project and all its audit history.
          This action cannot be undone.
        </p>

        {error && (
          <p className="text-red-400 text-sm bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2 mb-4">
            {error}
          </p>
        )}

        <div className="flex gap-3">
          <Link
            href={".."}
            className="flex-1 py-2.5 px-4 text-center bg-white/5 hover:bg-white/10 border border-white/10 text-white text-sm font-medium rounded-xl transition-colors"
          >
            Cancel
          </Link>
          <button
            onClick={handleDelete}
            disabled={loading}
            className="flex-1 flex items-center justify-center gap-2 py-2.5 px-4 bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white text-sm font-medium rounded-xl transition-colors"
          >
            <Trash2 size={15} />
            {loading ? "Deleting..." : "Delete project"}
          </button>
        </div>
      </div>
    </div>
  );
}
