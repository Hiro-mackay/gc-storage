import { cn, formatBytes } from '@/lib/utils';

describe('cn', () => {
  it('merges class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar');
  });

  it('resolves Tailwind conflicts', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4');
  });

  it('handles conditional classes', () => {
    const condition = false as boolean;
    expect(cn('base', condition && 'hidden', 'end')).toBe('base end');
  });

  it('returns empty string for no inputs', () => {
    expect(cn()).toBe('');
  });
});

describe('formatBytes', () => {
  it.each([
    [0, '0 B'],
    [1, '1 B'],
    [500, '500 B'],
    [1024, '1 KB'],
    [1536, '1.5 KB'],
    [1048576, '1 MB'],
    [1073741824, '1 GB'],
    [1099511627776, '1 TB'],
  ])('formats %d as %s', (input, expected) => {
    expect(formatBytes(input)).toBe(expected);
  });

  it('returns "0 B" for negative values', () => {
    expect(formatBytes(-1)).toBe('0 B');
    expect(formatBytes(-1024)).toBe('0 B');
  });
});
