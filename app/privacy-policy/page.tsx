import { Metadata } from "next";
import Link from "next/link";
import { Search } from "lucide-react";

export const metadata: Metadata = {
  title: "Privacy Policy — SeoXpert",
  description: "SeoXpert Privacy Policy — how we collect, use, and protect your data.",
};

const LAST_UPDATED = "June 9, 2026";
const CONTACT_EMAIL = "privacy@seoexpert.in";
const SITE = "https://seoexpert.in";

export default function PrivacyPolicy() {
  return (
    <div className="min-h-screen bg-[#050510] text-white">
      {/* Nav */}
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
        <h1 className="text-4xl font-black mb-2">Privacy Policy</h1>
        <p className="text-gray-500 text-sm mb-10">Last updated: {LAST_UPDATED}</p>

        <div className="space-y-8 text-gray-300 leading-relaxed text-[15px]">

          <section>
            <h2 className="text-white font-bold text-lg mb-3">1. Introduction</h2>
            <p>
              Welcome to SeoXpert ("<strong className="text-white">we</strong>", "<strong className="text-white">us</strong>", or "<strong className="text-white">our</strong>"). We operate the website {SITE} and provide SEO analysis, auditing, and keyword tracking services (collectively, the "<strong className="text-white">Service</strong>"). This Privacy Policy explains how we collect, use, disclose, and safeguard your information when you use our Service.
            </p>
            <p className="mt-3">By using the Service, you agree to the collection and use of information in accordance with this policy.</p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">2. Information We Collect</h2>
            <h3 className="text-white font-semibold mb-2">2.1 Information you provide directly</h3>
            <ul className="list-disc pl-5 space-y-1">
              <li>Account registration data (name, email address, password)</li>
              <li>Website URLs and domains you submit for analysis</li>
              <li>Billing information (processed securely via third-party payment processors)</li>
              <li>Communications you send us (support requests, feedback)</li>
            </ul>
            <h3 className="text-white font-semibold mb-2 mt-4">2.2 Information collected automatically</h3>
            <ul className="list-disc pl-5 space-y-1">
              <li>Log data (IP address, browser type, pages visited, timestamps)</li>
              <li>Device information (operating system, screen resolution)</li>
              <li>Usage data (features used, audit history, session duration)</li>
              <li>Cookies and similar tracking technologies (see Section 6)</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">3. How We Use Your Information</h2>
            <p>We use the information we collect to:</p>
            <ul className="list-disc pl-5 space-y-1 mt-2">
              <li>Provide, operate, and maintain the Service</li>
              <li>Process transactions and send related information (receipts, invoices)</li>
              <li>Send administrative communications (account updates, security alerts)</li>
              <li>Send promotional communications (with your consent; opt-out anytime)</li>
              <li>Monitor and analyse usage trends to improve the Service</li>
              <li>Detect, prevent, and address technical issues and fraudulent activity</li>
              <li>Comply with legal obligations</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">4. How We Share Your Information</h2>
            <p>We do <strong className="text-white">not</strong> sell your personal information. We may share data with:</p>
            <ul className="list-disc pl-5 space-y-1 mt-2">
              <li><strong className="text-white">Service providers</strong> — third parties that help us operate the Service (hosting, payment processing, email delivery, analytics), bound by confidentiality agreements</li>
              <li><strong className="text-white">Legal requirements</strong> — if required by law, court order, or government authority</li>
              <li><strong className="text-white">Business transfers</strong> — in connection with a merger, acquisition, or sale of assets</li>
              <li><strong className="text-white">With your consent</strong> — in any other case with your explicit consent</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">5. Data Retention</h2>
            <p>
              We retain your personal data for as long as your account is active or as needed to provide the Service. You may request deletion of your account and associated data at any time by contacting us at <a href={`mailto:${CONTACT_EMAIL}`} className="text-indigo-400 hover:underline">{CONTACT_EMAIL}</a>. Some data may be retained for up to 90 days in backups, and longer where required by law.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">6. Cookies</h2>
            <p>We use cookies and similar tracking technologies to:</p>
            <ul className="list-disc pl-5 space-y-1 mt-2">
              <li>Keep you logged in (session cookies)</li>
              <li>Remember your preferences</li>
              <li>Analyse Service usage (analytics cookies)</li>
            </ul>
            <p className="mt-3">You can control cookies through your browser settings. Disabling cookies may affect Service functionality.</p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">7. Data Security</h2>
            <p>
              We implement industry-standard security measures including HTTPS encryption, secure password hashing, and access controls. However, no method of transmission over the Internet is 100% secure. We cannot guarantee absolute security of your data.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">8. Your Rights</h2>
            <p>Depending on your location, you may have the right to:</p>
            <ul className="list-disc pl-5 space-y-1 mt-2">
              <li>Access the personal data we hold about you</li>
              <li>Correct inaccurate or incomplete data</li>
              <li>Request deletion of your data ("right to be forgotten")</li>
              <li>Object to or restrict processing of your data</li>
              <li>Data portability (receive your data in a structured format)</li>
              <li>Withdraw consent at any time (where processing is based on consent)</li>
            </ul>
            <p className="mt-3">To exercise these rights, contact us at <a href={`mailto:${CONTACT_EMAIL}`} className="text-indigo-400 hover:underline">{CONTACT_EMAIL}</a>.</p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">9. Third-Party Services</h2>
            <p>
              Our Service may contain links to third-party websites. We are not responsible for the privacy practices of those sites and encourage you to review their privacy policies. We use the following third-party services: Supabase (database/auth), and analytics providers. These services have their own privacy policies.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">10. Children's Privacy</h2>
            <p>
              Our Service is not directed at children under 13. We do not knowingly collect personal information from children under 13. If you believe we have inadvertently collected such information, please contact us immediately.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">11. Changes to This Policy</h2>
            <p>
              We may update this Privacy Policy from time to time. We will notify you of material changes by posting the new policy on this page and updating the "Last updated" date. Your continued use of the Service after any changes constitutes acceptance of the updated policy.
            </p>
          </section>

          <section>
            <h2 className="text-white font-bold text-lg mb-3">12. Contact Us</h2>
            <p>If you have questions about this Privacy Policy, please contact us:</p>
            <div className="mt-3 bg-white/3 border border-white/8 rounded-xl p-4 space-y-1">
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
