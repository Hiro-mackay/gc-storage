import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createWrapper } from '@/test/test-utils';
import { AppearanceTab } from '../components/appearance-tab';

vi.mock('../api/mutations', () => ({
  useUpdateProfileMutation: vi.fn(),
}));

vi.mock('@/stores/ui-store', () => ({
  useUIStore: vi.fn(),
}));

import { useUIStore } from '@/stores/ui-store';
import { useUpdateProfileMutation } from '../api/mutations';

function renderTab() {
  return render(<AppearanceTab />, { wrapper: createWrapper() });
}

describe('AppearanceTab', () => {
  beforeEach(() => {
    vi.mocked(useUIStore).mockReturnValue({
      theme: 'system',
      setTheme: vi.fn(),
    } as unknown as ReturnType<typeof useUIStore>);

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);
  });

  it('should render Theme label', () => {
    renderTab();
    expect(screen.getByText('Theme')).toBeInTheDocument();
  });

  it('should render System theme button', () => {
    renderTab();
    expect(screen.getByRole('button', { name: 'System' })).toBeInTheDocument();
  });

  it('should render Light theme button', () => {
    renderTab();
    expect(screen.getByRole('button', { name: 'Light' })).toBeInTheDocument();
  });

  it('should render Dark theme button', () => {
    renderTab();
    expect(screen.getByRole('button', { name: 'Dark' })).toBeInTheDocument();
  });

  it('should show System button as active when theme is system', () => {
    renderTab();
    const systemButton = screen.getByRole('button', { name: 'System' });
    expect(systemButton).toHaveAttribute('data-variant', 'default');
    expect(screen.getByRole('button', { name: 'Light' })).toHaveAttribute(
      'data-variant',
      'outline',
    );
    expect(screen.getByRole('button', { name: 'Dark' })).toHaveAttribute(
      'data-variant',
      'outline',
    );
  });

  it('should call setTheme and mutation when Light is clicked', async () => {
    const setTheme = vi.fn();
    const mutate = vi.fn();

    vi.mocked(useUIStore).mockReturnValue({
      theme: 'system',
      setTheme,
    } as unknown as ReturnType<typeof useUIStore>);

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate,
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    renderTab();

    await userEvent.click(screen.getByRole('button', { name: 'Light' }));

    expect(setTheme).toHaveBeenCalledWith('light');
    expect(mutate).toHaveBeenCalledWith({ theme: 'light' });
  });

  it('should call setTheme and mutation when Dark is clicked', async () => {
    const setTheme = vi.fn();
    const mutate = vi.fn();

    vi.mocked(useUIStore).mockReturnValue({
      theme: 'system',
      setTheme,
    } as unknown as ReturnType<typeof useUIStore>);

    vi.mocked(useUpdateProfileMutation).mockReturnValue({
      mutate,
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateProfileMutation>);

    renderTab();

    await userEvent.click(screen.getByRole('button', { name: 'Dark' }));

    expect(setTheme).toHaveBeenCalledWith('dark');
    expect(mutate).toHaveBeenCalledWith({ theme: 'dark' });
  });
});
