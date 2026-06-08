import { createClient } from "@/lib/supabase/server";
import { notFound } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Trash2 } from "lucide-react";
import AuditSection from "@/components/AuditSection";

interface Project {
  id: string;
  name: string;
  domain: string;
  created_at: string;
}

export default async function ProjectDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const supabase = await createClient();
  const session = (await supabase.auth.getSession()).data.session;
  const apiUrl = process.env.API_URL ?? "http://localhost:8090";

  if (!session?.access_token) notFound();

  const [projRes, auditsRes] = await Promise.all([
    fetch(`${apiUrl}/api/v1/projects/${id}`, {
      headers: { Authorization: `Bearer ${session.access_token}` },
      cache: "no-store",
    }),
    fetch(`${apiUrl}/api/v1/audits?project_id=${id}&limit=20`, {
      headers: { Authorization: `Bearer ${session.access_token}` },
      cache: "no-store",
    }).catch(() => null),
  ]);

  if (!projRes.ok) notFound();

  const project: Project = await projRes.json();
  const auditsData = auditsRes?.ok ? await auditsRes.json().catch(() => ({})) : {};
  const audits = auditsData.audits ?? [];

  return (
    <div className="max-w-4xl">
      <div className="flex items-center justify-between mb-8">
        <div>
          <Link
            href="/dashboard/projects"
            className="inline-flex items-center gap-2 text-gray-400 hover:text-white text-sm mb-3 transition-colors"
          >
            <ArrowLeft size={15} />
            Projects
          </Link>
          <h1 className="text-2xl font-bold text-white">{project.name}</h1>
          <p className="text-gray-400 text-sm mt-1">{project.domain}</p>
        </div>
        <Link
          href={`/dashboard/projects/${id}/delete`}
          className="flex items-center gap-2 px-3 py-2 text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-xl text-sm transition-all"
        >
          <Trash2 size={15} />
          Delete
        </Link>
      </div>

      {/* Audit trigger + results — client component for interactivity */}
      <AuditSection
        project={project}
        initialAudits={audits}
      />
    </div>
  );
}
