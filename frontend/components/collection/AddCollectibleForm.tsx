'use client'

import { useRouter } from 'next/navigation'
import { useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/field'
import { ImagePicker } from './ImagePicker'
import { ApiError } from '@/lib/api/errors'
import { addCollectible } from '@/lib/api/collectibles'
import { randomUUID } from '@/lib/uuid'
import { COLLECTION_STATUSES, STATUS_LABELS, type CollectibleImage } from '@/lib/api/types'

/** The eleven optional text and date attributes, in the order a collector tends to know them. */
const EMPTY = {
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

type FormValues = typeof EMPTY

/**
 * Record a new collectible.
 *
 * Client-side checking is deliberately thin. The server validates everything and reports every
 * problem at once; this form renders that verdict rather than forming its own, so there is exactly
 * one set of rules (Constitution Principle III, FR-020).
 */
export function AddCollectibleForm() {
  const router = useRouter()
  const [values, setValues] = useState<FormValues>(EMPTY)
  const [image, setImage] = useState<CollectibleImage | null>(null)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState<string | null>(null)

  /*
   * One submission key per collectible, not per attempt (FR-047).
   *
   * This is the whole point: a double-click or a retry after a lost response replays the same key
   * and returns the collectible already created, while a collector deliberately adding a second
   * identical copy starts a new form and so a new key. Regenerating per attempt would defeat it.
   */
  const submissionKey = useRef<string>(randomUUID())

  function set<K extends keyof FormValues>(key: K, value: FormValues[K]) {
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

  /** Blank optional values are omitted entirely, so "not recorded" stays distinct from "" (FR-007). */
  function optional(value: string): string | undefined {
    const trimmed = value.trim()
    return trimmed === '' ? undefined : trimmed
  }

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (saving) return

    setSaving(true)
    setFormError(null)
    setFieldErrors({})

    try {
      const created = await addCollectible({
        submissionKey: submissionKey.current,
        name: values.name,
        collectionStatus: values.collectionStatus as (typeof COLLECTION_STATUSES)[number],
        character: optional(values.character),
        series: optional(values.series),
        manufacturer: optional(values.manufacturer),
        category: optional(values.category),
        scale: optional(values.scale),
        edition: optional(values.edition),
        purchasePrice: optional(values.purchasePrice),
        purchaseDate: optional(values.purchaseDate),
        releaseDate: optional(values.releaseDate),
        notes: optional(values.notes),
        imageId: image?.id,
      })

      setSaved(created.name)
      // A fresh key: the next collectible is a new submission, not a retry of this one.
      submissionKey.current = randomUUID()
      setValues(EMPTY)
      setImage(null)
      router.refresh()
    } catch (err) {
      if (err instanceof ApiError) {
        // Everything the collector typed stays exactly where it is (FR-022). Losing a filled-in
        // form to a failed save is the thing this must never do.
        setFieldErrors(err.byField())
        setFormError(err.isUnauthenticated ? 'Your session has expired. Sign in and try again.' : err.message)
      } else {
        setFormError('This collectible could not be saved. Your details are still here — try again.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-8">
      {saved && (
        <div
          data-testid="add-success"
          role="status"
          className="rounded-lg border border-success/40 bg-success/10 px-4 py-3 text-sm text-ink"
        >
          <span aria-hidden="true" className="mr-2 text-success">
            ✓
          </span>
          <strong className="font-medium">{saved}</strong> was added to your vault.
        </div>
      )}

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
          {saving ? 'Adding…' : 'Add to my vault'}
        </Button>
        <p className="text-xs text-ink-faint">A name and a status are all you need.</p>
      </div>
    </form>
  )
}
