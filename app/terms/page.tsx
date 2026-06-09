import { Metadata } from "next";
import Link from "next/link";
import { Search } from "lucide-react";

export const metadata: Metadata = {
  title: "Terms of Service — SeoXpert",
  description: "SeoXpert Terms of Service — the rules and conditions governing your use of our platform.",
};

const LAST_UPDATED = "June 9, 2026";
const CONTACT_EMAIL = "legal@seoexpert.in";
const SITE = "https://seoexpert.in";

export default function TermsOfService() {
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

      <main className="max-w-3xl mx-auto px-6 py-16">
        <h1 className="text-4xl font-black mb-2">Terms of Service</h1>
        <p className="text-gray-500 text-sm mb-10">Last updated: {LAST_UPDATED}</p>

        <div className="space-y-8 text-gray-300 leading-relaxed text-[15px]">

          <section>
            <h2 className="text-white font-bold text-lg mb-3">1. Acceptance of Terms</h2>
            <p>
              By accessing or using SeoXpert ("{SITE}"), you agree to be bound by these Terms of Service ("<strong className="text-white">Terms</strong>"). If you do not agree to these Terms, do not use the Service. These Terms apply to all visitors, users, and others who access the Service.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">2. Description of Service</h2>
            <p>
              SeoXpert provides an SEO analytics and auditing platform including keyword tracking, technical site audits, domain analysis, and related tools (the "<strong className="text-white">Service</strong>"). We reserve the right to modify, suspend, or discontinue the Service at any time with or without notice.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">3. Accounts</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>You must provide accurate and complete information when creating an account.</li>
              <li>You are responsible for maintaining the security of your account and password.</li>
              <li>You must notify us immediately of any unauthorized use of your account.</li>
              <li>You may not use another person's account without permission.</li>
              <li>You must be at least 13 years old to use the Service.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">4. Acceptable Use</h2>
            <p>You agree not to:</p>
            <ul className="list-disc pl-5 space-y-2 mt-2">
              <li>Use the Service for any unlawful purpose or in violation of any regulations</li>
              <li>Attempt to gain unauthorized access to any part of the Service or its related systems</li>
              <li>Reverse engineer, decompile, or disassemble any part of the Service</li>
              <li>Submit websites containing malware, phishing, or illegal content for analysis</li>
              <li>Use automated means to access the Service beyond normal usage (scraping, crawling our platform)</li>
              <li>Resell or commercialise the Service without our written consent</li>
              <li>Interfere with or disrupt the integrity or performance of the Service</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">5. Intellectual Property</h2>
            <p>
              The Service and its original content, features, and functionality are owned by SeoXpert and are protected by international copyright, trademark, and other intellectual property laws. You may not copy, modify, distribute, or create derivative works based on our Service without our express written permission.
            </p>
            <p className="mt-3">
              You retain ownership of any data or content you submit to the Service. By submitting content, you grant us a limited licence to process it to provide the Service.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">6. Subscription and Payments</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>Some features require a paid subscription. Pricing is displayed on the pricing page.</li>
              <li>Subscriptions are billed in advance on a monthly or annual basis.</li>
              <li>All fees are non-refundable unless required by applicable law or stated otherwise.</li>
              <li>We reserve the right to change pricing with 30 days&apos; notice.</li>
              <li>Failure to pay may result in suspension or termination of your account.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">7. Disclaimer of Warranties</h2>
            <p>
              THE SERVICE IS PROVIDED "AS IS" AND "AS AVAILABLE" WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO IMPLIED WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, OR NON-INFRINGEMENT. WE DO NOT WARRANT THAT THE SERVICE WILL BE UNINTERRUPTED, ERROR-FREE, OR FREE OF VIRUSES OR OTHER HARMFUL COMPONENTS.
            </p>
            <p className="mt-3">
              SEO results and rankings depend on many factors outside our control. We do not guarantee specific search engine rankings or traffic outcomes.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">8. Limitation of Liability</h2>
            <p>
              TO THE MAXIMUM EXTENT PERMITTED BY LAW, SEOXPERT SHALL NOT BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, INCLUDING LOSS OF PROFITS, DATA, OR GOODWILL, ARISING FROM YOUR USE OF OR INABILITY TO USE THE SERVICE. OUR TOTAL LIABILITY SHALL NOT EXCEED THE AMOUNT YOU PAID US IN THE 12 MONTHS PRECEDING THE CLAIM.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">9. Indemnification</h2>
            <p>
              You agree to defend, indemnify, and hold harmless SeoXpert and its officers, directors, employees, and agents from any claims, damages, obligations, losses, liabilities, costs, or debt arising from your use of the Service or violation of these Terms.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">10. Termination</h2>
            <p>
              We may terminate or suspend your account and access to the Service immediately, without prior notice, if you breach these Terms. Upon termination, your right to use the Service ceases immediately. You may terminate your account at any time from your account settings.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">11. Governing Law</h2>
            <p>
              These Terms shall be governed by and construed in accordance with the laws of India, without regard to its conflict of law provisions. Any disputes shall be subject to the exclusive jurisdiction of courts in India.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">12. Changes to Terms</h2>
            <p>
              We reserve the right to update these Terms at any time. We will notify you of significant changes by email or by posting a notice on the Service. Your continued use after changes constitutes acceptance of the new Terms.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">13. Contact</h2>
            <div className="bg-white/3 border border-white/8 rounded-xl p-4 space-y-1">
              <p><strong className="text-white">SeoXpert</strong></p>
              <p>Email: <a href={`mailto:${CONTACT_EMAIL}`} className="text-indigo-400 hover:underline">{CONTACT_EMAIL}</a></p>
              <p>Website: <a href={SITE} className="text-indigo-400 hover:underline">{SITE}</a></p>
            </div>
          </section>

        </div>
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
