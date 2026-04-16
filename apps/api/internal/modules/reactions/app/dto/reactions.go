package dto

type SetReactionRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Value      int    `json:"value"`
}

type ReactionResponse struct {
	TargetType    string `json:"target_type"`
	TargetID      string `json:"target_id"`
	Value         int    `json:"value"`
	ReactionsUp   int    `json:"reactions_up"`
	ReactionsDown int    `json:"reactions_down"`
	ReactionScore int    `json:"reaction_score"`
}
