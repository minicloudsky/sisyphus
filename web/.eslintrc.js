module.exports = {
  extends: [require.resolve('@umijs/lint/dist/config/eslint')],
  rules: {
    '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    'react-hooks/exhaustive-deps': 'warn',
    'no-console': 'off',
  },
};
