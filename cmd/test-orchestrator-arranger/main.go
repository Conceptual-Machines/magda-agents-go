package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/agents/coordination"
	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/joho/godotenv"
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

	// Create orchestrator
	orchestrator := coordination.NewOrchestrator(cfg)

	// Test questions
	testQuestions := []string{
		"create a new track with piano instrument and add a C Am F G chord progression",
		"create a new track with Serum and add an E minor arpeggio",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for i, question := range testQuestions {
		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Test %d/%d: %s\n", i+1, len(testQuestions), question)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		startTime := time.Now()

		// Generate actions
		result, err := orchestrator.GenerateActions(ctx, question, nil)
		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		duration := time.Since(startTime)

		// Print results
		fmt.Printf("✅ Success! Duration: %v\n\n", duration)
		fmt.Printf("Actions (%d):\n", len(result.Actions))

		// Pretty print actions
		for j, action := range result.Actions {
			actionJSON, _ := json.MarshalIndent(action, "", "  ")
			fmt.Printf("  [%d] %s\n", j+1, string(actionJSON))

			// Check if this is a MIDI action and show note count
			if actionType, ok := action["action"].(string); ok && actionType == "add_midi" {
				if notes, ok := action["notes"].([]interface{}); ok {
					fmt.Printf("      → Contains %d MIDI notes\n", len(notes))
				} else if notes, ok := action["notes"].([]map[string]any); ok {
					fmt.Printf("      → Contains %d MIDI notes\n", len(notes))
				}
			}
		}

		if result.Usage != nil {
			fmt.Printf("\nUsage:\n")
			usageJSON, _ := json.MarshalIndent(result.Usage, "", "  ")
			fmt.Printf("  %s\n", string(usageJSON))
		}

		// Small delay between tests
		if i < len(testQuestions)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ All tests completed!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
