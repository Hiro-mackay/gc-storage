interface PasswordStrengthProps {
  password: string;
}

export function PasswordStrength({ password }: PasswordStrengthProps) {
  const rules = [
    { label: 'At least 8 characters', test: password.length >= 8 },
    { label: 'At least one uppercase letter', test: /[A-Z]/.test(password) },
    { label: 'At least one number', test: /[0-9]/.test(password) },
  ];

  return (
    <ul className="space-y-1 text-xs">
      {rules.map((rule) => (
        <li
          key={rule.label}
          className={rule.test ? 'text-green-600' : 'text-muted-foreground'}
        >
          {rule.test ? '\u2713' : '\u2717'} {rule.label}
        </li>
      ))}
    </ul>
  );
}
