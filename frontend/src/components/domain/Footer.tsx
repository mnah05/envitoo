import { Link } from "react-router-dom";

export function Footer() {
  return (
    <footer className="border-t bg-background">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-10 sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <div className="flex items-center gap-2">
          <span className="grid size-6 place-items-center rounded-md bg-primary text-primary-foreground text-xs font-bold">
            P
          </span>
          <span className="font-bold tracking-tight">EnviToo</span>
        </div>
        <nav className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-muted-foreground">
          <Link to="/discover" className="hover:text-foreground">
            Discover
          </Link>
          <Link to="/create" className="hover:text-foreground">
            Create event
          </Link>
          <a
            href="https://github.com/mnah05/envitoo"
            target="_blank"
            rel="noreferrer"
            className="hover:text-foreground"
          >
            GitHub
          </a>
        </nav>
        <p className="text-sm text-muted-foreground">
          © {new Date().getFullYear()} EnviToo
        </p>
      </div>
    </footer>
  );
}
