package domain

import "testing"

func TestPrincipalCapabilitiesFollowRatingThresholds(t *testing.T) {
	cases := []struct {
		name               string
		rating             int
		canCreateArticle   bool
		canComment         bool
		canChat            bool
		canReport          bool
		canPublishDirectly bool
		canModerate        bool
		canChatWithDevLead bool
	}{
		{name: "negative rating cannot create articles", rating: -1, canCreateArticle: false, canComment: true, canChat: true},
		{name: "below minus one hundred cannot comment or chat", rating: -101},
		{name: "default user can create and react later", rating: 0, canCreateArticle: true, canComment: true, canChat: true},
		{name: "rating ten can report", rating: 10, canCreateArticle: true, canComment: true, canChat: true, canReport: true},
		{name: "rating one thousand can publish directly", rating: 1000, canCreateArticle: true, canComment: true, canChat: true, canReport: true, canPublishDirectly: true},
		{name: "rating ten thousand can moderate", rating: 10000, canCreateArticle: true, canComment: true, canChat: true, canReport: true, canPublishDirectly: true, canModerate: true},
		{name: "over one hundred thousand can chat with dev lead", rating: 100001, canCreateArticle: true, canComment: true, canChat: true, canReport: true, canPublishDirectly: true, canModerate: true, canChatWithDevLead: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			principal := Principal{Role: RoleUser, Rating: tt.rating}

			if principal.CanCreateArticle() != tt.canCreateArticle {
				t.Fatalf("CanCreateArticle() = %v, want %v", principal.CanCreateArticle(), tt.canCreateArticle)
			}
			if principal.CanComment() != tt.canComment {
				t.Fatalf("CanComment() = %v, want %v", principal.CanComment(), tt.canComment)
			}
			if principal.CanChat() != tt.canChat {
				t.Fatalf("CanChat() = %v, want %v", principal.CanChat(), tt.canChat)
			}
			if principal.CanReport() != tt.canReport {
				t.Fatalf("CanReport() = %v, want %v", principal.CanReport(), tt.canReport)
			}
			if principal.CanPublishDirectly() != tt.canPublishDirectly {
				t.Fatalf("CanPublishDirectly() = %v, want %v", principal.CanPublishDirectly(), tt.canPublishDirectly)
			}
			if principal.CanModerate() != tt.canModerate {
				t.Fatalf("CanModerate() = %v, want %v", principal.CanModerate(), tt.canModerate)
			}
			if principal.CanChatWithDevLead() != tt.canChatWithDevLead {
				t.Fatalf("CanChatWithDevLead() = %v, want %v", principal.CanChatWithDevLead(), tt.canChatWithDevLead)
			}
		})
	}
}

func TestAdminBypassesRatingThresholds(t *testing.T) {
	principal := Principal{Role: RoleAdmin, Rating: -1000000}

	if !principal.CanCreateArticle() || !principal.CanComment() || !principal.CanChat() || !principal.CanReport() || !principal.CanPublishDirectly() || !principal.CanModerate() || !principal.CanChatWithDevLead() {
		t.Fatal("admin must bypass rating thresholds")
	}
}
