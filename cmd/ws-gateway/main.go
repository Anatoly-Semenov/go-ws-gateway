package main

import (
	"fmt"
	"os"

	"github.com/anatoly-dev/go-ws-gateway/cmd/ws-gateway/commands"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ws-gateway",
		Short: "WebSocket Gateway service",
		Long:  "A high-performance WebSocket Gateway service that handles Kafka messages and sends them to clients via WebSockets",
	}

	rootCmd.AddCommand(commands.NewServeCommand())
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
