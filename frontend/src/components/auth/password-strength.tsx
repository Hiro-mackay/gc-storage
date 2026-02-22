interface PasswordStrengthProps {
  password: string;
}

export function PasswordStrength({ password }: PasswordStrengthProps) {
  const hasUppercase = /[A-Z]/.test(password);
  const hasLowercase = /[a-z]/.test(password);
  const hasNumber = /[0-9]/.test(password);
  const charTypesCount = [hasUppercase, hasLowercase, hasNumber].filter(
    Boolean,
  ).length;

  const rules = [
    { label: 'At least 8 characters', test: password.length >= 8 },
    {
      label: `At least 2 character types (${charTypesCount}/3: uppercase, lowercase, number)`,
      test: charTypesCount >= 2,
    },
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
