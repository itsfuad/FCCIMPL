// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import fs from 'node:fs';

// Load Ferret TextMate grammar
const ferretGrammar = JSON.parse(fs.readFileSync('./src/syntax/fer.tmLanguage.json', 'utf-8'));
// Override the name to match what we use in code blocks
ferretGrammar.name = 'ferret';

// https://astro.build/config
export default defineConfig({
  output: 'static',
  experimental: {
    clientPrerender: true,
  },
  integrations: [
      starlight({
          title: 'Ferret',
          description: 'A modern, type-safe programming language',
          favicon: '/favicon.png',  // Change to '/favicon.png' or '/favicon.ico' if using different format
          social: [
              { icon: 'github', label: 'GitHub', href: 'https://github.com/itsfuad/FCCIMPL' }
          ],
          customCss: [
              './src/styles/custom.css',
          ],
          defaultLocale: 'root',
          locales: {
              root: {
                  label: 'English',
                  lang: 'en',
              },
          },
          expressiveCode: {
              themes: ['github-dark', 'github-light', 'ayu-dark', 'one-dark-pro'],
              shiki: {
                  langs: [ferretGrammar],
              },
          },
          components: {
              // Override the Head component to add View Transitions
              Head: './src/components/Head.astro',
              // Modern shadcn-style theme toggle
              ThemeSelect: './src/components/ThemeSelect.astro',
              // Use ferret image instead of text
              SiteTitle: './src/components/SiteTitle.astro',
              // Use custom navbar for all pages
              Header: './src/components/Header.astro',
          },

          sidebar: [
              {
                  label: 'Getting Started',
                  items: [
                      { label: 'Introduction', slug: 'getting-started/introduction' },
                      { label: 'Installation', slug: 'getting-started/installation' },
                      { label: 'Hello World', slug: 'getting-started/hello-world' },
                  ],
              },
              {
                  label: 'Playground',
                  link: '/playground/',
              },
              {
                  label: 'Basics',
                  items: [
                      { label: 'Variables & Constants', slug: 'language/variables' },
                      { label: 'Data Types', slug: 'language/types' },
                      { label: 'Operators', slug: 'language/operators' },
                      { label: 'Comments', slug: 'language/comments' },
                  ],
              },
              {
                  label: 'Control Flow',
                  items: [
                      { label: 'If Statements', slug: 'language/if-statements' },
                      { label: 'Loops', slug: 'language/loops' },
                      { label: 'When Expressions', slug: 'language/when' },
                  ],
              },
              {
                  label: 'Functions',
                  items: [
                      { label: 'Function Basics', slug: 'language/functions' },
                      { label: 'Parameters & Returns', slug: 'language/parameters' },
                  ],
              },
              {
                  label: 'Type System',
                  items: [
                      { label: 'Structs', slug: 'language/structs' },
                      { label: 'Enums', slug: 'language/enums' },
                      { label: 'Interfaces', slug: 'language/interfaces' },
                      { label: 'Optional Types', slug: 'language/optionals' },
                  ],
              },
              {
                  label: 'Advanced',
                  items: [
                      { label: 'Error Handling', slug: 'language/errors' },
                      { label: 'Generics', slug: 'language/generics' },
                  ],
              },
          ],
      }),
	],

  vite: {
    plugins: [],
  },
});