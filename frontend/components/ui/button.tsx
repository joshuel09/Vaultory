import { cva, type VariantProps } from 'class-variance-authority'
import { forwardRef } from 'react'
import { cn } from '@/lib/cn'

/**
 * Vaultory's button. Built on the shadcn/ui pattern but restyled to Vaultory's own identity rather
 * than left at the library's defaults (Technology Constraints).
 */
export const buttonClasses = cva(
  'inline-flex items-center justify-center gap-2 rounded-lg text-sm font-medium ' +
    'transition-colors disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        primary: 'bg-accent text-accent-ink hover:bg-accent/90',
        secondary: 'bg-surface-raised text-ink border border-edge hover:border-ink-faint',
        ghost: 'text-ink-muted hover:text-ink hover:bg-surface-raised',
      },
      size: {
        md: 'h-10 px-4',
        sm: 'h-8 px-3 text-xs',
        lg: 'h-12 px-6 text-base',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonClasses> {}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => (
    <button ref={ref} className={cn(buttonClasses({ variant, size }), className)} {...props} />
  ),
)
Button.displayName = 'Button'
