/* Lightweight starter; we'll switch to flat config + typescript-eslint later. */
module.exports = {
  root: true,
  env: { browser: true, es2022: true, node: true },
  extends: ['eslint:recommended'],
  ignorePatterns: ['dist', 'node_modules'],
  parserOptions: { ecmaVersion: 'latest', sourceType: 'module' },
}
