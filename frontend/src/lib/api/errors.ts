export function getApiErrorMessage(error: unknown, fallback = 'An error occurred'): string {
  if (error && typeof error === 'object' && 'error' in error) {
    const apiError = (error as { error?: { message?: string } }).error
    if (apiError?.message) return apiError.message
  }
  if (error instanceof Error) return error.message
  return fallback
}
