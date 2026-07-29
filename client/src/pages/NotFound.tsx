import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";

export function NotFound() {
  return (
    <div className="pt-14">
      <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-6 text-center">
        <div className="flex items-baseline gap-1 font-display text-[clamp(4rem,12vw,8rem)] font-semibold leading-none tracking-[-0.04em]">
          <span className="text-primary">4</span>
          <span className="text-muted-foreground/30">0</span>
          <span className="text-primary">4</span>
        </div>
        <div>
          <h1 className="display-title text-xl">Page not found</h1>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            The page you&apos;re looking for doesn&apos;t exist.
          </p>
        </div>
        <Button asChild>
          <Link to="/">Back home</Link>
        </Button>
      </PageContainer>
    </div>
  );
}
