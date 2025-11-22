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

    trailingSlash: 'ignore',
    integrations: [
        starlight({
            title: 'Ferret',
            description: 'A modern, type-safe programming language',
            favicon: '/favicon.png',  // Change to '/favicon.png' or '/favicon.ico' if using different format
            disable404Route: true, // Use custom 404 page instead of Starlight's
            lastUpdated: true,
            editLink: {
                baseUrl: 'https://github.com/itsfuad/FCCIMPL/edit/main/website/src/content/docs/',
            },
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
            // Use a single theme for both code snippets and playground for visual consistency.
            expressiveCode: {
                themes: ['one-dark-pro'], // Change this to your preferred built-in Shiki theme
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
                    label: 'Basics',
                    items: [
                        { label: 'Variables & Constants', slug: 'language/basics/variables' },
                        { label: 'Data Types', slug: 'language/basics/types' },
                        { label: 'Operators', slug: 'language/basics/operators' },
                        { label: 'Comments', slug: 'language/basics/comments' },
                    ],
                },
                {
                    label: 'Control Flow',
                    items: [
                        { label: 'If Statements', slug: 'language/control-flow/if-statements' },
                        { label: 'Loops', slug: 'language/control-flow/loops' },
                        { label: 'When Expressions', slug: 'language/control-flow/when' },
                    ],
                },
                {
                    label: 'Functions',
                    items: [
                        { label: 'Function Basics', slug: 'language/functions/functions' },
                        { label: 'Parameters & Returns', slug: 'language/functions/parameters' },
                    ],
                },
                {
                    label: 'Type System',
                    items: [
                        { label: 'Structs', slug: 'language/type-system/structs' },
                        { label: 'Enums', slug: 'language/type-system/enums' },
                        { label: 'Interfaces', slug: 'language/type-system/interfaces' },
                        { label: 'Optional Types', slug: 'language/type-system/optionals' },
                    ],
                },
                {
                    label: 'Advanced',
                    items: [
                        { label: 'Error Handling', slug: 'language/advanced/errors' },
                        { label: 'Generics', slug: 'language/advanced/generics' },
                    ],
                },
            ],
        }),
    ],

    vite: {
        plugins: [],
    },
});