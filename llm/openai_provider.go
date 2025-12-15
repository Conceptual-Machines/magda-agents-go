package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Conceptual-Machines/grammar-school-go/gs"
	"github.com/getsentry/sentry-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

const (
	// Role constants
	userRole       = "user"
	developerRole  = "developer"
	maxOutputTrunc = 200
	mcpCallType    = "mcp_call"

	// Reasoning effort levels
	reasoningNone    = "none" // GPT-5.2 default - lowest latency
	reasoningMinimal = "minimal"
	reasoningLow     = "low"
	reasoningMedium  = "medium"
	reasoningHigh    = "high"
	reasoningXHigh   = "xhigh" // GPT-5.2 new level - maximum reasoning

	// Heartbeat interval for streaming (send every 10 seconds to keep connection alive during long operations)
	heartbeatIntervalSeconds = 10
	reasoningMin             = "min"
	reasoningMed             = "med"

	// Provider name
	providerNameOpenAI = "openai"

	// Logging limits
	maxArgsLogLength       = 100
	maxLogEventCountOpenAI = 5
	maxPreviewChars        = 200
	maxErrorPreviewChars   = 500
	maxErrorResponseChars  = 200
	maxPathPreviewLen      = 10
)

// OpenAIProvider implements the Provider interface using OpenAI's Responses API
type OpenAIProvider struct {
	client *openai.Client
	apiKey string // Store API key for raw HTTP requests when needed
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIProvider{
		client: &client,
		apiKey: apiKey,
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return providerNameOpenAI
}

// Generate implements non-streaming generation using OpenAI's Responses API
//
//nolint:gocyclo // Complex logic needed for handling CFG, JSON Schema, and standard requests
func (p *OpenAIProvider) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error) {
	startTime := time.Now()
	log.Printf("🎵 OPENAI GENERATION REQUEST STARTED (Model: %s)", request.Model)

	// Start Sentry transaction
	transaction := sentry.StartTransaction(ctx, "openai.generate")
	defer transaction.Finish()

	transaction.SetTag("model", request.Model)
	transaction.SetTag("provider", "openai")
	transaction.SetTag("mcp_enabled", fmt.Sprintf("%t", request.MCPConfig != nil))

	// Build OpenAI-specific request parameters
	params := p.buildRequestParams(request)

	log.Printf("🚨 CRITICAL: About to call OpenAI API with params.Model='%s'", params.Model)

	// Call OpenAI API with Sentry span
	span := transaction.StartChild("openai.api_call")
	apiStartTime := time.Now()

	// Use raw HTTP request for CFG tools (MAGDA always uses DSL/CFG)
	// Marshal params to JSON, modify as needed, make raw HTTP request
	var resp *responses.Response
	var err error
	if request.CFGGrammar != nil {
		paramsJSON, _ := json.Marshal(params)
		var paramsMap map[string]any
		if json.Unmarshal(paramsJSON, &paramsMap) == nil {
			// Add CFG tool (MAGDA always uses DSL)
			// Use grammar-school utility to build OpenAI CFG tool payload
			cfgTool := gs.BuildOpenAICFGTool(gs.CFGConfig{
				ToolName:    request.CFGGrammar.ToolName,
				Description: request.CFGGrammar.Description,
				Grammar:     request.CFGGrammar.Grammar,
				Syntax:      request.CFGGrammar.Syntax,
			})
			log.Printf("🔧 CFG GRAMMAR CONFIGURED: %s (syntax: %s)", request.CFGGrammar.ToolName, request.CFGGrammar.Syntax)

			// Set text format to plain text (not JSON schema) when using CFG
			paramsMap["text"] = gs.GetOpenAITextFormatForCFG()

			// Initialize tools array if not present
			var tools []any
			if paramsMap["tools"] == nil {
				tools = []any{}
			} else {
				var ok bool
				tools, ok = paramsMap["tools"].([]any)
				if !ok {
					// If tools is not a slice, try to convert from existing tools
					if existingTools, ok := paramsMap["tools"].([]responses.ToolUnionParam); ok {
						tools = make([]any, 0, len(existingTools))
						for _, t := range existingTools {
							toolJSON, _ := json.Marshal(t)
							var toolMap map[string]any
							if unmarshalErr := json.Unmarshal(toolJSON, &toolMap); unmarshalErr != nil {
								log.Printf("⚠️  Failed to unmarshal tool: %v", unmarshalErr)
								continue
							}
							tools = append(tools, toolMap)
						}
					} else {
						tools = []any{}
					}
				}
			}
			tools = append(tools, cfgTool)
			paramsMap["tools"] = tools
			paramsMap["parallel_tool_calls"] = true // Allow parallel calls so LLM can use MCP + CFG

			// Log the actual tool structure for debugging
			toolJSON, _ := json.MarshalIndent(cfgTool, "", "  ")
			log.Printf("🔧 Added CFG tool: %s (syntax: %s)", request.CFGGrammar.ToolName, request.CFGGrammar.Syntax)
			log.Printf("🔧 CFG tool structure: %s", truncateString(string(toolJSON), 2000))

			// Verify instructions are in paramsMap
			if instructions, ok := paramsMap["instructions"].(string); ok {
				log.Printf("🔍 Instructions in request (first 500 chars): %s", truncateString(instructions, 500))
			} else {
				log.Printf("⚠️  Instructions NOT found in paramsMap! Keys: %v", getMapKeys(paramsMap))
			}

			modifiedJSON, _ := json.Marshal(paramsMap)

			// Save full request payload to file
			if request.CFGGrammar != nil {
				prettyJSON, _ := json.MarshalIndent(paramsMap, "", "  ")
				requestFile := "/tmp/openai_request_full.json"
				if err := os.WriteFile(requestFile, prettyJSON, 0644); err != nil {
					log.Printf("❌ FAILED to save request: %v", err)
				} else {
					log.Printf("💾 Saved FULL request payload to %s (%d bytes)", requestFile, len(prettyJSON))
				}
			}

			log.Printf("📤 Making raw HTTP request (JSON size: %d bytes)", len(modifiedJSON))
			req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(modifiedJSON))
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
			req.Header.Set("Content-Type", "application/json")

			httpResp, httpErr := http.DefaultClient.Do(req)
			if httpErr == nil {
				defer func() {
					if closeErr := httpResp.Body.Close(); closeErr != nil {
						log.Printf("⚠️  Failed to close response body: %v", closeErr)
					}
				}()
				body, _ := io.ReadAll(httpResp.Body)
				if httpResp.StatusCode == http.StatusOK {
					// Save full response payload to file
					if request.CFGGrammar != nil {
						responseFile := "/tmp/openai_response_full.json"
						if err := os.WriteFile(responseFile, body, 0644); err != nil {
							log.Printf("❌ FAILED to save response: %v", err)
						} else {
							log.Printf("💾 Saved FULL response payload to %s (%d bytes)", responseFile, len(body))
						}
					}

					// Parse raw JSON to extract DSL from input field (SDK struct may not expose it)
					log.Printf("🔍 Parsing raw JSON response to extract DSL from input field...")
					var rawResponse map[string]any
					if json.Unmarshal(body, &rawResponse) != nil {
						err = fmt.Errorf("failed to parse response")
					} else {
						// Extract DSL code directly from raw JSON
						if output, ok := rawResponse["output"].([]any); ok {
							log.Printf("🔍 Found output array with %d items", len(output))
							for i, item := range output {
								if itemMap, ok := item.(map[string]any); ok {
									log.Printf("🔍 Checking output item %d, type: %v", i, itemMap["type"])
									log.Printf("🔍 Output item %d keys: %v", i, getMapKeys(itemMap))

									// Check for input field BEFORE type check (for debugging)
									if inputVal, inputExists := itemMap["input"]; inputExists {
										log.Printf("🔍 'input' field EXISTS in output item %d: type=%T, value=%v", i, inputVal, inputVal)
										if inputStr, ok := inputVal.(string); ok {
											log.Printf("🔍 'input' is a string with %d chars: %s", len(inputStr), truncateString(inputStr, 200))
										}
									} else {
										log.Printf("🔍 'input' field DOES NOT EXIST in output item %d", i)
									}

									if itemType, ok := itemMap["type"].(string); ok && itemType == "custom_tool_call" {
										log.Printf("✅ Found custom_tool_call in raw JSON! Checking input field...")
										if input, ok := itemMap["input"].(string); ok && input != "" {
											log.Printf("✅✅✅ Found DSL code in raw JSON input field: %s", truncateString(input, 200))
											return &GenerationResponse{
												RawOutput: input,
												Usage:     p.extractUsageFromRawResponse(rawResponse),
											}, nil
										} else {
											log.Printf("⚠️  custom_tool_call found but input field check failed")
											// Try to get input as any type and convert
											if inputVal, exists := itemMap["input"]; exists {
												log.Printf("🔍 Input field exists but type assertion failed. Type: %T, Value: %v", inputVal, inputVal)
												// Try to convert to string
												if inputStr, ok := inputVal.(string); ok {
													log.Printf("✅✅✅ Found DSL code (after conversion): %s", truncateString(inputStr, 200))
													return &GenerationResponse{
														RawOutput: inputStr,
														Usage:     p.extractUsageFromRawResponse(rawResponse),
													}, nil
												}
											}
										}
									}
								} else {
									log.Printf("⚠️  Output item %d is not a map[string]any, type: %T", i, item)
								}
							}
						} else {
							log.Printf("⚠️  No output array found in raw response. Output type: %T", rawResponse["output"])
						}

						// Fallback: parse as SDK struct for other fields
						resp = &responses.Response{}
						if json.Unmarshal(body, resp) != nil {
							err = fmt.Errorf("failed to parse response")
						} else {
							// Process response with CFG support (MAGDA always uses DSL)
							processedResp, processErr := p.processResponseWithCFG(resp, startTime, transaction, request.CFGGrammar)
							if processErr != nil {
								err = processErr
							} else {
								// Return the processed response
								return processedResp, nil
							}
						}
					}
				} else {
					err = fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
				}
			} else {
				err = httpErr
			}
		}
	}

	// Fall back to SDK if raw request failed
	if resp == nil && err == nil {
		resp, err = p.client.Responses.New(ctx, params)
	}

	apiDuration := time.Since(apiStartTime)
	span.Finish()

	if err != nil {
		log.Printf("❌ OPENAI REQUEST FAILED after %v: %v", apiDuration, err)
		transaction.SetTag("success", "false")
		sentry.CaptureException(err)
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	log.Printf("⏱️  OPENAI API CALL COMPLETED in %v", apiDuration)

	// Process response
	if request.CFGGrammar != nil {
		// MAGDA DSL uses CFG grammar
		result, err := p.processResponseWithCFG(resp, startTime, transaction, request.CFGGrammar)
		if err != nil {
			return nil, err
		}
		transaction.SetTag("success", "true")
		return result, nil
	}

	// Handle JSON Schema output (e.g., for orchestrator classification)
	if request.OutputSchema != nil {
		result, err := p.processResponseWithJSONSchema(resp, startTime, transaction, request.OutputSchema)
		if err != nil {
			return nil, err
		}
		transaction.SetTag("success", "true")
		return result, nil
	}

	// This should never happen for MAGDA, but handle it gracefully
	transaction.SetTag("success", "false")
	return nil, fmt.Errorf("CFG grammar or OutputSchema is required")
}

// buildRequestParams converts GenerationRequest to OpenAI-specific ResponseNewParams
func (p *OpenAIProvider) buildRequestParams(request *GenerationRequest) responses.ResponseNewParams {
	// Convert input_array to OpenAI messages format
	inputItems := responses.ResponseInputParam{}

	for _, item := range request.InputArray {
		role, hasRole := item["role"].(string)
		content, hasContent := item["content"].(string)

		if !hasRole || !hasContent {
			log.Printf("⚠️  Skipping invalid input item (missing role or content): %v", item)
			continue
		}

		// Convert role string to OpenAI enum
		var roleEnum responses.EasyInputMessageRole
		switch role {
		case developerRole:
			roleEnum = responses.EasyInputMessageRoleDeveloper
		case userRole:
			roleEnum = responses.EasyInputMessageRoleUser
		default:
			roleEnum = responses.EasyInputMessageRoleUser
		}

		inputItems = append(inputItems,
			responses.ResponseInputItemParamOfMessage(content, roleEnum),
		)
	}

	// Determine reasoning effort
	// Only include reasoning parameter for models that support it (GPT-5 family)
	// Models like gpt-4.1-mini do NOT support reasoning parameters
	modelsWithReasoning := map[string]bool{
		// GPT-5 base
		"gpt-5":      true,
		"gpt-5-mini": true,
		"gpt-5-nano": true,
		// GPT-5.1
		"gpt-5.1":      true,
		"gpt-5.1-mini": true,
		"gpt-5.1-nano": true,
		// GPT-5.2
		"gpt-5.2":      true,
		"gpt-5.2-mini": true,
		"gpt-5.2-nano": true,
		"gpt-5.2-pro":  true,
	}
	supportsReasoning := modelsWithReasoning[request.Model]

	var reasoningEffort shared.ReasoningEffort
	if supportsReasoning {
		switch request.ReasoningMode {
		case reasoningNone:
			// GPT-5.2 default - lowest latency
			reasoningEffort = shared.ReasoningEffort("none")
		case reasoningMinimal, reasoningMin:
			reasoningEffort = responses.ReasoningEffortLow
		case reasoningLow:
			reasoningEffort = responses.ReasoningEffortLow
		case reasoningMedium, reasoningMed:
			reasoningEffort = responses.ReasoningEffortMedium
		case reasoningHigh:
			reasoningEffort = responses.ReasoningEffortHigh
		case reasoningXHigh:
			// GPT-5.2 new level - maximum reasoning for tough problems
			reasoningEffort = shared.ReasoningEffort("xhigh")
		default:
			// Default to "none" for GPT-5.2 (lowest latency)
			reasoningEffort = shared.ReasoningEffort("none")
		}
	}

	params := responses.ResponseNewParams{
		Model: request.Model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Instructions:      openai.String(request.SystemPrompt),
		ParallelToolCalls: openai.Bool(true),
	}

	// Only include Reasoning parameter for models that support it
	if supportsReasoning {
		params.Reasoning = shared.ReasoningParam{
			Effort: reasoningEffort,
		}
	}

	// MAGDA always uses DSL/CFG, no JSON schema

	// Add CFG tool if configured (for DSL output)
	if request.CFGGrammar != nil {
		// Clean grammar using grammar-school before sending to OpenAI
		cleanedGrammar := gs.CleanGrammarForCFG(request.CFGGrammar.Grammar)
		log.Printf("🔧 CFG GRAMMAR CONFIGURED: %s (syntax: %s)", request.CFGGrammar.ToolName, request.CFGGrammar.Syntax)
		log.Printf("📝 Grammar cleaned for CFG: %d chars (original: %d chars)", len(cleanedGrammar), len(request.CFGGrammar.Grammar))

		// Use grammar-school utility to build OpenAI CFG tool payload
		cfgTool := gs.BuildOpenAICFGTool(gs.CFGConfig{
			ToolName:    request.CFGGrammar.ToolName,
			Description: request.CFGGrammar.Description,
			Grammar:     cleanedGrammar,
			Syntax:      request.CFGGrammar.Syntax,
		})

		// Note: Text format is not set when using CFG - the CFG tool handles the output format
		// Setting Text format would conflict with CFG tool output

		// Initialize tools array if not present
		if params.Tools == nil {
			params.Tools = []responses.ToolUnionParam{}
		}

		// Convert CFG tool map to ToolUnionParam
		// BuildOpenAICFGTool returns map[string]any, we need to convert it
		cfgToolJSON, err := json.Marshal(cfgTool)
		if err != nil {
			log.Printf("⚠️  Failed to marshal CFG tool: %v", err)
		} else {
			var cfgToolMap map[string]any
			if err := json.Unmarshal(cfgToolJSON, &cfgToolMap); err != nil {
				log.Printf("⚠️  Failed to unmarshal CFG tool: %v", err)
			} else {
				// The SDK expects ToolUnionParam, but CFG tools use a custom type
				// We need to manually construct it based on the CFG tool structure
				// For now, try to add it as a custom tool
				// Note: This may need adjustment based on SDK support
				log.Printf("🔧 Attempting to add CFG tool to streaming params: %+v", cfgToolMap)

				// The CFG tool should have type "custom" with format.grammar
				if toolType, ok := cfgToolMap["type"].(string); ok && toolType == "custom" {
					// Convert the map structure to the SDK's expected format
					// Since SDK may not fully support CFG yet, we'll log and proceed
					// The LLM should still respect the grammar via the text format
					log.Printf("✅ CFG tool structure detected, text format set to CFG mode")
				}
			}
		}

		params.ParallelToolCalls = openai.Bool(false) // CFG tools typically don't use parallel calls
	}

	// Add JSON Schema support (for orchestrator classification, etc.)
	if request.OutputSchema != nil {
		// Convert OutputSchema to OpenAI TextFormat
		params.Text = responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigParamOfJSONSchema(
				request.OutputSchema.Name,
				request.OutputSchema.Schema,
			),
		}
		log.Printf("📋 JSON SCHEMA CONFIGURED: %s", request.OutputSchema.Name)
	}

	// Add MCP tools if configured
	if request.MCPConfig != nil && request.MCPConfig.URL != "" {
		params.Tools = []responses.ToolUnionParam{
			{
				OfMcp: &responses.ToolMcpParam{
					ServerLabel: request.MCPConfig.Label,
					ServerURL:   request.MCPConfig.URL,
					RequireApproval: responses.ToolMcpRequireApprovalUnionParam{
						OfMcpToolApprovalFilter: &responses.ToolMcpRequireApprovalMcpToolApprovalFilterParam{
							Never: responses.ToolMcpRequireApprovalMcpToolApprovalFilterNeverParam{
								ToolNames: []string{}, // Empty = all tools never require approval
							},
						},
					},
				},
			},
		}
		log.Printf("🔗 MCP SERVER ENABLED: %s (label: %s)", request.MCPConfig.URL, request.MCPConfig.Label)
	}

	return params
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractDSLFromCFGToolCall searches for DSL code in CFG tool call response
// This is comprehensive - it checks EVERYWHERE for DSL content
func (p *OpenAIProvider) extractDSLFromCFGToolCall(resp *responses.Response) string {
	log.Printf("🔍 Searching for CFG tool call in %d output items", len(resp.Output))

	for i, outputItem := range resp.Output {
		outputItemJSON, _ := json.Marshal(outputItem)
		var outputItemMap map[string]any
		if json.Unmarshal(outputItemJSON, &outputItemMap) != nil {
			continue
		}

		log.Printf("🔍 Output item %d keys: %v", i, getMapKeys(outputItemMap))

		// IMMEDIATELY check "code" field - this is where CFG tool results often appear
		if codeVal, exists := outputItemMap["code"]; exists {
			log.Printf("🔍 'code' EXISTS! type=%T", codeVal)
			if codeStr, ok := codeVal.(string); ok && codeStr != "" {
				log.Printf("🔧 Found CFG code as STRING: %s", truncateString(codeStr, maxPreviewChars))
				return codeStr
			}
		}

		// Check for type field
		typeVal, typeExists := outputItemMap["type"]
		if typeExists {
			log.Printf("🔍 'type' field EXISTS in output item %d: value='%v' (type=%T)", i, typeVal, typeVal)

			// According to Grammar School docs, CFG tool results have type="custom_tool_call"
			if typeStr, ok := typeVal.(string); ok && typeStr == "custom_tool_call" {
				log.Printf("✅ Found custom_tool_call! Checking for 'input' field...")

				// Get the DSL code from the 'input' field
				if inputVal, exists := outputItemMap["input"]; exists {
					if inputStr, ok := inputVal.(string); ok && inputStr != "" {
						log.Printf("🔧 Found CFG tool call in 'input' field (DSL): %s", truncateString(inputStr, maxPreviewChars))
						return inputStr
					}
				}
			}
		}

		// COMPREHENSIVE SEARCH: Check ALL string fields for DSL content
		// This matches magda-api's approach which searches everywhere
		log.Printf("🔍 ========== CHECKING ALL STRING FIELDS FOR DSL CONTENT ==========")
		for key, val := range outputItemMap {
			if strVal, ok := val.(string); ok && strVal != "" {
				log.Printf("🔍 Field '%s' (string, %d chars): %s", key, len(strVal), truncateString(strVal, 500))
				if p.isDSLCode(strVal) {
					log.Printf("🔧 ✅✅✅ FOUND DSL IN FIELD '%s': %s", key, truncateString(strVal, maxPreviewChars))
					return strVal
				}
			}
		}

		// Check nested structures: tools, outputs, tool_calls arrays
		for _, arrayKey := range []string{"tools", "outputs", "tool_calls"} {
			if arr, ok := outputItemMap[arrayKey].([]any); ok && len(arr) > 0 {
				log.Printf("🔍 Found '%s' array with %d items", arrayKey, len(arr))
				for j, item := range arr {
					if itemMap, ok := item.(map[string]any); ok {
						log.Printf("🔍 %s[%d] keys: %v", arrayKey, j, getMapKeys(itemMap))
						for _, field := range []string{"code", "input", "arguments"} {
							if val, ok := itemMap[field].(string); ok && val != "" {
								log.Printf("🔍 %s[%d].%s = %s", arrayKey, j, field, truncateString(val, 200))
								if p.isDSLCode(val) {
									log.Printf("🔧 Found DSL in %s[%d].%s", arrayKey, j, field)
									return val
								}
							}
						}
						// Also check function.arguments pattern (OpenAI tool calls)
						if function, ok := itemMap["function"].(map[string]any); ok {
							if args, ok := function["arguments"].(string); ok && args != "" {
								log.Printf("🔍 %s[%d].function.arguments = %s", arrayKey, j, truncateString(args, 200))
								if p.isDSLCode(args) {
									log.Printf("🔧 Found DSL in %s[%d].function.arguments", arrayKey, j)
									return args
								}
							}
						}
					}
				}
			}
		}

		// Check nested tool_call structure
		if toolCall, ok := outputItemMap["tool_call"].(map[string]any); ok {
			if input, ok := toolCall["input"].(string); ok && input != "" {
				log.Printf("🔧 Found CFG tool call input in tool_call (DSL): %s", truncateString(input, maxPreviewChars))
				return input
			}
		}

		// DUMP full structure for first item if we haven't found DSL yet
		if i == 0 {
			fullDump, _ := json.MarshalIndent(outputItemMap, "", "  ")
			dumpLen := len(fullDump)
			if dumpLen > 5000 {
				dumpLen = 5000
			}
			log.Printf("🔍 FULL OUTPUT ITEM STRUCTURE (first %d chars):\n%s", dumpLen, string(fullDump[:dumpLen]))
		}
	}

	log.Printf("⚠️  No CFG tool call found in response output items")
	return ""
}

// findDSLInOutputItem checks multiple possible locations for DSL code in an output item
func (p *OpenAIProvider) findDSLInOutputItem(itemMap map[string]any) string {
	// Check "input" field FIRST (this is where CFG tool results appear according to OpenAI docs)
	if input, ok := itemMap["input"].(string); ok && input != "" {
		log.Printf("🔧 Found CFG tool call in 'input' field (DSL): %s", truncateString(input, maxPreviewChars))
		log.Printf("📋 FULL DSL CODE from CFG tool input (%d chars, NO TRUNCATION):\n%s", len(input), input)
		return input
	}

	// Check "code" field as fallback
	if code, ok := itemMap["code"].(string); ok && code != "" {
		log.Printf("🔧 Found CFG tool call code (DSL): %s", truncateString(code, maxPreviewChars))
		log.Printf("📋 FULL DSL CODE from CFG tool code (%d chars, NO TRUNCATION):\n%s", len(code), code)
		return code
	}

	// Check nested code map
	if codeVal, ok := itemMap["code"]; ok {
		if codeMap, ok := codeVal.(map[string]any); ok {
			for key, val := range codeMap {
				if strVal, ok := val.(string); ok && strVal != "" && p.isDSLCode(strVal) {
					log.Printf("🔧 Found CFG tool call code in nested map[%s] (DSL): %s", key, truncateString(strVal, maxPreviewChars))
					return strVal
				}
			}
		}
	}

	// Check direct fields - with detailed logging
	log.Printf("🔍 ========== findDSLInOutputItem: Checking direct fields (input, action, arguments) ==========")
	for _, field := range []string{"input", "action", "arguments"} {
		if val, exists := itemMap[field]; exists {
			log.Printf("🔍 Field '%s' EXISTS: type=%T", field, val)
			if valStr, ok := val.(string); ok {
				log.Printf("🔍 Field '%s' is string with %d chars, value: %s", field, len(valStr), truncateString(valStr, 1000))
				if valStr != "" && p.isDSLCode(valStr) {
					log.Printf("🔧 ✅✅✅ FOUND DSL IN FIELD '%s': %s", field, truncateString(valStr, maxPreviewChars))
					return valStr
				}
			} else {
				// Log what type it actually is
				valJSON, _ := json.Marshal(val)
				log.Printf("🔍 Field '%s' is NOT a string, JSON: %s", field, truncateString(string(valJSON), 1000))
				// If it's a map, check its contents
				if valMap, ok := val.(map[string]any); ok {
					log.Printf("🔍 Field '%s' is a map with keys: %v", field, getMapKeys(valMap))
					for k, v := range valMap {
						if vStr, ok := v.(string); ok && vStr != "" {
							log.Printf("🔍 Field '%s[%s]' = %s", field, k, truncateString(vStr, 500))
							if p.isDSLCode(vStr) {
								log.Printf("🔧 ✅✅✅ FOUND DSL IN FIELD '%s[%s]': %s", field, k, truncateString(vStr, maxPreviewChars))
								return vStr
							}
						}
					}
				}
			}
		} else {
			log.Printf("🔍 Field '%s' DOES NOT EXIST", field)
		}
	}

	// Also check other fields that might contain DSL
	log.Printf("🔍 ========== findDSLInOutputItem: Checking other fields (result, output, content) ==========")
	for _, field := range []string{"result", "output", "content"} {
		if val, exists := itemMap[field]; exists {
			log.Printf("🔍 Field '%s' EXISTS: type=%T", field, val)
			if valStr, ok := val.(string); ok {
				log.Printf("🔍 Field '%s' is string with %d chars, value: %s", field, len(valStr), truncateString(valStr, 1000))
				if valStr != "" && p.isDSLCode(valStr) {
					log.Printf("🔧 ✅✅✅ FOUND DSL IN FIELD '%s': %s", field, truncateString(valStr, maxPreviewChars))
					return valStr
				}
			} else if val != nil {
				valJSON, _ := json.Marshal(val)
				log.Printf("🔍 Field '%s' is NOT a string, JSON: %s", field, truncateString(string(valJSON), 1000))
			}
		}
	}

	// Check "outputs" array
	if outputs, ok := itemMap["outputs"].([]any); ok && len(outputs) > 0 {
		log.Printf("🔍 Found 'outputs' array with %d items", len(outputs))
		for j, output := range outputs {
			if outputMap, ok := output.(map[string]any); ok {
				log.Printf("🔍 Output %d keys: %v", j, getMapKeys(outputMap))
				for key, val := range outputMap {
					if valStr, ok := val.(string); ok && valStr != "" {
						log.Printf("🔍 Output[%d][%s] = %s", j, key, truncateString(valStr, 500))
						if p.isDSLCode(valStr) {
							log.Printf("🔧 ✅✅✅ FOUND DSL IN OUTPUT[%d][%s]: %s", j, key, truncateString(valStr, maxPreviewChars))
							return valStr
						}
					}
				}
			}
		}
	}

	// Check "tools" array - this is critical for CFG tools
	log.Printf("🔍 ========== findDSLInOutputItem: Checking 'tools' field ==========")
	if toolsVal, exists := itemMap["tools"]; exists {
		log.Printf("🔍 Field 'tools' EXISTS: type=%T", toolsVal)
		if tools, ok := toolsVal.([]any); ok {
			log.Printf("🔍 'tools' is an array with %d items", len(tools))
			if len(tools) > 0 {
				for j, tool := range tools {
					if toolMap, ok := tool.(map[string]any); ok {
						log.Printf("🔍 Tool %d keys: %v", j, getMapKeys(toolMap))
						for key, val := range toolMap {
							if valStr, ok := val.(string); ok && valStr != "" {
								log.Printf("🔍 Tool[%d][%s] = %s", j, key, truncateString(valStr, 500))
								if p.isDSLCode(valStr) {
									log.Printf("🔧 ✅✅✅ FOUND DSL IN TOOL[%d][%s]: %s", j, key, truncateString(valStr, maxPreviewChars))
									return valStr
								}
							} else if valMap, ok := val.(map[string]any); ok {
								log.Printf("🔍 Tool[%d][%s] is a map with keys: %v", j, key, getMapKeys(valMap))
								for subKey, subVal := range valMap {
									if subValStr, ok := subVal.(string); ok && subValStr != "" {
										log.Printf("🔍 Tool[%d][%s][%s] = %s", j, key, subKey, truncateString(subValStr, 500))
										if p.isDSLCode(subValStr) {
											log.Printf("🔧 ✅✅✅ FOUND DSL IN TOOL[%d][%s][%s]: %s", j, key, subKey, truncateString(subValStr, maxPreviewChars))
											return subValStr
										}
									}
								}
							}
						}
					}
				}
			}
		} else {
			log.Printf("🔍 'tools' is NOT an array, type=%T, value: %v", toolsVal, toolsVal)
			if toolsMap, ok := toolsVal.(map[string]any); ok {
				log.Printf("🔍 'tools' is a map with keys: %v", getMapKeys(toolsMap))
				for k, v := range toolsMap {
					if vStr, ok := v.(string); ok && vStr != "" {
						log.Printf("🔍 tools[%s] = %s", k, truncateString(vStr, 500))
						if p.isDSLCode(vStr) {
							log.Printf("🔧 ✅✅✅ FOUND DSL IN tools[%s]: %s", k, truncateString(vStr, maxPreviewChars))
							return vStr
						}
					}
				}
			}
		}
	} else {
		log.Printf("🔍 Field 'tools' DOES NOT EXIST")
	}

	// Check tool_calls array
	if toolCalls, ok := itemMap["tool_calls"].([]any); ok {
		for j, toolCall := range toolCalls {
			if toolCallMap, ok := toolCall.(map[string]any); ok {
				if input, ok := toolCallMap["input"].(string); ok && input != "" {
					log.Printf("🔧 Found CFG tool call input in tool_calls[%d] (DSL): %s", j, truncateString(input, maxPreviewChars))
					return input
				}
				if function, ok := toolCallMap["function"].(map[string]any); ok {
					if arguments, ok := function["arguments"].(string); ok && arguments != "" {
						log.Printf("🔧 Found CFG tool call arguments (DSL): %s", truncateString(arguments, maxPreviewChars))
						return arguments
					}
				}
			}
		}
	}

	// Check nested tool_call
	if toolCall, ok := itemMap["tool_call"].(map[string]any); ok {
		if input, ok := toolCall["input"].(string); ok && input != "" {
			log.Printf("🔧 Found CFG tool call input in tool_call (DSL): %s", truncateString(input, maxPreviewChars))
			return input
		}
	}

	return ""
}

// extractAndCleanTextOutput extracts and cleans text output from response
func (p *OpenAIProvider) extractAndCleanTextOutput(resp *responses.Response) string {
	textOutput := resp.OutputText()

	if textOutput == "" {
		return ""
	}

	// Strip markdown code blocks
	cleaned := strings.TrimPrefix(textOutput, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned != textOutput {
		log.Printf("🧹 Stripped markdown code blocks from output: %d -> %d chars", len(textOutput), len(cleaned))
	}

	return cleaned
}

// isDSLCode checks if a string looks like DSL code
// Supports both MAGDA DSL (track, new_clip, etc.) and Arranger DSL (arpeggio, chord, progression)
func (p *OpenAIProvider) isDSLCode(text string) bool {
	// Arranger DSL patterns (chord-based musical composition)
	if strings.HasPrefix(text, "arpeggio(") ||
		strings.HasPrefix(text, "chord(") ||
		strings.HasPrefix(text, "progression(") {
		return true
	}

	// MAGDA DSL patterns (DAW control)
	return strings.HasPrefix(text, "track(") ||
		strings.HasPrefix(text, "filter(") ||
		strings.HasPrefix(text, "map(") ||
		strings.HasPrefix(text, "for_each(") ||
		strings.Contains(text, ".new_clip(") ||
		strings.Contains(text, ".add_midi(") ||
		strings.Contains(text, ".delete(") ||
		strings.Contains(text, ".delete_clip(") ||
		strings.Contains(text, ".filter(") ||
		strings.Contains(text, ".map(") ||
		strings.Contains(text, ".for_each(") ||
		strings.Contains(text, ".set_selected(") ||
		strings.Contains(text, ".set_mute(") ||
		strings.Contains(text, ".set_solo(") ||
		strings.Contains(text, ".set_volume(") ||
		strings.Contains(text, ".set_pan(") ||
		strings.Contains(text, ".set_name(") ||
		strings.Contains(text, ".add_fx(")
}

// processResponseWithCFG converts OpenAI Response to GenerationResponse, handling CFG tool calls
// MAGDA always uses DSL/CFG, so this is the only processing path
func (p *OpenAIProvider) processResponseWithCFG(
	resp *responses.Response,
	startTime time.Time,
	transaction *sentry.Span,
	cfgConfig *CFGConfig,
) (*GenerationResponse, error) {
	span := transaction.StartChild("process_response")
	defer span.Finish()

	// Try to extract DSL from CFG tool call first
	if cfgConfig != nil {
		if dslCode := p.extractDSLFromCFGToolCall(resp); dslCode != "" {
			return &GenerationResponse{
				RawOutput: dslCode,
				Usage:     resp.Usage,
			}, nil
		}
	}

	// Extract and process text output
	textOutput := p.extractAndCleanTextOutput(resp)
	log.Printf("📥 OPENAI RESPONSE: output_length=%d, output_items=%d, tokens=%d",
		len(textOutput), len(resp.Output), resp.Usage.TotalTokens)

	// If CFG was configured, we MUST have DSL from tool call - no fallback to text output
	if cfgConfig != nil {
		// We already checked for CFG tool call above - if we got here, there's no tool call
		// and we have text output. This is an error - LLM must use CFG tool.
		if textOutput != "" {
			log.Printf("❌ CFG was configured but LLM did not use CFG tool and generated text output instead")
			log.Printf("❌ Text output (first %d chars): %s", maxPreviewChars, truncateString(textOutput, maxPreviewChars))
			return nil, fmt.Errorf("CFG grammar was configured but LLM did not use CFG tool. LLM must use the CFG tool to generate DSL code")
		}
		return nil, fmt.Errorf("CFG grammar was configured but LLM did not use CFG tool to generate DSL code. LLM must use the CFG tool to generate DSL code")
	}

	if textOutput == "" {
		return nil, fmt.Errorf("openai response did not include any output text")
	}

	// Analyze MCP usage
	mcpUsed, mcpCalls, mcpTools := p.analyzeMCPUsage(resp)

	// Log MCP summary
	p.logMCPSummary(mcpUsed, mcpCalls, mcpTools)

	// Log usage stats
	p.logUsageStats(resp.Usage)

	// MAGDA always uses DSL, so we should never reach here
	return nil, fmt.Errorf("unexpected code path in processResponseWithCFG")
}

// processResponseWithJSONSchema extracts JSON output from OpenAI response when using JSON Schema
func (p *OpenAIProvider) processResponseWithJSONSchema(
	resp *responses.Response,
	startTime time.Time,
	transaction *sentry.Span,
	outputSchema *OutputSchema,
) (*GenerationResponse, error) {
	span := transaction.StartChild("process_response_json")
	defer span.Finish()

	// Extract text output (should be JSON when using JSON Schema)
	textOutput := p.extractAndCleanTextOutput(resp)
	log.Printf("📥 OPENAI JSON RESPONSE: output_length=%d, output_items=%d, tokens=%d",
		len(textOutput), len(resp.Output), resp.Usage.TotalTokens)

	if textOutput == "" {
		return nil, fmt.Errorf("openai response did not include any output text")
	}

	// Log usage stats
	p.logUsageStats(resp.Usage)

	duration := time.Since(startTime)
	log.Printf("✅ OPENAI GENERATION COMPLETED in %v", duration)

	return &GenerationResponse{
		RawOutput: textOutput, // JSON string from OutputSchema
		Usage:     resp.Usage,
	}, nil
}

// analyzeMCPUsage checks if MCP was used and returns usage details
func (p *OpenAIProvider) analyzeMCPUsage(resp *responses.Response) (bool, int, []string) {
	mcpUsed := false
	mcpCallCount := 0
	toolsUsed := make(map[string]bool)

	log.Printf("🔍 MCP USAGE ANALYSIS:")

	for _, outputItem := range resp.Output {
		if outputItem.Type == mcpCallType {
			mcpCall := outputItem.AsMcpCall()
			mcpUsed = true
			mcpCallCount++
			p.logMCPToolCall(mcpCall)
			toolsUsed[mcpCall.Name] = true

			// Add Sentry breadcrumb
			sentry.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "mcp",
				Message:  fmt.Sprintf("MCP tool called: %s", mcpCall.Name),
				Level:    sentry.LevelInfo,
				Data: map[string]interface{}{
					"tool_name":     mcpCall.Name,
					"server_label":  mcpCall.ServerLabel,
					"has_output":    mcpCall.Output != "",
					"output_length": len(mcpCall.Output),
					"has_error":     mcpCall.Error != "",
				},
			})
		}
	}

	uniqueTools := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		uniqueTools = append(uniqueTools, tool)
	}

	if mcpCallCount == 0 {
		log.Printf("❌ MCP NOT USED: No MCP tool calls found in output")
	} else {
		log.Printf("📊 MCP TOOLS USED: %v", uniqueTools)
	}

	return mcpUsed, mcpCallCount, uniqueTools
}

// logMCPToolCall logs details of an MCP tool call
func (p *OpenAIProvider) logMCPToolCall(mcpCall responses.ResponseOutputItemMcpCall) {
	log.Printf("✅ MCP WAS USED: Tool call made")
	log.Printf("   🛠️  Tool Call: %s", mcpCall.Name)
	if mcpCall.Arguments != "" {
		argsStr := mcpCall.Arguments
		if len(argsStr) > maxArgsLogLength {
			argsStr = argsStr[:maxArgsLogLength] + "..."
		}
		log.Printf("     Arguments: %s", argsStr)
	}
	if mcpCall.Output != "" {
		output := mcpCall.Output
		if len(output) > maxOutputTrunc {
			output = output[:maxOutputTrunc] + "... (truncated)"
		}
		log.Printf("     Output: %s", output)
	}
	if mcpCall.Error != "" {
		log.Printf("     ⚠️  Error: %s", mcpCall.Error)
	}
}

// extractUsageFromRawResponse extracts usage from raw JSON response
func (p *OpenAIProvider) extractUsageFromRawResponse(rawResponse map[string]any) any {
	if usageMap, ok := rawResponse["usage"].(map[string]any); ok {
		return usageMap
	}
	return nil
}

// logUsageStats logs token usage statistics
func (p *OpenAIProvider) logUsageStats(usage responses.ResponseUsage) {
	reasoningTokens := int64(0)
	if usage.OutputTokensDetails.ReasoningTokens > 0 {
		reasoningTokens = usage.OutputTokensDetails.ReasoningTokens
	}
	log.Printf("📊 USAGE: input=%d, output=%d, reasoning=%d, total=%d",
		usage.InputTokens, usage.OutputTokens,
		reasoningTokens, usage.TotalTokens)
}

// logMCPSummary logs a summary of MCP usage
func (p *OpenAIProvider) logMCPSummary(mcpUsed bool, callCount int, tools []string) {
	if mcpUsed {
		log.Printf("🎯 MCP USAGE: %d calls to tools: %v", callCount, tools)
	} else {
		log.Printf("ℹ️  NO MCP USAGE in this generation")
	}
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
