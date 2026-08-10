/*
Package gomad contains the main API for interacting with Gomad simulations.
Gomad is a simulation testing framework that aims to make it easier to write
reliable distributed systems code in go. See the README.md file for an
introduction to Gomad.

# Introduction

Distributed systems can fail in many surprising ways. Exhaustively thinking of
all things that can go wrong and writing tests for them is infeasible. Gomad
simulates the external components that a program might interact with, such as
the disk, network, clocks, and more. Gomad has an API to introduce failures in
these systems (like chaos testing) to test that a program can handle otherwise
difficult-to-reproduce failures.

# Writing and running gomad tests

Gomad tests are normal Go tests that are run in a Gomad simulation. To run Gomad
tests, use the 'gomad test' command with similar arguments as the standard 'go
test' command:

	# run an normal test
	go test -run TestName -v ./path/to/pkg
	# run a gomad test
	gomad test -run TestName -v ./path/to/pkg

The 'gomad' executable is built from cmd/tools/gomad in the Temporal repository.
To run Gomad tests from within a Go test, use the
[github.com/temporalio/gomad/metatesting] package.

# Gomad simulation

When tests run inside Gomad they run in a simulation and they do not interact
with the normal operating system. Gomad simulates:

  - Real time. The time is simulated by Gomad (like on the Go playground) so that
    it advances automatically when all goroutines are paused. This means tests run
    quickly even if they sleep for a long time.

  - The filesystem. Gomad implements its own filesystem that can simulate data
    loss when writes are not fsync-ed properly.

  - The network. Gomad implements its own network that can introduce extra latency
    and partition machines.

  - Machines. Inside of Gomad a single Go test can create multiple machines which
    are new instantiations of a Go program with their own global variables, their
    own disk, and their own network address.

Gomad implements its simulation at the Go runtime and system call level,
emulating Linux system calls. Gomad compiles all code with GOOS set to linux. To
interact with the simulation, programs can use the standard library with
functions like [time.Sleep], [os.OpenFile], [net.Dial], and all others working
as you would expect.

To control the simulation, this package has functions like [NewMachine] and
[Machine.Crash] to create and manipulate machines and [SetConnected] to
manipulate the network.

# Gomad internals

For a description of how Gomad works, see the design in
https://github.com/temporalio/gomad/blob/main/docs/design.md
*/
package gomad
