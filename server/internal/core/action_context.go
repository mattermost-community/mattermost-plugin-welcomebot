package core

// ActionContext passed from action buttons
type ActionContext struct {
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

// Action type for decoding action buttons
type Action struct {
	Context *ActionContext `json:"context"`
	UserID  string         `json:"user_id"`
}
