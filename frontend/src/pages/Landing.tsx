import { Link } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";

export function Landing() {
  return (
    <section className="flex min-h-[85dvh] items-center justify-center py-24">
      <PageContainer className="flex flex-col items-center text-center">
        <h1 className="display-hero max-w-3xl text-balance text-5xl tracking-tight sm:text-6xl">
          Delightful events
          <br />
          start here.
        </h1>

        <p className="mt-6 max-w-xl text-pretty text-lg text-muted-foreground">
          Set up an event page in minutes and open registrations today.
        </p>

        <div className="mt-8 flex flex-col items-center gap-4 sm:flex-row">
          <Button asChild size="lg" className="rounded-full px-6 text-base">
            <Link to="/create">
              Create your first event
              <ArrowRight data-icon="inline-end" />
            </Link>
          </Button>
          <Link
            to="/discover"
            className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            Discover events →
          </Link>
        </div>
      </PageContainer>
    </section>
  );
}