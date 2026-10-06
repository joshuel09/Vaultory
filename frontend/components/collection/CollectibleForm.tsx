'use client'

import { useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/field'
import { ImagePicker } from './ImagePicker'
import { ApiError } from '@/lib/api/errors'
import {
  COLLECTION_STATUSES,
  STATUS_LABELS,
  type Collectible,
  type CollectibleImageRef,
} from '@/lib/api/types'

/** The twelve attributes a collector types, as strings, in the order they tend to know them. */
export const EMPTY_VALUES = {
  name: '',
  collectionStatus: 'owned',
  character: '',
  series: '',
  manufacturer: '',
  category: '',
  scale: '',
  edition: '',
  purchasePrice: '',
  purchaseDate: '',
  releaseDate: '',
  notes: '',
}

export type CollectibleFormValues = typeof EMPTY_VALUES

/**
 * Fill the form from a stored collectible.
 *
 * An attribute the collector never supplied comes back null and becomes an empty field, never an
 * invented default (FR-003). The one exception is the status, which every collectible has.
 */
export function valuesOf(c: Collectible): CollectibleFormValues {
  return {
    name: c.name,
    collectionStatus: c.collectionStatus,
    character: c.character ?? '',
    series: c.series ?? '',
    manufacturer: c.manufacturer ?? '',
    category: c.category ?? '',
    scale: c.scale ?? '',
    edition: c.edition ?? '',
    purchasePrice: c.purchasePrice ?? '',
    purchaseDate: c.purchaseDate ?? '',
    releaseDate: c.releaseDate ?? '',
    notes: c.notes ?? '',
  }
}

/** Blank optional values are omitted entirely, so "not recorded" stays distinct from "" (FR-005). */
export function optional(value: string): string | undefined {
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}

interface Props {
  /** What the form opens with. Adding passes nothing; editing passes the stored values. */
  initialValues?: CollectibleFormValues
  initialImage?: CollectibleImageRef | null
  submitLabel: string
  /** Shown on the button while a submission is in flight. */
  submittingLabel: string
  /** Sits beside the submit button. */
  hint?: ReactNode
  /** Rendered above the fields — a success confirmation, or a version-conflict notice. */
  notice?: ReactNode
  /** Rendered below the submit row. Where editing puts its delete affordance (FR-022a). */
  footer?: ReactNode
  /** Clears the form after a successful submission. Adding does; editing navigates away instead. */
  resetOnSuccess?: boolean
  /**
   * Offered a failed submission before the form renders its own message. Returning true means the
   * parent has presented it — a version conflict is not a form error and deserves better than a
   * red banner (FR-027).
   */
  onError?: (error: unknown) => boolean
  onSubmit: (values: CollectibleFormValues, image: CollectibleImageRef | null) => Promise<void>
}

/**
 * The collectible form, shared by adding and editing.
 *
 * It exists in one copy on purpose. SC-008 requires that every rule rejecting a value when adding
 * rejects it when editing, and two 290-line components would make that a promise rather than a
 * property — the second one would drift the first time somebody changed only the one in front of
 * them.
 *
 * Client-side checking is deliberately thin. The server validates everything and reports every
 * problem at once; this renders that verdict rather than forming its own, so there is exactly one
 * set of rules (Constitution Principle III, FR-015).
 */
export function CollectibleForm({
  initialValues = EMPTY_VALUES,
  initialImage = null,
  submitLabel,
  submittingLabel,
  hint,
  notice,
  footer,
  resetOnSuccess = false,
  onError,
  onSubmit,
}: Props) {
  const [values, setValues] = useState<CollectibleFormValues>(initialValues)
  const [image, setImage] = useState<CollectibleImageRef | null>(initialImage)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  function set<K extends keyof CollectibleFormValues>(key: K, value: CollectibleFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }))
    // Clear a field's error as the collector addresses it, rather than leaving stale complaints on
    // screen.
    setFieldErrors((prev) => {
      if (!prev[key]) return prev
      const next = { ...prev }
      delete next[key]
      return next
    })
  }

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    /*
     * One submission at a time (FR-036).
     *
     * Adding needs this so a double-click does not depend on the submission key alone. Editing
     * needs it more: two requests carrying the same expectedVersion means the second arrives after
     * the first has already raised the version, comes back 409, and tells the collector the
     * collectible changed since they opened it — about their own save.
     */
    if (saving) return

    setSaving(true)
    setFormError(null)
    setFieldErrors({})

    try {
      await onSubmit(values, image)
      if (resetOnSuccess) {
        setValues(EMPTY_VALUES)
        setImage(null)
      }
    } catch (err) {
      // The parent gets first refusal. Everything the collector typed stays put either way.
      if (onError?.(err)) return
      if (err instanceof ApiError) {
        // Everything the collector typed stays exactly where it is (FR-035). Losing a filled-in
        // form to a failed save is the thing this must never do.
        setFieldErrors(err.byField())
        setFormError(
          err.isUnauthenticated ? 'Your session has expired. Sign in and try again.' : err.message,
        )
      } else {
        setFormError('This could not be saved. Your details are still here — try again.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-8">
      {notice}

      {formError && (
        <div
          data-testid="form-error"
          role="alert"
          className="rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-ink"
        >
          {formError}
        </div>
      )}

      <section className="space-y-5">
        <h2 className="text-sm font-medium uppercase tracking-wide text-ink-faint">The essentials</h2>

        <Field label="Name" required error={fieldErrors.name}>
          {({ id, describedBy, invalid }) => (
            <Input
              id={id}
              value={values.name}
              autoFocus
              placeholder="Kaiju Sentinel"
              aria-invalid={invalid || undefined}
              aria-describedby={describedBy}
              onChange={(e) => set('name', e.target.value)}
            />
          )}
        </Field>

        <Field label="Collection status" required error={fieldErrors.collectionStatus}>
          {({ id, describedBy, invalid }) => (
            <Select
              id={id}
              value={values.collectionStatus}
              aria-invalid={invalid || undefined}
              aria-describedby={describedBy}
              onChange={(e) => set('collectionStatus', e.target.value)}
            >
              {COLLECTION_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {STATUS_LABELS[s]}
                </option>
              ))}
            </Select>
          )}
        </Field>

        <ImagePicker image={image} onChange={setImage} />
      </section>

      <section className="space-y-5">
        <h2 className="text-sm font-medium uppercase tracking-wide text-ink-faint">
          The details <span className="normal-case tracking-normal text-ink-faint/70">— all optional</span>
        </h2>

        <div className="grid gap-5 sm:grid-cols-2">
          {(
            [
              ['character', 'Character', 'Sentinel Prime'],
              ['series', 'Series or franchise', 'Kaiju Wars'],
              ['manufacturer', 'Manufacturer', 'Apex Studio'],
              ['category', 'Category', 'Statue'],
              ['scale', 'Scale', '1/4'],
              ['edition', 'Edition or variant', 'Deluxe Exclusive'],
            ] as const
          ).map(([key, label, placeholder]) => (
            <Field key={key} label={label} error={fieldErrors[key]}>
              {({ id, describedBy, invalid }) => (
                <Input
                  id={id}
                  value={values[key]}
                  placeholder={placeholder}
                  aria-invalid={invalid || undefined}
                  aria-describedby={describedBy}
                  onChange={(e) => set(key, e.target.value)}
                />
              )}
            </Field>
          ))}

          <Field
            label="Purchase price"
            hint="Digits and up to two decimal places, like 249.99"
            error={fieldErrors.purchasePrice}
          >
            {({ id, describedBy, invalid }) => (
              <Input
                id={id}
                inputMode="decimal"
                value={values.purchasePrice}
                placeholder="249.99"
                aria-invalid={invalid || undefined}
                aria-describedby={describedBy}
                onChange={(e) => set('purchasePrice', e.target.value)}
              />
            )}
          </Field>

          <Field label="Purchase date" error={fieldErrors.purchaseDate}>
            {({ id, describedBy, invalid }) => (
              <Input
                id={id}
                type="date"
                value={values.purchaseDate}
                aria-invalid={invalid || undefined}
                aria-describedby={describedBy}
                onChange={(e) => set('purchaseDate', e.target.value)}
              />
            )}
          </Field>

          <Field
            label="Release date"
            hint="Past or future — a preorder and an already-released piece are both fine"
            error={fieldErrors.releaseDate}
          >
            {({ id, describedBy, invalid }) => (
              <Input
                id={id}
                type="date"
                value={values.releaseDate}
                aria-invalid={invalid || undefined}
                aria-describedby={describedBy}
                onChange={(e) => set('releaseDate', e.target.value)}
              />
            )}
          </Field>
        </div>

        <Field label="Notes" error={fieldErrors.notes}>
          {({ id, describedBy, invalid }) => (
            <Textarea
              id={id}
              rows={4}
              value={values.notes}
              placeholder="Box has a small dent on the lower left corner."
              aria-invalid={invalid || undefined}
              aria-describedby={describedBy}
              onChange={(e) => set('notes', e.target.value)}
            />
          )}
        </Field>
      </section>

      <div className="flex items-center gap-3 border-t border-edge pt-6">
        <Button type="submit" size="lg" disabled={saving}>
          {saving ? submittingLabel : submitLabel}
        </Button>
        {hint && <p className="text-xs text-ink-faint">{hint}</p>}
      </div>

      {footer}
    </form>
  )
}
