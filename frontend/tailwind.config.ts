import type { Config } from 'tailwindcss'

/**
 * Vaultory's visual language.
 *
 * Colours are declared as CSS custom properties in globals.css and referenced here, so a single
 * token set drives both the dark and the light appearance (FR-046). Nothing in a component names
 * a raw colour.
 */
const config: Config = {
  content: ['./app/**/*.{ts,tsx}', './components/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        canvas: 'rgb(var(--canvas) / <alpha-value>)',
        surface: 'rgb(var(--surface) / <alpha-value>)',
        'surface-raised': 'rgb(var(--surface-raised) / <alpha-value>)',
        edge: 'rgb(var(--edge) / <alpha-value>)',
        ink: 'rgb(var(--ink) / <alpha-value>)',
        'ink-muted': 'rgb(var(--ink-muted) / <alpha-value>)',
        'ink-faint': 'rgb(var(--ink-faint) / <alpha-value>)',
        accent: 'rgb(var(--accent) / <alpha-value>)',
        'accent-ink': 'rgb(var(--accent-ink) / <alpha-value>)',
        danger: 'rgb(var(--danger) / <alpha-value>)',
        'danger-ink': 'rgb(var(--danger-ink) / <alpha-value>)',
        success: 'rgb(var(--success) / <alpha-value>)',
      },
      fontFamily: {
        sans: ['var(--font-sans)', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        card: '0.875rem',
      },
      boxShadow: {
        // A lifted card, not a drop shadow. Subtle and purposeful, per Principle I.
        card: '0 1px 2px rgb(0 0 0 / 0.28), 0 8px 24px -12px rgb(0 0 0 / 0.45)',
        'card-hover': '0 2px 4px rgb(0 0 0 / 0.3), 0 16px 40px -16px rgb(0 0 0 / 0.55)',
      },
      aspectRatio: {
        // The gallery frame. Matches imaging.RenditionWidth x RenditionHeight exactly, so the
        // card and the rendition cannot drift apart (FR-014).
        collectible: '4 / 5',
      },
    },
  },
  plugins: [],
}

export default config
