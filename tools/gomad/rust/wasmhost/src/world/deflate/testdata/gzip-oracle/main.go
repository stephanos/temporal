// Independent, test-only oracle for the pinned stock Go 1.27.1 codec.
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"

	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "world" {
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			return err
		}
		model, err := world.New(world.Config{Seed: 7, Limits: world.CompressedSnapshotProfile().Limits})
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			id, err := model.Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "test", Kind: "stream", Key: fmt.Sprintf("stream.%d", i)}, Payload: bytes.Repeat([]byte("world-payload-"), i%100)})
			if err != nil {
				return err
			}
			if _, err := model.Ready(world.Readiness{RequestID: id, At: world.InitialTime, Kind: "ready", Payload: []byte("ready")}); err != nil {
				return err
			}
		}
		if _, err := model.Quiesce(); err != nil {
			return err
		}
		encoded, err := world.EncodeSnapshot(model.Snapshot())
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(encoded)
		return err
	}
	if len(os.Args) > 1 && os.Args[1] == "decode" {
		limit, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			return err
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		reader, err := gzip.NewReader(bytes.NewReader(input))
		if err != nil {
			return err
		}
		reader.Multistream(false)
		raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
		if err != nil {
			return err
		}
		if int64(len(raw)) > limit {
			return fmt.Errorf("expanded bound")
		}
		if err := reader.Close(); err != nil {
			return err
		}
		_, err = os.Stdout.Write(raw)
		return err
	}
	level := gzip.DefaultCompression
	if len(os.Args) > 1 {
		var err error
		level, err = strconv.Atoi(os.Args[1])
		if err != nil {
			return err
		}
	}
	writer, err := gzip.NewWriterLevel(os.Stdout, level)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, os.Stdin); err != nil {
		return err
	}
	return writer.Close()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
