import next from 'eslint-config-next'

export default [
  { ignores: ['node_modules/**', '.next/**', 'out/**', 'coverage/**', 'lib/types/api.ts'] },
  ...next(),
  {
    rules: {
      // Principle III: `any` SHOULD NOT be used unless there is a documented technical reason.
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unsafe-assignment': 'off',
    },
  },
]
