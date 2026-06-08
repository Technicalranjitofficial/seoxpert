import { createClient } from "@/lib/supabase/server";
import Link from "next/link";
import { Plus, Globe } from "lucide-react";

interface Project {
  id: string;
  name: string;
  domain: string;
  created_at: string;
}

export default async function ProjectsPage() {
  const supabase = await createClient();
  const session = (await supabase.auth.getSession()).data.session;
  const apiUrl = process.env.API_URL ?? "http://localhost:8090";

  let projects: Project[] = [];
  if (session?.access_token) {
    const res = await fetch(`${apiUrl}/api/v1/projects`, {
      headers: { Authorization: `Bearer ${session.access_token}` },
      cache: "no-store",
    });
    if (res.ok) {
      const d = await res.json().catch(() => ({}));
      projects = d.projects ?? [];
    }
  }

  return (
    <div className="max-w-4xl">
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-2xl font-bold text-white">Projects</h1>
        <Link
          href="/dashboard/projects/new"
          className="flex items-center gap-2 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-xl transition-colors"
        >
          <Plus size={16} />
          New project
        </Link>
      </div>

      {projects.length === 0 ? (
        <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-16 text-center">
          <Globe size={48} className="text-gray-600 mx-auto mb-4" />
          <h2 className="text-white font-semibold text-lg mb-2">No projects yet</h2>
          <p className="text-gray-400 text-sm mb-6">
            Add your first website to start running SEO audits.
          </p>
          <Link
            href="/dashboard/projects/new"
            className="inline-flex items-center gap-2 px-5 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium rounded-xl transition-colors"
          >
            <Plus size={16} />
            Add your first project
          </Link>
        </div>
      ) : (
        <div className="grid gap-4">
          {projects.map((p) => (
            <Link
              key={p.id}
              href={`/dashboard/projects/${p.id}`}
              className="flex items-center justify-between p-5 bg-white/[0.03] border border-white/10 rounded-2xl hover:border-indigo-500/30 transition-all group"
            >
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center">
                  <Globe size={18} className="text-indigo-400" />
                </div>
                <div>
                  <p className="text-white font-medium group-hover:text-indigo-300 transition-colors">
                    {p.name}
                  </p>
                  <p className="text-gray-500 text-sm">{p.domain}</p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-gray-500 text-xs">
                  {new Date(p.created_at).toLocaleDateString()}
                </p>
                <p className="text-indigo-400 text-xs mt-1 group-hover:text-indigo-300">
                  View audits →
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
