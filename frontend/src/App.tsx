import { Routes, Route, Outlet, useLocation } from "react-router-dom";
import { lazy, Suspense, useEffect } from "react";
import { Header } from "@/components/domain/Header";
import { Footer } from "@/components/domain/Footer";
import { Loading } from "@/components/domain/Layout";
import { useSmoothScroll } from "@/hooks/use-smooth-scroll";
import { ProtectedRoute, PublicOnlyRoute } from "@/routes/ProtectedRoute";
import { ErrorBoundary } from "@/components/ui/error-boundary";

const Landing = lazy(() =>
  import("@/pages/Landing").then((m) => ({ default: m.Landing })),
);
const Discover = lazy(() =>
  import("@/pages/Discover").then((m) => ({ default: m.Discover })),
);
const EventDetail = lazy(() =>
  import("@/pages/EventDetail").then((m) => ({ default: m.EventDetail })),
);
const GuestTicket = lazy(() =>
  import("@/pages/GuestTicket").then((m) => ({ default: m.GuestTicket })),
);
const SignInPage = lazy(() =>
  import("@/pages/SignIn").then((m) => ({ default: m.SignInPage })),
);
const SignUpPage = lazy(() =>
  import("@/pages/SignUp").then((m) => ({ default: m.SignUpPage })),
);
const MyTickets = lazy(() =>
  import("@/pages/MyTickets").then((m) => ({ default: m.MyTickets })),
);
const CreateEvent = lazy(() =>
  import("@/pages/CreateEvent").then((m) => ({ default: m.CreateEvent })),
);
const ManageEvents = lazy(() =>
  import("@/pages/ManageEvents").then((m) => ({ default: m.ManageEvents })),
);
const EventDashboard = lazy(() =>
  import("@/pages/EventDashboard").then((m) => ({ default: m.EventDashboard })),
);
const EditEvent = lazy(() =>
  import("@/pages/EditEvent").then((m) => ({ default: m.EditEvent })),
);
const NotFound = lazy(() =>
  import("@/pages/NotFound").then((m) => ({ default: m.NotFound })),
);

function SiteLayout() {
  const { pathname } = useLocation();
  const smoothScroll = pathname === "/" || pathname.startsWith("/discover");
  useSmoothScroll(smoothScroll);
  useEffect(() => window.scrollTo(0, 0), [pathname]);

  return (
    <div className="flex min-h-dvh flex-col">
      <Header />
      <main className="flex-1">
        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>
      </main>
      <Footer />
    </div>
  );
}

export function App() {
  return (
    <ErrorBoundary>
      <Suspense fallback={<Loading />}>
        <Routes>
          <Route element={<SiteLayout />}>
            <Route path="/" element={<Landing />} />
            <Route path="/discover" element={<Discover />} />
            <Route path="/e/:slug" element={<EventDetail />} />
            <Route path="/ticket" element={<GuestTicket />} />

            <Route element={<ProtectedRoute />}>
              <Route path="/home" element={<MyTickets />} />
              <Route path="/create" element={<CreateEvent />} />
              <Route path="/manage" element={<ManageEvents />} />
              <Route path="/manage/:id" element={<EventDashboard />} />
              <Route path="/manage/:id/edit" element={<EditEvent />} />
            </Route>

            <Route path="*" element={<NotFound />} />
          </Route>

          <Route element={<PublicOnlyRoute />}>
            <Route path="/signin/*" element={<SignInPage />} />
            <Route path="/signup/*" element={<SignUpPage />} />
          </Route>
        </Routes>
      </Suspense>
    </ErrorBoundary>
  );
}
