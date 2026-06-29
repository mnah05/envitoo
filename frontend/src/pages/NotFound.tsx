import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";

export function NotFound() {
  return (
    <div className="pt-16">
      <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
        <p className="text-6xl font-bold tracking-tight text-primary">404</p>
        <h1 className="text-2xl font-bold">Page not found</h1>
        <p className="text-muted-foreground">The page you're looking for doesn't exist.</p>
        <Button asChild>
          <Link to="/">Back home</Link>
        </Button>
      </PageContainer>
    </div>
  );
}
