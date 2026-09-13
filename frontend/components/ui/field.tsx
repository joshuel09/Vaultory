import { forwardRef, useId } from 'react'
import { cn } from '@/lib/cn'

interface FieldProps {
  label: string
  hint?: string
  error?: string
  required?: boolean
  children: (props: { id: string; describedBy: string | undefined; invalid: boolean }) => React.ReactNode
}

/**
 * A labelled form field that wires up its own accessibility.
 *
 * The error is announced and linked to the input rather than merely coloured red, so a collector
 * using a screen reader — or one who cannot distinguish the colour — still learns what is wrong
 * (FR-044, FR-045).
 */
export function Field({ label, hint, error, required, children }: FieldProps) {
  const id = useId()
  const errorId = `${id}-error`
  const hintId = `${id}-hint`
  const describedBy = [error ? errorId : null, hint ? hintId : null].filter(Boolean).join(' ') || undefined

  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium text-ink">
        {label}
        {required && (
          <span className="ml-1 text-accent" aria-hidden="true">
            *
          </span>
        )}
        {required && <span className="sr-only"> (required)</span>}
      </label>
      {children({ id, describedBy, invalid: Boolean(error) })}
      {hint && !error && (
        <p id={hintId} className="text-xs text-ink-faint">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} role="alert" className="flex items-start gap-1.5 text-xs text-danger">
          {/* A glyph as well as the colour, so the error does not depend on seeing red. */}
          <span aria-hidden="true">!</span>
          <span>{error}</span>
        </p>
      )}
    </div>
  )
}

export const Input = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        'w-full rounded-lg border bg-surface px-3 py-2 text-sm text-ink placeholder:text-ink-faint',
        'border-edge focus:border-accent',
        props['aria-invalid'] && 'border-danger',
        className,
      )}
      {...props}
    />
  ),
)
Input.displayName = 'Input'

export const Textarea = forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        'w-full rounded-lg border bg-surface px-3 py-2 text-sm text-ink placeholder:text-ink-faint',
        'border-edge focus:border-accent',
        props['aria-invalid'] && 'border-danger',
        className,
      )}
      {...props}
    />
  ),
)
Textarea.displayName = 'Textarea'

export const Select = forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, ...props }, ref) => (
    <select
      ref={ref}
      className={cn(
        'w-full rounded-lg border bg-surface px-3 py-2 text-sm text-ink',
        'border-edge focus:border-accent',
        props['aria-invalid'] && 'border-danger',
        className,
      )}
      {...props}
    />
  ),
)
Select.displayName = 'Select'
