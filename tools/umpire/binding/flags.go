package binding

import "flag"

// RegisterFlags registers the flags that name a deployment, the same set under the same names for
// every command that binds Cases to one; subject is what the help text says binds, "the Case" or
// "the Cases".
func RegisterFlags(flags *flag.FlagSet, deployment *Deployment, subject string) {
	flags.StringVar(&deployment.GRPCAddress, "grpc", "", "frontend gRPC address")
	flags.StringVar(&deployment.HTTPAddress, "http", "", "frontend HTTP address")
	flags.StringVar(&deployment.Namespace, "namespace", "", "namespace "+subject+" bind to")
	flags.StringVar(&deployment.TaskQueue, "task-queue", "", "task queue "+subject+" bind to")
	flags.StringVar(&deployment.NexusEndpoint, "nexus-endpoint", "", "Nexus endpoint "+subject+" bind to")
	flags.StringVar(&deployment.HandlerTaskQueue, "handler-task-queue", "", "task queue "+subject+"' Nexus handler polls when one is bound apart from the caller's (default <task-queue>-handler)")
	flags.BoolVar(&deployment.Create, "create", false, "create the named resources and delete them on exit")
}

// Missing names the required deployment flags the caller left empty, in registration order.
func Missing(deployment Deployment) []string {
	var missing []string
	for _, required := range []struct{ name, value string }{
		{"--grpc", deployment.GRPCAddress},
		{"--http", deployment.HTTPAddress},
		{"--namespace", deployment.Namespace},
		{"--task-queue", deployment.TaskQueue},
	} {
		if required.value == "" {
			missing = append(missing, required.name)
		}
	}
	return missing
}
