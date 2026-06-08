-- Migration: 003_audits

CREATE TABLE IF NOT EXISTS public.audits (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID        NOT NULL REFERENCES public.projects(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    status        TEXT        NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending','running','completed','failed')),
    total_pages   INT         NOT NULL DEFAULT 0,
    crawled_pages INT         NOT NULL DEFAULT 0,
    score         INT         NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 100),
    issues        INT         NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_audits_project_id ON public.audits (project_id);
CREATE INDEX IF NOT EXISTS idx_audits_user_id    ON public.audits (user_id);
CREATE INDEX IF NOT EXISTS idx_audits_status     ON public.audits (status);
CREATE INDEX IF NOT EXISTS idx_audits_created_at ON public.audits (created_at DESC);

ALTER TABLE public.audits ENABLE ROW LEVEL SECURITY;

CREATE POLICY "users_own_audits" ON public.audits
    FOR ALL USING (auth.uid() = user_id);

-- ── Audit Issues ──────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS public.audit_issues (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_id    UUID    NOT NULL REFERENCES public.audits(id) ON DELETE CASCADE,
    url         TEXT    NOT NULL,
    check_type  TEXT    NOT NULL,
    severity    TEXT    NOT NULL CHECK (severity IN ('critical','warning','info')),
    title       TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    suggestion  TEXT    NOT NULL DEFAULT ''
);

-- Partitioned by audit for fast per-audit queries
CREATE INDEX IF NOT EXISTS idx_audit_issues_audit_id  ON public.audit_issues (audit_id);
CREATE INDEX IF NOT EXISTS idx_audit_issues_severity  ON public.audit_issues (audit_id, severity);
CREATE INDEX IF NOT EXISTS idx_audit_issues_check_type ON public.audit_issues (check_type);

ALTER TABLE public.audit_issues ENABLE ROW LEVEL SECURITY;

-- Issues are readable through their parent audit's user ownership check.
CREATE POLICY "users_own_audit_issues" ON public.audit_issues
    FOR ALL USING (
        EXISTS (
            SELECT 1 FROM public.audits
            WHERE audits.id = audit_issues.audit_id
            AND audits.user_id = auth.uid()
        )
    );

-- Enable Supabase Realtime on audits so the dashboard gets live progress updates.
ALTER PUBLICATION supabase_realtime ADD TABLE public.audits;
