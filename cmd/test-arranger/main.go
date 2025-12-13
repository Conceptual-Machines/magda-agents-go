package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/joho/godotenv"

	// Import the arranger agent (it's in package services)
	arranger "github.com/Conceptual-Machines/magda-agents-go/agents/arranger"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️  Warning: Could not load .env file: %v", err)
		log.Println("   Continuing with environment variables...")
	}

	// Get OpenAI API key from environment
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("❌ ERROR: OPENAI_API_KEY is not set in environment!")
	}

	// Create config
	cfg := &config.Config{
		OpenAIAPIKey: openAIKey,
	}

	// Create basic arranger agent (no MCP for now)
	agent := arranger.NewBasicArrangerAgent(cfg)

	// Test questions
	testQuestions := []string{
		"add an e minor arpeggio",
		"create a C major chord",
		"add a chord progression: C, Am, F, G",
	}

	ctx := context.Background()

	for i, question := range testQuestions {
		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Test %d/%d: %s\n", i+1, len(testQuestions), question)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		startTime := time.Now()

		// Generate actions
		result, err := agent.GenerateActions(ctx, question)
		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		duration := time.Since(startTime)

		// Print results
		fmt.Printf("✅ Success! Duration: %v\n\n", duration)
		fmt.Printf("Actions (%d):\n", len(result.Actions))
		for j, action := range result.Actions {
			actionJSON, _ := json.MarshalIndent(action, "", "  ")
			fmt.Printf("  [%d] %s\n", j+1, string(actionJSON))
		}

		if result.Usage != nil {
			fmt.Printf("\nUsage:\n")
			usageJSON, _ := json.MarshalIndent(result.Usage, "", "  ")
			fmt.Printf("  %s\n", string(usageJSON))
		}

		if result.MCPUsed {
			fmt.Printf("\nMCP: Used (%d calls)\n", result.MCPCalls)
		}

		// Small delay between tests
		if i < len(testQuestions)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ All tests completed!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

