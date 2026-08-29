import type { Lead } from "../lib/types";

export function LeadsTable({ leads }: { leads: Lead[] }) {
  if (leads.length === 0) {
    return <p className="empty-state">No leads captured yet.</p>;
  }
  return (
    <table className="leads-table">
      <thead>
        <tr>
          <th>Email</th>
          <th>Name</th>
          <th>Captured</th>
        </tr>
      </thead>
      <tbody>
        {leads.map((lead) => (
          <tr key={lead.id}>
            <td>{lead.email}</td>
            <td>{lead.name || "—"}</td>
            <td>{new Date(lead.createdAt).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
