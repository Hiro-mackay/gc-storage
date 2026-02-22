import { useEffect, useState } from 'react'
import { Link, useSearch } from '@tanstack/react-router'
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api/client'

export function VerifyEmailPage() {
  const search = useSearch({ strict: false }) as { token?: string }
  const [status, setStatus] = useState<'verifying' | 'success' | 'error'>(
    'verifying'
  )
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!search.token) {
      setStatus('error')
      setError('Verification token is missing.')
      return
    }

    const verify = async () => {
      try {
        const { error: apiError } = await api.POST('/auth/email/verify', {
          params: { query: { token: search.token! } },
        })

        if (apiError) {
          setStatus('error')
          setError(apiError.error?.message ?? 'Verification failed.')
          return
        }

        setStatus('success')
      } catch {
        setStatus('error')
        setError('Network error. Please try again.')
      }
    }

    verify()
  }, [search.token])

  return (
    <Card>
      <CardHeader>
        <CardTitle>Email Verification</CardTitle>
      </CardHeader>
      <CardContent>
        {status === 'verifying' && (
          <p className="text-sm text-muted-foreground">Verifying your email...</p>
        )}
        {status === 'success' && (
          <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-700">
            Your email has been verified successfully. You can now log in.
          </div>
        )}
        {status === 'error' && (
          <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            {error}
          </div>
        )}
      </CardContent>
      <CardFooter>
        <Link to="/login" className="w-full">
          <Button className="w-full">Go to Login</Button>
        </Link>
      </CardFooter>
    </Card>
  )
}
