import {readFileSync} from 'node:fs';
const task='.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.4.md';
const text=readFileSync(task,'utf8'),start=text.indexOf('## Description\n')+'## Description\n'.length,end=text.indexOf('\n## Acceptance',start);
if(start<'## Description\n'.length||end<start)throw new Error('Missing task sections');
const context=readFileSync('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-4/conductor-source-acceptance-20261008/review-context.md','utf8');
if(text.includes('### Current conductor review context'))throw new Error('Context already appended');
process.stdout.write(text.slice(start,end).trimEnd()+'\n\n### Current conductor review context (2026-10-08)\n\n'+context.replace(/^# Current task4 source-acceptance review\n\n/,'')+'\n');
