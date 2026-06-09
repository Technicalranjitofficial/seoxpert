import { Metadata } from "next";
import Link from "next/link";
import { Search, Target, Zap, Shield, Users, BarChart3, Globe, Brain, ArrowRight } from "lucide-react";

export const metadata: Metadata = {
  title: "About SeoXpert — AI-Powered SEO Platform",
  description: "Learn about SeoXpert — our mission to make enterprise-grade SEO accessible to every business and agency.",
};

const values = [
  {
    icon: Brain,
    title: "AI-First Approach",
    desc: "We build every feature with AI at the core — from keyword intelligence to content optimization — so you always stay ahead of the curve.",
  },
  {
    icon: Target,
    title: "Results That Matter",
    desc: "We obsess over actionable insights, not vanity metrics. Every data point we surface is one that can directly improve your rankings.",
  },
  {
    icon: Shield,
    title: "Transparent & Honest",
    desc: "No black-box scores or made-up metrics. We show you exactly what we checked, what we found, and how to fix it.",
  },
  {
    icon: Zap,
    title: "Speed & Reliability",
    desc: "Our crawler and rank-tracker are built on distributed infrastructure to deliver fast, consistent results at any scale.",
  },
  {
    icon: Users,
    title: "Built for Agencies",
    desc: "Multi-project dashboards, white-label reports, and team collaboration — everything an SEO agency needs in one platform.",
  },
  {
    icon: Globe,
    title: "Global Coverage",
    desc: "Track rankings across 100+ countries, analyse SERP features in any locale, and audit websites in any language.",
  },
];

const milestones = [
  { year: "2024", title: "Concept & Research", desc: "Identified the gap: enterprise SEO tools are too expensive; cheap tools are too shallow. SeoXpert was born." },
  { year: "2025", title: "Private Beta", desc: "Launched technical SEO auditor and rank tracker to a closed group of early-adopter agencies." },
  { year: "2026", title: "Public Launch", desc: "Full platform launch including AI keyword intelligence, domain overview, and content optimizer." },
];

export default function About() {
  return (
    <div className="min-h-screen bg-[#050510] text-white">
      <header className="flex items-center justify-between px-6 md:px-12 py-6 border-b border-white/5">
        <Link href="/" className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center">
            <Search size={16} className="text-white" />
          </div>
          <span className="text-xl font-black tracking-tight">Seo<span className="text-indigo-400">Xpert</span></span>
        </Link>
        <nav className="hidden md:flex items-center gap-6 text-sm text-gray-400">
          <Link href="/blog" className="hover:text-white transition-colors">Blog</Link>
          <Link href="/tools" className="hover:text-white transition-colors">Free Tools</Link>
          <Link href="/contact" className="hover:text-white transition-colors">Contact</Link>
        </nav>
        <div className="flex items-center gap-3">
          <Link href="/login" className="text-sm text-gray-300 hover:text-white px-4 py-2 rounded-lg hover:bg-white/5 transition-colors">Sign In</Link>
          <Link href="/signup" className="text-sm font-semibold bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white px-4 py-2 rounded-lg transition-all">Sign Up Free</Link>
        </div>
      </header>

      <main>
        {/* Hero */}
        <section className="text-center px-6 pt-20 pb-16 max-w-4xl mx-auto">
          <div className="inline-flex items-center gap-2 border border-indigo-500/30 bg-indigo-500/10 px-4 py-2 rounded-full text-xs text-indigo-300 mb-8">
            <BarChart3 size={12} />
            Our Story
          </div>
          <h1 className="text-5xl md:text-6xl font-black leading-tight mb-6">
            We&apos;re on a mission to<br />
            <span className="bg-gradient-to-r from-indigo-400 to-violet-400 bg-clip-text text-transparent">
              democratize SEO
            </span>
          </h1>
          <p className="text-gray-400 text-lg max-w-2xl mx-auto">
            Enterprise SEO tools cost thousands of dollars per month. Basic tools give you shallow, misleading data. SeoXpert bridges that gap — giving every business and agency access to the depth and intelligence they need to rank higher.
          </p>
        </section>

        {/* Mission statement */}
        <section className="max-w-5xl mx-auto px-6 mb-20">
          <div
            className="rounded-2xl p-8 md:p-12 border border-indigo-500/20"
            style={{ background: "radial-gradient(ellipse 80% 60% at 50% 0%, rgba(99,102,241,0.12) 0%, transparent 70%)" }}
          >
            <h2 className="text-2xl font-bold text-center mb-4">What We Believe</h2>
            <p className="text-gray-300 text-center text-lg max-w-2xl mx-auto leading-relaxed">
              Great SEO should not require a €10,000/month Semrush Enterprise contract. Every indie founder, small business owner, and growing agency deserves access to accurate technical audits, real keyword data, and AI-powered insights. That is why we built SeoXpert.
            </p>
          </div>
        </section>

        {/* Values */}
        <section className="max-w-6xl mx-auto px-6 mb-20">
          <h2 className="text-3xl font-black text-center mb-12">Our Values</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {values.map(({ icon: Icon, title, desc }) => (
              <div key={title} className="bg-white/3 border border-white/8 rounded-xl p-6 hover:border-indigo-500/30 transition-colors">
                <div className="w-10 h-10 rounded-lg bg-indigo-500/20 flex items-center justify-center mb-4">
                  <Icon size={20} className="text-indigo-400" />
                </div>
                <h3 className="font-bold text-white mb-2">{title}</h3>
                <p className="text-gray-400 text-sm leading-relaxed">{desc}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Timeline */}
        <section className="max-w-3xl mx-auto px-6 mb-20">
          <h2 className="text-3xl font-black text-center mb-12">Our Journey</h2>
          <div className="space-y-6">
            {milestones.map(({ year, title, desc }) => (
              <div key={year} className="flex gap-6">
                <div className="flex flex-col items-center">
                  <div className="w-12 h-12 rounded-xl bg-indigo-600/30 border border-indigo-500/40 flex items-center justify-center text-indigo-300 font-bold text-sm shrink-0">{year}</div>
                  <div className="w-px flex-1 bg-white/8 mt-2" />
                </div>
                <div className="pb-6">
                  <h3 className="font-bold text-white mb-1">{title}</h3>
                  <p className="text-gray-400 text-sm">{desc}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* CTA */}
        <section className="text-center px-6 pb-20">
          <h2 className="text-3xl font-black mb-4">Ready to grow your organic traffic?</h2>
          <p className="text-gray-400 mb-8">Join agencies and businesses already using SeoXpert to dominate search.</p>
          <div className="flex justify-center gap-4 flex-wrap">
            <Link
              href="/signup"
              className="inline-flex items-center gap-2 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white font-semibold px-6 py-3 rounded-xl transition-all"
            >
              Start for Free <ArrowRight size={16} />
            </Link>
            <Link
              href="/contact"
              className="inline-flex items-center gap-2 border border-white/15 hover:border-white/30 text-gray-300 hover:text-white font-semibold px-6 py-3 rounded-xl transition-all"
            >
              Contact Us
            </Link>
          </div>
        </section>
      </main>

      <footer className="border-t border-white/5 py-8 text-center text-gray-600 text-sm">
        <p>© {new Date().getFullYear()} SeoXpert. All rights reserved.</p>
        <div className="flex justify-center gap-6 mt-3">
          <Link href="/privacy-policy" className="hover:text-gray-400 transition-colors">Privacy Policy</Link>
          <Link href="/terms" className="hover:text-gray-400 transition-colors">Terms of Service</Link>
          <Link href="/about" className="hover:text-gray-400 transition-colors">About</Link>
          <Link href="/contact" className="hover:text-gray-400 transition-colors">Contact</Link>
        </div>
      </footer>
    </div>
  );
}
