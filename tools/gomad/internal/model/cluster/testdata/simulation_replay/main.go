package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := fmt.Println("ok"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	replayedPayload := os.Getenv("GOMAD_SIMULATION_REPLAY_PAYLOAD")
	if replayedPayload == "" {
		replayedPayload = "two"
	}
	if replayedPayload != "one" && replayedPayload != "two" {
		return fmt.Errorf("invalid replay payload %q", replayedPayload)
	}
	payload := "one"
	listening := make(chan struct{}, 1)
	acknowledged := make(chan struct{}, 1)
	const serverBoot gomadsim.BootID = "simulation-replay-server"
	const clientBoot gomadsim.BootID = "simulation-replay-client"
	if err := gomadsim.RegisterBoot(serverBoot, func(_ context.Context, node gomadsim.NodeContext) (retErr error) {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, listener.Close()) }()
		listening <- struct{}{}
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, conn.Close()) }()
		body := make([]byte, 3)
		if _, err := io.ReadFull(conn, body); err != nil {
			return err
		}
		if string(body) != payload {
			return fmt.Errorf("received stream payload=%q, want %q", body, payload)
		}
		if n, err := conn.Write([]byte("ack")); n != 3 || err != nil {
			return fmt.Errorf("acknowledgement write=%d,%v", n, err)
		}
		<-acknowledged
		return nil
	}); err != nil {
		return err
	}
	if err := gomadsim.RegisterBoot(clientBoot, func(ctx context.Context, _ gomadsim.NodeContext) (retErr error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, conn.Close()) }()
		if n, err := conn.Write([]byte(payload)); n != 3 || err != nil {
			return fmt.Errorf("stream payload write=%d,%v", n, err)
		}
		ack := make([]byte, 3)
		if _, err := io.ReadFull(conn, ack); err != nil || string(ack) != "ack" {
			return fmt.Errorf("acknowledgement=%q,%v", ack, err)
		}
		acknowledged <- struct{}{}
		return nil
	}); err != nil {
		return err
	}
	spec := gomadsim.Spec{
		Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendInProcess, Fidelity: gomadsim.FidelitySimulationModel,
		Seed: 89, Limits: gomadsim.DefaultLimits(),
		Nodes: []gomadsim.NodeSpec{
			{ID: "client", Boot: clientBoot, Address: "10.0.0.2"},
			{ID: "server", Boot: serverBoot, Address: "10.0.0.1"},
		},
		Links: []gomadsim.LinkSpec{
			{From: "client", To: "server", Enabled: true},
			{From: "server", To: "client", Enabled: true},
		},
	}
	scenario := func(ctx context.Context, cluster gomadsim.Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		select {
		case <-listening:
		case <-ctx.Done():
			return ctx.Err()
		}
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		for _, handle := range []gomadsim.NodeHandle{client, server} {
			terminal, err := cluster.Wait(ctx, handle)
			if err != nil {
				return err
			}
			if terminal.State != gomadsim.NodeStateExited {
				return fmt.Errorf("node %s finished as %s: %s", handle.Node, terminal.State, terminal.Reason)
			}
		}
		return nil
	}
	recorded, err := gomadsim.Run(ctx, spec, scenario)
	if err != nil {
		return err
	}
	if recorded.Outcome != gomadsim.OutcomeCompleted {
		return fmt.Errorf("recorded simulation outcome=%s: %s", recorded.Outcome, recorded.Reason)
	}
	plan, err := gomadsim.ReplayPlanFor(recorded.Record)
	if err != nil {
		return err
	}
	spec.Replay = &plan
	payload = replayedPayload
	replayed, err := gomadsim.Run(ctx, spec, scenario)
	if err != nil {
		return err
	}
	if replayed.Outcome != gomadsim.OutcomeCompleted || replayed.Record.Identity != recorded.Record.Identity {
		return fmt.Errorf("network-only replay outcome=%s: %s, record=%s, want %s", replayed.Outcome, replayed.Reason, replayed.Record.Identity, recorded.Record.Identity)
	}
	return nil
}
