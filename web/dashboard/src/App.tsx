import { BrowserRouter, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./lib/AuthContext";
import { AuthPage } from "./pages/AuthPage";
import { VideoListPage } from "./pages/VideoListPage";
import { VideoDetailPage } from "./pages/VideoDetailPage";
import { ErrorBoundary } from "./components/ErrorBoundary";

function Shell() {
  const auth = useAuth();

  if (auth.status === "loading") {
    return <div className="page">Loading…</div>;
  }
  if (auth.status === "needs-setup") {
    return <AuthPage mode="needs-setup" />;
  }
  if (auth.status === "logged-out") {
    return <AuthPage mode="logged-out" />;
  }

  return (
    <BrowserRouter>
      <header className="app-header">
        <span className="app-brand">clipfolio</span>
        <span className="app-user">
          {auth.email}
          <button type="button" className="link-button" onClick={auth.logout}>
            Sign out
          </button>
        </span>
      </header>
      <main>
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<VideoListPage />} />
            <Route path="/videos/:id" element={<VideoDetailPage />} />
          </Routes>
        </ErrorBoundary>
      </main>
    </BrowserRouter>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  );
}
