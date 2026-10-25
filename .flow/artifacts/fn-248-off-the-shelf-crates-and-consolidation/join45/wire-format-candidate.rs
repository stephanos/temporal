use ::umpire_catalog::{WireField, WireKind, WireMutation, WireValue, GenerationError};
/// Raw request fields, including descriptor fields ignored by semantic catalogs.
pub fn workflow_request_fields(method: WorkflowServiceMethod) -> Vec<WireField> {
    match method {
        WorkflowServiceMethod::StartActivityExecution => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.run_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.run_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_input"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_input"
                .into(), kind : WireKind::Bool, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_outcome"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_outcome"
                .into(), kind : WireKind::Bool, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.long_poll_token"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.long_poll_token"
                .into(), kind : WireKind::Bytes, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_heartbeat_details"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_heartbeat_details"
                .into(), kind : WireKind::Bool, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_last_failure"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_last_failure"
                .into(), kind : WireKind::Bool, optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::DescribeActivityExecution => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.run_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.run_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_input"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_input"
                .into(), kind : WireKind::Bool, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_outcome"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_outcome"
                .into(), kind : WireKind::Bool, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.long_poll_token"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.long_poll_token"
                .into(), kind : WireKind::Bytes, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_heartbeat_details"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_heartbeat_details"
                .into(), kind : WireKind::Bool, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_last_failure"
                .into(), declaration :
                "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_last_failure"
                .into(), kind : WireKind::Bool, optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::PollActivityExecution => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityExecutionRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityExecutionRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityExecutionRequest.activity_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityExecutionRequest.activity_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityExecutionRequest.run_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityExecutionRequest.run_id"
                .into(), kind : WireKind::String, optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::RequestCancelActivityExecution => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.activity_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.activity_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.run_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.run_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.identity"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.identity"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.request_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.request_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.reason"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.reason"
                .into(), kind : WireKind::String, optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::PollActivityTaskQueue => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue.name"
                .into(), declaration : "temporal.api.taskqueue.v1.TaskQueue.name".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue.kind"
                .into(), declaration : "temporal.api.taskqueue.v1.TaskQueue.kind".into(),
                kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32, 3i32] },
                optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue.normal_name"
                .into(), declaration : "temporal.api.taskqueue.v1.TaskQueue.normal_name"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.poller_group_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.poller_group_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.identity"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.identity"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_instance_key"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_instance_key"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_control_task_queue"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_control_task_queue"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata.max_tasks_per_second"
                .into(), declaration :
                "temporal.api.taskqueue.v1.TaskQueueMetadata.max_tasks_per_second"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata.max_tasks_per_second.value"
                .into(), declaration : "google.protobuf.DoubleValue.value".into(), kind :
                WireKind::Float, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities.build_id"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionCapabilities.build_id".into(), kind
                : WireKind::String, optional : false, refusal : None }, WireField { path
                :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities.use_versioning"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionCapabilities.use_versioning".into(),
                kind : WireKind::Bool, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities.deployment_series_name"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionCapabilities.deployment_series_name"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options"
                .into(), declaration :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options.deployment_name"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.deployment_name"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options.build_id"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.build_id".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options.worker_versioning_mode"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.worker_versioning_mode"
                .into(), kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32] },
                optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::RespondActivityTaskCompleted => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.task_token"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.task_token"
                .into(), kind : WireKind::Bytes, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.metadata.key"
                .into(), declaration : "temporal.api.common.v1.Payload.MetadataEntry.key"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.metadata.value"
                .into(), declaration :
                "temporal.api.common.v1.Payload.MetadataEntry.value".into(), kind :
                WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.external_payloads.size_bytes"
                .into(), declaration :
                "temporal.api.common.v1.Payload.ExternalPayloadDetails.size_bytes"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.identity"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.identity"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.resource_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.resource_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version.build_id"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionStamp.build_id".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version.use_versioning"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionStamp.use_versioning".into(), kind :
                WireKind::Bool, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment.series_name"
                .into(), declaration :
                "temporal.api.deployment.v1.Deployment.series_name".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment.build_id"
                .into(), declaration : "temporal.api.deployment.v1.Deployment.build_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options.deployment_name"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.deployment_name"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options.build_id"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.build_id".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options.worker_versioning_mode"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.worker_versioning_mode"
                .into(), kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32] },
                optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::RespondActivityTaskFailed => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.task_token"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.task_token"
                .into(), kind : WireKind::Bytes, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.message"
                .into(), declaration : "temporal.api.failure.v1.Failure.message".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.source"
                .into(), declaration : "temporal.api.failure.v1.Failure.source".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.stack_trace"
                .into(), declaration : "temporal.api.failure.v1.Failure.stack_trace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.encoded_attributes".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.metadata.key"
                .into(), declaration : "temporal.api.common.v1.Payload.MetadataEntry.key"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.metadata.value"
                .into(), declaration :
                "temporal.api.common.v1.Payload.MetadataEntry.value".into(), kind :
                WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.external_payloads.size_bytes"
                .into(), declaration :
                "temporal.api.common.v1.Payload.ExternalPayloadDetails.size_bytes"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause"
                .into(), declaration : "temporal.api.failure.v1.Failure.cause".into(),
                kind : WireKind::Message, optional : true, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.message"
                .into(), declaration : "temporal.api.failure.v1.Failure.message".into(),
                kind : WireKind::String, optional : false, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.source"
                .into(), declaration : "temporal.api.failure.v1.Failure.source".into(),
                kind : WireKind::String, optional : false, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.stack_trace"
                .into(), declaration : "temporal.api.failure.v1.Failure.stack_trace"
                .into(), kind : WireKind::String, optional : false, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.encoded_attributes"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.encoded_attributes".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.cause"
                .into(), declaration : "temporal.api.failure.v1.Failure.cause".into(),
                kind : WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.application_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.application_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.timeout_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.timeout_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.canceled_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.canceled_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.terminated_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.terminated_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.server_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.server_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.reset_workflow_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.reset_workflow_failure_info".into(),
                kind : WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.activity_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.activity_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.child_workflow_execution_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.child_workflow_execution_failure_info"
                .into(), kind : WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.nexus_operation_execution_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.nexus_operation_execution_failure_info"
                .into(), kind : WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause.nexus_handler_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.nexus_handler_failure_info".into(), kind
                : WireKind::Message, optional : true, refusal :
                Some("recursive descriptor edge; deeper fields were not expanded".into())
                }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.application_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.type"
                .into(), declaration :
                "temporal.api.failure.v1.ApplicationFailureInfo.type".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.non_retryable"
                .into(), declaration :
                "temporal.api.failure.v1.ApplicationFailureInfo.non_retryable".into(),
                kind : WireKind::Bool, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details"
                .into(), declaration :
                "temporal.api.failure.v1.ApplicationFailureInfo.details".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.next_retry_delay"
                .into(), declaration :
                "temporal.api.failure.v1.ApplicationFailureInfo.next_retry_delay".into(),
                kind : WireKind::Message, optional : true, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.next_retry_delay.seconds"
                .into(), declaration : "google.protobuf.Duration.seconds".into(), kind :
                WireKind::Signed, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.next_retry_delay.nanos"
                .into(), declaration : "google.protobuf.Duration.nanos".into(), kind :
                WireKind::Signed, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.category"
                .into(), declaration :
                "temporal.api.failure.v1.ApplicationFailureInfo.category".into(), kind :
                WireKind::Enum { defined : vec![0i32, 1i32] }, optional : false, refusal
                : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.timeout_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.timeout_type"
                .into(), declaration :
                "temporal.api.failure.v1.TimeoutFailureInfo.timeout_type".into(), kind :
                WireKind::Enum { defined : vec![0i32, 1i32, 2i32, 3i32, 4i32] }, optional
                : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details"
                .into(), declaration :
                "temporal.api.failure.v1.TimeoutFailureInfo.last_heartbeat_details"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.canceled_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details"
                .into(), declaration :
                "temporal.api.failure.v1.CanceledFailureInfo.details".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.identity"
                .into(), declaration :
                "temporal.api.failure.v1.CanceledFailureInfo.identity".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.terminated_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.terminated_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.terminated_failure_info.identity"
                .into(), declaration :
                "temporal.api.failure.v1.TerminatedFailureInfo.identity".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.server_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.server_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.server_failure_info.non_retryable"
                .into(), declaration :
                "temporal.api.failure.v1.ServerFailureInfo.non_retryable".into(), kind :
                WireKind::Bool, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.reset_workflow_failure_info".into(),
                kind : WireKind::Message, optional : true, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details"
                .into(), declaration :
                "temporal.api.failure.v1.ResetWorkflowFailureInfo.last_heartbeat_details"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal :
                Some("descriptor depth limit reached; deeper fields were not expanded"
                .into()) }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.activity_failure_info".into(), kind :
                WireKind::Message, optional : true, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.scheduled_event_id"
                .into(), declaration :
                "temporal.api.failure.v1.ActivityFailureInfo.scheduled_event_id".into(),
                kind : WireKind::Signed, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.started_event_id"
                .into(), declaration :
                "temporal.api.failure.v1.ActivityFailureInfo.started_event_id".into(),
                kind : WireKind::Signed, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.identity"
                .into(), declaration :
                "temporal.api.failure.v1.ActivityFailureInfo.identity".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.activity_type"
                .into(), declaration :
                "temporal.api.failure.v1.ActivityFailureInfo.activity_type".into(), kind
                : WireKind::Message, optional : true, refusal : None }, WireField { path
                :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.activity_type.name"
                .into(), declaration : "temporal.api.common.v1.ActivityType.name".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.activity_id"
                .into(), declaration :
                "temporal.api.failure.v1.ActivityFailureInfo.activity_id".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.retry_state"
                .into(), declaration :
                "temporal.api.failure.v1.ActivityFailureInfo.retry_state".into(), kind :
                WireKind::Enum { defined : vec![0i32, 1i32, 2i32, 3i32, 4i32, 5i32, 6i32,
                7i32] }, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.child_workflow_execution_failure_info"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.namespace"
                .into(), declaration :
                "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_execution"
                .into(), declaration :
                "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.workflow_execution"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_execution.workflow_id"
                .into(), declaration :
                "temporal.api.common.v1.WorkflowExecution.workflow_id".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_execution.run_id"
                .into(), declaration : "temporal.api.common.v1.WorkflowExecution.run_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_type"
                .into(), declaration :
                "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.workflow_type"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_type.name"
                .into(), declaration : "temporal.api.common.v1.WorkflowType.name".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.initiated_event_id"
                .into(), declaration :
                "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.initiated_event_id"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.started_event_id"
                .into(), declaration :
                "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.started_event_id"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.retry_state"
                .into(), declaration :
                "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.retry_state"
                .into(), kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32, 3i32,
                4i32, 5i32, 6i32, 7i32] }, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.nexus_operation_execution_failure_info"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.scheduled_event_id"
                .into(), declaration :
                "temporal.api.failure.v1.NexusOperationFailureInfo.scheduled_event_id"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.endpoint"
                .into(), declaration :
                "temporal.api.failure.v1.NexusOperationFailureInfo.endpoint".into(), kind
                : WireKind::String, optional : false, refusal : None }, WireField { path
                :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.service"
                .into(), declaration :
                "temporal.api.failure.v1.NexusOperationFailureInfo.service".into(), kind
                : WireKind::String, optional : false, refusal : None }, WireField { path
                :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.operation"
                .into(), declaration :
                "temporal.api.failure.v1.NexusOperationFailureInfo.operation".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.operation_id"
                .into(), declaration :
                "temporal.api.failure.v1.NexusOperationFailureInfo.operation_id".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.operation_token"
                .into(), declaration :
                "temporal.api.failure.v1.NexusOperationFailureInfo.operation_token"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_handler_failure_info"
                .into(), declaration :
                "temporal.api.failure.v1.Failure.nexus_handler_failure_info".into(), kind
                : WireKind::Message, optional : true, refusal : None }, WireField { path
                :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_handler_failure_info.type"
                .into(), declaration :
                "temporal.api.failure.v1.NexusHandlerFailureInfo.type".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_handler_failure_info.retry_behavior"
                .into(), declaration :
                "temporal.api.failure.v1.NexusHandlerFailureInfo.retry_behavior".into(),
                kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32] }, optional :
                false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.identity"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.identity"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.resource_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.resource_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.metadata.key"
                .into(), declaration : "temporal.api.common.v1.Payload.MetadataEntry.key"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.metadata.value"
                .into(), declaration :
                "temporal.api.common.v1.Payload.MetadataEntry.value".into(), kind :
                WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.external_payloads.size_bytes"
                .into(), declaration :
                "temporal.api.common.v1.Payload.ExternalPayloadDetails.size_bytes"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version.build_id"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionStamp.build_id".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version.use_versioning"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionStamp.use_versioning".into(), kind :
                WireKind::Bool, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment.series_name"
                .into(), declaration :
                "temporal.api.deployment.v1.Deployment.series_name".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment.build_id"
                .into(), declaration : "temporal.api.deployment.v1.Deployment.build_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options.deployment_name"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.deployment_name"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options.build_id"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.build_id".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options.worker_versioning_mode"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.worker_versioning_mode"
                .into(), kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32] },
                optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.cause"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.cause"
                .into(), kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32, 3i32]
                }, optional : false, refusal : None }
            ]
        }
        WorkflowServiceMethod::RespondActivityTaskCanceled => {
            vec![
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.task_token"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.task_token"
                .into(), kind : WireKind::Bytes, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads"
                .into(), declaration : "temporal.api.common.v1.Payloads.payloads".into(),
                kind : WireKind::Repeated, optional : false, refusal : None }, WireField
                { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.metadata"
                .into(), declaration : "temporal.api.common.v1.Payload.metadata".into(),
                kind : WireKind::Map, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.metadata.key"
                .into(), declaration : "temporal.api.common.v1.Payload.MetadataEntry.key"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.metadata.value"
                .into(), declaration :
                "temporal.api.common.v1.Payload.MetadataEntry.value".into(), kind :
                WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.data"
                .into(), declaration : "temporal.api.common.v1.Payload.data".into(), kind
                : WireKind::Bytes, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.external_payloads"
                .into(), declaration : "temporal.api.common.v1.Payload.external_payloads"
                .into(), kind : WireKind::Repeated, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.external_payloads.size_bytes"
                .into(), declaration :
                "temporal.api.common.v1.Payload.ExternalPayloadDetails.size_bytes"
                .into(), kind : WireKind::Signed, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.identity"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.identity"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.namespace"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.namespace"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.resource_id"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.resource_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version.build_id"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionStamp.build_id".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version.use_versioning"
                .into(), declaration :
                "temporal.api.common.v1.WorkerVersionStamp.use_versioning".into(), kind :
                WireKind::Bool, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment.series_name"
                .into(), declaration :
                "temporal.api.deployment.v1.Deployment.series_name".into(), kind :
                WireKind::String, optional : false, refusal : None }, WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment.build_id"
                .into(), declaration : "temporal.api.deployment.v1.Deployment.build_id"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options"
                .into(), declaration :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options"
                .into(), kind : WireKind::Message, optional : true, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options.deployment_name"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.deployment_name"
                .into(), kind : WireKind::String, optional : false, refusal : None },
                WireField { path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options.build_id"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.build_id".into(),
                kind : WireKind::String, optional : false, refusal : None }, WireField {
                path :
                "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options.worker_versioning_mode"
                .into(), declaration :
                "temporal.api.deployment.v1.WorkerDeploymentOptions.worker_versioning_mode"
                .into(), kind : WireKind::Enum { defined : vec![0i32, 1i32, 2i32] },
                optional : false, refusal : None }
            ]
        }
        _ => {
            vec![
                WireField { path : method.name().into(), declaration : method.name()
                .into(), kind : WireKind::Message, optional : false, refusal :
                Some("request root is outside the live probe surface".into()) }
            ]
        }
    }
}
/// Apply one typed descriptor mutation to a decoded request.
pub fn mutate_workflow_request(
    method: WorkflowServiceMethod,
    bytes: &[u8],
    path: &str,
    mutation: &WireMutation,
) -> Result<Vec<u8>, GenerationError> {
    match method {
        WorkflowServiceMethod::StartActivityExecution => {
            let mut message = <temporal::api::workflowservice::v1::DescribeActivityExecutionRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_start_activity_execution(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::DescribeActivityExecution => {
            let mut message = <temporal::api::workflowservice::v1::DescribeActivityExecutionRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_describe_activity_execution(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::PollActivityExecution => {
            let mut message = <temporal::api::workflowservice::v1::PollActivityExecutionRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_poll_activity_execution(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::RequestCancelActivityExecution => {
            let mut message = <temporal::api::workflowservice::v1::RequestCancelActivityExecutionRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_request_cancel_activity_execution(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::PollActivityTaskQueue => {
            let mut message = <temporal::api::workflowservice::v1::PollActivityTaskQueueRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_poll_activity_task_queue(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::RespondActivityTaskCompleted => {
            let mut message = <temporal::api::workflowservice::v1::RespondActivityTaskCompletedRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_respond_activity_task_completed(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::RespondActivityTaskFailed => {
            let mut message = <temporal::api::workflowservice::v1::RespondActivityTaskFailedRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_respond_activity_task_failed(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        WorkflowServiceMethod::RespondActivityTaskCanceled => {
            let mut message = <temporal::api::workflowservice::v1::RespondActivityTaskCanceledRequest as ::prost::Message>::decode(
                    bytes,
                )
                .map_err(|_| GenerationError::InvalidEncoding)?;
            mutate_respond_activity_task_canceled(&mut message, path, mutation)?;
            Ok(::prost::Message::encode_to_vec(&message))
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_start_activity_execution(
    message: &mut temporal::api::workflowservice::v1::DescribeActivityExecutionRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.activity_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.run_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.run_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_input" => {
            let node0 = message;
            WireValue::mutate(&mut node0.include_input, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_outcome" => {
            let node0 = message;
            WireValue::mutate(&mut node0.include_outcome, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.long_poll_token" => {
            let node0 = message;
            WireValue::mutate(&mut node0.long_poll_token, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_heartbeat_details" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.include_heartbeat_details,
                &WireKind::Bool,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_last_failure" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.include_last_failure,
                &WireKind::Bool,
                mutation,
            )?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_describe_activity_execution(
    message: &mut temporal::api::workflowservice::v1::DescribeActivityExecutionRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.activity_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.run_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.run_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_input" => {
            let node0 = message;
            WireValue::mutate(&mut node0.include_input, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_outcome" => {
            let node0 = message;
            WireValue::mutate(&mut node0.include_outcome, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.long_poll_token" => {
            let node0 = message;
            WireValue::mutate(&mut node0.long_poll_token, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_heartbeat_details" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.include_heartbeat_details,
                &WireKind::Bool,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.include_last_failure" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.include_last_failure,
                &WireKind::Bool,
                mutation,
            )?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_poll_activity_execution(
    message: &mut temporal::api::workflowservice::v1::PollActivityExecutionRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.PollActivityExecutionRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityExecutionRequest.activity_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.activity_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityExecutionRequest.run_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.run_id, &WireKind::String, mutation)?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_request_cancel_activity_execution(
    message: &mut temporal::api::workflowservice::v1::RequestCancelActivityExecutionRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.activity_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.activity_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.run_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.run_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.identity" => {
            let node0 = message;
            WireValue::mutate(&mut node0.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.request_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.request_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.reason" => {
            let node0 = message;
            WireValue::mutate(&mut node0.reason, &WireKind::String, mutation)?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_poll_activity_task_queue(
    message: &mut temporal::api::workflowservice::v1::PollActivityTaskQueueRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.task_queue = None,
                WireMutation::Empty => node0.task_queue = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue.name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.task_queue.is_none()) {
                return Ok(());
            }
            let node1 = node0.task_queue.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue.kind" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.task_queue.is_none()) {
                return Ok(());
            }
            let node1 = node0.task_queue.get_or_insert_with(Default::default);
            WireValue::mutate(
                &mut node1.kind,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32, 3i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue.normal_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.task_queue.is_none()) {
                return Ok(());
            }
            let node1 = node0.task_queue.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.normal_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.poller_group_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.poller_group_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.identity" => {
            let node0 = message;
            WireValue::mutate(&mut node0.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_instance_key" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.worker_instance_key,
                &WireKind::String,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_control_task_queue" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.worker_control_task_queue,
                &WireKind::String,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.task_queue_metadata = None,
                WireMutation::Empty => {
                    node0.task_queue_metadata = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata.max_tasks_per_second" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.task_queue_metadata.is_none())
            {
                return Ok(());
            }
            let node1 = node0.task_queue_metadata.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent => node1.max_tasks_per_second = None,
                WireMutation::Empty => {
                    node1.max_tasks_per_second = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue_metadata.max_tasks_per_second.value" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.task_queue_metadata.is_none())
            {
                return Ok(());
            }
            let node1 = node0.task_queue_metadata.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.max_tasks_per_second.is_none())
            {
                return Ok(());
            }
            let node2 = node1.max_tasks_per_second.get_or_insert_with(Default::default);
            WireValue::mutate(&mut *node2, &WireKind::Float, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.worker_version_capabilities = None,
                WireMutation::Empty => {
                    node0.worker_version_capabilities = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version_capabilities.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .worker_version_capabilities
                .get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities.use_versioning" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version_capabilities.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .worker_version_capabilities
                .get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.use_versioning, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.worker_version_capabilities.deployment_series_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version_capabilities.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .worker_version_capabilities
                .get_or_insert_with(Default::default);
            WireValue::mutate(
                &mut node1.deployment_series_name,
                &WireKind::String,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment_options = None,
                WireMutation::Empty => {
                    node0.deployment_options = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options.deployment_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.deployment_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.deployment_options.worker_versioning_mode" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(
                &mut node1.worker_versioning_mode,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32],
                },
                mutation,
            )?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_respond_activity_task_completed(
    message: &mut temporal::api::workflowservice::v1::RespondActivityTaskCompletedRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.task_token" => {
            let node0 = message;
            WireValue::mutate(&mut node0.task_token, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.result = None,
                WireMutation::Empty => node0.result = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node1.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node1.payloads.clear();
                    node1.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.metadata" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node2.metadata.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.metadata.clear();
                    for index in 0..length {
                        node2.metadata.insert(index.to_string(), Default::default());
                    }
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.metadata.key" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.0, &WireKind::String, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.metadata.value" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.1, &WireKind::Bytes, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.data" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node2.data, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.external_payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            match mutation {
                WireMutation::Absent | WireMutation::Empty => {
                    node2.external_payloads.clear()
                }
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.external_payloads.clear();
                    node2.external_payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.result.payloads.external_payloads.size_bytes" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.result.is_none()) {
                return Ok(());
            }
            let node1 = node0.result.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent)
                && (node2.external_payloads.is_empty())
            {
                return Ok(());
            }
            if node2.external_payloads.is_empty() {
                node2.external_payloads.push(Default::default());
            }
            let node3 = node2
                .external_payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node3.size_bytes, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.identity" => {
            let node0 = message;
            WireValue::mutate(&mut node0.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.resource_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.resource_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.worker_version = None,
                WireMutation::Empty => node0.worker_version = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version.is_none())
            {
                return Ok(());
            }
            let node1 = node0.worker_version.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.worker_version.use_versioning" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version.is_none())
            {
                return Ok(());
            }
            let node1 = node0.worker_version.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.use_versioning, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment = None,
                WireMutation::Empty => node0.deployment = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment.series_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.deployment.is_none()) {
                return Ok(());
            }
            let node1 = node0.deployment.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.series_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.deployment.is_none()) {
                return Ok(());
            }
            let node1 = node0.deployment.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment_options = None,
                WireMutation::Empty => {
                    node0.deployment_options = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options.deployment_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.deployment_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest.deployment_options.worker_versioning_mode" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(
                &mut node1.worker_versioning_mode,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32],
                },
                mutation,
            )?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_respond_activity_task_failed(
    message: &mut temporal::api::workflowservice::v1::RespondActivityTaskFailedRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.task_token" => {
            let node0 = message;
            WireValue::mutate(&mut node0.task_token, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.failure = None,
                WireMutation::Empty => node0.failure = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.message" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.message, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.source" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.source, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.stack_trace" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.stack_trace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent => node1.encoded_attributes = None,
                WireMutation::Empty => {
                    node1.encoded_attributes = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.metadata" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.encoded_attributes.is_none())
            {
                return Ok(());
            }
            let node2 = node1.encoded_attributes.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node2.metadata.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.metadata.clear();
                    for index in 0..length {
                        node2.metadata.insert(index.to_string(), Default::default());
                    }
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.metadata.key" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.encoded_attributes.is_none())
            {
                return Ok(());
            }
            let node2 = node1.encoded_attributes.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.0, &WireKind::String, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.metadata.value" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.encoded_attributes.is_none())
            {
                return Ok(());
            }
            let node2 = node1.encoded_attributes.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.1, &WireKind::Bytes, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.data" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.encoded_attributes.is_none())
            {
                return Ok(());
            }
            let node2 = node1.encoded_attributes.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node2.data, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.external_payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.encoded_attributes.is_none())
            {
                return Ok(());
            }
            let node2 = node1.encoded_attributes.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => {
                    node2.external_payloads.clear()
                }
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.external_payloads.clear();
                    node2.external_payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.encoded_attributes.external_payloads.size_bytes" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node1.encoded_attributes.is_none())
            {
                return Ok(());
            }
            let node2 = node1.encoded_attributes.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (node2.external_payloads.is_empty())
            {
                return Ok(());
            }
            if node2.external_payloads.is_empty() {
                node2.external_payloads.push(Default::default());
            }
            let node3 = node2
                .external_payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node3.size_bytes, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.cause" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent => node1.cause = None,
                WireMutation::Empty => node1.cause = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.type" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.r#type, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.non_retryable" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.non_retryable, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.details = None,
                WireMutation::Empty => node2.details = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.details.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent) && (node2.details.is_none()) {
                return Ok(());
            }
            let node3 = node2.details.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node3.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node3.payloads.clear();
                    node3.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.next_retry_delay" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.next_retry_delay = None,
                WireMutation::Empty => node2.next_retry_delay = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.next_retry_delay.seconds" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.next_retry_delay.is_none())
            {
                return Ok(());
            }
            let node3 = node2.next_retry_delay.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node3.seconds, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.next_retry_delay.nanos" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.next_retry_delay.is_none())
            {
                return Ok(());
            }
            let node3 = node2.next_retry_delay.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node3.nanos, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.application_failure_info.category" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ApplicationFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.category,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.timeout_type" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.timeout_type,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32, 3i32, 4i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.last_heartbeat_details = None,
                WireMutation::Empty => {
                    node2.last_heartbeat_details = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.timeout_failure_info.last_heartbeat_details.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::TimeoutFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node3 = node2
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node3.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node3.payloads.clear();
                    node3.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.details = None,
                WireMutation::Empty => node2.details = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.details.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent) && (node2.details.is_none()) {
                return Ok(());
            }
            let node3 = node2.details.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node3.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node3.payloads.clear();
                    node3.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.canceled_failure_info.identity" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::CanceledFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.terminated_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::TerminatedFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::TerminatedFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.terminated_failure_info.identity" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::TerminatedFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::TerminatedFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::TerminatedFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::TerminatedFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.server_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ServerFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ServerFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.server_failure_info.non_retryable" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ServerFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ServerFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ServerFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ServerFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.non_retryable, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.last_heartbeat_details = None,
                WireMutation::Empty => {
                    node2.last_heartbeat_details = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.reset_workflow_failure_info.last_heartbeat_details.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ResetWorkflowFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node3 = node2
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node3.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node3.payloads.clear();
                    node3.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.scheduled_event_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.scheduled_event_id,
                &WireKind::Signed,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.started_event_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.started_event_id, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.identity" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.activity_type" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.activity_type = None,
                WireMutation::Empty => node2.activity_type = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.activity_type.name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.activity_type.is_none())
            {
                return Ok(());
            }
            let node3 = node2.activity_type.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node3.name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.activity_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.activity_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.activity_failure_info.retry_state" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ActivityFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.retry_state,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32, 3i32, 4i32, 5i32, 6i32, 7i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.namespace" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_execution" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.workflow_execution = None,
                WireMutation::Empty => {
                    node2.workflow_execution = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_execution.workflow_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.workflow_execution.is_none())
            {
                return Ok(());
            }
            let node3 = node2.workflow_execution.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node3.workflow_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_execution.run_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.workflow_execution.is_none())
            {
                return Ok(());
            }
            let node3 = node2.workflow_execution.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node3.run_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_type" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            match mutation {
                WireMutation::Absent => node2.workflow_type = None,
                WireMutation::Empty => node2.workflow_type = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.workflow_type.name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            if matches!(mutation, WireMutation::Absent)
                && (node2.workflow_type.is_none())
            {
                return Ok(());
            }
            let node3 = node2.workflow_type.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node3.name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.initiated_event_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.initiated_event_id,
                &WireKind::Signed,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.started_event_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.started_event_id, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.child_workflow_execution_failure_info.retry_state" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::ChildWorkflowExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.retry_state,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32, 3i32, 4i32, 5i32, 6i32, 7i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.scheduled_event_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.scheduled_event_id,
                &WireKind::Signed,
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.endpoint" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.endpoint, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.service" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.service, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.operation" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.operation, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.operation_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.operation_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_operation_execution_failure_info.operation_token" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusOperationExecutionFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.operation_token, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_handler_failure_info" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) {
                if matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(_))
                ) {
                    node1.failure_info = None;
                }
            } else {
                if !matches!(mutation, WireMutation::Empty) {
                    return Err(GenerationError::InapplicableMutation);
                }
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(
                        Default::default(),
                    ),
                );
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_handler_failure_info.type" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(&mut node2.r#type, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.failure.nexus_handler_failure_info.retry_behavior" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.failure.is_none()) {
                return Ok(());
            }
            let node1 = node0.failure.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent)
                && (!matches!(
                    node1.failure_info,
                    Some(temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(_))
                ))
            {
                return Ok(());
            }
            if !matches!(
                node1.failure_info,
                Some(temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(_))
            ) {
                node1.failure_info = Some(
                    temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(
                        Default::default(),
                    ),
                );
            }
            let Some(
                temporal::api::failure::v1::failure::FailureInfo::NexusHandlerFailureInfo(
                    node2,
                ),
            ) = node1.failure_info.as_mut() else {
                return Err(GenerationError::UnsupportedType)
            };
            WireValue::mutate(
                &mut node2.retry_behavior,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.identity" => {
            let node0 = message;
            WireValue::mutate(&mut node0.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.resource_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.resource_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.last_heartbeat_details = None,
                WireMutation::Empty => {
                    node0.last_heartbeat_details = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node1.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node1.payloads.clear();
                    node1.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.metadata" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node2.metadata.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.metadata.clear();
                    for index in 0..length {
                        node2.metadata.insert(index.to_string(), Default::default());
                    }
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.metadata.key" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.0, &WireKind::String, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.metadata.value" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.1, &WireKind::Bytes, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.data" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node2.data, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.external_payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            match mutation {
                WireMutation::Absent | WireMutation::Empty => {
                    node2.external_payloads.clear()
                }
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.external_payloads.clear();
                    node2.external_payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.last_heartbeat_details.payloads.external_payloads.size_bytes" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.last_heartbeat_details.is_none())
            {
                return Ok(());
            }
            let node1 = node0
                .last_heartbeat_details
                .get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent)
                && (node2.external_payloads.is_empty())
            {
                return Ok(());
            }
            if node2.external_payloads.is_empty() {
                node2.external_payloads.push(Default::default());
            }
            let node3 = node2
                .external_payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node3.size_bytes, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.worker_version = None,
                WireMutation::Empty => node0.worker_version = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version.is_none())
            {
                return Ok(());
            }
            let node1 = node0.worker_version.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.worker_version.use_versioning" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version.is_none())
            {
                return Ok(());
            }
            let node1 = node0.worker_version.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.use_versioning, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment = None,
                WireMutation::Empty => node0.deployment = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment.series_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.deployment.is_none()) {
                return Ok(());
            }
            let node1 = node0.deployment.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.series_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.deployment.is_none()) {
                return Ok(());
            }
            let node1 = node0.deployment.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment_options = None,
                WireMutation::Empty => {
                    node0.deployment_options = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options.deployment_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.deployment_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.deployment_options.worker_versioning_mode" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(
                &mut node1.worker_versioning_mode,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32],
                },
                mutation,
            )?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest.cause" => {
            let node0 = message;
            WireValue::mutate(
                &mut node0.cause,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32, 3i32],
                },
                mutation,
            )?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn mutate_respond_activity_task_canceled(
    message: &mut temporal::api::workflowservice::v1::RespondActivityTaskCanceledRequest,
    path: &str,
    mutation: &WireMutation,
) -> Result<(), GenerationError> {
    match path {
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.task_token" => {
            let node0 = message;
            WireValue::mutate(&mut node0.task_token, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.details = None,
                WireMutation::Empty => node0.details = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node1.payloads.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node1.payloads.clear();
                    node1.payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.metadata" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            match mutation {
                WireMutation::Absent | WireMutation::Empty => node2.metadata.clear(),
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.metadata.clear();
                    for index in 0..length {
                        node2.metadata.insert(index.to_string(), Default::default());
                    }
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.metadata.key" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.0, &WireKind::String, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.metadata.value" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent) && (node2.metadata.is_empty()) {
                return Ok(());
            }
            let key2 = node2.metadata.keys().next().cloned().unwrap_or_default();
            let mut entry2 = (
                key2.clone(),
                node2.metadata.remove(&key2).unwrap_or_default(),
            );
            let node3 = &mut entry2;
            WireValue::mutate(&mut node3.1, &WireKind::Bytes, mutation)?;
            node2.metadata.insert(entry2.0, entry2.1);
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.data" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node2.data, &WireKind::Bytes, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.external_payloads" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            match mutation {
                WireMutation::Absent | WireMutation::Empty => {
                    node2.external_payloads.clear()
                }
                WireMutation::LimitPlusOne { .. } => {
                    let length = wire_collection_len(mutation)?;
                    node2.external_payloads.clear();
                    node2.external_payloads.resize_with(length, Default::default);
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.details.payloads.external_payloads.size_bytes" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.details.is_none()) {
                return Ok(());
            }
            let node1 = node0.details.get_or_insert_with(Default::default);
            if matches!(mutation, WireMutation::Absent) && (node1.payloads.is_empty()) {
                return Ok(());
            }
            if node1.payloads.is_empty() {
                node1.payloads.push(Default::default());
            }
            let node2 = node1
                .payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            if matches!(mutation, WireMutation::Absent)
                && (node2.external_payloads.is_empty())
            {
                return Ok(());
            }
            if node2.external_payloads.is_empty() {
                node2.external_payloads.push(Default::default());
            }
            let node3 = node2
                .external_payloads
                .first_mut()
                .ok_or(GenerationError::UnsupportedType)?;
            WireValue::mutate(&mut node3.size_bytes, &WireKind::Signed, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.identity" => {
            let node0 = message;
            WireValue::mutate(&mut node0.identity, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.namespace" => {
            let node0 = message;
            WireValue::mutate(&mut node0.namespace, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.resource_id" => {
            let node0 = message;
            WireValue::mutate(&mut node0.resource_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.worker_version = None,
                WireMutation::Empty => node0.worker_version = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version.is_none())
            {
                return Ok(());
            }
            let node1 = node0.worker_version.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.worker_version.use_versioning" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.worker_version.is_none())
            {
                return Ok(());
            }
            let node1 = node0.worker_version.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.use_versioning, &WireKind::Bool, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment = None,
                WireMutation::Empty => node0.deployment = Some(Default::default()),
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment.series_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.deployment.is_none()) {
                return Ok(());
            }
            let node1 = node0.deployment.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.series_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent) && (node0.deployment.is_none()) {
                return Ok(());
            }
            let node1 = node0.deployment.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options" => {
            let node0 = message;
            match mutation {
                WireMutation::Absent => node0.deployment_options = None,
                WireMutation::Empty => {
                    node0.deployment_options = Some(Default::default());
                }
                _ => return Err(GenerationError::InapplicableMutation),
            }
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options.deployment_name" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.deployment_name, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options.build_id" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(&mut node1.build_id, &WireKind::String, mutation)?;
            Ok(())
        }
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest.deployment_options.worker_versioning_mode" => {
            let node0 = message;
            if matches!(mutation, WireMutation::Absent)
                && (node0.deployment_options.is_none())
            {
                return Ok(());
            }
            let node1 = node0.deployment_options.get_or_insert_with(Default::default);
            WireValue::mutate(
                &mut node1.worker_versioning_mode,
                &WireKind::Enum {
                    defined: vec![0i32, 1i32, 2i32],
                },
                mutation,
            )?;
            Ok(())
        }
        _ => Err(GenerationError::UnsupportedType),
    }
}
fn wire_collection_len(mutation: &WireMutation) -> Result<usize, GenerationError> {
    let WireMutation::LimitPlusOne { limit } = mutation else {
        return Err(GenerationError::InapplicableMutation)
    };
    let length = limit.checked_add(1).ok_or(GenerationError::SizeLimit)?;
    if length > 4096 {
        return Err(GenerationError::SizeLimit);
    }
    usize::try_from(length).map_err(|_| GenerationError::SizeLimit)
}
