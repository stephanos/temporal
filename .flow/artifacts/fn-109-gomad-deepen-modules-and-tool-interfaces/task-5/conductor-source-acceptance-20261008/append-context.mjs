import { readFileSync } from 'node:fs';
const text = readFileSync(0, 'utf8');
const heading = '## Description\n', start = text.indexOf(heading) + heading.length, end = text.indexOf('\n## Acceptance', start);
if (start < heading.length || end < start) throw new Error('Missing task sections');
if (text.includes('### Current conductor review context')) throw new Error('Context already appended');
const context = readFileSync('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-5/conductor-source-acceptance-20261008/review-context.md', 'utf8');
process.stdout.write(text.slice(start, end).trimEnd() + '\n\n### Current conductor review context (2026-10-08)\n\n' + context.replace(/^# Current task5 source-acceptance review\n\n/, '') + '\n');
