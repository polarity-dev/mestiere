import jsEslint from "@eslint/js"
import stylistic from "@stylistic/eslint-plugin"
import astro from "eslint-plugin-astro"
import { defineConfig, globalIgnores } from "eslint/config"
import globals from "globals"
import { fileURLToPath } from "node:url"
import tsEslint from "typescript-eslint"

export default defineConfig(
  jsEslint.configs.recommended,
  tsEslint.configs.strictTypeChecked,
  astro.configs["flat/recommended"],
  globalIgnores([
    "**/node_modules/",
    "dist/",
    "lavagna/dist/",
    ".astro/",
  ]),
  {
    plugins: {
      "@stylistic": stylistic,
    },
    languageOptions: {
      parserOptions: {
        tsconfigRootDir: fileURLToPath(new URL(".", import.meta.url)),
        projectService: true,
        extraFileExtensions: [".astro"],
      },
      globals: {
        ...globals.node,
      },
    },
    rules: {
      curly: ["error"],
      eqeqeq: ["warn", "always"],
      "no-empty": ["error", { allowEmptyCatch: true }],
      "no-undef": ["error"],
      "object-shorthand": "error",
      "prefer-const": "error",
      "@typescript-eslint/await-thenable": "error",
      "@typescript-eslint/ban-ts-comment": [
        "error",
        {
          "ts-expect-error": "allow-with-description",
        },
      ],
      "@typescript-eslint/class-literal-property-style": "error",
      "@typescript-eslint/consistent-type-assertions": "error",
      "@typescript-eslint/no-confusing-void-expression": [
        "error",
        {
          ignoreArrowShorthand: true,
        },
      ],
      "@typescript-eslint/no-misused-promises": [
        "error",
        {
          checksVoidReturn: false,
        },
      ],
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
        },
      ],
      "@typescript-eslint/only-throw-error": "off",
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/restrict-template-expressions": [
        "error",
        {
          allowNumber: true,
          allowBoolean: true,
        },
      ],
      "@typescript-eslint/no-deprecated": "error",
      "@stylistic/list-style": "error",
      "@stylistic/arrow-parens": ["error", "as-needed"],
      "@stylistic/arrow-spacing": [
        "error",
        {
          before: true,
          after: true,
        },
      ],
      "@stylistic/brace-style": [
        "error",
        "1tbs",
        {
          allowSingleLine: false,
        },
      ],
      "@stylistic/comma-dangle": ["error", "always-multiline"],
      "@stylistic/comma-spacing": [
        "error",
        {
          before: false,
          after: true,
        },
      ],
      "@stylistic/computed-property-spacing": ["error", "never"],
      "@stylistic/dot-location": ["error", "property"],
      "@stylistic/eol-last": ["error", "always"],
      "@stylistic/function-call-spacing": ["error", "never"],
      "@stylistic/indent": ["error", 2],
      "@stylistic/key-spacing": [
        "error",
        {
          beforeColon: false,
          afterColon: true,
          mode: "strict",
        },
      ],
      "@stylistic/keyword-spacing": [
        "error",
        {
          before: true,
          after: true,
        },
      ],
      "@stylistic/linebreak-style": ["error", "unix"],
      "@stylistic/new-parens": ["error", "always"],
      "@stylistic/no-floating-decimal": "error",
      "@stylistic/no-multi-spaces": "error",
      "@stylistic/no-multiple-empty-lines": ["error", { max: 1 }],
      "@stylistic/no-trailing-spaces": "error",
      "@stylistic/no-whitespace-before-property": "error",
      "@stylistic/padded-blocks": ["error", "never"],
      "@stylistic/quote-props": ["error", "as-needed"],
      "@stylistic/quotes": [
        "error",
        "double",
        {
          avoidEscape: true,
        },
      ],
      "@stylistic/rest-spread-spacing": ["error", "never"],
      "@stylistic/semi": ["error", "never"],
      "@stylistic/space-before-blocks": ["error", "always"],
      "@stylistic/space-before-function-paren": [
        "error",
        {
          anonymous: "never",
          named: "never",
          asyncArrow: "never",
          catch: "always",
        },
      ],
      "@stylistic/space-in-parens": ["error", "never"],
      "@stylistic/space-infix-ops": "error",
      "@stylistic/space-unary-ops": [
        "error",
        {
          words: true,
          nonwords: false,
        },
      ],
      "@stylistic/spaced-comment": [
        "error",
        "always",
        {
          markers: ["/"],
        },
      ],
      "@stylistic/switch-colon-spacing": [
        "error",
        {
          after: true,
          before: false,
        },
      ],
      "@stylistic/template-curly-spacing": ["error", "never"],
      "@stylistic/template-tag-spacing": ["error", "never"],
      "@stylistic/wrap-iife": ["error", "inside"],
      "@stylistic/member-delimiter-style": [
        "error",
        {
          multiline: {
            delimiter: "none",
            requireLast: false,
          },
          singleline: {
            delimiter: "comma",
            requireLast: false,
          },
        },
      ],
      "@stylistic/type-annotation-spacing": "error",
    },
  },
  {
    files: ["src/scripts/**/*.ts", "public/scripts/**/*.js", "**/*.astro/*.ts", "lavagna/layout.ts"],
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  },
  {
    files: ["**/*.astro"],
    languageOptions: {
      parserOptions: { projectService: false, project: "./tsconfig.json" },
    },
    rules: {
      "@typescript-eslint/no-unsafe-return": "off",
    },
  },
  {
    files: ["public/scripts/**/*.js"],
    extends: [tsEslint.configs.disableTypeChecked],
  },
)
