// The gateway's sign-in routes (ADR 0013). They are pages, not API calls: the
// browser itself goes there, so the provider can show its own sign-in page.
export const loginPath = "/auth/login"
export const logoutPath = "/auth/logout"

// signInRedirect returns where to send the browser after an API response, or
// undefined to stay. A 401 means the session expired or was never there, and
// no screen can do anything with it but start over from sign-in.
export function signInRedirect(status: number): string | undefined {
  return status === 401 ? loginPath : undefined
}
