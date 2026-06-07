"use client";
import { useState, useEffect, useRef } from "react";

function TimeBox({ value, label }: { value: number; label: string }) {
  return (
    <div className="glass px-6 py-5 text-center min-w-[80px]">
      <div className="text-4xl font-black text-white tabular-nums">
        {String(value).padStart(2, "0")}
      </div>
      <div className="text-xs text-gray-500 uppercase tracking-widest mt-1">{label}</div>
    </div>
  );
}

export default function Countdown() {
  const target = useRef(new Date(Date.now() + 45 * 24 * 60 * 60 * 1000));
  const calc = () => {
    const diff = Math.max(0, target.current.getTime() - Date.now());
    return {
      days: Math.floor(diff / 86400000),
      hours: Math.floor((diff % 86400000) / 3600000),
      minutes: Math.floor((diff % 3600000) / 60000),
      seconds: Math.floor((diff % 60000) / 1000),
    };
  };
  const [t, setT] = useState(calc);
  useEffect(() => {
    const id = setInterval(() => setT(calc()), 1000);
    return () => clearInterval(id);
  }, []);
  return (
    <div className="flex justify-center gap-3 mb-12 flex-wrap">
      <TimeBox value={t.days} label="Days" />
      <TimeBox value={t.hours} label="Hours" />
      <TimeBox value={t.minutes} label="Mins" />
      <TimeBox value={t.seconds} label="Secs" />
    </div>
  );
}
