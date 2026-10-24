package runner

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	"umpire.local/model/host"
)

func writeProgram(ctx context.Context, args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("umpire program", flag.ContinueOnError)
	flags.SetOutput(stderr)
	model := flags.String("model", "", "the guest wasm")
	world := flags.String("world", "", "a binary google.protobuf.Any: the Reset configuration")
	output := flags.String("out", "", "the binary google.protobuf.Any program")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("program accepts no positional arguments")
	}
	if *model == "" || *world == "" || *output == "" {
		return errors.New("--model, --world and --out are required")
	}
	wasm, err := os.ReadFile(*model)
	if err != nil {
		return fmt.Errorf("read the model: %w", err)
	}
	configuration, err := readAny(*world)
	if err != nil {
		return fmt.Errorf("read the world: %w", err)
	}
	guest, err := host.New(ctx, wasm)
	if err != nil {
		return err
	}
	reset, resetErr := guest.Reset(ctx, &modelv1.ResetRequest{Configuration: configuration})
	if err := errors.Join(resetErr, guest.Close(ctx)); err != nil {
		return err
	}
	if reset.GetProgram() == nil {
		return errors.New("Reset returned no program")
	}
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(reset.GetProgram())
	if err != nil {
		return fmt.Errorf("marshal the program: %w", err)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		return fmt.Errorf("write the program: %w", err)
	}
	return nil
}
