'use client'

import { useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { ApiError } from '@/lib/api/errors'
import { describeObviousProblem, uploadImage } from '@/lib/api/images'
import type { CollectibleImage } from '@/lib/api/types'

interface Props {
  image: CollectibleImage | null
  onChange: (image: CollectibleImage | null) => void
}

/**
 * Choose one photograph for a collectible (FR-008).
 *
 * The upload happens here and on its own, which is what lets a refused image leave the rest of the
 * form untouched: the collector fixes the photo or proceeds without one, and nothing they typed is
 * lost (FR-013).
 */
export function ImagePicker({ image, onChange }: Props) {
  const input = useRef<HTMLInputElement | null>(null)
  const [uploading, setUploading] = useState(false)
  const [problem, setProblem] = useState<string | null>(null)

  async function handleFile(file: File) {
    setProblem(null)

    // A courtesy check so an obviously wrong file fails instantly rather than after a 10 MB
    // upload. The server still decides — it re-checks by decoding the bytes (FR-009).
    const obvious = describeObviousProblem(file)
    if (obvious) {
      setProblem(obvious)
      if (input.current) input.current.value = ''
      return
    }

    setUploading(true)
    try {
      onChange(await uploadImage(file))
    } catch (err) {
      // The server's message names the limit or the accepted formats; show that rather than
      // inventing wording of our own (FR-009, FR-010).
      setProblem(err instanceof ApiError ? err.message : 'That image could not be uploaded.')
      onChange(null)
      if (input.current) input.current.value = ''
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="space-y-2">
      <span className="block text-sm font-medium text-ink">Photograph</span>

      <div className="flex items-start gap-4">
        <div className="h-28 w-[5.6rem] shrink-0 overflow-hidden rounded-lg border border-edge bg-surface-raised">
          {image ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={image.renditionUrl}
              alt="The photograph you chose"
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-xs text-ink-faint">
              None
            </div>
          )}
        </div>

        <div className="space-y-2">
          <input
            ref={input}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="sr-only"
            id="collectible-image"
            disabled={uploading}
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) void handleFile(file)
            }}
          />
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={uploading}
              onClick={() => input.current?.click()}
            >
              {uploading ? 'Uploading…' : image ? 'Choose a different photo' : 'Choose a photo'}
            </Button>
            {image && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={uploading}
                onClick={() => {
                  onChange(null)
                  setProblem(null)
                  if (input.current) input.current.value = ''
                }}
              >
                Remove
              </Button>
            )}
          </div>
          <p className="text-xs text-ink-faint">
            Optional. JPEG, PNG, or WebP, up to 10 MB.
          </p>
        </div>
      </div>

      {problem && (
        <p role="alert" data-testid="image-problem" className="flex items-start gap-1.5 text-xs text-danger">
          <span aria-hidden="true">!</span>
          <span>
            {problem} You can choose another photo, or save this collectible without one.
          </span>
        </p>
      )}
    </div>
  )
}
