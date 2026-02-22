import createClient from 'openapi-fetch';
import type { paths } from './schema';

/**
 * CookieからCSRFトークンを取得する
 */
function getCSRFToken(): string | undefined {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

export const api = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include',
  headers: {},
});

// リクエストごとにCSRFトークンヘッダーを動的に付与する
api.use({
  onRequest({ request }) {
    const method = request.method.toUpperCase();
    if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
      const csrfToken = getCSRFToken();
      if (csrfToken) {
        request.headers.set('X-CSRF-Token', csrfToken);
      }
    }
    return request;
  },
});
