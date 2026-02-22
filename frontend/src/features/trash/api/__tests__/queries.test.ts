import { renderHook, waitFor } from '@testing-library/react';
import { api } from '@/lib/api/client';
import { createWrapper } from '@/test/test-utils';
import { useTrashItems } from '../queries';

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

describe('useTrashItems', () => {
  const mockItems = [
    { id: 'file-1', name: 'report.pdf', daysUntilExpiry: 5 },
    { id: 'file-2', name: 'data.csv', daysUntilExpiry: 2 },
  ];

  it('calls GET /trash and returns items', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: { items: mockItems, nextCursor: null } },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashItems(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.GET).toHaveBeenCalledWith('/trash', expect.anything());
    expect(result.current.data).toEqual({
      items: mockItems,
      nextCursor: null,
    });
  });

  it('returns empty items array when data is undefined', async () => {
    mockApi.GET.mockResolvedValueOnce({
      data: { data: undefined },
      error: undefined,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashItems(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ items: [] });
  });

  it('throws on API error', async () => {
    const apiError = { error: { message: 'Unauthorized' } };
    mockApi.GET.mockResolvedValueOnce({
      data: undefined,
      error: apiError,
      response: new Response(),
    } as never);

    const { result } = renderHook(() => useTrashItems(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBe(apiError);
  });
});
