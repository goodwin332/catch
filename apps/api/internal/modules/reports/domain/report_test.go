package domain

import "testing"

func TestNormalizeRequiresDetailsForOther(t *testing.T) {
	if _, _, _, _, err := Normalize("article", "id", "other", ""); err == nil {
		t.Fatal("other reason without details must be rejected")
	}
}

func TestRequiredDecisions(t *testing.T) {
	if RequiredDecisions(TargetTypeComment, DecisionAccept) != 3 {
		t.Fatal("comment accept threshold must be 3")
	}
	if RequiredDecisions(TargetTypeArticle, DecisionReject) != 10 {
		t.Fatal("article reject threshold must be 10")
	}
}
