"use client";
import { useState } from "react";
import { Bell, Lock, CheckCircle2 } from "lucide-react";

export default function EmailForm() {
  const [email, setEmail] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/)) {
      setError("Please enter a valid email.");
      return;
    }
    setError("");
    setSubmitted(true);
  };

  if (submitted) {
    return (
      <div className="flex items-center justify-center gap-3 glass px-8 py-5 max-w-md mx-auto border border-emerald-500/30">
        <CheckCircle2 size={22} className="text-emerald-400 shrink-0" />
        <div className="text-left">
          <p className="text-white font-semibold">You&apos;re on the list!</p>
          <p className="text-gray-400 text-sm">We&apos;ll notify you at launch.</p>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col sm:flex-row gap-3 max-w-md mx-auto">
      <div className="flex-1">
        <input
          type="email"
          value={email}
          onChange={e => setEmail(e.target.value)}
          placeholder="Enter your email address"
          className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3.5 text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500/60 transition-colors"
        />
        {error && <p className="text-red-400 text-xs mt-1 text-left">{error}</p>}
      </div>
      <button
        type="submit"
        className="flex items-center justify-center gap-2 bg-gradient-to-r from-indigo-600 to-violet-600 text-white font-semibold px-6 py-3.5 rounded-xl whitespace-nowrap"
      >
        <Bell size={16} /> Notify Me
      </button>
      <p className="text-gray-600 text-xs mt-3 flex items-center justify-center gap-1.5 sm:hidden">
        <Lock size={11} /> No spam. Early access perks for subscribers.
      </p>
    </form>
  );
}
