package dto

type RequestEmailCodeRequest struct {
	Email string `json:"email"`
}

type RequestEmailCodeResponse struct {
	Status  string `json:"status"`
	DevCode string `json:"dev_code,omitempty"`
}

type VerifyEmailCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type DevLoginRequest struct {
	Email string `json:"email,omitempty"`
}

type CurrentUserResponse struct {
	User         UserDTO         `json:"user"`
	Capabilities CapabilitiesDTO `json:"capabilities"`
}

type UserDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Rating      int    `json:"rating"`
	Role        string `json:"role"`
}

type CapabilitiesDTO struct {
	CanCreateArticle   bool `json:"can_create_article"`
	CanComment         bool `json:"can_comment"`
	CanChat            bool `json:"can_chat"`
	CanReport          bool `json:"can_report"`
	CanPublishDirectly bool `json:"can_publish_directly"`
	CanModerate        bool `json:"can_moderate"`
	CanChatWithDevLead bool `json:"can_chat_with_dev_lead"`
}
