import { useState, type FormEvent } from "react";
import { api, APIError } from "../lib/api";
import { useAuth } from "../lib/AuthContext";

export function AuthPage({ mode }: { mode: "needs-setup" | "logged-out" }) {
  const auth = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [setupToken, setSetupToken] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      if (mode === "needs-setup") {
        await api.setup(email, password, setupToken);
      } else {
        await api.login(email, password);
      }
      await auth.refresh();
    } catch (err) {
      setError(err instanceof APIError ? err.message : "Something went wrong");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>clipfolio</h1>
        <p className="auth-subtitle">
          {mode === "needs-setup" ? "Create the admin account for this deploy." : "Sign in to your dashboard."}
        </p>
        <form onSubmit={onSubmit}>
          <label>
            Email
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
          </label>
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              minLength={8}
              required
            />
          </label>
          {mode === "needs-setup" && (
            <label>
              Setup token <span className="hint">(only if CLIPFOLIO_SETUP_TOKEN is set)</span>
              <input type="text" value={setupToken} onChange={(e) => setSetupToken(e.target.value)} />
            </label>
          )}
          {error && <p className="form-error">{error}</p>}
          <button type="submit" disabled={busy}>
            {mode === "needs-setup" ? "Create account" : "Sign in"}
          </button>
        </form>
      </div>
    </div>
  );
}
