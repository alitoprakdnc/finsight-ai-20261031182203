module.exports = {
  root: true,
  env: {
    browser: true,
    es2022: true,
    node: true,
  },
  parser: '@typescript-eslint/parser',
  parserOptions: {
    project: ['./frontend/tsconfig.json'],
    tsconfigRootDir: __dirname,
    ecmaVersion: 2022,
    sourceType: 'module',
    ecmaFeatures: {
      jsx: true,
    },
  },
  plugins: [
    '@typescript-eslint',
    'react',
    'react-hooks',
    'jsx-a11y',
    'import',
    'security',
    'prettier',
  ],
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
    'plugin:jsx-a11y/recommended',
    'plugin:import/errors',
    'plugin:import/warnings',
    'plugin:import/typescript',
    'plugin:security/recommended',
    'plugin:prettier/recommended',
    'next/core-web-vitals',
  ],
  settings: {
    react: {
      version: 'detect',
    },
    'import/resolver': {
      node: {
        extensions: ['.js', '.jsx', '.ts', '.tsx'],
      },
      typescript: {
        project: './frontend/tsconfig.json',
      },
    },
  },
  ignorePatterns: ['node_modules/', '.next/', 'dist/', 'build/', 'public/'],
  rules: {
    // General JavaScript/TypeScript rules
    'no-console': ['warn', { allow: ['error', 'warn'] }],
    'no-debugger': 'error',
    'prefer-const': 'error',
    'no-var': 'error',
    'object-shorthand': ['error', 'always'],
    'arrow-body-style': ['error', 'as-needed'],
    'no-unused-vars': 'off', // handled by @typescript-eslint
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_' },
    ],
    '@typescript-eslint/explicit-function-return-type': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/no-floating-promises': 'error',
    '@typescript-eslint/strict-boolean-expressions': 'error',
    '@typescript-eslint/prefer-nullish-coalescing': 'error',
    '@typescript-eslint/prefer-optional-chain': 'error',
    '@typescript-eslint/no-misused-promises': [
      'error',
      { checksVoidReturn: true },
    ],

    // React specific
    'react/react-in-jsx-scope': 'off', // Next.js auto‑imports React
    'react/prop-types': 'off',
    'react/jsx-boolean-value': ['error', 'never'],
    'react/jsx-curly-brace-presence': [
      'error',
      { props: 'never', children: 'never' },
    ],
    'react/jsx-no-useless-fragment': 'error',
    'react/jsx-key': 'error',
    'react/self-closing-comp': 'error',

    // Hooks
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',

    // Accessibility
    'jsx-a11y/anchor-is-valid': 'off',
    'jsx-a11y/no-autofocus': 'error',
    'jsx-a11y/label-has-associated-control': [
      'error',
      { assert: 'either' },
    ],

    // Import ordering
    'import/order': [
      'error',
      {
        groups: [
          'builtin',
          'external',
          'internal',
          ['parent', 'sibling'],
          'index',
          'object',
          'type',
        ],
        pathGroups: [
          {
            pattern: 'react',
            group: 'external',
            position: 'before',
          },
          {
            pattern: '@/**',
            group: 'internal',
            position: 'after',
          },
        ],
        pathGroupsExcludedImportTypes: ['react'],
        alphabetize: { order: 'asc', caseInsensitive: true },
        'newlines-between': 'always',
      },
    ],

    // Security
    'security/detect-object-injection': 'off', // false positives with DI patterns

    // Prettier integration
    'prettier/prettier': [
      'error',
      {
        endOfLine: 'auto',
        singleQuote: true,
        trailingComma: 'all',
        semi: true,
        tabWidth: 2,
        useTabs: false,
      },
    ],
  },
  overrides: [
    {
      files: ['**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}'],
      env: { jest: true },
      plugins: ['jest'],
      extends: ['plugin:jest/recommended'],
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
      },
    },
    {
      files: ['cypress/**/*.ts', 'cypress/**/*.tsx'],
      env: { 'cypress/globals': true },
      plugins: ['cypress'],
      extends: ['plugin:cypress/recommended'],
      rules: {
        'no-console': 'off',
      },
    },
  ],
};