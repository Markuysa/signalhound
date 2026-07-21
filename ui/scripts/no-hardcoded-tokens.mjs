/* Fails the lint if a raw hex colour appears anywhere in src/. Colours live once in
   tokens.css and are referenced through Tailwind — a hex literal in a component is the
   defect CLAUDE.md calls out, so this is a gate, not a nit. tokens.css itself is the one
   place hex is allowed. */
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

const HEX = /#[0-9a-fA-F]{3,8}\b/;
const ALLOW = new Set(['src/styles/tokens.css']);
const bad = [];

function walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      walk(p);
      continue;
    }
    if (!/\.(ts|tsx|css)$/.test(name)) continue;
    const rel = p.replace(/^.*?(src\/.*)$/, '$1');
    if (ALLOW.has(rel)) continue;
    readFileSync(p, 'utf8')
      .split('\n')
      .forEach((line, i) => {
        if (HEX.test(line)) bad.push(`${rel}:${i + 1}  ${line.trim()}`);
      });
  }
}

walk('src');
if (bad.length) {
  console.error('Hardcoded colour(s) found — use a token from tokens.css:\n' + bad.join('\n'));
  process.exit(1);
}
console.log('no-hardcoded-tokens: clean');
