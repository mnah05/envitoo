import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";

export function NotFound() {
  return (
    <div className="pt-16">
      <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
        <p className="display-hero text-6xl text-primary sm:text-7xl">404</p>
        <h1 className="display-title text-2xl">Page not found</h1>
        <p className="text-lede max-w-sm text-[1.0625rem]">
          The page you're looking for doesn't exist.
        </p>
        <Button asChild className="mt-2 rounded-full">
          <Link to="/">Back home</Link>
        </Button>
      </PageContainer>
    </div>
  );
}
