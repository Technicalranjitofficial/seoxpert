import { createClient } from "@/lib/supabase/server";

export default async function SettingsPage() {
  const supabase = await createClient();
  const {
    data: { user },
  } = await supabase.auth.getUser();

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold text-white mb-8">Settings</h1>

      <div className="bg-white/[0.03] border border-white/10 rounded-2xl p-6 space-y-4">
        <h2 className="text-white font-semibold">Account</h2>
        <div>
          <label className="block text-xs font-medium text-gray-400 mb-1">Email</label>
          <p className="text-white bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm">
            {user?.email}
          </p>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-400 mb-1">Plan</label>
          <span className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-indigo-500/10 border border-indigo-500/20 text-indigo-300 text-sm rounded-lg">
            ✦ Free plan
          </span>
        </div>
      </div>
    </div>
  );
}
