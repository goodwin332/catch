//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"catch/apps/api/internal/app/bootstrap"
	"catch/apps/api/internal/app/composition"
	"catch/apps/api/internal/app/config"
	"catch/apps/api/internal/platform/db"
	"catch/apps/api/internal/platform/logger"
	"catch/apps/api/internal/platform/mail"
	"catch/apps/api/internal/platform/outbox"
	"catch/apps/api/internal/platform/search"
)

type testSession struct {
	UserID string
	Cookie *http.Cookie
	CSRF   *http.Cookie
}

func TestModerationApprovalPublishesLowRatingDraft(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "moderation-author@catch.local")
	admin := loginUser(t, server, cfg, "moderation-admin@catch.local")
	setUserRoleAndRating(t, container, admin.UserID, "admin", 10000)

	draft := createDraft(t, server, author, "Проверка модерации", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Материал для очереди модерации."}]}`)
	submitted := submitDraft(t, server, author, draft.ID)
	if submitted.Status != "in_moderation" || !submitted.ModerationRequired {
		t.Fatalf("submitted draft status = %q moderation_required=%v, want in_moderation true", submitted.Status, submitted.ModerationRequired)
	}

	listResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/moderation/submissions?limit=10", http.MethodGet, "", http.StatusOK, admin, nil)
	defer listResponse.Body.Close()
	var submissions struct {
		Items []struct {
			ID        string `json:"id"`
			ArticleID string `json:"article_id"`
			Status    string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&submissions); err != nil {
		t.Fatalf("decode moderation submissions: %v", err)
	}
	if len(submissions.Items) != 1 {
		t.Fatalf("submissions length = %d, want 1", len(submissions.Items))
	}
	if submissions.Items[0].ArticleID != draft.ID || submissions.Items[0].Status != "pending" {
		t.Fatalf("submission = %+v, want pending for article %s", submissions.Items[0], draft.ID)
	}

	approveResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/moderation/submissions/"+submissions.Items[0].ID+"/approve", http.MethodPost, "", http.StatusOK, admin, nil)
	defer approveResponse.Body.Close()
	var approved struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(approveResponse.Body).Decode(&approved); err != nil {
		t.Fatalf("decode approved submission: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("approved status = %q, want approved", approved.Status)
	}

	published := submitDraft(t, server, author, draft.ID)
	if published.Status != "published" || published.ModerationRequired {
		t.Fatalf("published draft status = %q moderation_required=%v, want published false", published.Status, published.ModerationRequired)
	}

	publicResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/"+draft.ID, http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer publicResponse.Body.Close()
}

func TestReportsChatAndMediaWorkflows(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "workflow-author@catch.local")
	reporter := loginUser(t, server, cfg, "workflow-reporter@catch.local")
	admin := loginUser(t, server, cfg, "workflow-admin@catch.local")
	setUserRoleAndRating(t, container, author.UserID, "user", 1000)
	setUserRoleAndRating(t, container, reporter.UserID, "user", 20)
	setUserRoleAndRating(t, container, admin.UserID, "admin", 10000)

	fileID := uploadPNG(t, server, author)
	unreferencedFileID := uploadPNG(t, server, author)
	draft := createDraft(t, server, author, "Проверка жалоб и файлов", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Материал с вложенным файлом."},{"type":"image","file_id":"`+fileID+`"}]}`)
	published := submitDraft(t, server, author, draft.ID)
	if published.Status != "published" {
		t.Fatalf("published status = %q, want published", published.Status)
	}

	comment := createComment(t, server, reporter, draft.ID, "Первый отзыв по статье")
	editResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/comments/"+comment.ID, http.MethodPatch, `{"body":"Обновлённый отзыв по статье"}`, http.StatusOK, reporter, nil)
	defer editResponse.Body.Close()
	var editedComment struct {
		Body     string `json:"body"`
		EditedAt string `json:"edited_at"`
	}
	if err := json.NewDecoder(editResponse.Body).Decode(&editedComment); err != nil {
		t.Fatalf("decode edited comment: %v", err)
	}
	if editedComment.Body != "Обновлённый отзыв по статье" || editedComment.EditedAt == "" {
		t.Fatalf("edited comment = %+v, want updated body and edited_at", editedComment)
	}
	permalinkResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/comments/"+comment.ID, http.MethodGet, "", http.StatusOK, testSession{}, nil)
	permalinkResponse.Body.Close()

	articleReactionResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reactions", http.MethodPost, `{"target_type":"article","target_id":"`+draft.ID+`","value":1}`, http.StatusOK, reporter, nil)
	defer articleReactionResponse.Body.Close()
	var articleReaction struct {
		ReactionsUp   int `json:"reactions_up"`
		ReactionScore int `json:"reaction_score"`
	}
	if err := json.NewDecoder(articleReactionResponse.Body).Decode(&articleReaction); err != nil {
		t.Fatalf("decode article reaction: %v", err)
	}
	if articleReaction.ReactionsUp != 1 || articleReaction.ReactionScore != 1 {
		t.Fatalf("article reaction = %+v, want one positive reaction", articleReaction)
	}

	commentReactionResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reactions", http.MethodPost, `{"target_type":"comment","target_id":"`+comment.ID+`","value":-1}`, http.StatusOK, author, nil)
	commentReactionResponse.Body.Close()

	publicArticleResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/"+draft.ID, http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer publicArticleResponse.Body.Close()
	var publicArticle struct {
		ReactionScore int `json:"reaction_score"`
	}
	if err := json.NewDecoder(publicArticleResponse.Body).Decode(&publicArticle); err != nil {
		t.Fatalf("decode public article: %v", err)
	}
	if publicArticle.ReactionScore != 1 {
		t.Fatalf("public article reaction score = %d, want 1", publicArticle.ReactionScore)
	}

	contentResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/media/files/"+fileID+"/content", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	data, err := io.ReadAll(contentResponse.Body)
	contentResponse.Body.Close()
	if err != nil {
		t.Fatalf("read media content: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("media content is empty")
	}
	previewResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/media/files/"+fileID+"/preview", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	previewData, err := io.ReadAll(previewResponse.Body)
	previewResponse.Body.Close()
	if err != nil {
		t.Fatalf("read media preview: %v", err)
	}
	if len(previewData) == 0 || previewResponse.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("media preview content-type=%q size=%d, want image/png with body", previewResponse.Header.Get("Content-Type"), len(previewData))
	}
	deleted, err := container.Media.CleanupUnreferenced(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("cleanup unreferenced media: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted media count = %d, want 1", deleted)
	}
	deletedContentResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/media/files/"+unreferencedFileID+"/content", http.MethodGet, "", http.StatusNotFound, testSession{}, nil)
	deletedContentResponse.Body.Close()
	referencedContentResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/media/files/"+fileID+"/content", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	referencedContentResponse.Body.Close()

	reportResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reports", http.MethodPost, `{"target_type":"article","target_id":"`+draft.ID+`","reason":"advertising"}`, http.StatusCreated, reporter, nil)
	defer reportResponse.Body.Close()
	var report struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(reportResponse.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Status != "pending" {
		t.Fatalf("report status = %q, want pending", report.Status)
	}

	decisionResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reports/"+report.ID+"/decisions", http.MethodPost, `{"decision":"accept"}`, http.StatusOK, admin, nil)
	defer decisionResponse.Body.Close()
	var decision struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(decisionResponse.Body).Decode(&decision); err != nil {
		t.Fatalf("decode report decision: %v", err)
	}
	if decision.Status != "accepted" {
		t.Fatalf("decision status = %q, want accepted", decision.Status)
	}
	reportOutbox := &recordingArticleIndexer{}
	processOutboxOnce(t, container, reportOutbox)
	if !reportOutbox.deletedArticle(draft.ID) {
		t.Fatalf("deleted search documents = %v, want article %s removed", reportOutbox.deleted, draft.ID)
	}

	removedResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/"+draft.ID, http.MethodGet, "", http.StatusNotFound, testSession{}, nil)
	removedResponse.Body.Close()

	commentReportResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reports", http.MethodPost, `{"target_type":"comment","target_id":"`+comment.ID+`","reason":"insult"}`, http.StatusCreated, author, nil)
	defer commentReportResponse.Body.Close()
	var commentReport struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(commentReportResponse.Body).Decode(&commentReport); err != nil {
		t.Fatalf("decode comment report: %v", err)
	}

	commentDecisionResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reports/"+commentReport.ID+"/decisions", http.MethodPost, `{"decision":"accept"}`, http.StatusOK, admin, nil)
	commentDecisionResponse.Body.Close()

	commentsResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/"+draft.ID+"/comments", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer commentsResponse.Body.Close()
	var comments struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(commentsResponse.Body).Decode(&comments); err != nil {
		t.Fatalf("decode comments: %v", err)
	}
	if len(comments.Items) != 1 || comments.Items[0].ID != comment.ID || comments.Items[0].Status != "deleted" {
		t.Fatalf("comments = %+v, want deleted reported comment", comments.Items)
	}

	conversationResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/chat/conversations", http.MethodPost, `{"recipient_id":"`+reporter.UserID+`"}`, http.StatusCreated, author, nil)
	defer conversationResponse.Body.Close()
	var conversation struct {
		ID        string   `json:"id"`
		MemberIDs []string `json:"member_ids"`
	}
	if err := json.NewDecoder(conversationResponse.Body).Decode(&conversation); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	if len(conversation.MemberIDs) != 2 {
		t.Fatalf("member ids = %v, want two members", conversation.MemberIDs)
	}

	messageResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/chat/conversations/"+conversation.ID+"/messages", http.MethodPost, `{"body":"Привет, видел отчёт по месту?"}`, http.StatusCreated, author, nil)
	defer messageResponse.Body.Close()
	var message struct {
		ID     string `json:"id"`
		Body   string `json:"body"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(messageResponse.Body).Decode(&message); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if message.Body == "" || message.Status != "sent" {
		t.Fatalf("message = %+v, want sent message", message)
	}
	secondMessageResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/chat/conversations/"+conversation.ID+"/messages", http.MethodPost, `{"body":"Добавил вторую точку маршрута."}`, http.StatusCreated, author, nil)
	defer secondMessageResponse.Body.Close()
	var secondMessage struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(secondMessageResponse.Body).Decode(&secondMessage); err != nil {
		t.Fatalf("decode second message: %v", err)
	}

	listMessagesResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/chat/conversations/"+conversation.ID+"/messages?limit=10", http.MethodGet, "", http.StatusOK, reporter, nil)
	defer listMessagesResponse.Body.Close()
	var messages struct {
		Items []struct {
			Body string `json:"body"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listMessagesResponse.Body).Decode(&messages); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(messages.Items) != 2 || messages.Items[0].Body != message.Body {
		t.Fatalf("messages = %+v, want two sent messages", messages.Items)
	}
	listAfterResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/chat/conversations/"+conversation.ID+"/messages?after_id="+message.ID, http.MethodGet, "", http.StatusOK, reporter, nil)
	defer listAfterResponse.Body.Close()
	var messagesAfter struct {
		Items []struct {
			Body string `json:"body"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listAfterResponse.Body).Decode(&messagesAfter); err != nil {
		t.Fatalf("decode messages after: %v", err)
	}
	if len(messagesAfter.Items) != 1 || messagesAfter.Items[0].Body != secondMessage.Body {
		t.Fatalf("messages after = %+v, want second message only", messagesAfter.Items)
	}

	notificationsResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/notifications?limit=10", http.MethodGet, "", http.StatusOK, reporter, nil)
	defer notificationsResponse.Body.Close()
	var notifications struct {
		Items []struct {
			EventType   string `json:"event_type"`
			TargetID    string `json:"target_id"`
			UnreadCount int    `json:"unread_count"`
		} `json:"items"`
	}
	if err := json.NewDecoder(notificationsResponse.Body).Decode(&notifications); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	foundChatNotification := false
	for _, notification := range notifications.Items {
		if notification.EventType == "chat.message.created" && notification.TargetID == conversation.ID {
			foundChatNotification = true
			if notification.UnreadCount != 2 {
				t.Fatalf("chat notification unread_count = %d, want 2", notification.UnreadCount)
			}
		}
	}
	if !foundChatNotification {
		t.Fatalf("chat notification for conversation %s not found in %+v", conversation.ID, notifications.Items)
	}

	readResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/chat/conversations/"+conversation.ID+"/read", http.MethodPost, "", http.StatusNoContent, reporter, nil)
	readResponse.Body.Close()
}

func TestMediaRejectsInvalidUpload(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "invalid-media-author@catch.local")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "note.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("plain text is not an allowed media payload")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	headers := map[string]string{"Content-Type": writer.FormDataContentType()}
	response := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/media/files", http.MethodPost, body.String(), http.StatusUnprocessableEntity, author, headers)
	response.Body.Close()
}

func TestFeedCursorPagination(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "pagination-author@catch.local")
	setUserRoleAndRating(t, container, author.UserID, "user", 1000)

	firstDraft := createDraft(t, server, author, "Первая статья пагинации", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Первый материал для проверки курсора."}]}`)
	submitDraft(t, server, author, firstDraft.ID)
	secondDraft := createDraft(t, server, author, "Вторая статья пагинации", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Второй материал для проверки курсора."}]}`)
	submitDraft(t, server, author, secondDraft.ID)

	firstPageResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/feed?limit=1", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer firstPageResponse.Body.Close()
	var firstPage struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.NewDecoder(firstPageResponse.Body).Decode(&firstPage); err != nil {
		t.Fatalf("decode first feed page: %v", err)
	}
	if len(firstPage.Items) != 1 || firstPage.NextCursor == "" {
		t.Fatalf("first page = %+v, want one item and next cursor", firstPage)
	}

	secondPageResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/feed?limit=1&cursor="+firstPage.NextCursor, http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer secondPageResponse.Body.Close()
	var secondPage struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.NewDecoder(secondPageResponse.Body).Decode(&secondPage); err != nil {
		t.Fatalf("decode second feed page: %v", err)
	}
	if len(secondPage.Items) != 1 {
		t.Fatalf("second page = %+v, want one item", secondPage)
	}
	if secondPage.Items[0].ID == firstPage.Items[0].ID {
		t.Fatalf("cursor pagination returned duplicate article %s", secondPage.Items[0].ID)
	}

	invalidCursorResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/feed?cursor=not-a-valid-cursor", http.MethodGet, "", http.StatusUnprocessableEntity, testSession{}, nil)
	invalidCursorResponse.Body.Close()

	firstSearchResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/search?q=%D0%BF%D0%B0%D0%B3%D0%B8%D0%BD%D0%B0%D1%86%D0%B8%D0%B8&limit=1", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer firstSearchResponse.Body.Close()
	var firstSearchPage struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.NewDecoder(firstSearchResponse.Body).Decode(&firstSearchPage); err != nil {
		t.Fatalf("decode first search page: %v", err)
	}
	if len(firstSearchPage.Items) != 1 || firstSearchPage.NextCursor == "" {
		t.Fatalf("first search page = %+v, want one item and next cursor", firstSearchPage)
	}

	secondSearchResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/search?q=%D0%BF%D0%B0%D0%B3%D0%B8%D0%BD%D0%B0%D1%86%D0%B8%D0%B8&limit=1&cursor="+firstSearchPage.NextCursor, http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer secondSearchResponse.Body.Close()
	var secondSearchPage struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(secondSearchResponse.Body).Decode(&secondSearchPage); err != nil {
		t.Fatalf("decode second search page: %v", err)
	}
	if len(secondSearchPage.Items) != 1 || secondSearchPage.Items[0].ID == firstSearchPage.Items[0].ID {
		t.Fatalf("second search page = %+v, want next distinct search item", secondSearchPage)
	}
}

func TestSearchPeopleAndTagQueries(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "search-author@catch.local")
	setUserRoleAndRating(t, container, author.UserID, "user", 1000)

	profileResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/profile/me", http.MethodPatch, `{"username":"river-guide","display_name":"Речной проводник"}`, http.StatusOK, author, nil)
	profileResponse.Body.Close()

	peopleResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/search/people?q=@river&limit=10", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer peopleResponse.Body.Close()
	var people struct {
		Items []struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
		} `json:"items"`
	}
	if err := json.NewDecoder(peopleResponse.Body).Decode(&people); err != nil {
		t.Fatalf("decode people search: %v", err)
	}
	if len(people.Items) != 1 || people.Items[0].UserID != author.UserID || people.Items[0].Username != "river-guide" {
		t.Fatalf("people search = %+v, want updated author profile", people.Items)
	}

	draft := createDraft(t, server, author, "Поиск по тегам", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Материал для проверки поиска по тегам."}]}`)
	submitDraft(t, server, author, draft.ID)

	tagResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/search?q=%23%D1%82%D0%B5%D1%81%D1%82&limit=10", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer tagResponse.Body.Close()
	var tagResults struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(tagResponse.Body).Decode(&tagResults); err != nil {
		t.Fatalf("decode tag search: %v", err)
	}
	if len(tagResults.Items) != 1 || tagResults.Items[0].ID != draft.ID {
		t.Fatalf("tag search = %+v, want article %s", tagResults.Items, draft.ID)
	}
}

func TestPopularFeedRanksRecentArticleActivity(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "popular-author@catch.local")
	reader := loginUser(t, server, cfg, "popular-reader@catch.local")
	setUserRoleAndRating(t, container, author.UserID, "user", 1000)
	setUserRoleAndRating(t, container, reader.UserID, "user", 20)

	quietDraft := createDraft(t, server, author, "Тихая статья", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Материал без активности."}]}`)
	submitDraft(t, server, author, quietDraft.ID)
	activeDraft := createDraft(t, server, author, "Активная статья", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Материал с активностью."}]}`)
	submitDraft(t, server, author, activeDraft.ID)

	reactionResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reactions", http.MethodPost, `{"target_type":"article","target_id":"`+activeDraft.ID+`","value":1}`, http.StatusOK, reader, nil)
	reactionResponse.Body.Close()
	createComment(t, server, reader, activeDraft.ID, "Полезный материал")

	popularResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/feed/popular?limit=10", http.MethodGet, "", http.StatusOK, testSession{}, nil)
	defer popularResponse.Body.Close()
	var popular struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(popularResponse.Body).Decode(&popular); err != nil {
		t.Fatalf("decode popular feed: %v", err)
	}
	if len(popular.Items) < 2 || popular.Items[0].ID != activeDraft.ID {
		t.Fatalf("popular feed = %+v, want active article first", popular.Items)
	}
}

func TestSocialNotificationsAreProduced(t *testing.T) {
	server, container, cfg := newIntegrationHTTPServer(t)
	defer server.Close()
	defer container.Close()

	author := loginUser(t, server, cfg, "social-author@catch.local")
	follower := loginUser(t, server, cfg, "social-follower@catch.local")
	setUserRoleAndRating(t, container, author.UserID, "user", 1000)
	setUserRoleAndRating(t, container, follower.UserID, "user", 20)

	followResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/subscriptions/"+author.UserID, http.MethodPost, "", http.StatusOK, follower, nil)
	followResponse.Body.Close()

	draft := createDraft(t, server, author, "Социальные уведомления", `{"type":"catch.article","version":1,"blocks":[{"type":"paragraph","text":"Материал для проверки уведомлений подписчикам."}]}`)
	published := submitDraft(t, server, author, draft.ID)
	if published.Status != "published" {
		t.Fatalf("published status = %q, want published", published.Status)
	}
	socialOutbox := &recordingArticleIndexer{}
	processOutboxOnce(t, container, socialOutbox)
	if !socialOutbox.indexedArticle(draft.ID) {
		t.Fatalf("indexed search documents = %+v, want article %s indexed", socialOutbox.indexed, draft.ID)
	}

	bookmarkResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/bookmarks/items", http.MethodPost, `{"article_id":"`+draft.ID+`"}`, http.StatusNoContent, follower, nil)
	bookmarkResponse.Body.Close()

	reactionResponse := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/reactions", http.MethodPost, `{"target_type":"article","target_id":"`+draft.ID+`","value":1}`, http.StatusOK, follower, nil)
	reactionResponse.Body.Close()

	authorNotifications := listNotificationEvents(t, server, author)
	assertNotification(t, authorNotifications, "subscription.followed", "user", follower.UserID)
	assertNotification(t, authorNotifications, "bookmark.added", "article", draft.ID)
	assertNotification(t, authorNotifications, "rating.changed", "article", draft.ID)

	followerNotifications := listNotificationEvents(t, server, follower)
	assertNotification(t, followerNotifications, "article.published", "article", draft.ID)
}

type draftResponse struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	ModerationRequired bool   `json:"moderation_required"`
}

type commentResponse struct {
	ID string `json:"id"`
}

type notificationEvent struct {
	EventType  string `json:"event_type"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

func listNotificationEvents(t *testing.T, server *httptest.Server, session testSession) []notificationEvent {
	t.Helper()

	response := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/notifications?limit=20", http.MethodGet, "", http.StatusOK, session, nil)
	defer response.Body.Close()
	var payload struct {
		Items []notificationEvent `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	return payload.Items
}

func assertNotification(t *testing.T, notifications []notificationEvent, eventType, targetType, targetID string) {
	t.Helper()

	for _, notification := range notifications {
		if notification.EventType == eventType && notification.TargetType == targetType && notification.TargetID == targetID {
			return
		}
	}
	t.Fatalf("notification %s/%s/%s not found in %+v", eventType, targetType, targetID, notifications)
}

type recordingArticleIndexer struct {
	indexed []search.ArticleDocument
	deleted []string
}

func (r *recordingArticleIndexer) IndexArticle(_ context.Context, article search.ArticleDocument) error {
	r.indexed = append(r.indexed, article)
	return nil
}

func (r *recordingArticleIndexer) DeleteArticle(_ context.Context, articleID string) error {
	r.deleted = append(r.deleted, articleID)
	return nil
}

func (r *recordingArticleIndexer) indexedArticle(articleID string) bool {
	for _, article := range r.indexed {
		if article.ID == articleID {
			return true
		}
	}
	return false
}

func (r *recordingArticleIndexer) deletedArticle(articleID string) bool {
	for _, deleted := range r.deleted {
		if deleted == articleID {
			return true
		}
	}
	return false
}

func processOutboxOnce(t *testing.T, container *composition.Container, articleSearch search.ArticleIndexer) {
	t.Helper()

	log := logger.New(config.EnvTest)
	worker := outbox.NewWorker(container.DB, outbox.NewNotificationHandler(container.DB, articleSearch, mail.NoopSender{}, log), log, "integration-test")
	if _, err := worker.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("process outbox: %v", err)
	}
}

func newIntegrationHTTPServer(t *testing.T) (*httptest.Server, *composition.Container, config.Config) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	cfg := testConfig(databaseURL)
	log := logger.New(config.EnvTest)
	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	if err := db.ApplyMigrations(ctx, container.DB, migrationsDir(t)); err != nil {
		container.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	cleanupIdentityTables(t, container)

	return httptest.NewServer(bootstrap.NewRouter(container)), container, cfg
}

func loginUser(t *testing.T, server *httptest.Server, cfg config.Config, email string) testSession {
	t.Helper()

	response := doJSON(t, server.Client(), server.URL+"/api/v1/dev/auth/login", http.MethodPost, `{"email":"`+email+`"}`, http.StatusOK, nil)
	defer response.Body.Close()
	sessionCookie := findCookie(t, response.Cookies(), cfg.Auth.SessionCookieName)
	csrfCookie := findCookie(t, response.Cookies(), cfg.Auth.CSRFCookieName)

	var payload struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode dev login: %v", err)
	}
	if payload.User.ID == "" {
		t.Fatal("dev login returned empty user id")
	}
	return testSession{UserID: payload.User.ID, Cookie: sessionCookie, CSRF: csrfCookie}
}

func setUserRoleAndRating(t *testing.T, container *composition.Container, userID, role string, rating int) {
	t.Helper()

	if _, err := container.DB.Exec(context.Background(), `
		update users
		set role = $2, rating = $3, updated_at = now()
		where id = $1
	`, userID, role, rating); err != nil {
		t.Fatalf("update user role/rating: %v", err)
	}
}

func createDraft(t *testing.T, server *httptest.Server, session testSession, title, content string) draftResponse {
	t.Helper()

	body := `{"title":"` + title + `","content":` + content + `,"tags":["тест"]}`
	response := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/drafts", http.MethodPost, body, http.StatusCreated, session, nil)
	defer response.Body.Close()
	var draft draftResponse
	if err := json.NewDecoder(response.Body).Decode(&draft); err != nil {
		t.Fatalf("decode draft: %v", err)
	}
	if draft.ID == "" {
		t.Fatal("draft id is empty")
	}
	return draft
}

func submitDraft(t *testing.T, server *httptest.Server, session testSession, articleID string) draftResponse {
	t.Helper()

	response := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/drafts/"+articleID+"/submit", http.MethodPost, `{}`, http.StatusOK, session, nil)
	defer response.Body.Close()
	var draft draftResponse
	if err := json.NewDecoder(response.Body).Decode(&draft); err != nil {
		t.Fatalf("decode submitted draft: %v", err)
	}
	return draft
}

func createComment(t *testing.T, server *httptest.Server, session testSession, articleID, body string) commentResponse {
	t.Helper()

	response := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/articles/"+articleID+"/comments", http.MethodPost, `{"body":"`+body+`"}`, http.StatusCreated, session, nil)
	defer response.Body.Close()
	var comment commentResponse
	if err := json.NewDecoder(response.Body).Decode(&comment); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	if comment.ID == "" {
		t.Fatal("comment id is empty")
	}
	return comment
}

func uploadPNG(t *testing.T, server *httptest.Server, session testSession) string {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "tiny.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
		0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	headers := map[string]string{"Content-Type": writer.FormDataContentType()}
	response := doJSONWithHeaders(t, server.Client(), server.URL+"/api/v1/media/files", http.MethodPost, body.String(), http.StatusCreated, session, headers)
	defer response.Body.Close()
	var file struct {
		ID         string `json:"id"`
		PreviewURL string `json:"preview_url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&file); err != nil {
		t.Fatalf("decode uploaded file: %v", err)
	}
	if file.ID == "" {
		t.Fatal("uploaded file id is empty")
	}
	if file.PreviewURL == "" {
		t.Fatal("uploaded image preview_url is empty")
	}
	return file.ID
}

func doJSONWithHeaders(t *testing.T, client *http.Client, url, method, body string, wantStatus int, session testSession, headers map[string]string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	if session.Cookie != nil {
		request.AddCookie(session.Cookie)
	}
	if session.CSRF != nil {
		request.AddCookie(session.CSRF)
		request.Header.Set("X-CSRF-Token", session.CSRF.Value)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("call %s: %v", url, err)
	}
	if response.StatusCode != wantStatus {
		defer response.Body.Close()
		t.Fatalf("%s status = %d, want %d", url, response.StatusCode, wantStatus)
	}
	return response
}
