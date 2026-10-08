import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {git,stock,write} from './capture.mjs';
const paths=git(['ls-files','tools/gomad3']).toString().trim().split('\n').filter(p=>p.endsWith('.go')),argv=[stock+'/gofmt','-l',...paths],r=spawnSync(argv[0],argv.slice(1),{encoding:'utf8',timeout:600000});write('fmt-check.json',{argv,exit:r.status,stdout:r.stdout,stderr:r.stderr,files:paths.length,mutating:false});assert.equal(r.status,0);assert.equal(r.stdout,'');console.log('gofmt -l: '+paths.length+' tracked nested-module Go files, no differences.');
