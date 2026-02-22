import { useEffect, useRef } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useOAuthLoginMutation } from '@/features/auth/api/mutations'

interface OAuthSearchParams {
  code?: string
  state?: string
  error?: string
  error_description?: string
}

function validateOAuthState(provider: string, state?: string): boolean {
  const savedState = sessionStorage.getItem(`oauth_state_${provider}`)
  sessionStorage.removeItem(`oauth_state_${provider}`)
  return !!state && state === savedState
}

export function useOAuthCallback(
  provider: string,
  search: OAuthSearchParams,
) {
  const navigate = useNavigate()
  const mutation = useOAuthLoginMutation()

  // Validate CSRF state once (ref prevents double-execution in StrictMode)
  const stateCheckRef = useRef<{ valid: boolean } | null>(null)
  if (stateCheckRef.current === null) {
    stateCheckRef.current = {
      valid: validateOAuthState(provider, search.state),
    }
  }

  // Derive validation errors synchronously during render (not in an effect)
  const validationError = search.error
    ? (search.error_description ?? 'Authorization was denied.')
    : !stateCheckRef.current?.valid
      ? 'Invalid OAuth state. Please try again.'
      : !search.code
        ? 'No authorization code received.'
        : null

  const processedRef = useRef(false)

  useEffect(() => {
    if (validationError || !search.code || processedRef.current) return
    processedRef.current = true

    mutation.mutate(
      { provider, code: search.code },
      { onSuccess: () => navigate({ to: '/files' }) },
    )
    // mutation.mutate and navigate are stable references
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validationError, search.code, provider])

  return {
    error: validationError ?? (mutation.error ? mutation.error.message : null),
    isPending: !validationError && mutation.isPending,
  }
}
