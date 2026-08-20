// Package gomadv3sim defines the application-facing Gomad v3 cluster contract.
// Its in-process backend provides lifecycle, runtime-domain, and node-aware
// virtual-network fidelity. It composes separate durable-volume, fault,
// scenario, history, oracle, artifact, and replay modules. Its process backend
// adds fresh package initialization and hard isolation while reusing the same
// detached models.
package gomadv3sim
