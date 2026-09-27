/**
 * The action name for an authentication request, with any dynamic segment dropped.
 *
 * `/api/auth/reset-password/:token` would otherwise put the token straight into the log — which
 * FR-016 forbids, and which the logger was doing until a check against the running stack found 45
 * of them.
 *
 * A whitelist rather than a list of things to redact: only lowercase word segments survive, so the
 * next dynamic segment somebody adds is dropped without anyone having to remember to redact it.
 */
export function actionOf(pathname: string): string {
  return pathname
    .replace(/^\/api\/auth\//, '')
    .split('/')
    .filter((segment) => /^[a-z][a-z0-9-]*$/.test(segment))
    .join('/')
}
