import fs from 'node:fs';
import { fileURLToPath } from 'node:url';

// Load the Ferret TextMate grammar
const grammar = JSON.parse(
  fs.readFileSync(
    fileURLToPath(new URL('./syntax/fer.tmLanguage.json', import.meta.url)),
    'utf-8'
  )
);

// Export as a proper Shiki language with the grammar spread
export const ferretLang = {
  name: 'ferret',  // Language ID used in code blocks
  ...grammar       // Spread the entire grammar (includes scopeName, patterns, repository, etc.)
};
