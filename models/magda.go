package models

// MagdaActionsOutput represents the structured output from MAGDA LLM
type MagdaActionsOutput struct {
	Actions []map[string]interface{} `json:"actions"`
}
