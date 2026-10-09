import fs from 'node:fs';
import path from 'node:path';
import {repo, base, git, hash} from './run.mjs';

const paths = git('ls-tree', '-r', '--name-only', base, '--', 'tools/gomad3', 'tools/gomad3integration').trim().split('\n');
const inventory = paths.map(relative => ({path: relative, sha256: hash(fs.readFileSync(path.join(repo, relative)))}));
const inputPattern = /\/(schema|toolchain\/version|internal\/gomadtool\/generation|deterministicio\/boundary|target\/internal\/livecap|internal\/compatibilitypack\/(requests|authoring))\/|\/(choice\/(trace|tape)\.go|toolchain\/runtime\/|qualification\/tests\.generator\.json)|\/cmd\/gomadtool\/(version|protocol|boundary|qualification_manifest)\.go$|\/Makefile$|\/internal\/compatibilitypack\/(working-directories\.json|schema\.go|v2_selection\.go)$/;
const inputs = inventory.filter(entry => inputPattern.test(entry.path));
const outputs = inventory.filter(entry => /generated|\/qualification\/tests\.json$|\/internal\/compatibilitypack\/(packs|reports)\/|\/internal\/compatibilitypack\/generation\.json$|\/boundary\/(compiler-spec|expected-intercepts|report|upgrade)-/.test(entry.path));
const changedInputs = inputs.filter(entry => entry.sha256 !== hash(git('show', base + ':' + entry.path))));
if (changedInputs.length !== 0) throw Error('generator input differs before production edit');
console.log(JSON.stringify({base, selectors: {inputs: inputPattern.source, outputs: 'generated consumers, qualification/tests.json, packs/reports/generation and boundary compiler/report/upgrade consumers'}, generator_inputs: inputs, generated_outputs: outputs, actual_generation: false, mode: 'pre-edit inventory; all paths selected from baseline git ls-tree'}));
for (const relative of ['.turbo/plans/gomad3-glossary-update.md', '.turbo/technical-debt.md']) console.log(JSON.stringify({user_untracked: relative, sha256: hash(fs.readFileSync(path.join(repo, relative)))}));
