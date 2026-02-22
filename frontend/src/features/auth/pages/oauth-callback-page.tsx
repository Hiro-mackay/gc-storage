import { Link, useParams, useSearch } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useOAuthCallback } from '@/features/auth/hooks/use-oauth-callback'

export function OAuthCallbackPage() {
  const { provider } = useParams({ strict: false }) as { provider: string }
  const search = useSearch({ strict: false }) as {
    code?: string
    state?: string
    error?: string
    error_description?: string
  }

  const { error, isPending } = useOAuthCallback(provider, search)

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Authentication failed</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">{error}</p>
        </CardContent>
        <CardFooter>
          <Button asChild className="w-full">
            <Link to="/login">Back to login</Link>
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (isPending) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-8">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="mt-4 text-sm text-muted-foreground">
            Completing authentication...
          </p>
        </CardContent>
      </Card>
    )
  }

  return null
}
