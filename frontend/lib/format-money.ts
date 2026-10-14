/**
 * Render an exact decimal amount as a collector reads money.
 *
 * The value arrives as a string and stays one. It is grouped with separators for legibility and
 * never parsed into a number: `Number("0.10")` is the beginning of exactly the floating-point
 * problem the whole backend is built to avoid, and doing it in the last step would undo all of it
 * (FR-033, Principle IV).
 */
export function formatMoney(exact: string): string {
  const [whole = '0', fraction = '00'] = exact.split('.')
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return `${grouped}.${fraction.padEnd(2, '0')}`
}
