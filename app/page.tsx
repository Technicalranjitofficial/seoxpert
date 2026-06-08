import {
  Search, BarChart3, Link2, FileSearch, TrendingUp, Globe,
  Zap, Shield, Brain, Target, ArrowRight, CheckCircle2,
  ChevronRight, Star, Users, Award, Sparkles, Lock,
} from "lucide-react";
import Countdown from "./Countdown";
import EmailForm from "./EmailForm";

const features = [
  { icon: Brain, title: "AI Keyword Intelligence", desc: "Uncover high-intent keywords with AI-driven search intent clustering, difficulty scoring, and traffic potential forecasting.", tag: "Keyword Research", color: "bg-indigo-600" },
  { icon: FileSearch, title: "Technical SEO Auditor", desc: "Deep crawl for Core Web Vitals, broken links, schema errors, duplicate content, and 200+ technical SEO factors.", tag: "Site Audit", color: "bg-violet-600" },
  { icon: Link2, title: "Backlink Intelligence", desc: "Analyze your full backlink profile, discover link-building opportunities, and monitor toxic links in real time.", tag: "Link Analysis", color: "bg-cyan-600" },
  { icon: BarChart3, title: "Rank Tracker", desc: "Track keyword rankings daily across Google, Bing, and Yahoo for any location, device, or language worldwide.", tag: "Rank Tracking", color: "bg-emerald-600" },
  { icon: Target, title: "Competitor Intelligence", desc: "Reverse-engineer competitor strategies — their top keywords, backlinks, content gaps, and SERP positioning.", tag: "Competitive Analysis", color: "bg-orange-600" },
  { icon: Sparkles, title: "AI Content Optimizer", desc: "AI-generated content briefs, NLP keyword recommendations, and real-time on-page SEO scoring as you write.", tag: "Content SEO", color: "bg-pink-600" },
  { icon: Globe, title: "Local SEO Manager", desc: "Optimize for local search with GBP insights, citation tracking, review monitoring, and geo-targeted rank tracking.", tag: "Local SEO", color: "bg-teal-600" },
  { icon: TrendingUp, title: "Traffic Analytics", desc: "Unified organic traffic dashboard with GSC integration, CTR analysis, impression trends, and revenue attribution.", tag: "Analytics", color: "bg-blue-600" },
  { icon: Shield, title: "Algorithm Monitor", desc: "Instant alerts for Google algorithm updates and their impact on your rankings before your competitors notice.", tag: "Monitoring", color: "bg-red-600" },
];

const stats = [
  { label: "Keywords Tracked", value: "50M+" },
  { label: "Websites Analyzed", value: "2M+" },
  { label: "Backlinks Indexed", value: "10B+" },
  { label: "Avg. Traffic Increase", value: "312%" },
];

const upcoming = [
  "White-label SEO reports for agencies",
  "Automated SEO task recommendations",
  "Chrome extension for on-page analysis",
  "API access for developers",
  "Team collaboration & client portals",
  "CMS integrations (WordPress, Webflow, Shopify)",
];

export default function Home() {
  return (
    <main
      className="min-h-screen text-white overflow-x-hidden"
      style={{ background: "radial-gradient(ellipse 80% 50% at 15% 0%, rgba(99,102,241,0.12) 0%, transparent 55%), radial-gradient(ellipse 60% 40% at 85% 60%, rgba(139,92,246,0.08) 0%, transparent 50%), #050510" }}
    >
      {/* Header */}
      <header className="flex items-center justify-between px-6 md:px-12 py-6 border-b border-white/5">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center">
            <Search size={16} className="text-white" />
          </div>
          <span className="text-xl font-black tracking-tight">Seo<span className="gradient-text">Xpert</span></span>
        </div>
        <div className="flex items-center gap-3">
          <span className="hidden md:flex items-center gap-2 text-xs text-indigo-400 glass px-3 py-1.5 rounded-full">
            <Zap size={12} /> Launching Soon
          </span>
          <a
            href="/login"
            className="text-sm text-gray-300 hover:text-white px-4 py-2 rounded-lg hover:bg-white/5 transition-colors"
          >
            Sign In
          </a>
          <a
            href="/signup"
            className="text-sm font-semibold bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white px-4 py-2 rounded-lg transition-all"
          >
            Sign Up
          </a>
        </div>
      </header>

      {/* Hero */}
      <section className="text-center px-6 pt-20 pb-16 max-w-5xl mx-auto">
        <div className="inline-flex items-center gap-2 glass px-4 py-2 rounded-full text-xs text-indigo-300 mb-8 border border-indigo-500/20">
          <Sparkles size={12} className="text-indigo-400" />
          The future of SEO is AI-powered
          <ChevronRight size={12} />
        </div>
        <h1 className="text-5xl md:text-7xl font-black leading-[1.05] mb-6 tracking-tight">
          Dominate Search<br />
          <span className="gradient-text">With AI-Powered SEO</span>
        </h1>
        <p className="text-gray-400 text-lg md:text-xl max-w-2xl mx-auto leading-relaxed mb-10">
          SeoXpert combines cutting-edge AI with enterprise SEO tools to help agencies, startups, and businesses rank higher, attract more organic traffic, and outperform competitors — all from one platform.
        </p>
        <Countdown />
        <EmailForm />
        <p className="text-gray-600 text-xs mt-3 hidden sm:flex items-center justify-center gap-1.5">
          <Lock size={11} /> No spam. Early access perks for subscribers.
        </p>
      </section>

      {/* Stats */}
      <section className="max-w-5xl mx-auto px-6 pb-16">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {stats.map(s => (
            <div key={s.label} className="glass p-6 text-center">
              <div className="text-3xl font-black gradient-text mb-1">{s.value}</div>
              <div className="text-gray-500 text-sm">{s.label}</div>
            </div>
          ))}
        </div>
      </section>

      {/* Features */}
      <section className="max-w-6xl mx-auto px-6 pb-20">
        <div className="text-center mb-12">
          <span className="text-indigo-400 text-sm font-semibold uppercase tracking-widest block mb-3">Everything You Need</span>
          <h2 className="text-4xl md:text-5xl font-black text-white mb-4">
            One Platform, Every<br /><span className="gradient-text">SEO Superpower</span>
          </h2>
          <p className="text-gray-400 max-w-xl mx-auto">
            From keyword discovery to technical audits, backlinks to content optimization — SeoXpert has every tool you need to win at SEO.
          </p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {features.map(f => (
            <div key={f.title} className="glass p-6">
              <div className={`inline-flex p-3 rounded-xl mb-4 ${f.color}`}>
                <f.icon size={22} className="text-white" />
              </div>
              <span className="text-xs font-semibold text-indigo-400 uppercase tracking-widest mb-2 block">{f.tag}</span>
              <h3 className="text-white font-bold text-lg mb-2">{f.title}</h3>
              <p className="text-gray-400 text-sm leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Coming Up */}
      <section className="max-w-4xl mx-auto px-6 pb-20">
        <div className="glass p-8 md:p-12 border border-indigo-500/10">
          <div className="flex items-center gap-3 mb-8">
            <div className="w-10 h-10 rounded-xl bg-indigo-600/20 flex items-center justify-center">
              <Zap size={20} className="text-indigo-400" />
            </div>
            <div>
              <h2 className="text-2xl font-black text-white">What&apos;s Coming</h2>
              <p className="text-gray-500 text-sm">Planned for v1.0 and beyond</p>
            </div>
          </div>
          <div className="grid sm:grid-cols-2 gap-3">
            {upcoming.map(item => (
              <div key={item} className="flex items-center gap-3">
                <CheckCircle2 size={16} className="text-indigo-400 shrink-0" />
                <span className="text-gray-300 text-sm">{item}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Why SeoXpert */}
      <section className="max-w-5xl mx-auto px-6 pb-20">
        <div className="text-center mb-10">
          <h2 className="text-4xl font-black text-white">Why <span className="gradient-text">SeoXpert</span>?</h2>
        </div>
        <div className="grid md:grid-cols-3 gap-6">
          {[
            { icon: Brain, title: "AI-First Approach", desc: "Every feature is built with AI at the core. Get smarter recommendations that actually move rankings." },
            { icon: Users, title: "Built for Agencies", desc: "Multi-client dashboards, white-label reports, team permissions, and audit templates for SEO agencies." },
            { icon: Award, title: "Enterprise-Grade Data", desc: "50M+ keywords, 10B+ backlinks, and real-time SERP data updated daily — the freshest data in the industry." },
          ].map(item => (
            <div key={item.title} className="glass p-8 text-center">
              <div className="w-12 h-12 rounded-xl bg-indigo-600/20 flex items-center justify-center mx-auto mb-5">
                <item.icon size={22} className="text-indigo-400" />
              </div>
              <h3 className="text-white font-bold text-lg mb-2">{item.title}</h3>
              <p className="text-gray-400 text-sm leading-relaxed">{item.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Testimonial */}
      <section className="max-w-3xl mx-auto px-6 pb-20 text-center">
        <div className="glass p-10 border border-indigo-500/10">
          <div className="flex justify-center mb-4">
            {[...Array(5)].map((_, i) => <Star key={i} size={18} className="text-yellow-400 fill-yellow-400" />)}
          </div>
          <blockquote className="text-white text-xl font-medium leading-relaxed mb-6">
            &ldquo;SeoXpert is exactly what the SEO industry has been waiting for — an AI-native platform that doesn&apos;t just give you data, it tells you what to do with it.&rdquo;
          </blockquote>
          <div className="flex items-center justify-center gap-3">
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center text-sm font-bold">R</div>
            <div className="text-left">
              <p className="text-white text-sm font-semibold">Ranjit Kumar</p>
              <p className="text-gray-500 text-xs">Founder, SeoXpert</p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="max-w-3xl mx-auto px-6 pb-24 text-center">
        <h2 className="text-4xl md:text-5xl font-black text-white mb-4">Ready to <span className="gradient-text">Rank #1</span>?</h2>
        <p className="text-gray-400 mb-8 text-lg">Join thousands of SEO professionals on the early access list. Get lifetime discounts and priority access at launch.</p>
        <a href="#" className="inline-flex items-center gap-2 bg-gradient-to-r from-indigo-600 to-violet-600 text-white font-bold px-8 py-4 rounded-xl text-lg">
          Get Early Access <ArrowRight size={20} />
        </a>
      </section>

      {/* Footer */}
      <footer className="border-t border-white/5 px-6 py-8 text-center text-gray-600 text-sm">
        <p>© {new Date().getFullYear()} SeoXpert. All rights reserved.</p>
        <p className="mt-1 text-xs">SEO platform · Keyword research · Technical audit · Rank tracking · Backlink analysis · AI SEO tools</p>
      </footer>
    </main>
  );
}
