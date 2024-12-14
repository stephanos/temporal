#!/usr/bin/env bash

UPDATE_ID=6c7841ad-8a12-4c6d-bf1e-82386abad415
WF_ID=workflowupdate-1:0193c24e-a937-7e15-a9d6-71339b94ece1-0-0-height-0-child-aff580e0-9db0-4fc6-975c-6333206d9c47
RUN_ID=0193c24e-b430-76ed-ab38-73004187925e

cat cluster_a.log | grep ${UPDATE_ID} | jq > ${UPDATE_ID}_logs_a.json
cat cluster_b.log | grep ${UPDATE_ID} | jq > ${UPDATE_ID}_logs_b.json

go run ./cmd/tools/tdbg/main.go --address localhost:7233 \
  w describe --wid ${WF_ID} --rid ${RUN_ID} > ${UPDATE_ID}_state_a.json
go run ./cmd/tools/tdbg/main.go --address localhost:8233 \
  w describe --wid ${WF_ID} --rid ${RUN_ID} > ${UPDATE_ID}_state_b.json

../cli/temporal --address localhost:7233 workflow show \
  --workflow-id ${WF_ID} --output json > ${UPDATE_ID}_events_a.json
../cli/temporal --address localhost:8233 workflow show \
  --workflow-id ${WF_ID} --output json > ${UPDATE_ID}_events_b.json
