#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const modelsPath = path.join(__dirname, 'wailsjs/go/models.ts');

if (!fs.existsSync(modelsPath)) {
  console.log('models.ts not found, skipping fix');
  process.exit(0);
}

let content = fs.readFileSync(modelsPath, 'utf-8');

content = content.replace(/Record<string,\s*number>/g, '{[key: string]: number}');
content = content.replace(/Record<string,\s*string>/g, '{[key: string]: string}');
content = content.replace(/Record<string,\s*any>/g, '{[key: string]: any}');

fs.writeFileSync(modelsPath, content, 'utf-8');
console.log('Fixed TypeScript type conflicts in models.ts');
