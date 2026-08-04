import { Link } from "react-router-dom";
import { usePageMeta } from "@/hooks/use-page-meta";
import { PageContainer } from "@/components/domain/Layout";

interface LegalSection {
  heading: string;
  body: string[];
}

interface LegalDoc {
  title: string;
  updated: string;
  intro: string;
  sections: LegalSection[];
}

const CONTACT_EMAIL = "hello@envitoo.app";

const PRIVACY: LegalDoc = {
  title: "Privacy Policy",
  updated: "August 4, 2026",
  intro:
    "This policy explains what EnviToo collects, why we collect it, and the choices you have. It applies to everyone who visits the site, registers for an event, or hosts events on EnviToo.",
  sections: [
    {
      heading: "1. What we collect",
      body: [
        "Account information: when you sign up through Clerk, we store your name, email address, and profile picture on our servers.",
        "Event information: when you create an event, we store everything you enter — title, description, schedule, capacity, location, cover image, and any volunteer roles.",
        "Registration information: when you register for an event (signed-in or as a guest), we store your name, email address, registration status, and check-in time.",
        "Technical data: web server logs (IP address, user agent, request path) are kept briefly to operate, secure, and debug the service.",
        "We do not use analytics trackers, advertising cookies, or third-party behavioural tracking.",
      ],
    },
    {
      heading: "2. How we use it",
      body: [
        "To run the service: process registrations, generate and validate QR tickets, send confirmation, ticket, OTP, and cancellation emails, and let hosts manage check-ins and volunteers.",
        "To keep the service safe: rate limiting, spam and abuse prevention.",
        "We never sell your personal data.",
      ],
    },
    {
      heading: "3. Third parties",
      body: [
        "Clerk — authentication and account management.",
        "Resend — transactional email delivery (tickets, confirmations, one-time codes).",
        "Supabase — Postgres database hosting.",
        "Cloudflare R2 — storage of uploaded cover images.",
        "Photon / OpenStreetMap — location search and map tiles when you search for or display event locations.",
        "Google Fonts — typography on the website.",
      ],
    },
    {
      heading: "4. How long we keep it",
      body: [
        "Account data is kept while your account is active. Event and registration data is kept to support event logistics (check-in, tickets, refunds of cancelled events).",
        "If you want your data deleted, email " + CONTACT_EMAIL + " and we will remove what we can within 30 days.",
      ],
    },
    {
      heading: "5. Your rights",
      body: [
        "You can request access to, correction of, or deletion of your personal data by contacting " + CONTACT_EMAIL + ".",
        "You can cancel your registrations from your dashboard at any time.",
        "If you signed up through Clerk, you can delete your Clerk account from the Clerk account page; contact us to also remove event and registration records.",
      ],
    },
    {
      heading: "6. Children",
      body: [
        "EnviToo is not directed at children under 13, and we do not knowingly collect their personal data. If you believe a child has registered, contact " + CONTACT_EMAIL + ".",
      ],
    },
    {
      heading: "7. Changes",
      body: [
        "If we change this policy in a material way, we will update the date above and, where practical, notify active hosts by email.",
      ],
    },
    {
      heading: "8. Contact",
      body: ["Questions about privacy: " + CONTACT_EMAIL + "."],
    },
  ],
};

const TERMS: LegalDoc = {
  title: "Terms of Service",
  updated: "August 4, 2026",
  intro:
    "By using EnviToo you agree to these terms. They are written in plain language so everyone — hosts, guests, and volunteers — knows where they stand.",
  sections: [
    {
      heading: "1. The service",
      body: [
        "EnviToo lets people create event pages, publish them (publicly on Discover or privately via a link), open registrations, issue QR tickets, and manage check-in and volunteers.",
        "The service is provided free of charge on an as-is basis while we grow it.",
      ],
    },
    {
      heading: "2. Your account",
      body: [
        "You are responsible for keeping your login credentials safe.",
        "You must be at least 13 years old to use EnviToo.",
        "You may not use the service to misrepresent yourself, create accounts for others without permission, or attempt to disrupt the service.",
      ],
    },
    {
      heading: "3. Hosting events",
      body: [
        "You are responsible for the events you publish: their legality, accuracy, safety, and your own compliance with any laws that apply (including, for HR hosts, employment and data-protection rules).",
        "You may not publish events that are unlawful, promote violence or hate, or are fraudulent.",
        "You may cancel your events from the dashboard. When you cancel, registrations close, check-in is disabled, and existing ticket holders are notified by email.",
      ],
    },
    {
      heading: "4. Registering and tickets",
      body: [
        "Registering for an event means you plan to attend; tickets are non-transferable and tied to your registration.",
        "If an event is cancelled, your ticket becomes invalid and the host is responsible for any further communication with you.",
        "When you register with an email address that already matches an account, the registration is linked to that account.",
      ],
    },
    {
      heading: "5. Content",
      body: [
        "You keep ownership of the event content and text you submit. You grant EnviToo a licence to store, display, and process that content to provide the service.",
        "Guests' registration data belongs to the guest; hosts may use it to run their event and for no other purpose without consent.",
      ],
    },
    {
      heading: "6. Acceptable use",
      body: [
        "Don't spam, scrape at scale, attempt to break or overload the service, or use it to send unsolicited email.",
        "Don't register for events you don't intend to attend, or create tickets/registrations in bulk to disrupt a host's event.",
        "We may remove content or suspend accounts that violate these terms.",
      ],
    },
    {
      heading: "7. Disclaimers and liability",
      body: [
        "EnviToo is provided as-is, without warranties of any kind. We work hard to keep it up, but we don't guarantee uninterrupted availability.",
        "To the maximum extent permitted by law, EnviToo is not liable for damages arising from your use of the service, including missed or cancelled events. Events are organised by their hosts, not by EnviToo.",
      ],
    },
    {
      heading: "8. Changes",
      body: [
        "We may update these terms from time to time; the version above is the current one.",
        "If you don't agree with a change, stop using the service. Continued use after a change means you accept it.",
      ],
    },
    {
      heading: "9. Contact",
      body: ["Questions about these terms: " + CONTACT_EMAIL + "."],
    },
  ],
};

export function PrivacyPolicy() {
  return <LegalDocView doc={PRIVACY} />;
}

export function TermsOfService() {
  return <LegalDocView doc={TERMS} />;
}

function LegalDocView({ doc }: { doc: LegalDoc }) {
  usePageMeta({ title: `${doc.title} — EnviToo` });

  return (
    <div className="pt-14">
      <PageContainer className="py-10">
        <div className="mx-auto max-w-2xl">
          <p className="text-eyebrow">Legal</p>
          <h1 className="display-title mt-2 text-3xl sm:text-4xl">{doc.title}</h1>
          <p className="mt-2 text-sm text-muted-foreground">Last updated {doc.updated}</p>
          <p className="text-lede mt-6">{doc.intro}</p>

          <div className="mt-10 flex flex-col gap-10">
            {doc.sections.map((section) => (
              <section key={section.heading}>
                <h2 className="display-headline text-xl">{section.heading}</h2>
                <div className="mt-3 flex flex-col gap-3">
                  {section.body.map((paragraph, i) => (
                    <p
                      key={i}
                      className="text-sm leading-relaxed text-muted-foreground"
                    >
                      {paragraph}
                    </p>
                  ))}
                </div>
              </section>
            ))}
          </div>

          <div className="mt-12 border-t border-border pt-6">
            <Link
              to="/"
              className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              Back to EnviToo
            </Link>
          </div>
        </div>
      </PageContainer>
    </div>
  );
}
