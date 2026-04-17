package dto

type BookmarkListResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type BookmarkListsResponse struct {
	Items []BookmarkListResponse `json:"items"`
}

type BookmarkedArticleResponse struct {
	ListID       string   `json:"list_id"`
	ListName     string   `json:"list_name"`
	ArticleID    string   `json:"article_id"`
	AuthorID     string   `json:"author_id"`
	Title        string   `json:"title"`
	Excerpt      string   `json:"excerpt"`
	Tags         []string `json:"tags"`
	PublishedAt  string   `json:"published_at"`
	BookmarkedAt string   `json:"bookmarked_at"`
}

type BookmarkedArticlesResponse struct {
	Items []BookmarkedArticleResponse `json:"items"`
}

type CreateBookmarkListRequest struct {
	Name string `json:"name"`
}

type AddBookmarkRequest struct {
	ListID    string `json:"list_id,omitempty"`
	ArticleID string `json:"article_id"`
}

type RemoveBookmarkRequest struct {
	ListID    string `json:"list_id"`
	ArticleID string `json:"article_id"`
}

type FollowResponse struct {
	AuthorID  string `json:"author_id"`
	Following bool   `json:"following"`
}
