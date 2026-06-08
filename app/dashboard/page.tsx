import { createClient } from "@/lib/supabase/server";
import Link from "next/link";
import { Plus, Globe, AlertTriangle, TrendingUp } from "lucide-react";

export default async function DashboardPage() {
  const supabase = await createClient();
  const {
    data: { user },
  } = await supabase.auth.getUser();

  // Fetch projects and recent audits for this user
  const apiUrl = process.env.API_URL ?? "http://localhost:8090";
  const accessToken = (await supabase.auth.getSession()).data.session
    ?.access_token;

  let projects: Project[] = [];
  let recentAudits: Audit[] = [];

  if (accessToken) {
    const [projRes, auditRes] = await Promise.all([
      fetch(`${apiUrl}/api/v1/projects`, {
        headers: { Authorization: `Bearer ${accessToken}` },
        cache: "no-store",
      }),
      fetch(`${apiUrl}/api/v1/audits?limit=5`, {
        headers: { Authorization: `Bearer ${accessToken}` },
        cache: "no-store",
      }).catch(() => null),
    ]);

    if (projRes.ok) {
      const d = await projRes.json().catch(() => ({}));
      projects = d.data ?? d.projects ?? [];
    }
    if (auditRes?.ok) {
      const d = await auditRes.json().catch(() => ({}));
      recentAudits = d.data ?? d.audits ?? [];
    }
  }

  return (
    <div className="max-w-6xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Overview</h1>
          <p className="text-gray-400 text-sm mt-1">
            Welcome back, {user?.email}
          </p>
        </div>
        <Link
          href="/dashboard/projects/new"
          className="flex items-center gap-2 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-xl transition-colors"
        >
          <Plus size={16} />
          New project
        </Link>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        <StatCard
          icon={<Globe size={20} className="text-indigo-400" />}
          label="Projects"
          value={projects.length}
        />
        <StatCard
          icon={<AlertTriangle size={20} className="text-amber-400" />}
          label="Audits run"
          value={recentAudits.length}
        />
        <StatCard
          icon={<TrendingUp size={20} className="text-emerald-400" />}
          label="Avg score"
          value={
            recentAudits.length
              ? Math.round(
                  recentAudits.reduce((s: number, a: Audit) => s + (a.score ?? 0), 0) /
                    recentAudits.length
                ) + "%"
              : "—"
          }
        />
      </div>

      {/* Projects list */}
      <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-6">
        <h2 className="text-white font-semibold mb-4">Your projects</h2>
        {projects.length === 0 ? (
          <div className="text-center py-12">
            <Globe size={40} className="text-gray-600 mx-auto mb-3" />
            <p className="text-gray-400">No projects yet.</p>
            <Link
              href="/dashboard/projects/new"
              className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-indigo-600/20 border border-indigo-500/30 text-indigo-300 text-sm rounded-xl hover:bg-indigo-600/30 transition-colors"
            >
              <Plus size={15} />
              Create your first project
            </Link>
          </div>
        ) : (
          <div className="space-y-3">
            {projects.map((p: Project) => (
              <Link
                key={p.id}
                href={`/dashboard/projects/${p.id}`}
                className="flex items-center justify-between p-4 bg-white/[0.03] border border-white/10 rounded-xl hover:border-indigo-500/30 transition-colors group"
              >
                <div>
                  <p className="text-white font-medium group-hover:text-indigo-300 transition-colors">
                    {p.name}
                  </p>
                  <p className="text-gray-500 text-sm">{p.domain}</p>
                </div>
                <span className="text-gray-500 text-xs">
                  {new Date(p.created_at).toLocaleDateString()}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string | number;
}) {
  return (
    <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-5">
      <div className="flex items-center gap-3 mb-3">
        <div className="w-9 h-9 rounded-lg bg-white/5 flex items-center justify-center">
          {icon}
        </div>
        <span className="text-gray-400 text-sm">{label}</span>
      </div>
      <p className="text-3xl font-bold text-white">{value}</p>
    </div>
  );
}

interface Project {
  id: string;
  name: string;
  domain: string;
  created_at: string;
}

interface Audit {
  id: string;
  score?: number;
}
