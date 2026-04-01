package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/y2k-pixel-blog/backend/internal/model"
)

// Repository handles all database operations
type Repository struct {
	db *sql.DB
}

// New creates a new Repository
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// --- Board operations ---

// ListBoards returns all top-level boards with children
func (r *Repository) ListBoards() ([]*model.Board, error) {
	rows, err := r.db.Query(`
		SELECT id, slug, name, color, icon, sort_order, parent_id
		FROM boards
		WHERE parent_id IS NULL
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []*model.Board
	for rows.Next() {
		b := &model.Board{}
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &b.Color, &b.Icon, &b.Order, &b.ParentID); err != nil {
			return nil, err
		}
		children, _ := r.listChildBoards(b.ID)
		b.Children = children
		count, _ := r.countBoardPosts(b.ID)
		b.PostCount = count
		boards = append(boards, b)
	}
	return boards, nil
}

func (r *Repository) listChildBoards(parentID int) ([]*model.Board, error) {
	rows, err := r.db.Query(`
		SELECT id, slug, name, color, icon, sort_order, parent_id
		FROM boards
		WHERE parent_id = $1
		ORDER BY sort_order ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []*model.Board
	for rows.Next() {
		b := &model.Board{}
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &b.Color, &b.Icon, &b.Order, &b.ParentID); err != nil {
			return nil, err
		}
		count, _ := r.countBoardPosts(b.ID)
		b.PostCount = count
		boards = append(boards, b)
	}
	return boards, nil
}

func (r *Repository) countBoardPosts(boardID int) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post_boards WHERE board_id = $1`, boardID).Scan(&count)
	return count, err
}

// GetBoardBySlug returns a board by its slug
func (r *Repository) GetBoardBySlug(slug string) (*model.Board, error) {
	b := &model.Board{}
	err := r.db.QueryRow(`
		SELECT id, slug, name, color, icon, sort_order, parent_id
		FROM boards WHERE slug = $1
	`, slug).Scan(&b.ID, &b.Slug, &b.Name, &b.Color, &b.Icon, &b.Order, &b.ParentID)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// --- Board content listing ---

// ListBoardItems returns posts and sub-boards in a board
func (r *Repository) ListBoardItems(boardID, page, limit int, sort, order, contentType string) ([]model.BoardListItem, int, error) {
	// Whitelist sort columns to prevent SQL injection
	sortCol := "p.date"
	switch sort {
	case "title":
		sortCol = "p.title"
	case "score":
		sortCol = "p.score"
	}
	sortDir := "DESC"
	if order == "asc" {
		sortDir = "ASC"
	}

	// Build dynamic query — public only shows published posts
	baseQuery := `
		FROM posts p
		JOIN post_boards pb ON p.id = pb.post_id
		WHERE pb.board_id = $1 AND p.status = 'published'
	`
	args := []interface{}{boardID}
	argIdx := 2

	if contentType != "" {
		baseQuery += fmt.Sprintf(` AND p.type = $%d`, argIdx)
		args = append(args, contentType)
		argIdx++
	}

	// Count total
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) `+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch paginated items
	offset := (page - 1) * limit
	selectQuery := `
		SELECT p.slug, p.title, p.type, p.date, p.tags, p.excerpt, p.read_time, p.score
	` + baseQuery + ` ORDER BY ` + sortCol + ` ` + sortDir + `
		LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []model.BoardListItem
	for rows.Next() {
		item := model.BoardListItem{}
		var tagsStr, date sql.NullString
		var readTime sql.NullInt64
		var score sql.NullFloat64
		if err := rows.Scan(&item.Slug, &item.Title, &item.Type, &date, &tagsStr, &item.Excerpt, &readTime, &score); err != nil {
			return nil, 0, err
		}
		if date.Valid {
			item.Date = formatDateOnly(date.String)
		}
		if readTime.Valid {
			item.ReadTime = int(readTime.Int64)
		}
		if score.Valid {
			item.Score = score.Float64
		}
		if tagsStr.Valid && tagsStr.String != "" {
			item.Tags = parseTags(tagsStr.String)
		}
		items = append(items, item)
	}

	// Prepend sub-boards at the top
	subBoards, err := r.listChildBoards(boardID)
	if err == nil {
		for i := len(subBoards) - 1; i >= 0; i-- {
			sb := subBoards[i]
			items = append([]model.BoardListItem{{
				Slug:      sb.Slug,
				Name:      sb.Name,
				Type:      "board",
				Icon:      sb.Icon,
				Color:     sb.Color,
				PostCount: sb.PostCount,
			}}, items...)
			total++
		}
	}

	return items, total, nil
}

// --- Post operations ---

// GetPostBySlug returns a full post by slug, with all type-specific fields populated
func (r *Repository) GetPostBySlug(slug string) (*model.Post, error) {
	post := &model.Post{}
	var tagsStr, dateStr sql.NullString
	var radarJSON, pagesJSON []byte

	err := r.db.QueryRow(`
		SELECT id, slug, title, type, status, date, tags, excerpt,
		       content_raw, content_html, custom_footer, custom_css, css_enabled,
		       read_time, word_count, icon, sort_order, show_in_menu,
		       cover, summary, score, radar_charts, photo_pages
		FROM posts WHERE slug = $1
	`, slug).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Type, &post.Status, &dateStr,
		&tagsStr, &post.Excerpt, &post.ContentRaw, &post.Content,
		&post.CustomFooter, &post.CustomCSS, &post.CSSEnabled,
		&post.ReadTime, &post.WordCount,
		&post.Icon, &post.Order, &post.ShowInMenu,
		&post.Cover, &post.Summary, &post.Score, &radarJSON, &pagesJSON,
	)
	if err != nil {
		return nil, err
	}

	// Parse date — normalize to YYYY-MM-DD
	if dateStr.Valid {
		post.Date = formatDateOnly(dateStr.String)
	}

	// Parse comma-separated tags
	if tagsStr.Valid && tagsStr.String != "" {
		post.Tags = parseTags(tagsStr.String)
	}

	// Load boards from junction table
	post.Boards, _ = r.getPostBoards(post.ID)

	// Parse JSONB fields based on type
	if post.Type == "rating" && len(radarJSON) > 0 {
		var charts []model.RadarChart
		if err := json.Unmarshal(radarJSON, &charts); err == nil {
			post.RadarCharts = charts
		}
	}

	if post.Type == "photo" && len(pagesJSON) > 0 {
		var pages []model.PhotoPage
		if err := json.Unmarshal(pagesJSON, &pages); err == nil {
			post.Pages = pages
		}
	}

	// Strip fields not relevant to this type
	switch post.Type {
	case "article":
		post.Cover = ""
		post.Summary = ""
		post.RadarCharts = nil
		post.Pages = nil
		post.Icon = ""
		post.ShowInMenu = false
	case "photo":
		post.Cover = ""
		post.Summary = ""
		post.RadarCharts = nil
		post.Content = ""
		post.ContentRaw = ""
		post.Icon = ""
		post.ShowInMenu = false
	case "rating":
		post.Pages = nil
		post.Icon = ""
		post.ShowInMenu = false
	case "page":
		post.Cover = ""
		post.Summary = ""
		post.RadarCharts = nil
		post.Pages = nil
	}

	return post, nil
}

// getPostBoards returns board slugs for a given post
func (r *Repository) getPostBoards(postID int) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT b.slug FROM boards b
		JOIN post_boards pb ON b.id = pb.board_id
		WHERE pb.post_id = $1
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, nil
}

// --- Search ---

// Search performs a search query based on type, optionally scoped to a board
func (r *Repository) Search(query, searchType string, boardSlug string, limit int) ([]model.SearchResult, error) {
	var sqlQuery string
	var args []interface{}

	// Base columns
	cols := `p.slug, p.title, p.type, p.date, p.tags, p.excerpt, p.score`

	// Board scoping
	boardJoin := ""
	boardWhere := ""
	argIdx := 1

	if boardSlug != "" {
		boardJoin = ` JOIN post_boards pb ON p.id = pb.post_id JOIN boards b ON b.id = pb.board_id`
		boardWhere = fmt.Sprintf(` AND b.slug = $%d`, argIdx)
		args = append(args, boardSlug)
		argIdx++
	}

	switch searchType {
	case "tag":
		sqlQuery = fmt.Sprintf(`
			SELECT %s FROM posts p%s
			WHERE p.status = 'published' AND p.tags ILIKE '%%' || $%d || '%%' %s
			ORDER BY p.date DESC
			LIMIT $%d`, cols, boardJoin, argIdx, boardWhere, argIdx+1)
		args = append(args, query, limit)

	case "fulltext":
		// pg_trgm trigram search — works with Chinese, Japanese, Korean
		sqlQuery = fmt.Sprintf(`
			SELECT %s FROM posts p%s
			WHERE p.status = 'published'
			  AND (p.title ILIKE '%%' || $%d || '%%'
			    OR p.content_raw ILIKE '%%' || $%d || '%%'
			    OR p.excerpt ILIKE '%%' || $%d || '%%'
			    OR p.tags ILIKE '%%' || $%d || '%%') %s
			ORDER BY
			  GREATEST(
			    similarity(p.title, $%d),
			    similarity(p.excerpt, $%d)
			  ) DESC,
			  p.date DESC
			LIMIT $%d`, cols, boardJoin, argIdx, argIdx, argIdx, argIdx, boardWhere, argIdx, argIdx, argIdx+1)
		args = append(args, query, limit)

	case "board_browse":
		// Just list everything in the board
		sqlQuery = fmt.Sprintf(`
			SELECT %s FROM posts p%s
			WHERE p.status = 'published' %s
			ORDER BY p.date DESC
			LIMIT $%d`, cols, boardJoin, boardWhere, argIdx)
		args = append(args, limit)

	default:
		// Default search — trigram ILIKE on title/tags/excerpt (GIN indexed)
		sqlQuery = fmt.Sprintf(`
			SELECT %s FROM posts p%s
			WHERE p.status = 'published' AND (p.title ILIKE '%%' || $%d || '%%'
			   OR p.tags ILIKE '%%' || $%d || '%%'
			   OR p.excerpt ILIKE '%%' || $%d || '%%') %s
			ORDER BY
			  GREATEST(
			    similarity(p.title, $%d),
			    similarity(p.tags, $%d)
			  ) DESC,
			  p.date DESC
			LIMIT $%d`, cols, boardJoin, argIdx, argIdx, argIdx, boardWhere, argIdx, argIdx, argIdx+1)
		args = append(args, query, limit)
	}

	rows, err := r.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		sr := model.SearchResult{}
		var tagsStr, dateStr sql.NullString
		var score sql.NullFloat64
		if err := rows.Scan(&sr.Slug, &sr.Title, &sr.Type, &dateStr, &tagsStr, &sr.Excerpt, &score); err != nil {
			return nil, err
		}
		if dateStr.Valid {
			sr.Date = formatDateOnly(dateStr.String)
		}
		if tagsStr.Valid && tagsStr.String != "" {
			sr.Tags = parseTags(tagsStr.String)
		}
		if score.Valid {
			sr.Score = score.Float64
		}
		sr.Boards, _ = r.getPostBoardsBySlug(sr.Slug)
		results = append(results, sr)
	}
	return results, nil
}

// ListAllTags returns all unique tags with their usage counts
func (r *Repository) ListAllTags() ([]model.TagItem, error) {
	// Use unnest to split comma-separated tags and count occurrences
	rows, err := r.db.Query(`
		SELECT TRIM(tag) AS tag_name, COUNT(*) AS cnt
		FROM (
			SELECT UNNEST(STRING_TO_ARRAY(tags, ',')) AS tag
			FROM posts
			WHERE tags != '' AND status = 'published'
		) sub
		GROUP BY TRIM(tag)
		ORDER BY cnt DESC, tag_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.TagItem
	for rows.Next() {
		t := model.TagItem{}
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// SearchByTag returns all posts with a specific tag
func (r *Repository) SearchByTag(tag string, limit int) ([]model.SearchResult, error) {
	return r.Search(tag, "tag", "", limit)
}

// getPostBoardsBySlug returns board slugs for a post identified by slug
func (r *Repository) getPostBoardsBySlug(postSlug string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT b.slug FROM boards b
		JOIN post_boards pb ON b.id = pb.board_id
		JOIN posts p ON p.id = pb.post_id
		WHERE p.slug = $1
	`, postSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, nil
}

// --- Menu ---

// ListTopLevelBoardsForMenu returns boards for the default menu state
func (r *Repository) ListTopLevelBoardsForMenu() ([]model.MenuBoard, error) {
	rows, err := r.db.Query(`
		SELECT slug, name, color, icon, sort_order
		FROM boards
		WHERE parent_id IS NULL
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []model.MenuBoard
	for rows.Next() {
		b := model.MenuBoard{}
		if err := rows.Scan(&b.Slug, &b.Name, &b.Color, &b.Icon, &b.Order); err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	return boards, nil
}

// ListMenuPages returns pages with showInMenu=true
func (r *Repository) ListMenuPages() ([]model.MenuPage, error) {
	rows, err := r.db.Query(`
		SELECT slug, title, icon, sort_order
		FROM posts
		WHERE type = 'page' AND show_in_menu = true
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []model.MenuPage
	for rows.Next() {
		p := model.MenuPage{}
		if err := rows.Scan(&p.Slug, &p.Title, &p.Icon, &p.Order); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// --- AI Cache ---

// GetAICache returns cached AI summary if exists
func (r *Repository) GetAICache(title string) (*model.AICache, error) {
	cache := &model.AICache{}
	err := r.db.QueryRow(`
		SELECT id, slug, title, tags, summary_text, model_used, created_at
		FROM ai_cache
		WHERE title = $1
	`, title).Scan(&cache.ID, &cache.Slug, &cache.Title, &cache.Tags,
		&cache.SummaryText, &cache.ModelUsed, &cache.CreatedAt)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

// SaveAICache stores an AI-generated summary
func (r *Repository) SaveAICache(slug, title, tags, summary, modelUsed string) error {
	_, err := r.db.Exec(`
		INSERT INTO ai_cache (slug, title, tags, summary_text, model_used)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (title) DO UPDATE SET
			summary_text = EXCLUDED.summary_text,
			model_used = EXCLUDED.model_used,
			created_at = NOW()
	`, slug, title, tags, summary, modelUsed)
	return err
}

// DeleteAICacheBySlug deletes cached AI summary for a slug
func (r *Repository) DeleteAICacheBySlug(slug string) error {
	_, err := r.db.Exec(`DELETE FROM ai_cache WHERE slug = $1`, slug)
	return err
}

// --- Auth ---

// GetUserByUsername returns user for authentication
func (r *Repository) GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(`
		SELECT id, username, password_hash, created_at
		FROM users WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdatePassword changes a user's password hash
func (r *Repository) UpdatePassword(username, newPasswordHash string) error {
	result, err := r.db.Exec(`UPDATE users SET password_hash = $1 WHERE username = $2`, newPasswordHash, username)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- Settings ---

// GetSetting returns a single setting value
func (r *Repository) GetSetting(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	return value, err
}

// SetSetting upserts a setting
func (r *Repository) SetSetting(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}

// GetAllSettings returns all settings as a map
func (r *Repository) GetAllSettings() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, nil
}

// HasAdminUser checks if any admin user exists
func (r *Repository) HasAdminUser() (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count > 0, err
}

// CreateUser creates a new admin user
func (r *Repository) CreateUser(username, passwordHash string) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id
	`, username, passwordHash).Scan(&id)
	return id, err
}

// --- Admin CRUD ---

func (r *Repository) CreatePost(slug, title, postType, date string, tags []string,
	excerpt, contentRaw, customFooter, customCSS string, cssEnabled bool,
	readTime, wordCount int,
	icon string, order int, showInMenu bool,
	cover, summary string, score float64, radarJSON, pagesJSON []byte,
	boardSlugs []string, status string) (int, error) {

	if status == "" {
		status = "published"
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	tagsStr := strings.Join(tags, ",")

	var postID int
	err = tx.QueryRow(`
		INSERT INTO posts (slug, title, type, status, date, tags, excerpt,
			content_raw, content_html, custom_footer, custom_css, css_enabled,
			read_time, word_count,
			icon, sort_order, show_in_menu, cover, summary, score,
			radar_charts, photo_pages)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id
	`, slug, title, postType, status, date, tagsStr, excerpt,
		contentRaw, contentRaw, customFooter, customCSS, cssEnabled,
		readTime, wordCount,
		icon, order, showInMenu, cover, summary, score,
		radarJSON, pagesJSON,
	).Scan(&postID)
	if err != nil {
		return 0, err
	}

	// Link to boards
	for _, boardSlug := range boardSlugs {
		_, err = tx.Exec(`
			INSERT INTO post_boards (post_id, board_id)
			SELECT $1, id FROM boards WHERE slug = $2
			ON CONFLICT DO NOTHING
		`, postID, boardSlug)
		if err != nil {
			return 0, err
		}
	}

	return postID, tx.Commit()
}

// UpdatePost updates an existing post's fields (only non-nil fields)
func (r *Repository) UpdatePost(slug string, title, date *string, tags *[]string,
	excerpt, contentRaw, customFooter *string,
	icon *string, order *int, showInMenu *bool,
	cover, summary *string, radarJSON, pagesJSON []byte,
	boardSlugs *[]string, status *string,
	customCSS *string, cssEnabled *bool) error {

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Build dynamic UPDATE
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	addField := func(col string, val interface{}) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}

	if title != nil {
		addField("title", *title)
	}
	if date != nil {
		addField("date", *date)
	}
	if tags != nil {
		addField("tags", strings.Join(*tags, ","))
	}
	if excerpt != nil {
		addField("excerpt", *excerpt)
	}
	if contentRaw != nil {
		addField("content_raw", *contentRaw)
		addField("content_html", *contentRaw)
		// Recalculate word count and read time
		wc := len([]rune(*contentRaw))
		rt := (wc + 399) / 400
		addField("word_count", wc)
		addField("read_time", rt)
	}
	if customFooter != nil {
		addField("custom_footer", *customFooter)
	}
	if customCSS != nil {
		addField("custom_css", *customCSS)
	}
	if cssEnabled != nil {
		addField("css_enabled", *cssEnabled)
	}
	if icon != nil {
		addField("icon", *icon)
	}
	if order != nil {
		addField("sort_order", *order)
	}
	if showInMenu != nil {
		addField("show_in_menu", *showInMenu)
	}
	if cover != nil {
		addField("cover", *cover)
	}
	if summary != nil {
		addField("summary", *summary)
	}
	if radarJSON != nil {
		addField("radar_charts", radarJSON)
	}
	if pagesJSON != nil {
		addField("photo_pages", pagesJSON)
	}
	if status != nil {
		addField("status", *status)
		if *status != "trashed" {
			addField("trashed_at", nil)
		}
	}

	if len(setClauses) == 0 && boardSlugs == nil {
		return nil // nothing to update
	}

	if len(setClauses) > 0 {
		query := fmt.Sprintf("UPDATE posts SET %s WHERE slug = $%d",
			strings.Join(setClauses, ", "), argIdx)
		args = append(args, slug)

		_, err = tx.Exec(query, args...)
		if err != nil {
			return err
		}
	}

	// Update board links if provided
	if boardSlugs != nil {
		// Get post ID
		var postID int
		err = tx.QueryRow(`SELECT id FROM posts WHERE slug = $1`, slug).Scan(&postID)
		if err != nil {
			return err
		}

		// Remove old links
		_, err = tx.Exec(`DELETE FROM post_boards WHERE post_id = $1`, postID)
		if err != nil {
			return err
		}

		// Add new links
		for _, bs := range *boardSlugs {
			_, err = tx.Exec(`
				INSERT INTO post_boards (post_id, board_id)
				SELECT $1, id FROM boards WHERE slug = $2
				ON CONFLICT DO NOTHING
			`, postID, bs)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// DeletePost permanently removes a post and its board associations
func (r *Repository) DeletePost(slug string) error {
	// Delete board links first
	r.db.Exec(`DELETE FROM post_boards WHERE post_id = (SELECT id FROM posts WHERE slug = $1)`, slug)
	result, err := r.db.Exec(`DELETE FROM posts WHERE slug = $1`, slug)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// TrashPost moves a post to the recycle bin (soft delete)
func (r *Repository) TrashPost(slug string) error {
	result, err := r.db.Exec(`UPDATE posts SET status = 'trashed', trashed_at = NOW() WHERE slug = $1`, slug)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RestorePost restores a post from the recycle bin
func (r *Repository) RestorePost(slug string) error {
	result, err := r.db.Exec(`UPDATE posts SET status = 'published', trashed_at = NULL WHERE slug = $1 AND status = 'trashed'`, slug)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListPostsByStatus returns posts filtered by status
func (r *Repository) ListPostsByStatus(status string, limit int) ([]model.SearchResult, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(`
		SELECT slug, title, type, date, tags, excerpt, score
		FROM posts WHERE status = $1
		ORDER BY CASE WHEN $1 = 'trashed' THEN trashed_at ELSE date END DESC
		LIMIT $2
	`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		sr := model.SearchResult{}
		var tagsStr, dateStr sql.NullString
		var score sql.NullFloat64
		if err := rows.Scan(&sr.Slug, &sr.Title, &sr.Type, &dateStr, &tagsStr, &sr.Excerpt, &score); err != nil {
			return nil, err
		}
		if dateStr.Valid {
			sr.Date = formatDateOnly(dateStr.String)
		}
		if tagsStr.Valid && tagsStr.String != "" {
			sr.Tags = parseTags(tagsStr.String)
		}
		if score.Valid {
			sr.Score = score.Float64
		}
		sr.Boards, _ = r.getPostBoardsBySlug(sr.Slug)
		results = append(results, sr)
	}
	return results, nil
}

// EmptyTrash permanently deletes all trashed posts
func (r *Repository) EmptyTrash() (int, error) {
	// Delete board links first
	r.db.Exec(`DELETE FROM post_boards WHERE post_id IN (SELECT id FROM posts WHERE status = 'trashed')`)
	result, err := r.db.Exec(`DELETE FROM posts WHERE status = 'trashed'`)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// CreateOrUpdateBoard creates or updates a board
func (r *Repository) CreateOrUpdateBoard(slug, name, color, icon string, order int, parentSlug *string) (int, error) {
	if color == "" {
		color = "#8b7aab"
	}
	if icon == "" {
		icon = "folder"
	}

	var boardID int
	err := r.db.QueryRow(`
		INSERT INTO boards (slug, name, color, icon, sort_order, parent_id)
		VALUES ($1, $2, $3, $4, $5, (SELECT id FROM boards WHERE slug = $6))
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			color = EXCLUDED.color,
			icon = EXCLUDED.icon,
			sort_order = EXCLUDED.sort_order,
			parent_id = EXCLUDED.parent_id
		RETURNING id
	`, slug, name, color, icon, order, parentSlug).Scan(&boardID)
	return boardID, err
}

// --- Helpers ---

// parseTags splits a comma-separated tag string into a slice, trimming spaces
func parseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// formatDateOnly normalizes a date string (possibly with time/timezone) to YYYY-MM-DD
func formatDateOnly(raw string) string {
	// Try common formats and extract date part
	for _, layout := range []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	// Fallback: return first 10 chars if looks like a date
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

