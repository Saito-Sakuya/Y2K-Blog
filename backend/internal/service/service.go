package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/y2k-pixel-blog/backend/internal/model"
	"github.com/y2k-pixel-blog/backend/internal/repository"
)

var ErrNotFound = errors.New("not found")
var ErrUnauthorized = errors.New("unauthorized")

// Service holds business logic
type Service struct {
	repo       *repository.Repository
	httpClient *http.Client
}

// New creates a new Service
func New(repo *repository.Repository) *Service {
	return &Service{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Boards ---

// ListBoards returns all top-level boards with their children
func (s *Service) ListBoards() ([]*model.Board, error) {
	return s.repo.ListBoards()
}

// GetBoardContent returns content list for a board with pagination
func (s *Service) GetBoardContent(slug string, page, limit int, sort, order, contentType string) (*model.BoardDetailResponse, error) {
	board, err := s.repo.GetBoardBySlug(slug)
	if err != nil {
		return nil, ErrNotFound
	}

	// Clamp pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, total, err := s.repo.ListBoardItems(board.ID, page, limit, sort, order, contentType)
	if err != nil {
		return nil, err
	}

	totalPages := (total + limit - 1) / limit

	return &model.BoardDetailResponse{
		Board: board,
		Items: items,
		Pagination: model.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// --- Posts ---

// GetPost returns a single post by slug
func (s *Service) GetPost(slug string) (*model.Post, error) {
	post, err := s.repo.GetPostBySlug(slug)
	if err != nil {
		return nil, ErrNotFound
	}
	return post, nil
}

// --- Search ---

// Search performs content search with advanced syntax parsing
//
// Syntax:
//   keyword        → match title + tags + excerpt
//   #tag           → filter by tag name
//   ~text          → fulltext body search
//   @board         → browse all posts in a board
//   @board keyword → search title within a board
//   @board #tag    → filter tag within a board
//   @board ~text   → fulltext search within a board
func (s *Service) Search(query string, limit int) (*model.SearchResponse, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return &model.SearchResponse{Query: query, Results: []model.SearchResult{}, Total: 0}, nil
	}

	searchType := "default"
	boardSlug := ""
	cleanQuery := query
	syntaxLabel := "default"

	// Parse @board prefix
	if strings.HasPrefix(query, "@") {
		rest := query[1:]
		// Split on first space: @board keyword
		parts := strings.SplitN(rest, " ", 2)
		boardSlug = strings.TrimSpace(parts[0])

		if len(parts) == 1 || strings.TrimSpace(parts[1]) == "" {
			// Just @board — browse mode
			searchType = "board_browse"
			cleanQuery = ""
			syntaxLabel = "@board"
		} else {
			// @board + sub-query
			subQuery := strings.TrimSpace(parts[1])
			if strings.HasPrefix(subQuery, "#") {
				searchType = "tag"
				cleanQuery = subQuery[1:]
				syntaxLabel = "@board #tag"
			} else if strings.HasPrefix(subQuery, "~") {
				searchType = "fulltext"
				cleanQuery = subQuery[1:]
				syntaxLabel = "@board ~fulltext"
			} else {
				searchType = "default"
				cleanQuery = subQuery
				syntaxLabel = "@board keyword"
			}
		}
	} else if strings.HasPrefix(query, "#") {
		searchType = "tag"
		cleanQuery = query[1:]
		syntaxLabel = "#tag"
	} else if strings.HasPrefix(query, "~") {
		searchType = "fulltext"
		cleanQuery = query[1:]
		syntaxLabel = "~fulltext"
	}

	cleanQuery = strings.TrimSpace(cleanQuery)

	// For board_browse, we don't need a query string
	if searchType != "board_browse" && cleanQuery == "" {
		return &model.SearchResponse{Query: query, Results: []model.SearchResult{}, Total: 0}, nil
	}

	results, err := s.repo.Search(cleanQuery, searchType, boardSlug, limit)
	if err != nil {
		return nil, err
	}

	if results == nil {
		results = []model.SearchResult{}
	}

	return &model.SearchResponse{
		Query:   query,
		Syntax:  syntaxLabel,
		Board:   boardSlug,
		Results: results,
		Total:   len(results),
	}, nil
}

// ListTags returns all tags with usage counts
func (s *Service) ListTags() (*model.TagListResponse, error) {
	tags, err := s.repo.ListAllTags()
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []model.TagItem{}
	}
	return &model.TagListResponse{
		Tags:  tags,
		Total: len(tags),
	}, nil
}

// --- Menu ---

// GetMenu returns menu data (top-level boards + showInMenu pages)
func (s *Service) GetMenu() (*model.MenuResponse, error) {
	boards, err := s.repo.ListTopLevelBoardsForMenu()
	if err != nil {
		return nil, err
	}
	if boards == nil {
		boards = []model.MenuBoard{}
	}

	pages, err := s.repo.ListMenuPages()
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []model.MenuPage{}
	}

	return &model.MenuResponse{
		Boards: boards,
		Pages:  pages,
	}, nil
}

// --- AI Summary ---

// GetAISummary returns AI-generated summary (cached or new)
func (s *Service) GetAISummary(title, tags string) (*model.AISummaryResponse, error) {
	// Check cache first
	cached, err := s.repo.GetAICache(title)
	if err == nil && cached != nil {
		return &model.AISummaryResponse{
			Title:       cached.Title,
			Summary:     cached.SummaryText,
			Status:      "ready",
			Cached:      true,
			Model:       cached.ModelUsed,
			GeneratedAt: cached.CreatedAt.Format(time.RFC3339),
		}, nil
	}

	// Try to generate via AI
	apiURL := os.Getenv("AI_API_URL")
	apiKey := os.Getenv("AI_API_KEY")
	aiModel := os.Getenv("AI_MODEL")
	if aiModel == "" {
		aiModel = "deepseek-chat"
	}

	// If no AI API configured, return placeholder
	if apiURL == "" || apiKey == "" {
		return &model.AISummaryResponse{
			Title:   title,
			Summary: fmt.Sprintf("《%s》是一部优秀的作品。（AI 简述未配置，请设置 AI_API_URL 和 AI_API_KEY 环境变量）", title),
			Status:  "ready",
			Cached:  false,
			Model:   "placeholder",
		}, nil
	}

	// Call OpenAI-compatible API
	summary, err := s.callAI(apiURL, apiKey, aiModel, title, tags)
	if err != nil {
		return &model.AISummaryResponse{
			Title:   title,
			Status:  "error",
			Message: fmt.Sprintf("AI 生成失败: %v", err),
			Cached:  false,
		}, nil
	}

	// Cache the result
	slug := strings.ReplaceAll(strings.ToLower(title), " ", "-")
	_ = s.repo.SaveAICache(slug, title, tags, summary, aiModel)

	return &model.AISummaryResponse{
		Title:       title,
		Summary:     summary,
		Status:      "ready",
		Cached:      false,
		Model:       aiModel,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// callAI sends a request to an OpenAI-compatible API
func (s *Service) callAI(apiURL, apiKey, aiModel, title, tags string) (string, error) {
	systemPrompt := "你是一个简洁客观的作品简介生成器。请用中文为以下作品生成一段约150字的客观介绍，包含作品类型、核心主题和主要特色。不要使用主观评价词汇。"
	userPrompt := fmt.Sprintf("作品名称：%s\n标签：%s", title, tags)

	reqBody := map[string]interface{}{
		"model": aiModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  300,
		"temperature": 0.7,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Ensure URL ends with /chat/completions
	url := strings.TrimRight(apiURL, "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("AI API returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse OpenAI response
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI returned no choices")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// --- Auth ---

// Login authenticates an admin user and returns JWT tokens
func (s *Service) Login(username, password string) (*model.LoginResponse, error) {
	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrUnauthorized
	}

	// Generate JWT
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "y2k-pixel-blog-dev-secret"
	}

	expiresIn := 3600 // 1 hour
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate refresh token (longer-lived)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})

	refreshStr, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &model.LoginResponse{
		Token:        tokenStr,
		RefreshToken: refreshStr,
		ExpiresIn:    expiresIn,
	}, nil
}

// ChangePassword verifies old password and updates to new one
func (s *Service) ChangePassword(username, oldPassword, newPassword string) error {
	// Validate new password
	if len(newPassword) < 6 {
		return fmt.Errorf("new password must be at least 6 characters")
	}

	// Verify old password
	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.repo.UpdatePassword(username, string(hash))
}

// --- Admin ---

// DeleteAICache removes cached AI summary for a slug
func (s *Service) DeleteAICache(slug string) error {
	return s.repo.DeleteAICacheBySlug(slug)
}

// CreatePost creates a new post
func (s *Service) CreatePost(req *model.CreatePostRequest) (int, error) {
	// Validate type
	validTypes := map[string]bool{"article": true, "photo": true, "rating": true, "page": true}
	if !validTypes[req.Type] {
		return 0, fmt.Errorf("invalid type: %s", req.Type)
	}

	// Calculate word count and read time
	wordCount := len([]rune(req.Content))
	readTime := (wordCount + 399) / 400

	// Serialize JSON fields
	var radarJSON, pagesJSON []byte
	if len(req.RadarCharts) > 0 {
		radarJSON, _ = json.Marshal(req.RadarCharts)
	}
	if len(req.Pages) > 0 {
		pagesJSON, _ = json.Marshal(req.Pages)
	}

	return s.repo.CreatePost(
		req.Slug, req.Title, req.Type, req.Date, req.Tags,
		req.Excerpt, req.Content, req.CustomFooter, req.CustomCSS, req.CSSEnabled,
		readTime, wordCount,
		req.Icon, req.Order, req.ShowInMenu,
		req.Cover, req.Summary, req.Score, radarJSON, pagesJSON,
		req.Boards, req.Status,
	)
}

// UpdatePost updates an existing post
func (s *Service) UpdatePost(slug string, req *model.UpdatePostRequest) error {
	var radarJSON, pagesJSON []byte
	if req.RadarCharts != nil {
		radarJSON, _ = json.Marshal(*req.RadarCharts)
	}
	if req.Pages != nil {
		pagesJSON, _ = json.Marshal(*req.Pages)
	}

	return s.repo.UpdatePost(
		slug, req.Title, req.Date, req.Tags,
		req.Excerpt, req.Content, req.CustomFooter,
		req.Icon, req.Order, req.ShowInMenu,
		req.Cover, req.Summary, radarJSON, pagesJSON,
		req.Boards, req.Status,
		req.CustomCSS, req.CSSEnabled,
	)
}

// DeletePost permanently deletes a post
func (s *Service) DeletePost(slug string) error {
	return s.repo.DeletePost(slug)
}

// TrashPost moves a post to the recycle bin
func (s *Service) TrashPost(slug string) error {
	return s.repo.TrashPost(slug)
}

// RestorePost restores a post from the recycle bin
func (s *Service) RestorePost(slug string) error {
	return s.repo.RestorePost(slug)
}

// ListPostsByStatus lists posts by status (draft, trashed, published)
func (s *Service) ListPostsByStatus(status string, limit int) ([]model.SearchResult, error) {
	validStatuses := map[string]bool{"published": true, "draft": true, "trashed": true}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}
	return s.repo.ListPostsByStatus(status, limit)
}

// EmptyTrash permanently deletes all trashed posts
func (s *Service) EmptyTrash() (int, error) {
	return s.repo.EmptyTrash()
}

// --- Preview ---

// GeneratePreviewToken creates a short-lived JWT for previewing a draft/post
func (s *Service) GeneratePreviewToken(slug string) (string, error) {
	// Verify post exists
	post, err := s.repo.GetPostBySlug(slug)
	if err != nil {
		return "", fmt.Errorf("post not found: %w", err)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "y2k-pixel-blog-dev-secret"
	}

	// Token valid for 15 minutes
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"purpose": "preview",
		"slug":    post.Slug,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenStr, nil
}

// GetPostByPreviewToken validates a preview token and returns the full post
func (s *Service) GetPostByPreviewToken(tokenStr string) (*model.Post, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "y2k-pixel-blog-dev-secret"
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired preview token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check purpose
	purpose, _ := claims["purpose"].(string)
	if purpose != "preview" {
		return nil, fmt.Errorf("token is not a preview token")
	}

	slug, _ := claims["slug"].(string)
	if slug == "" {
		return nil, fmt.Errorf("token missing slug")
	}

	// Get the post regardless of status (draft, trashed, published)
	return s.repo.GetPostBySlug(slug)
}

// CreateBoard creates or updates a board
func (s *Service) CreateBoard(req *model.CreateBoardRequest) (int, error) {
	return s.repo.CreateOrUpdateBoard(
		req.Slug, req.Name, req.Color, req.Icon, req.Order, req.Parent,
	)
}

// --- Setup & Settings ---

// CheckSetupNeeded returns whether the initial setup is required
func (s *Service) CheckSetupNeeded() (*model.SetupStatusResponse, error) {
	// Check if setup_completed flag exists
	completed, err := s.repo.GetSetting("setup_completed")
	if err == nil && completed == "true" {
		return &model.SetupStatusResponse{NeedsSetup: false}, nil
	}

	// Also check if admin user exists (legacy: might have been set up manually)
	hasAdmin, err := s.repo.HasAdminUser()
	if err != nil {
		return nil, err
	}
	if hasAdmin {
		return &model.SetupStatusResponse{NeedsSetup: false}, nil
	}

	return &model.SetupStatusResponse{
		NeedsSetup: true,
		Message:    "欢迎！请完成初始设置以开始使用 Y2K Pixel Blog。",
	}, nil
}

// Initialize performs the one-time setup
func (s *Service) Initialize(req *model.SetupRequest) error {
	// Guard: check if already initialized
	status, err := s.CheckSetupNeeded()
	if err != nil {
		return err
	}
	if !status.NeedsSetup {
		return fmt.Errorf("setup already completed")
	}

	// 1. Create admin user
	hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	_, err = s.repo.CreateUser(req.AdminUsername, string(hash))
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// 2. Save site settings
	settings := map[string]string{
		"site_title":       req.SiteTitle,
		"site_description": req.SiteDescription,
		"site_footer":      req.SiteFooter,
		"setup_completed":  "true",
	}
	if req.AIApiURL != "" {
		settings["ai_api_url"] = req.AIApiURL
	}
	if req.AIApiKey != "" {
		settings["ai_api_key"] = req.AIApiKey
	}
	if req.AIModel != "" {
		settings["ai_model"] = req.AIModel
	}

	for k, v := range settings {
		if err := s.repo.SetSetting(k, v); err != nil {
			return fmt.Errorf("failed to save setting %s: %w", k, err)
		}
	}

	// 3. Create initial board if provided
	if req.FirstBoardSlug != "" && req.FirstBoardName != "" {
		icon := req.FirstBoardIcon
		if icon == "" {
			icon = "folder"
		}
		_, err = s.repo.CreateOrUpdateBoard(req.FirstBoardSlug, req.FirstBoardName, "#8b7aab", icon, 1, nil)
		if err != nil {
			return fmt.Errorf("failed to create initial board: %w", err)
		}
	}

	return nil
}

// GetSettings returns all site settings
func (s *Service) GetSettings() (*model.SiteSettings, error) {
	all, err := s.repo.GetAllSettings()
	if err != nil {
		return nil, err
	}

	settings := &model.SiteSettings{
		SiteTitle:             all["site_title"],
		SiteDescription:       all["site_description"],
		SiteFooter:            all["site_footer"],
		SiteLogoURL:           all["site_logo_url"],
		SiteLicense:           all["site_license"],
		SiteLicenseURL:        all["site_license_url"],
		AIApiURL:              all["ai_api_url"],
		AIModel:               all["ai_model"],
		SetupCompleted:        all["setup_completed"],
		GlobalCSS:             all["global_css"],
		CustomCSSEnabledTypes: all["custom_css_enabled_types"],
		FrontendDomain:        all["frontend_domain"],
		AdminDomain:           all["admin_domain"],
		FrontendSSLEnabled:    all["frontend_ssl_enabled"],
		FrontendSSLHasCert:    all["frontend_ssl_cert_pem"] != "" && all["frontend_ssl_key_pem"] != "",
		FrontendSSLMode:       all["frontend_ssl_mode"],
		AdminSSLEnabled:       all["admin_ssl_enabled"],
		AdminSSLHasCert:       all["admin_ssl_cert_pem"] != "" && all["admin_ssl_key_pem"] != "",
		AdminSSLMode:          all["admin_ssl_mode"],
		AcmeEmail:             all["acme_email"],
	}

	// Mask API key
	if key, ok := all["ai_api_key"]; ok && key != "" {
		if len(key) > 8 {
			settings.AIApiKey = key[:4] + "****" + key[len(key)-4:]
		} else {
			settings.AIApiKey = "****"
		}
	}

	return settings, nil
}

// UpdateSettings updates site settings
func (s *Service) UpdateSettings(req *model.UpdateSettingsRequest) error {
	if req.SiteTitle != nil {
		if err := s.repo.SetSetting("site_title", *req.SiteTitle); err != nil {
			return err
		}
	}
	if req.SiteDescription != nil {
		if err := s.repo.SetSetting("site_description", *req.SiteDescription); err != nil {
			return err
		}
	}
	if req.SiteFooter != nil {
		if err := s.repo.SetSetting("site_footer", *req.SiteFooter); err != nil {
			return err
		}
	}
	if req.SiteLogoURL != nil {
		if err := s.repo.SetSetting("site_logo_url", *req.SiteLogoURL); err != nil {
			return err
		}
	}
	if req.SiteLicense != nil {
		if err := s.repo.SetSetting("site_license", *req.SiteLicense); err != nil {
			return err
		}
	}
	if req.SiteLicenseURL != nil {
		if err := s.repo.SetSetting("site_license_url", *req.SiteLicenseURL); err != nil {
			return err
		}
	}
	if req.AIApiURL != nil {
		if err := s.repo.SetSetting("ai_api_url", *req.AIApiURL); err != nil {
			return err
		}
	}
	if req.AIApiKey != nil {
		if err := s.repo.SetSetting("ai_api_key", *req.AIApiKey); err != nil {
			return err
		}
	}
	if req.AIModel != nil {
		if err := s.repo.SetSetting("ai_model", *req.AIModel); err != nil {
			return err
		}
	}
	if req.GlobalCSS != nil {
		if err := s.repo.SetSetting("global_css", *req.GlobalCSS); err != nil {
			return err
		}
	}
	if req.CustomCSSEnabledTypes != nil {
		if err := s.repo.SetSetting("custom_css_enabled_types", *req.CustomCSSEnabledTypes); err != nil {
			return err
		}
	}
	if req.FrontendDomain != nil {
		if err := s.repo.SetSetting("frontend_domain", *req.FrontendDomain); err != nil {
			return err
		}
	}
	if req.AdminDomain != nil {
		if err := s.repo.SetSetting("admin_domain", *req.AdminDomain); err != nil {
			return err
		}
	}
	if req.FrontendSSLMode != nil {
		if err := s.repo.SetSetting("frontend_ssl_mode", *req.FrontendSSLMode); err != nil {
			return err
		}
	}
	if req.AdminSSLMode != nil {
		if err := s.repo.SetSetting("admin_ssl_mode", *req.AdminSSLMode); err != nil {
			return err
		}
	}
	if req.AcmeEmail != nil {
		if err := s.repo.SetSetting("acme_email", *req.AcmeEmail); err != nil {
			return err
		}
	}
	return nil
}

// ConfigureSSL saves SSL certificate and key for a target, validates they form a valid pair
func (s *Service) ConfigureSSL(target, certPEM, keyPEM string, enabled bool) error {
	if target != "frontend" && target != "admin" {
		return fmt.Errorf("invalid target")
	}

	// Validate cert and key are a valid PEM pair
	if certPEM != "" && keyPEM != "" {
		_, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			return fmt.Errorf("invalid certificate/key pair: %v", err)
		}
	}

	prefix := target + "_"
	if err := s.repo.SetSetting(prefix+"ssl_cert_pem", certPEM); err != nil {
		return err
	}
	if err := s.repo.SetSetting(prefix+"ssl_key_pem", keyPEM); err != nil {
		return err
	}

	enabledStr := "false"
	if enabled && certPEM != "" && keyPEM != "" {
		enabledStr = "true"
	}
	if err := s.repo.SetSetting(prefix+"ssl_enabled", enabledStr); err != nil {
		return err
	}

	return nil
}

// SSLFullConfig holds all SSL configuration for the TLS server
type SSLFullConfig struct {
	FrontCert, FrontKey, FrontDomain, FrontMode string
	AdminCert, AdminKey, AdminDomain, AdminMode string
	FrontEnabled, AdminEnabled                  bool
	AcmeEmail                                   string
}

// GetSSLConfig returns SSL configurations for the TLS server
func (s *Service) GetSSLConfig() (*SSLFullConfig, error) {
	all, err := s.repo.GetAllSettings()
	if err != nil {
		return nil, err
	}
	return &SSLFullConfig{
		FrontCert:    all["frontend_ssl_cert_pem"],
		FrontKey:     all["frontend_ssl_key_pem"],
		FrontDomain:  all["frontend_domain"],
		FrontMode:    all["frontend_ssl_mode"],
		FrontEnabled: all["frontend_ssl_enabled"] == "true",
		AdminCert:    all["admin_ssl_cert_pem"],
		AdminKey:     all["admin_ssl_key_pem"],
		AdminDomain:  all["admin_domain"],
		AdminMode:    all["admin_ssl_mode"],
		AdminEnabled: all["admin_ssl_enabled"] == "true",
		AcmeEmail:    all["acme_email"],
	}, nil
}

// RemoveSSL clears SSL configuration for a target
func (s *Service) RemoveSSL(target string) error {
	if target != "frontend" && target != "admin" {
		return fmt.Errorf("invalid target")
	}

	prefix := target + "_"
	if err := s.repo.SetSetting(prefix+"ssl_cert_pem", ""); err != nil {
		return err
	}
	if err := s.repo.SetSetting(prefix+"ssl_key_pem", ""); err != nil {
		return err
	}
	if err := s.repo.SetSetting(prefix+"ssl_enabled", "false"); err != nil {
		return err
	}
	return nil
}
