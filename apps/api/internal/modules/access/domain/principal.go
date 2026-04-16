package domain

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Principal struct {
	UserID   string
	Role     Role
	Rating   int
	Sanction Sanction
}

type Sanction struct {
	CreateArticlesBlocked bool
	CommentsBlocked       bool
	ChatBlocked           bool
	ReportsBlocked        bool
}

func (p Principal) IsAdmin() bool {
	return p.Role == RoleAdmin
}

func (p Principal) CanCreateArticle() bool {
	return p.IsAdmin() || (p.Rating >= 0 && !p.Sanction.CreateArticlesBlocked)
}

func (p Principal) CanComment() bool {
	return p.IsAdmin() || (p.Rating >= -100 && !p.Sanction.CommentsBlocked)
}

func (p Principal) CanChat() bool {
	return p.IsAdmin() || (p.Rating >= -100 && !p.Sanction.ChatBlocked)
}

func (p Principal) CanReport() bool {
	return p.IsAdmin() || (p.Rating >= 10 && !p.Sanction.ReportsBlocked)
}

func (p Principal) CanPublishDirectly() bool {
	return p.IsAdmin() || p.Rating >= 1000
}

func (p Principal) CanModerate() bool {
	return p.IsAdmin() || p.Rating >= 10000
}

func (p Principal) CanChatWithDevLead() bool {
	return p.IsAdmin() || p.Rating > 100000
}
