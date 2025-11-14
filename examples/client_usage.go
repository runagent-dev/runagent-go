package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/runagent-dev/runagent-go/pkg/client"
	"github.com/runagent-dev/runagent-go/pkg/types"
)

func exampleLocalExecution() {
	fmt.Println("=== Local Agent Execution ===")

	c, err := client.NewClient("my-agent", "process", &client.ClientOptions{
		Local: true,
		Host:  "localhost",
		Port:  8450,
	})
	if err != nil {
		log.Printf("Failed to create client: %v\n", err)
		return
	}
	defer c.Close()

	result, err := c.Run(context.Background(), map[string]interface{}{
		"input": "Hello from Go!",
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n\n", result)
}

func exampleRemoteExecution() {
	fmt.Println("=== Remote Agent Execution ===")

	c, err := client.FromEnv("my-agent", "process")
	if err != nil {
		log.Printf("Failed to create client: %v\n", err)
		return
	}
	defer c.Close()

	result, err := c.Run(context.Background(), map[string]interface{}{
		"prompt": "What is 2+2?",
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n\n", result)
}

func exampleStreamExecution() {
	fmt.Println("=== Stream Execution ===")

	c, err := client.NewClient("my-agent", "stream", &client.ClientOptions{
		Local: true,
		Host:  "localhost",
		Port:  8450,
	})
	if err != nil {
		log.Printf("Failed to create client: %v\n", err)
		return
	}
	defer c.Close()

	iterator, err := c.RunStream(context.Background(), map[string]interface{}{
		"prompt": "Generate a list",
	})
	if err != nil {
		log.Printf("Failed to start stream: %v\n", err)
		return
	}

	for {
		chunk, ok, err := iterator.Next(context.Background())
		if err != nil {
			log.Printf("Stream error: %v\n", err)
			break
		}
		if !ok {
			break
		}
		fmt.Printf("Chunk: %v\n", chunk)
	}
	fmt.Println()
}

func exampleExtraParams() {
	fmt.Println("=== Using Extra Params ===")

	c, err := client.NewClient("my-agent", "process", &client.ClientOptions{
		Host: "localhost",
		Port: 8450,
		ExtraParams: map[string]interface{}{
			"trace_id": "abc123",
			"user_id":  "user-42",
		},
	})
	if err != nil {
		log.Printf("Failed to create client: %v\n", err)
		return
	}
	defer c.Close()

	params := c.GetExtraParams()
	fmt.Printf("Extra params: %v\n\n", params)
}

func exampleErrorHandling() {
	fmt.Println("=== Error Handling ===")

	c, err := client.NewClient("my-agent", "process", &client.ClientOptions{
		APIKey: "invalid-key",
	})
	if err != nil {
		log.Printf("Failed to create client: %v\n", err)
		return
	}
	defer c.Close()

	_, err = c.Run(context.Background(), map[string]interface{}{})
	if err != nil {
		if runAgentErr, ok := err.(*types.RunAgentError); ok {
			fmt.Printf("Error Type: %s\n", runAgentErr.Type)
			fmt.Printf("Error Code: %s\n", runAgentErr.Code)
			fmt.Printf("Error Message: %s\n\n", runAgentErr.Message)
		} else {
			fmt.Printf("Unexpected error: %v\n\n", err)
		}
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "local":
			exampleLocalExecution()
		case "remote":
			exampleRemoteExecution()
		case "stream":
			exampleStreamExecution()
		case "extra":
			exampleExtraParams()
		case "error":
			exampleErrorHandling()
		default:
			fmt.Println("Usage: go run client_usage.go [local|remote|stream|extra|error]")
		}
	} else {
		fmt.Println("RunAgent Go SDK Examples\n")
		exampleLocalExecution()
		fmt.Println("Set RUNAGENT_API_KEY and run with 'remote' argument to try remote execution")
	}
}
