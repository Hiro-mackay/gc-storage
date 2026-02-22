import { render, screen, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { PendingInvitationsPage } from '../pending-invitations-page';

vi.mock('@/lib/api/client', () => ({
  api: {
    GET: vi.fn(),
    POST: vi.fn(),
    PATCH: vi.fn(),
    DELETE: vi.fn(),
  },
}));

const mockApi = vi.mocked(api);

beforeEach(() => {
  vi.clearAllMocks();
});

function renderPage() {
  return render(<PendingInvitationsPage />, { wrapper: createWrapper() });
}

describe('PendingInvitationsPage', () => {
  it('should display "No pending invitations" when list is empty', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: [] },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No pending invitations')).toBeInTheDocument();
    });
  });

  it('should display page heading', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: [] },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(
        screen.getByRole('heading', { name: 'Pending Invitations' }),
      ).toBeInTheDocument();
    });
  });

  it('should display group name for each invitation', async () => {
    const mockPending = [
      {
        invitation: { id: 'inv-1', role: 'viewer', expiresAt: null },
        group: { id: 'g-1', name: 'Alpha Team' },
      },
      {
        invitation: { id: 'inv-2', role: 'contributor', expiresAt: null },
        group: { id: 'g-2', name: 'Beta Squad' },
      },
    ];

    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockPending },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Alpha Team')).toBeInTheDocument();
      expect(screen.getByText('Beta Squad')).toBeInTheDocument();
    });
  });

  it('should display role badges for each invitation', async () => {
    const mockPending = [
      {
        invitation: { id: 'inv-1', role: 'viewer', expiresAt: null },
        group: { id: 'g-1', name: 'Alpha Team' },
      },
    ];

    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockPending },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('viewer')).toBeInTheDocument();
    });
  });

  it('should show "Check your email" message when invitations are present', async () => {
    const mockPending = [
      {
        invitation: { id: 'inv-1', role: 'viewer', expiresAt: null },
        group: { id: 'g-1', name: 'Alpha Team' },
      },
    ];

    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockPending },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(
        screen.getByText('Check your email to accept or decline invitations'),
      ).toBeInTheDocument();
    });
  });

  it('should display "Your Invitations" card title when invitations are present', async () => {
    const mockPending = [
      {
        invitation: { id: 'inv-1', role: 'viewer', expiresAt: null },
        group: { id: 'g-1', name: 'Alpha Team' },
      },
    ];

    mockApi.GET.mockResolvedValueOnce({
      data: { data: mockPending },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('Your Invitations')).toBeInTheDocument();
    });
  });

  it('should return empty array when data is null and show no invitations message', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: null },
      error: undefined,
      response: new Response(),
    } as never);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText('No pending invitations')).toBeInTheDocument();
    });
  });
});
