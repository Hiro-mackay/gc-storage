type Role = 'viewer' | 'contributor' | 'content_manager';

interface RoleDropdownProps {
  value: Role;
  onChange: (role: Role) => void;
  disabled?: boolean;
}

const ROLES: { value: Role; label: string }[] = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'contributor', label: 'Contributor' },
  { value: 'content_manager', label: 'Content Manager' },
];

export function RoleDropdown({ value, onChange, disabled }: RoleDropdownProps) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value as Role)}
      disabled={disabled}
      className="h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
    >
      {ROLES.map((r) => (
        <option key={r.value} value={r.value}>
          {r.label}
        </option>
      ))}
    </select>
  );
}
