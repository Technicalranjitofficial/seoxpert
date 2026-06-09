"use client";
import { useState } from "react";
import Link from "next/link";
import { Search, Mail, MessageSquare, Clock, CheckCircle2 } from "lucide-react";

const faqs = [
  { q: "How do I cancel my subscription?", a: "You can cancel anytime from your account settings. Your access continues until the end of the billing period." },
  { q: "Can I get a refund?", a: "We offer a 7-day money-back guarantee for first-time subscribers. Contact us within 7 days of your first payment." },
  { q: "Do you offer agency or team plans?", a: "Yes! Our Agency and Enterprise plans support multiple users and projects. Check the pricing page for details." },
  { q: "Is there a free trial?", a: "We offer a free tier with limited crawls and keyword tracking. No credit card required to start." },
  { q: "Can I add multiple websites?", a: "Yes. Depending on your plan you can add up to 5, 25, or unlimited projects." },
  { q: "What data sources do you use for domain analysis?", a: "We use RDAP (domain registry), the Wayback Machine CDX API, and live HTTP checks. No third-party paid APIs required." },
];

export default function Contact() {
  const [form, setForm] = useState({ name: "", email: "", subject: "", message: "" });
  const [sent, setSent] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSending(true);
    setError("");
    try {
      const res = await fetch("/api/contact", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      if (!res.ok) throw new Error("Failed to send");
      setSent(true);
    } catch {
      setError("Something went wrong. Please email us directly at support@seoexpert.in");
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="min-h-screen bg-[#050510] text-white">
      <header className="flex items-center justify-between px-6 md:px-12 py-6 border-b border-white/5">
        <Link href="/" className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center">
            <Search size={16} className="text-white" />
          </div>
          <span className="text-xl font-black tracking-tight">Seo<span className="text-indigo-400">Xpert</span></span>
        </Link>
        <Link href="/" className="text-sm text-gray-400 hover:text-white transition-colors">← Back to Home</Link>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-16">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-black mb-3">Get in Touch</h1>
          <p className="text-gray-400 text-lg">Have a question, feedback, or a partnership idea? We&apos;d love to hear from you.</p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-5 gap-10">
          {/* Contact info + FAQ */}
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-white/3 border border-white/8 rounded-xl p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-9 h-9 rounded-lg bg-indigo-500/20 flex items-center justify-center">
                  <Mail size={16} className="text-indigo-400" />
                </div>
                <div>
                  <p className="text-xs text-gray-500 uppercase tracking-wider">Email</p>
                  <a href="mailto:support@seoexpert.in" className="text-white text-sm hover:text-indigo-400 transition-colors">support@seoexpert.in</a>
                </div>
              </div>
            </div>

            <div className="bg-white/3 border border-white/8 rounded-xl p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-9 h-9 rounded-lg bg-violet-500/20 flex items-center justify-center">
                  <Clock size={16} className="text-violet-400" />
                </div>
                <div>
                  <p className="text-xs text-gray-500 uppercase tracking-wider">Response Time</p>
                  <p className="text-white text-sm">Within 24–48 hours</p>
                </div>
              </div>
            </div>

            <div className="bg-white/3 border border-white/8 rounded-xl p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-9 h-9 rounded-lg bg-cyan-500/20 flex items-center justify-center">
                  <MessageSquare size={16} className="text-cyan-400" />
                </div>
                <div>
                  <p className="text-xs text-gray-500 uppercase tracking-wider">For Billing & Account</p>
                  <a href="mailto:billing@seoexpert.in" className="text-white text-sm hover:text-indigo-400 transition-colors">billing@seoexpert.in</a>
                </div>
              </div>
            </div>
          </div>

          {/* Form */}
          <div className="lg:col-span-3">
            {sent ? (
              <div className="flex flex-col items-center justify-center h-full py-16 text-center">
                <CheckCircle2 size={48} className="text-emerald-400 mb-4" />
                <h2 className="text-xl font-bold mb-2">Message Sent!</h2>
                <p className="text-gray-400">We&apos;ll get back to you within 24–48 hours.</p>
                <button onClick={() => { setSent(false); setForm({ name: "", email: "", subject: "", message: "" }); }} className="mt-6 text-sm text-indigo-400 hover:underline">Send another message</button>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm text-gray-400 mb-1.5">Name</label>
                    <input
                      type="text"
                      required
                      placeholder="Your name"
                      value={form.name}
                      onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white placeholder-gray-600 focus:outline-none focus:border-indigo-500 transition-colors"
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-gray-400 mb-1.5">Email</label>
                    <input
                      type="email"
                      required
                      placeholder="you@example.com"
                      value={form.email}
                      onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white placeholder-gray-600 focus:outline-none focus:border-indigo-500 transition-colors"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1.5">Subject</label>
                  <select
                    value={form.subject}
                    onChange={e => setForm(f => ({ ...f, subject: e.target.value }))}
                    required
                    className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white focus:outline-none focus:border-indigo-500 transition-colors"
                  >
                    <option value="" className="bg-gray-900">Select a subject</option>
                    <option value="general" className="bg-gray-900">General Inquiry</option>
                    <option value="billing" className="bg-gray-900">Billing & Subscription</option>
                    <option value="bug" className="bg-gray-900">Bug Report</option>
                    <option value="feature" className="bg-gray-900">Feature Request</option>
                    <option value="partnership" className="bg-gray-900">Partnership</option>
                    <option value="other" className="bg-gray-900">Other</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1.5">Message</label>
                  <textarea
                    required
                    rows={5}
                    placeholder="Tell us more..."
                    value={form.message}
                    onChange={e => setForm(f => ({ ...f, message: e.target.value }))}
                    className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2.5 text-sm text-white placeholder-gray-600 focus:outline-none focus:border-indigo-500 transition-colors resize-none"
                  />
                </div>
                {error && <p className="text-red-400 text-sm">{error}</p>}
                <button
                  type="submit"
                  disabled={sending}
                  className="w-full bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 disabled:opacity-50 text-white font-semibold py-3 rounded-xl transition-all"
                >
                  {sending ? "Sending…" : "Send Message"}
                </button>
              </form>
            )}
          </div>
        </div>

        {/* FAQ */}
        <section className="mt-20">
          <h2 className="text-2xl font-black text-center mb-8">Frequently Asked Questions</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {faqs.map(({ q, a }) => (
              <div key={q} className="bg-white/3 border border-white/8 rounded-xl p-5">
                <h3 className="font-semibold text-white mb-2 text-sm">{q}</h3>
                <p className="text-gray-400 text-sm leading-relaxed">{a}</p>
              </div>
            ))}
          </div>
        </section>
      </main>

      <footer className="border-t border-white/5 py-8 text-center text-gray-600 text-sm mt-10">
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
