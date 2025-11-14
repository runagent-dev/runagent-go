package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/runagent-dev/runagent-go/pkg/client"
)

func main() {
	agentClient, err := client.New(
		"159689a8-d465-4329-a6ce-78bfdb5252ff",
		"agno_stream",
		true,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer agentClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stream, err := agentClient.RunStream(ctx, map[string]interface{}{
		"input": "Benefits of a long drive",
	})
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	defer stream.Close()

	for {
		data, ok, err := stream.Next(ctx)
		if err != nil {
			log.Fatalf("Stream error: %v", err)
		}
		if !ok {
			break
		}

		if dataMap, ok := data.(map[string]interface{}); ok {
			if content, ok := dataMap["content"]; ok {
				fmt.Print(content)
			}
		}
	}
	fmt.Println()
}
