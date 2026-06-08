-- Migration: 001_waitlist
-- Run this in Supabase SQL editor or via psql against supabase-db

CREATE TABLE IF NOT EXISTS public.waitlist (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT        NOT NULL,
    source      TEXT        NOT NULL DEFAULT 'landing_page',
    notified    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT waitlist_email_unique UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS idx_waitlist_created_at ON public.waitlist (created_at DESC);

-- RLS: only service_role can read/write waitlist (no public access)
ALTER TABLE public.waitlist ENABLE ROW LEVEL SECURITY;

CREATE POLICY "service_role_waitlist" ON public.waitlist
    USING (auth.role() = 'service_role');
