package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/y2k-pixel-blog/backend/internal/middleware"
	"github.com/y2k-pixel-blog/backend/internal/model"
	"github.com/y2k-pixel-blog/backend/internal/service"
)

// Handler holds all HTTP handlers
type Handler struct {
	svc *service.Service
}

// New creates a new Handler
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// ListBoards handles GET /api/boards
func (h *Handler) ListBoards(c *gin.Context) {
	boards, err := h.svc.ListBoards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to list boards",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, model.BoardListResponse{Boards: boards})
}

// GetBoardContent handles GET /api/boards/:slug
func (h *Handler) GetBoardContent(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sort := c.DefaultQuery("sort", "date")
	order := c.DefaultQuery("order", "desc")
	contentType := c.Query("type")

	result, err := h.svc.GetBoardContent(slug, page, limit, sort, order, contentType)
	if err != nil {
		if err == service.ErrNotFound {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Error: "board not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get board content",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetPost handles GET /api/posts/:slug
func (h *Handler) GetPost(c *gin.Context) {
	slug := c.Param("slug")

	post, err := h.svc.GetPost(slug)
	if err != nil {
		if err == service.ErrNotFound {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Error: "post not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get post",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, post)
}

// Search handles GET /api/search
func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if query == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "query parameter 'q' is required",
			Code:  "BAD_REQUEST",
		})
		return
	}

	result, err := h.svc.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "search failed",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListTags handles GET /api/tags
func (h *Handler) ListTags(c *gin.Context) {
	result, err := h.svc.ListTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to list tags",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetMenu handles GET /api/menu
func (h *Handler) GetMenu(c *gin.Context) {
	menu, err := h.svc.GetMenu()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get menu",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, menu)
}

// GetCSSConfig handles GET /api/css-config (public)
func (h *Handler) GetCSSConfig(c *gin.Context) {
	settings, err := h.svc.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get CSS config",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"globalCSS":             settings.GlobalCSS,
		"customCSSEnabledTypes": settings.CustomCSSEnabledTypes,
	})
}

// GetAISummary handles GET /api/ai/summary
func (h *Handler) GetAISummary(c *gin.Context) {
	title := c.Query("title")
	tags := c.Query("tags")

	if title == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "query parameter 'title' is required",
			Code:  "BAD_REQUEST",
		})
		return
	}

	result, err := h.svc.GetAISummary(title, tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get AI summary",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Login handles POST /api/admin/login
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username     string `json:"username" binding:"required"`
		Password     string `json:"password" binding:"required"`
		CaptchaToken string `json:"captchaToken"`
		CaptchaAnswer *int  `json:"captchaAnswer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid request body",
			Code:  "BAD_REQUEST",
		})
		return
	}

	// Get rate limiter from context
	rl, exists := c.Get("rateLimiter")
	if exists {
		rateLimiter := rl.(*middleware.RateLimiter)
		ip := c.ClientIP()
		failCount, _, needsCaptcha := rateLimiter.GetIPStatus(ip)

		// Check if captcha is required
		if needsCaptcha {
			if req.CaptchaToken == "" || req.CaptchaAnswer == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":        "captcha required due to failed attempts",
					"code":         "CAPTCHA_REQUIRED",
					"needsCaptcha": true,
					"failCount":    failCount,
				})
				return
			}

			// Verify captcha
			if !rateLimiter.VerifyCaptcha(req.CaptchaToken, *req.CaptchaAnswer) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":        "incorrect captcha answer",
					"code":         "CAPTCHA_INVALID",
					"needsCaptcha": true,
					"failCount":    failCount,
				})
				return
			}
		}
	}

	result, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		// Record failed attempt
		resp := gin.H{
			"error": "invalid credentials",
			"code":  "UNAUTHORIZED",
		}
		if exists {
			rateLimiter := rl.(*middleware.RateLimiter)
			ip := c.ClientIP()
			failCount, blocked := rateLimiter.RecordFail(ip)
			resp["failCount"] = failCount
			if blocked {
				resp["error"] = "Too many failed attempts. IP blocked for 15 minutes."
				resp["code"] = "IP_BLOCKED"
				c.JSON(http.StatusTooManyRequests, resp)
				return
			}
			_, _, needsCaptcha := rateLimiter.GetIPStatus(ip)
			resp["needsCaptcha"] = needsCaptcha
		}
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	// Record success — reset counter
	if exists {
		rateLimiter := rl.(*middleware.RateLimiter)
		rateLimiter.RecordSuccess(c.ClientIP())
	}
	c.JSON(http.StatusOK, result)
}

// CreatePost handles POST /api/admin/posts
func (h *Handler) CreatePost(c *gin.Context) {
	var req model.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid request body: " + err.Error(),
			Code:  "BAD_REQUEST",
		})
		return
	}

	id, err := h.svc.CreatePost(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to create post: " + err.Error(),
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "slug": req.Slug, "message": "post created"})
}

// UpdatePost handles PUT /api/admin/posts/:slug
func (h *Handler) UpdatePost(c *gin.Context) {
	slug := c.Param("slug")
	var req model.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid request body: " + err.Error(),
			Code:  "BAD_REQUEST",
		})
		return
	}

	if err := h.svc.UpdatePost(slug, &req); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to update post: " + err.Error(),
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"slug": slug, "message": "post updated"})
}

// DeletePost handles DELETE /api/admin/posts/:slug
func (h *Handler) DeletePost(c *gin.Context) {
	slug := c.Param("slug")
	if err := h.svc.DeletePost(slug); err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: "post not found",
			Code:  "NOT_FOUND",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"slug": slug, "message": "post deleted"})
}

// CreateBoard handles POST /api/admin/boards
func (h *Handler) CreateBoard(c *gin.Context) {
	var req model.CreateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid request body: " + err.Error(),
			Code:  "BAD_REQUEST",
		})
		return
	}

	id, err := h.svc.CreateBoard(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to create board: " + err.Error(),
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "slug": req.Slug, "message": "board created"})
}

// DeleteAICache handles DELETE /api/admin/ai-cache/:slug
func (h *Handler) DeleteAICache(c *gin.Context) {
	slug := c.Param("slug")
	if err := h.svc.DeleteAICache(slug); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to delete AI cache",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AI cache cleared", "slug": slug})
}

// ChangePassword handles PUT /api/admin/password
func (h *Handler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "oldPassword and newPassword are required",
			Code:  "BAD_REQUEST",
		})
		return
	}

	// Get username from JWT context
	username, _ := c.Get("username")
	usernameStr, ok := username.(string)
	if !ok || usernameStr == "" {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error: "invalid session",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if err := h.svc.ChangePassword(usernameStr, req.OldPassword, req.NewPassword); err != nil {
		code := "BAD_REQUEST"
		status := http.StatusBadRequest
		if err.Error() == "current password is incorrect" {
			code = "WRONG_PASSWORD"
			status = http.StatusForbidden
		}
		c.JSON(status, model.ErrorResponse{
			Error: err.Error(),
			Code:  code,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// --- Trash & Drafts ---

// TrashPost handles POST /api/admin/posts/:slug/trash
func (h *Handler) TrashPost(c *gin.Context) {
	slug := c.Param("slug")
	if err := h.svc.TrashPost(slug); err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: "post not found",
			Code:  "NOT_FOUND",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "post moved to trash", "slug": slug})
}

// RestorePost handles POST /api/admin/posts/:slug/restore
func (h *Handler) RestorePost(c *gin.Context) {
	slug := c.Param("slug")
	if err := h.svc.RestorePost(slug); err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: "post not found in trash",
			Code:  "NOT_FOUND",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "post restored", "slug": slug})
}

// ListPostsByStatus handles GET /api/admin/posts?status=draft|trashed|published
func (h *Handler) ListPostsByStatus(c *gin.Context) {
	status := c.DefaultQuery("status", "published")
	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil {
		limit = l
	}

	results, err := h.svc.ListPostsByStatus(status, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: err.Error(),
			Code:  "BAD_REQUEST",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"results": results,
		"total":   len(results),
	})
}

// EmptyTrash handles DELETE /api/admin/trash
func (h *Handler) EmptyTrash(c *gin.Context) {
	count, err := h.svc.EmptyTrash()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to empty trash",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "trash emptied", "deleted": count})
}

// --- Preview ---

// GeneratePreviewToken handles POST /api/admin/preview/:slug
func (h *Handler) GeneratePreviewToken(c *gin.Context) {
	slug := c.Param("slug")
	token, err := h.svc.GeneratePreviewToken(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: err.Error(),
			Code:  "NOT_FOUND",
		})
		return
	}

	// Build preview URL: DB frontend_domain > env FRONTEND_URL > localhost
	frontendURL := ""
	settings, _ := h.svc.GetSettings()
	if settings != nil && settings.FrontendDomain != "" {
		scheme := "https"
		if settings.FrontendSSLEnabled != "true" {
			scheme = "http"
		}
		frontendURL = scheme + "://" + settings.FrontendDomain
	}
	if frontendURL == "" {
		frontendURL = os.Getenv("FRONTEND_URL")
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	previewURL := frontendURL + "/preview/" + token

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"slug":       slug,
		"expiresIn":  900,
		"previewUrl": previewURL,
	})
}

// GetPreview handles GET /api/preview/:token (public, token-secured)
func (h *Handler) GetPreview(c *gin.Context) {
	token := c.Param("token")
	post, err := h.svc.GetPostByPreviewToken(token)
	if err != nil {
		status := http.StatusUnauthorized
		code := "INVALID_TOKEN"
		if err.Error() == "invalid or expired preview token" {
			code = "TOKEN_EXPIRED"
		}
		c.JSON(status, model.ErrorResponse{
			Error: err.Error(),
			Code:  code,
		})
		return
	}
	c.JSON(http.StatusOK, post)
}

// --- Setup & Settings ---

// GetSetupStatus handles GET /api/setup/status
func (h *Handler) GetSetupStatus(c *gin.Context) {
	status, err := h.svc.CheckSetupNeeded()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to check setup status",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, status)
}

// Initialize handles POST /api/setup/initialize
func (h *Handler) Initialize(c *gin.Context) {
	var req model.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid request: " + err.Error(),
			Code:  "BAD_REQUEST",
		})
		return
	}

	// Password validation
	if len(req.AdminPassword) < 6 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "密码长度至少 6 位",
			Code:  "BAD_REQUEST",
		})
		return
	}

	if err := h.svc.Initialize(&req); err != nil {
		code := "INTERNAL_ERROR"
		status := http.StatusInternalServerError
		if err.Error() == "setup already completed" {
			code = "ALREADY_SETUP"
			status = http.StatusConflict
		}
		c.JSON(status, model.ErrorResponse{
			Error: err.Error(),
			Code:  code,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "🎉 初始化完成！请使用新账户登录。",
		"username": req.AdminUsername,
	})
}

// GetSettings handles GET /api/admin/settings
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.svc.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get settings",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, settings)
}

// UpdateSettings handles PUT /api/admin/settings
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req model.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid request: " + err.Error(),
			Code:  "BAD_REQUEST",
		})
		return
	}

	if err := h.svc.UpdateSettings(&req); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to update settings",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

// ConfigureSSL handles PUT /api/admin/ssl
func (h *Handler) ConfigureSSL(c *gin.Context) {
	var req model.SSLConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "target, certPEM and keyPEM are required",
			Code:  "BAD_REQUEST",
		})
		return
	}

	if err := h.svc.ConfigureSSL(req.Target, req.CertPEM, req.KeyPEM, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_CERT",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "SSL configured successfully",
		"restartRequired": true,
	})
}

// RemoveSSL handles DELETE /api/admin/ssl
func (h *Handler) RemoveSSL(c *gin.Context) {
	target := c.DefaultQuery("target", "")
	if err := h.svc.RemoveSSL(target); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to remove SSL",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":         "SSL removed",
		"restartRequired": true,
	})
}

// GetSiteConfig handles GET /api/site-config — public site branding (no auth required)
func (h *Handler) GetSiteConfig(c *gin.Context) {
	settings, err := h.svc.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to get settings",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	// Check if setup has been completed
	setupStatus, _ := h.svc.CheckSetupNeeded()
	isSetup := true
	if setupStatus != nil && setupStatus.NeedsSetup {
		isSetup = false
	}

	c.JSON(http.StatusOK, gin.H{
		"siteTitle":       settings.SiteTitle,
		"siteDescription": settings.SiteDescription,
		"siteLogoUrl":     settings.SiteLogoURL,
		"siteLicense":     settings.SiteLicense,
		"siteLicenseUrl":  settings.SiteLicenseURL,
		"siteFooter":      settings.SiteFooter,
		"isSetup":         isSetup,
	})
}

// --- SEO Handlers ---

// helper: get base URL from settings
func (h *Handler) getBaseURL() string {
	settings, _ := h.svc.GetSettings()
	if settings != nil && settings.FrontendDomain != "" {
		scheme := "http"
		if settings.FrontendSSLEnabled == "true" {
			scheme = "https"
		}
		return scheme + "://" + settings.FrontendDomain
	}
	return "http://localhost:3000"
}

// RSS feed types
type rssChannel struct {
	XMLName       xml.Name  `xml:"channel"`
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Category    string `xml:"category,omitempty"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

// RSSFeed handles GET /feed.xml
func (h *Handler) RSSFeed(c *gin.Context) {
	baseURL := h.getBaseURL()
	settings, _ := h.svc.GetSettings()

	siteTitle := "Y2K Pixel Blog"
	siteDesc := ""
	if settings != nil {
		if settings.SiteTitle != "" {
			siteTitle = settings.SiteTitle
		}
		siteDesc = settings.SiteDescription
	}

	posts, err := h.svc.ListPostsByStatus("published", 50)
	if err != nil {
		c.XML(http.StatusInternalServerError, nil)
		return
	}

	var items []rssItem
	for _, p := range posts {
		link := fmt.Sprintf("%s/posts/%s", baseURL, p.Slug)
		cat := ""
		if len(p.Tags) > 0 {
			cat = strings.Join(p.Tags, ", ")
		}
		pubDate := ""
		if p.Date != "" {
			if t, err := time.Parse("2006-01-02", p.Date); err == nil {
				pubDate = t.Format(time.RFC1123Z)
			}
		}
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        link,
			Description: p.Excerpt,
			PubDate:     pubDate,
			GUID:        link,
			Category:    cat,
		})
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         siteTitle,
			Link:          baseURL,
			Description:   siteDesc,
			Language:      "zh-CN",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	xmlBytes, _ := xml.MarshalIndent(feed, "", "  ")
	c.String(http.StatusOK, xml.Header+string(xmlBytes))
}

// Sitemap types
type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapIndex struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// Sitemap handles GET /sitemap.xml
func (h *Handler) Sitemap(c *gin.Context) {
	baseURL := h.getBaseURL()

	urls := []sitemapURL{
		{Loc: baseURL + "/", ChangeFreq: "daily", Priority: "1.0"},
	}

	// Add all boards
	boards, _ := h.svc.ListBoards()
	for _, b := range boards {
		urls = append(urls, sitemapURL{
			Loc:        fmt.Sprintf("%s/boards/%s", baseURL, b.Slug),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})
	}

	// Add all published posts
	posts, _ := h.svc.ListPostsByStatus("published", 1000)
	for _, p := range posts {
		lastMod := ""
		if p.Date != "" {
			lastMod = p.Date
		}
		urls = append(urls, sitemapURL{
			Loc:        fmt.Sprintf("%s/posts/%s", baseURL, p.Slug),
			LastMod:    lastMod,
			ChangeFreq: "monthly",
			Priority:   "0.6",
		})
	}

	sitemap := sitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	xmlBytes, _ := xml.MarshalIndent(sitemap, "", "  ")
	c.String(http.StatusOK, xml.Header+string(xmlBytes))
}

// RobotsTxt handles GET /robots.txt
func (h *Handler) RobotsTxt(c *gin.Context) {
	baseURL := h.getBaseURL()
	body := fmt.Sprintf(`User-agent: *
Allow: /

Sitemap: %s/sitemap.xml
`, baseURL)
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400")
	c.String(http.StatusOK, body)
}

// OpenGraph handles GET /api/og/:slug — returns OG metadata for a post
func (h *Handler) OpenGraph(c *gin.Context) {
	slug := c.Param("slug")
	post, err := h.svc.GetPost(slug)
	if err != nil || post == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error: "post not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	baseURL := h.getBaseURL()
	settings, _ := h.svc.GetSettings()
	siteName := "Y2K Pixel Blog"
	if settings != nil && settings.SiteTitle != "" {
		siteName = settings.SiteTitle
	}

	ogType := "article"
	if post.Type == "page" {
		ogType = "website"
	}

	c.JSON(http.StatusOK, gin.H{
		"title":       post.Title,
		"description": post.Excerpt,
		"url":         fmt.Sprintf("%s/posts/%s", baseURL, post.Slug),
		"siteName":    siteName,
		"type":        ogType,
		"image":       post.Cover,
		"tags":        post.Tags,
		"publishedAt": post.Date,
	})
}
