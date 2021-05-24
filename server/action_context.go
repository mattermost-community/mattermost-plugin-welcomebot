package main

// ActionContext passed from action buttons
type ActionContext struct {
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	Action string `json:"action"`
	DirectMessagePost string `json:"direct_message_post"`
}

// Action type for decoding action buttons
type Action struct {
	UserID  string         `json:"user_id"`
	Context *ActionContext `json:"context"`
}
