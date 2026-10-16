/**
 * What a collector is told when a reset succeeded but the sign-in that follows it did not.
 *
 * The reset itself worked, and that is the part they most need to hear: without it they assume the
 * whole thing failed, go back to their inbox, and follow a link that has already been spent —
 * which answers "that link is no longer valid" and looks like the product losing their password
 * twice over.
 *
 * `throttled` gets its own wording because the two cases ask different things of the collector.
 * Sign-in is limited per address, so somebody who mistyped their old password a few times before
 * resetting it will have that attempt refused as well; telling them to sign in now would be
 * sending them into a second refusal (issue #41).
 */
export function ResetCompleteNotice({ throttled }: { throttled: boolean }) {
  return (
    <div
      data-testid="reset-complete"
      role="status"
      className="mb-6 rounded-lg border border-accent/40 bg-accent/10 px-4 py-3 text-sm text-ink"
    >
      <p className="font-medium">Your password has been changed.</p>
      <p className="mt-1 text-ink-muted">
        {throttled ? (
          <>
            There have been too many sign-in attempts from your network, so we could not sign you in
            straight away. Wait a few minutes, then sign in with your new password.
          </>
        ) : (
          <>Sign in with your new password to get back to your collection.</>
        )}
      </p>
    </div>
  )
}
