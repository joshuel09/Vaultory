import type { Collectible, ErrorResponse, FieldError, VersionConflictResponse } from './types'

/**
 * A failed request, carrying whatever the server said about it.
 *
 * Field errors are surfaced as-is: the server decides what is wrong and how to say it, and the
 * frontend renders that verdict rather than forming its own (Constitution Principle II).
 */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields: FieldError[]
  /**
   * The collectible as it now stands, present only on a version conflict.
   *
   * Carried in the 409 body so a refused edit can show the collector what the collectible actually
   * says without a second request, which would open a window in which it changes again (FR-027).
   */
  readonly current?: Collectible

  constructor(
    status: number,
    code: string,
    message: string,
    fields: FieldError[] = [],
    current?: Collectible,
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
    this.current = current
  }

  /**
   * True when somebody changed this collectible after the collector opened it (FR-027).
   *
   * Not a validation failure and not something retrying fixes: the collector has to see what it
   * now says and decide again.
   */
  get isVersionConflict(): boolean {
    return this.status === 409 && this.code === 'version_conflict'
  }

  /** True when the collector's session could not be established (FR-029). */
  get isUnauthenticated(): boolean {
    return this.status === 401
  }

  /** Field errors keyed by field name, for rendering against inputs. */
  byField(): Record<string, string> {
    const out: Record<string, string> = {}
    for (const f of this.fields) {
      out[f.field] = f.message
    }
    return out
  }
}

/** Build an ApiError from a failing response, falling back when the body is not our envelope. */
export async function toApiError(response: Response): Promise<ApiError> {
  let body: (ErrorResponse & Partial<VersionConflictResponse>) | undefined
  try {
    body = (await response.json()) as ErrorResponse & Partial<VersionConflictResponse>
  } catch {
    // A proxy or a crash can produce a non-JSON failure. Say something useful anyway.
  }
  const error = body?.error
  return new ApiError(
    response.status,
    error?.code ?? 'unknown_error',
    error?.message ?? 'Something went wrong. Please try again.',
    error?.fields ?? [],
    body?.current,
  )
}
