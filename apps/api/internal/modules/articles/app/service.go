package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/articles/app/dto"
	"catch/apps/api/internal/modules/articles/domain"
	"catch/apps/api/internal/modules/articles/ports"
	"catch/apps/api/internal/platform/db"
	httpx "catch/apps/api/internal/platform/http"
	"catch/apps/api/internal/platform/search"
)

type Service struct {
	tx     *db.TxManager
	repo   ports.Repository
	search search.ArticleSearcher
	now    func() time.Time
}

func NewService(tx *db.TxManager, repo ports.Repository) *Service {
	return NewServiceWithSearch(tx, repo, search.NoopArticleSearcher{})
}

func NewServiceWithSearch(tx *db.TxManager, repo ports.Repository, articleSearch search.ArticleSearcher) *Service {
	if articleSearch == nil {
		articleSearch = search.NoopArticleSearcher{}
	}
	return &Service{tx: tx, repo: repo, search: articleSearch, now: time.Now}
}

func (s *Service) CreateDraft(ctx context.Context, actor accessdomain.Principal, request dto.CreateDraftRequest) (dto.ArticleDraftResponse, error) {
	if !actor.CanCreateArticle() {
		return dto.ArticleDraftResponse{}, httpx.Forbidden("Недостаточно рейтинга для создания статей")
	}

	title, content, tags, err := validateArticleInput(request.Title, request.Content, request.Tags)
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}

	var draft domain.Draft
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.repo.CreateDraft(ctx, ports.CreateDraftInput{
			AuthorID: actor.UserID,
			Title:    title,
			Content:  content,
			Excerpt:  excerptFromDocument(content),
			Tags:     tags,
		})
		if err != nil {
			return err
		}
		draft = created
		return nil
	})
	if err != nil {
		return dto.ArticleDraftResponse{}, err
	}

	return mapDraft(draft), nil
}

func (s *Service) GetMyDraft(ctx context.Context, actor accessdomain.Principal, articleID string) (dto.ArticleDraftResponse, error) {
	draft, err := s.repo.FindDraftForAuthor(ctx, articleID, actor.UserID)
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}
	return mapDraft(draft), nil
}

func (s *Service) ListMyArticles(ctx context.Context, actor accessdomain.Principal, limit int) (dto.ArticleDraftListResponse, error) {
	articles, err := s.repo.ListForAuthor(ctx, actor.UserID, normalizeLimit(limit))
	if err != nil {
		return dto.ArticleDraftListResponse{}, err
	}
	items := make([]dto.ArticleDraftResponse, 0, len(articles))
	for _, article := range articles {
		items = append(items, mapDraft(article))
	}
	return dto.ArticleDraftListResponse{Items: items}, nil
}

func (s *Service) GetPublishedArticle(ctx context.Context, articleID string) (dto.PublicArticleResponse, error) {
	article, err := s.repo.FindPublished(ctx, articleID, s.now())
	if err != nil {
		return dto.PublicArticleResponse{}, mapArticleError(err)
	}
	return mapPublicArticle(article), nil
}

func (s *Service) ListFeed(ctx context.Context, limit int, cursorValue string) (dto.ArticleListResponse, error) {
	cursor, err := decodeCursor(cursorValue)
	if err != nil {
		return dto.ArticleListResponse{}, err
	}
	articles, err := s.repo.ListPublished(ctx, ports.ListPublishedInput{Limit: normalizeLimit(limit) + 1, Now: s.now(), Cursor: cursor})
	if err != nil {
		return dto.ArticleListResponse{}, err
	}
	return mapArticleList(articles, normalizeLimit(limit)), nil
}

func (s *Service) ListPopularFeed(ctx context.Context, limit int) (dto.ArticleListResponse, error) {
	now := s.now()
	articles, err := s.repo.ListPopular(ctx, ports.ListPopularInput{
		Limit: normalizeLimit(limit),
		Now:   now,
		Since: now.AddDate(0, 0, -14),
	})
	if err != nil {
		return dto.ArticleListResponse{}, err
	}
	return mapArticleList(articles, normalizeLimit(limit)), nil
}

func (s *Service) ListMyFeed(ctx context.Context, actor accessdomain.Principal, limit int, cursorValue string) (dto.ArticleListResponse, error) {
	cursor, err := decodeCursor(cursorValue)
	if err != nil {
		return dto.ArticleListResponse{}, err
	}
	articles, err := s.repo.ListPersonalizedFeed(ctx, ports.PersonalizedFeedInput{UserID: actor.UserID, Limit: normalizeLimit(limit) + 1, Now: s.now(), Cursor: cursor})
	if err != nil {
		return dto.ArticleListResponse{}, err
	}
	return mapArticleList(articles, normalizeLimit(limit)), nil
}

func (s *Service) Search(ctx context.Context, query string, limit int, cursorValue string) (dto.ArticleListResponse, error) {
	cleaned := strings.TrimSpace(query)
	if strings.HasPrefix(cleaned, "#") {
		cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, "#"))
	}
	if len([]rune(cleaned)) < 3 {
		return dto.ArticleListResponse{}, httpx.ValidationError("Поиск начинается с 3 символов", map[string]any{"q": "too_short"})
	}
	if response, ok := s.searchWithExternalIndex(ctx, cleaned, normalizeLimit(limit), cursorValue); ok {
		return response, nil
	}
	cursor, err := decodeCursor(cursorValue)
	if err != nil {
		return dto.ArticleListResponse{}, err
	}

	articles, err := s.repo.SearchPublished(ctx, ports.SearchPublishedInput{Query: cleaned, Limit: normalizeLimit(limit) + 1, Now: s.now(), Cursor: cursor})
	if err != nil {
		return dto.ArticleListResponse{}, err
	}
	return mapArticleList(articles, normalizeLimit(limit)), nil
}

func (s *Service) searchWithExternalIndex(ctx context.Context, query string, limit int, cursorValue string) (dto.ArticleListResponse, bool) {
	offset, ok := decodeSearchOffsetCursor(cursorValue)
	if !ok {
		return dto.ArticleListResponse{}, false
	}
	result, err := s.search.SearchArticles(ctx, search.ArticleSearchRequest{Query: query, Limit: limit, Offset: offset})
	if err != nil {
		return dto.ArticleListResponse{}, false
	}
	articles, err := s.repo.ListPublishedByIDs(ctx, ports.ListPublishedByIDsInput{IDs: result.IDs, Now: s.now()})
	if err != nil {
		return dto.ArticleListResponse{}, false
	}
	response := mapArticleList(articles, limit)
	if result.NextOffset != nil {
		response.NextCursor = encodeSearchOffsetCursor(*result.NextOffset)
	}
	return response, true
}

func (s *Service) UpdateDraft(ctx context.Context, actor accessdomain.Principal, articleID string, request dto.UpdateDraftRequest) (dto.ArticleDraftResponse, error) {
	existing, err := s.repo.FindDraftForAuthor(ctx, articleID, actor.UserID)
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}
	if existing.Status != domain.ArticleStatusDraft && existing.Status != domain.ArticleStatusArchived {
		return dto.ArticleDraftResponse{}, mapArticleError(domain.ErrArticleNotEditable)
	}

	title := existing.Title
	if request.Title != nil {
		normalizedTitle, err := domain.NormalizeTitle(*request.Title)
		if err != nil {
			return dto.ArticleDraftResponse{}, mapArticleError(err)
		}
		title = normalizedTitle
	}

	content := existing.Content
	if request.Content != nil {
		normalizedContent, err := domain.ValidateDocument(*request.Content)
		if err != nil {
			return dto.ArticleDraftResponse{}, mapArticleError(err)
		}
		content = normalizedContent
	}

	tags := existing.Tags
	if request.Tags != nil {
		normalizedTags, err := domain.NormalizeTags(request.Tags)
		if err != nil {
			return dto.ArticleDraftResponse{}, mapArticleError(err)
		}
		tags = normalizedTags
	}

	var draft domain.Draft
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		updated, err := s.repo.UpdateDraftRevision(ctx, ports.UpdateDraftRevisionInput{
			ArticleID: articleID,
			AuthorID:  actor.UserID,
			Title:     title,
			Content:   content,
			Excerpt:   excerptFromDocument(content),
			Tags:      tags,
		})
		if err != nil {
			return err
		}
		draft = updated
		return nil
	})
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}

	return mapDraft(draft), nil
}

func (s *Service) SubmitDraft(ctx context.Context, actor accessdomain.Principal, articleID string, request dto.SubmitDraftRequest) (dto.ArticleDraftResponse, error) {
	existing, err := s.repo.FindDraftForAuthor(ctx, articleID, actor.UserID)
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}
	if existing.Status != domain.ArticleStatusDraft && existing.Status != domain.ArticleStatusReadyToPublish {
		return dto.ArticleDraftResponse{}, mapArticleError(domain.ErrArticleNotEditable)
	}

	scheduledAt, err := parsePublishAt(request.PublishAt, s.now())
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}

	input := ports.SubmitDraftInput{
		ArticleID:          articleID,
		AuthorID:           actor.UserID,
		RevisionStatus:     domain.RevisionStatusSubmitted,
		ArticleStatus:      domain.ArticleStatusInModeration,
		ModerationRequired: true,
		ScheduledAt:        scheduledAt,
	}

	if actor.CanPublishDirectly() || existing.Status == domain.ArticleStatusReadyToPublish {
		now := s.now()
		publishedAt := now
		if scheduledAt != nil {
			publishedAt = *scheduledAt
		}
		publishedToday, err := s.repo.CountPublishedByAuthorSince(ctx, ports.CountPublishedByAuthorSinceInput{
			AuthorID: actor.UserID,
			Since:    startOfDay(now),
		})
		if err != nil {
			return dto.ArticleDraftResponse{}, err
		}
		if publishedToday >= 3 {
			return dto.ArticleDraftResponse{}, mapArticleError(domain.ErrDailyPublishLimit)
		}
		input.RevisionStatus = domain.RevisionStatusPublished
		input.ArticleStatus = domain.ArticleStatusPublished
		input.ModerationRequired = false
		input.PublishedAt = &publishedAt
		input.RewardPublication = existing.Status == domain.ArticleStatusReadyToPublish
	}

	var draft domain.Draft
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		submitted, err := s.repo.SubmitDraft(ctx, input)
		if err != nil {
			return err
		}
		draft = submitted
		return nil
	})
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}

	return mapDraft(draft), nil
}

func (s *Service) ArchiveDraft(ctx context.Context, actor accessdomain.Principal, articleID string) (dto.ArticleDraftResponse, error) {
	existing, err := s.repo.FindDraftForAuthor(ctx, articleID, actor.UserID)
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}
	if existing.Status != domain.ArticleStatusDraft &&
		existing.Status != domain.ArticleStatusInModeration &&
		existing.Status != domain.ArticleStatusReadyToPublish &&
		existing.Status != domain.ArticleStatusArchived {
		return dto.ArticleDraftResponse{}, mapArticleError(domain.ErrArticleNotEditable)
	}

	var draft domain.Draft
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		archived, err := s.repo.ArchiveDraft(ctx, ports.ArchiveDraftInput{ArticleID: articleID, AuthorID: actor.UserID})
		if err != nil {
			return err
		}
		draft = archived
		return nil
	})
	if err != nil {
		return dto.ArticleDraftResponse{}, mapArticleError(err)
	}
	return mapDraft(draft), nil
}

func validateArticleInput(titleValue string, contentValue json.RawMessage, tagValues []string) (string, json.RawMessage, []string, error) {
	title, err := domain.NormalizeTitle(titleValue)
	if err != nil {
		return "", nil, nil, err
	}
	content, err := domain.ValidateDocument(contentValue)
	if err != nil {
		return "", nil, nil, err
	}
	tags, err := domain.NormalizeTags(tagValues)
	if err != nil {
		return "", nil, nil, err
	}
	return title, content, tags, nil
}

func parsePublishAt(value *string, now time.Time) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, domain.ErrPublishWindow
	}
	if !parsed.After(now) || parsed.After(now.AddDate(0, 1, 0)) {
		return nil, domain.ErrPublishWindow
	}
	return &parsed, nil
}

func excerptFromDocument(value json.RawMessage) string {
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return ""
	}
	text := strings.Join(extractDocumentText(decoded, 400), " ")
	text = strings.TrimSpace(spacePattern.ReplaceAllString(text, " "))
	if utf8.RuneCountInString(text) <= 240 {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:240])) + "..."
}

func mapDraft(draft domain.Draft) dto.ArticleDraftResponse {
	return dto.ArticleDraftResponse{
		ID:                 draft.ID,
		Status:             string(draft.Status),
		CurrentRevisionID:  draft.CurrentRevisionID,
		ModerationRequired: draft.ModerationRequired,
		Title:              draft.Title,
		Content:            draft.Content,
		Excerpt:            draft.Excerpt,
		CoverURL:           coverURLFromDocument(draft.Content),
		Tags:               draft.Tags,
		Version:            draft.Version,
		ScheduledAt:        formatTimePtr(draft.ScheduledAt),
		PublishedAt:        formatTimePtr(draft.PublishedAt),
	}
}

func mapPublicArticle(article domain.Draft) dto.PublicArticleResponse {
	publishedAt := ""
	if article.PublishedAt != nil {
		publishedAt = article.PublishedAt.Format(time.RFC3339)
	}
	return dto.PublicArticleResponse{
		ID:            article.ID,
		AuthorID:      article.AuthorID,
		Title:         article.Title,
		Content:       article.Content,
		Excerpt:       article.Excerpt,
		CoverURL:      coverURLFromDocument(article.Content),
		Tags:          article.Tags,
		ReactionsUp:   article.ReactionsUp,
		ReactionsDown: article.ReactionsDown,
		ReactionScore: article.ReactionScore,
		PublishedAt:   publishedAt,
	}
}

func mapArticleList(articles []domain.Draft, limit int) dto.ArticleListResponse {
	hasNext := len(articles) > limit
	visible := articles
	if hasNext {
		visible = articles[:limit]
	}
	items := make([]dto.ArticleListItem, 0, len(visible))
	for _, article := range visible {
		publishedAt := ""
		if article.PublishedAt != nil {
			publishedAt = article.PublishedAt.Format(time.RFC3339)
		}
		items = append(items, dto.ArticleListItem{
			ID:            article.ID,
			AuthorID:      article.AuthorID,
			Title:         article.Title,
			Excerpt:       article.Excerpt,
			CoverURL:      coverURLFromDocument(article.Content),
			Tags:          article.Tags,
			ReactionsUp:   article.ReactionsUp,
			ReactionsDown: article.ReactionsDown,
			ReactionScore: article.ReactionScore,
			PublishedAt:   publishedAt,
		})
	}
	nextCursor := ""
	if hasNext && len(visible) > 0 {
		nextCursor = encodeCursor(visible[len(visible)-1])
	}
	return dto.ArticleListResponse{Items: items, NextCursor: nextCursor}
}

func encodeCursor(article domain.Draft) string {
	if article.PublishedAt == nil {
		return ""
	}
	raw := fmt.Sprintf("%d|%s|%s", article.SortRank, article.PublishedAt.UTC().Format(time.RFC3339Nano), article.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func encodeSearchOffsetCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("search|%d", offset)))
}

func decodeSearchOffsetCursor(value string) (int, bool) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return 0, true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(clean)
	if err != nil {
		return 0, false
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 || parts[0] != "search" {
		return 0, false
	}
	var offset int
	if _, err := fmt.Sscanf(parts[1], "%d", &offset); err != nil || offset < 0 {
		return 0, false
	}
	return offset, true
}

func decodeCursor(value string) (*ports.ListCursor, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(clean)
	if err != nil {
		return nil, httpx.ValidationError("Курсор указан некорректно", map[string]any{"cursor": "invalid"})
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 {
		return nil, httpx.ValidationError("Курсор указан некорректно", map[string]any{"cursor": "invalid"})
	}
	var rank int
	if _, err := fmt.Sscanf(parts[0], "%d", &rank); err != nil {
		return nil, httpx.ValidationError("Курсор указан некорректно", map[string]any{"cursor": "invalid"})
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return nil, httpx.ValidationError("Курсор указан некорректно", map[string]any{"cursor": "invalid"})
	}
	if strings.TrimSpace(parts[2]) == "" {
		return nil, httpx.ValidationError("Курсор указан некорректно", map[string]any{"cursor": "invalid"})
	}
	return &ports.ListCursor{Rank: rank, PublishedAt: publishedAt, ID: parts[2]}, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func formatTimePtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func mapArticleError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidTitle):
		return httpx.ValidationError("Заголовок должен быть от 3 до 160 символов", map[string]any{"title": "invalid"})
	case errors.Is(err, domain.ErrInvalidDocument):
		return httpx.ValidationError("Документ статьи указан некорректно", map[string]any{"content": "invalid"})
	case errors.Is(err, domain.ErrTooManyTags):
		return httpx.ValidationError("У статьи может быть максимум 10 тегов", map[string]any{"tags": "too_many"})
	case errors.Is(err, domain.ErrInvalidTag):
		return httpx.ValidationError("Тег указан некорректно", map[string]any{"tags": "invalid"})
	case errors.Is(err, domain.ErrArticleNotFound):
		return httpx.NewError(404, httpx.CodeNotFound, "Статья не найдена")
	case errors.Is(err, domain.ErrArticleNotEditable):
		return httpx.NewError(409, httpx.CodeConflict, "Статью нельзя редактировать в текущем статусе")
	case errors.Is(err, domain.ErrPublishWindow):
		return httpx.ValidationError("Дата публикации должна быть в будущем и не дальше одного месяца", map[string]any{"publish_at": "invalid"})
	case errors.Is(err, domain.ErrDailyPublishLimit):
		return httpx.NewError(429, httpx.CodeRateLimited, "Можно публиковать не больше трёх статей в сутки")
	default:
		return err
	}
}

var spacePattern = regexp.MustCompile(`\s+`)

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func extractDocumentText(value any, maxRunes int) []string {
	parts := make([]string, 0)
	collectDocumentText(value, maxRunes, &parts)
	return parts
}

func collectDocumentText(value any, maxRunes int, parts *[]string) {
	if utf8.RuneCountInString(strings.Join(*parts, " ")) >= maxRunes {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"text", "body", "caption", "title"} {
			if text, ok := typed[key].(string); ok && strings.TrimSpace(text) != "" {
				*parts = append(*parts, text)
			}
		}
		for _, nested := range typed {
			collectDocumentText(nested, maxRunes, parts)
		}
	case []any:
		for _, nested := range typed {
			collectDocumentText(nested, maxRunes, parts)
		}
	}
}

func coverURLFromDocument(value json.RawMessage) string {
	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return ""
	}
	blocks, _ := decoded["blocks"].([]any)
	for _, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]any)
		if !ok || block["type"] != "image" {
			continue
		}
		for _, key := range []string{"preview_url", "url"} {
			if url, ok := block[key].(string); ok && strings.TrimSpace(url) != "" {
				return strings.TrimSpace(url)
			}
		}
	}
	return ""
}
