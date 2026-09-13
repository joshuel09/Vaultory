import type { ErrorResponse, FieldError } from './types'

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

  constructor(status: number, code: string, message: string, fields: FieldError[] = []) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
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
  let body: ErrorResponse | undefined
  try {
    body = (await response.json()) as ErrorResponse
  } catch {
    // A proxy or a crash can produce a non-JSON failure. Say something useful anyway.
  }
  const error = body?.error
  return new ApiError(
    response.status,
    error?.code ?? 'unknown_error',
    error?.message ?? 'Something went wrong. Please try again.',
    error?.fields ?? [],
  )
}
