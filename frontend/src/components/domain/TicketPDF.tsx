import { Document, Image, Link, Page, StyleSheet, Text, View, pdf } from "@react-pdf/renderer";
import type { Event, Registration } from "@/lib/api";
import { formatEventDate, formatFullDate, formatPrice } from "@/lib/format";

export type TicketPdfInput = {
  event: Event;
  reg: Registration;
  token: string;
};

const GOLD = "#f5c518";
const INK = "#07070d";
const MUTE = "#6e6e8a";
const LINE = "#d4d4e0";

const styles = StyleSheet.create({
  page: {
    fontFamily: "Helvetica",
    fontSize: 11,
    color: INK,
    paddingTop: 64,
    paddingBottom: 96,
    paddingHorizontal: 48,
    backgroundColor: "#ffffff",
  },
  goldTop: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    height: 10,
    backgroundColor: GOLD,
  },
  goldBottom: {
    position: "absolute",
    bottom: 0,
    left: 0,
    right: 0,
    height: 10,
    backgroundColor: GOLD,
  },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    borderBottomWidth: 1,
    borderBottomColor: LINE,
    paddingBottom: 14,
    marginBottom: 24,
  },
  brand: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  brandMark: {
    width: 22,
    height: 22,
    borderRadius: 6,
    backgroundColor: GOLD,
  },
  brandName: {
    fontFamily: "Helvetica-Bold",
    fontSize: 14,
    letterSpacing: 0.4,
  },
  ticketId: {
    fontSize: 9,
    color: MUTE,
  },
  title: {
    fontFamily: "Helvetica-Bold",
    fontSize: 22,
    lineHeight: 1.2,
    marginBottom: 6,
  },
  dateLine: {
    fontSize: 12,
    color: MUTE,
    marginBottom: 18,
  },
  cover: {
    width: "100%",
    height: 150,
    borderRadius: 8,
    marginBottom: 20,
    objectFit: "cover",
  },
  divider: {
    height: 1,
    backgroundColor: LINE,
    marginVertical: 16,
  },
  body: {
    flexDirection: "row",
    justifyContent: "space-between",
  },
  leftCol: {
    width: "62%",
    gap: 14,
  },
  rightCol: {
    width: "30%",
    alignItems: "center",
    gap: 8,
  },
  label: {
    fontSize: 8,
    letterSpacing: 1,
    color: MUTE,
    textTransform: "uppercase",
    marginBottom: 3,
  },
  value: {
    fontSize: 11,
  },
  valueBold: {
    fontSize: 13,
    fontFamily: "Helvetica-Bold",
  },
  subValue: {
    fontSize: 10,
    color: MUTE,
  },
  qr: {
    width: 130,
    height: 130,
  },
  qrSub: {
    fontSize: 8,
    color: MUTE,
    textAlign: "center",
  },
  footer: {
    position: "absolute",
    bottom: 40,
    left: 48,
    right: 48,
    textAlign: "center",
  },
  footerHint: {
    fontSize: 9,
    color: MUTE,
    marginBottom: 4,
  },
  footerUrl: {
    fontSize: 9,
    color: INK,
    textDecoration: "none",
  },
});

function TicketDocument({
  event,
  reg,
  token,
  qrDataUrl,
  coverDataUrl,
}: TicketPdfInput & { qrDataUrl: string; coverDataUrl?: string }) {
  const venue = event.is_virtual
    ? "Virtual event"
    : [event.venue_name, event.location_address].filter(Boolean).join(" — ") || "Location TBA";
  const slugUrl = `${window.location.origin}/e/${event.slug}`;

  return (
    <Document>
      <Page size="A4" style={styles.page}>
        <View style={styles.goldTop} fixed />
        <View style={styles.goldBottom} fixed />

        <View style={styles.header}>
          <View style={styles.brand}>
            <View style={styles.brandMark} />
            <Text style={styles.brandName}>ENViTOO</Text>
          </View>
          <Text style={styles.ticketId}>
            TICKET · {token.slice(0, 8)}…{token.slice(-6)}
          </Text>
        </View>

        {coverDataUrl ? <Image style={styles.cover} src={coverDataUrl} /> : null}

        <Text style={styles.title}>{event.title}</Text>
        <Text style={styles.dateLine}>
          {formatFullDate(event.starts_at)} · {formatEventDate(event.starts_at, event.ends_at)}
        </Text>

        <View style={styles.divider} />

        <View style={styles.body}>
          <View style={styles.leftCol}>
            <View>
              <Text style={styles.label}>Attendee</Text>
              <Text style={styles.valueBold}>{reg.name}</Text>
              <Text style={styles.subValue}>{reg.email}</Text>
            </View>
            <View>
              <Text style={styles.label}>When</Text>
              <Text style={styles.value}>{formatEventDate(event.starts_at, event.ends_at)}</Text>
            </View>
            <View>
              <Text style={styles.label}>Where</Text>
              <Text style={styles.value}>{venue}</Text>
            </View>
            <View>
              <Text style={styles.label}>Admission</Text>
              <Text style={styles.value}>
                {formatPrice(event.price_in_paise, event.currency)}
                {event.requires_approval ? " · Approval required" : ""}
              </Text>
            </View>
          </View>

          <View style={styles.rightCol}>
            <Image style={styles.qr} src={qrDataUrl} />
            <Text style={styles.qrSub}>
              {token.slice(0, 8)}…{token.slice(-6)}
            </Text>
          </View>
        </View>

        <View style={styles.footer}>
          <Text style={styles.footerHint}>Present this QR at check-in · Keep this ticket for your records</Text>
          <Link src={slugUrl} style={styles.footerUrl}>
            {slugUrl}
          </Link>
        </View>
      </Page>
    </Document>
  );
}

async function toBlob(input: TicketPdfInput, qrDataUrl: string, coverDataUrl?: string) {
  return pdf(
    TicketDocument({ ...input, qrDataUrl, coverDataUrl }),
  ).toBlob();
}

/**
 * Generate a professional ticket PDF and return it as a Blob.
 * Tries embedding the event cover image; if that fails (CORS / fetch error),
 * retries without the cover so the PDF still generates.
 */
export async function generateTicketPdf(input: TicketPdfInput, qrDataUrl: string): Promise<Blob> {
  const { cover_image_url } = input.event;
  if (cover_image_url) {
    try {
      const res = await fetch(cover_image_url, { mode: "cors" });
      if (!res.ok) throw new Error("cover fetch failed");
      const blob = await res.blob();
      const coverDataUrl = await new Promise<string>((resolve, reject) => {
        const fr = new FileReader();
        fr.onload = () => resolve(fr.result as string);
        fr.onerror = () => reject(fr.error);
        fr.readAsDataURL(blob);
      });
      return toBlob(input, qrDataUrl, coverDataUrl);
    } catch {
      // fall through — retry without cover
    }
  }
  return toBlob(input, qrDataUrl);
}