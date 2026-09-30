// Command parity emits the existing Go models' tables for Quint's agreement checks.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/standaloneactivity"
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

func main() {
	models := []umpire.Model{
		nexuscaller.NexusProduct, nexuscaller.NexusProtocol,
		standaloneactivity.ActivityProduct, standaloneactivity.ActivityProtocol,
		worker.PollingMachine,
	}
	tables := make([]*umpire.Table, 0, len(models))
	for _, model := range models {
		table, err := model.Table()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		tables = append(tables, table)
	}
	if err := json.NewEncoder(os.Stdout).Encode(tables); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
