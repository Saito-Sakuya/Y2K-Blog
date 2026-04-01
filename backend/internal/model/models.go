package model

import "time"

// Board represents a content category/folder
type Board struct {
	ID        int       `json:"id" db:"id"`
	Slug      string    `json:"slug" db:"slug"`
	Name      string    `json:"name" db:"name"`
	Color     string    `json:"color" db:"color"`
	Icon      string    `json:"icon" db:"icon"`
	Order     int       `json:"order" db:"sort_order"`
	ParentID  *int      `json:"parentId,omitempty" db:"parent_id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	// Computed fields (not stored in DB directly)
	PostCount int      `json:"postCount,omitempty"`
	Children  []*Board `json:"children,omitempty"`
}

// Post represents any content type: article, photo, rating, page
type Post struct {
	ID           int       `json:"id" db:"id"`
	Slug         string    `json:"slug" db:"slug"`
	Title        string    `json:"title" db:"title"`
	Type         string    `json:"type" db:"type"` // article, photo, rating, page
	Status       string    `json:"status" db:"status"` // published, draft, trashed
	Date         string    `json:"date" db:"date"`
	Tags         []string  `json:"tags" db:"-"`
	TagsRaw      string    `json:"-" db:"tags"` // comma-separated in DB
	Boards       []string  `json:"boards" db:"-"`
	BoardsRaw    string    `json:"-" db:"boards"` // comma-separated in DB
	Excerpt      string    `json:"excerpt" db:"excerpt"`
	ContentRaw   string    `json:"contentRaw,omitempty" db:"content_raw"`
	Content      string    `json:"content,omitempty" db:"content_html"`
	CustomFooter string    `json:"customFooter,omitempty" db:"custom_footer"`
	CustomCSS    string    `json:"customCSS,omitempty" db:"custom_css"`
	CSSEnabled   bool      `json:"cssEnabled" db:"css_enabled"`
	ReadTime     int       `json:"readTime,omitempty" db:"read_time"`
	WordCount    int       `json:"wordCount,omitempty" db:"word_count"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`

	// Page-specific fields
	Icon       string `json:"icon,omitempty" db:"icon"`
	Order      int    `json:"order,omitempty" db:"sort_order"`
	ShowInMenu bool   `json:"showInMenu,omitempty" db:"show_in_menu"`

	// Rating-specific fields
	Cover       string       `json:"cover,omitempty" db:"cover"`
	Summary     string       `json:"summary,omitempty" db:"summary"`
	Score       float64      `json:"score,omitempty" db:"score"`
	RadarCharts []RadarChart `json:"radarCharts,omitempty" db:"-"`

	// Photo-specific fields
	Pages []PhotoPage `json:"pages,omitempty" db:"-"`
}

// PhotoPage represents a single page in a photo essay
type PhotoPage struct {
	Image   string `json:"image"`
	Text    string `json:"text"`    // rendered HTML
	TextRaw string `json:"textRaw"` // raw markdown
}

// RadarChart represents a radar chart group in a rating
type RadarChart struct {
	Name string     `json:"name"`
	Axes []RadarAxis `json:"axes"`
}

// RadarAxis represents a single axis/dimension in a radar chart
type RadarAxis struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// AICache represents cached AI-generated summaries
type AICache struct {
	ID          int       `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`
	Title       string    `json:"title" db:"title"`
	Tags        string    `json:"tags" db:"tags"`
	SummaryText string    `json:"summary" db:"summary_text"`
	ModelUsed   string    `json:"model" db:"model_used"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// User represents an admin user
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// --- API Response Types ---

// BoardListResponse is the response for GET /api/boards
type BoardListResponse struct {
	Boards []*Board `json:"boards"`
}

// BoardDetailResponse is the response for GET /api/boards/:slug
type BoardDetailResponse struct {
	Board      *Board          `json:"board"`
	Items      []BoardListItem `json:"items"`
	Pagination Pagination      `json:"pagination"`
}

// BoardListItem can be a post or a sub-board
type BoardListItem struct {
	Slug      string   `json:"slug"`
	Title     string   `json:"title,omitempty"`
	Name      string   `json:"name,omitempty"` // for sub-boards
	Type      string   `json:"type"`           // article, photo, rating, page, board
	Date      string   `json:"date,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Excerpt   string   `json:"excerpt,omitempty"`
	ReadTime  int      `json:"readTime,omitempty"`
	Score     float64  `json:"score,omitempty"` // rating only
	Icon      string   `json:"icon,omitempty"`
	Color     string   `json:"color,omitempty"`
	PostCount int      `json:"postCount,omitempty"`
}

// SearchResponse is the response for GET /api/search
type SearchResponse struct {
	Query   string         `json:"query"`
	Syntax  string         `json:"syntax,omitempty"` // parsed syntax type for debugging
	Board   string         `json:"board,omitempty"`  // scoped board if @board used
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// SearchResult is a single search result
type SearchResult struct {
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Date      string   `json:"date,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Excerpt   string   `json:"excerpt,omitempty"`
	Boards    []string `json:"boards,omitempty"`
	Score     float64  `json:"score,omitempty"`
	Relevance float64  `json:"relevance,omitempty"`
}

// TagItem represents a tag with its usage count
type TagItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// TagListResponse is the response for GET /api/tags
type TagListResponse struct {
	Tags  []TagItem `json:"tags"`
	Total int       `json:"total"`
}

// MenuResponse is the response for GET /api/menu
type MenuResponse struct {
	Boards []MenuBoard `json:"boards"`
	Pages  []MenuPage  `json:"pages"`
}

// MenuBoard is a board entry in the menu
type MenuBoard struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
}

// MenuPage is a page entry in the menu
type MenuPage struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
}

// AISummaryResponse is the response for GET /api/ai/summary
type AISummaryResponse struct {
	Title       string `json:"title"`
	Summary     string `json:"summary,omitempty"`
	Status      string `json:"status,omitempty"` // "ready" or "generating"
	Message     string `json:"message,omitempty"`
	Cached      bool   `json:"cached"`
	Model       string `json:"model,omitempty"`
	GeneratedAt string `json:"generatedAt,omitempty"`
}

// Pagination is pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// ErrorResponse is the standard error format
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// LoginRequest is the request body for POST /api/admin/login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the response for POST /api/admin/login
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// --- Admin Request Types ---

// CreatePostRequest is the body for POST /api/admin/posts
type CreatePostRequest struct {
	Slug         string       `json:"slug" binding:"required"`
	Title        string       `json:"title" binding:"required"`
	Type         string       `json:"type" binding:"required"` // article, photo, rating, page
	Date         string       `json:"date" binding:"required"`
	Tags         []string     `json:"tags"`
	Boards       []string     `json:"boards"`
	Excerpt      string       `json:"excerpt"`
	Content      string       `json:"content"`     // raw markdown
	CustomFooter string       `json:"customFooter"`
	Icon         string       `json:"icon"`         // page
	Order        int          `json:"order"`        // page
	ShowInMenu   bool         `json:"showInMenu"`   // page
	Cover        string       `json:"cover"`        // rating
	Summary      string       `json:"summary"`      // rating
	Score        float64      `json:"score"`        // rating (0.0-10.0)
	RadarCharts  []RadarChart `json:"radarCharts"`  // rating
	Pages        []PhotoPage  `json:"pages"`        // photo
	Status       string       `json:"status"`       // published (default), draft
	CustomCSS    string       `json:"customCSS"`    // custom CSS for this post
	CSSEnabled   bool         `json:"cssEnabled"`   // whether custom CSS is active
}

// UpdatePostRequest is the body for PUT /api/admin/posts/:slug
type UpdatePostRequest struct {
	Title        *string       `json:"title"`
	Date         *string       `json:"date"`
	Tags         *[]string     `json:"tags"`
	Boards       *[]string     `json:"boards"`
	Excerpt      *string       `json:"excerpt"`
	Content      *string       `json:"content"`
	CustomFooter *string       `json:"customFooter"`
	Icon         *string       `json:"icon"`
	Order        *int          `json:"order"`
	ShowInMenu   *bool         `json:"showInMenu"`
	Cover        *string       `json:"cover"`
	Summary      *string       `json:"summary"`
	RadarCharts  *[]RadarChart `json:"radarCharts"`
	Pages        *[]PhotoPage  `json:"pages"`
	Status       *string       `json:"status"` // published, draft
	CustomCSS    *string       `json:"customCSS"`
	CSSEnabled   *bool         `json:"cssEnabled"`
}

// CreateBoardRequest is the body for POST /api/admin/boards
type CreateBoardRequest struct {
	Slug   string  `json:"slug" binding:"required"`
	Name   string  `json:"name" binding:"required"`
	Color  string  `json:"color"`
	Icon   string  `json:"icon"`
	Order  int     `json:"order"`
	Parent *string `json:"parent"` // parent board slug
}

// --- Setup / Settings Types ---

// SetupStatusResponse is the response for GET /api/setup/status
type SetupStatusResponse struct {
	NeedsSetup bool   `json:"needsSetup"`
	Message    string `json:"message,omitempty"`
}

// SetupRequest is the body for POST /api/setup/initialize
type SetupRequest struct {
	// Site settings
	SiteTitle       string `json:"siteTitle" binding:"required"`
	SiteDescription string `json:"siteDescription"`
	SiteFooter      string `json:"siteFooter"`

	// Admin account
	AdminUsername string `json:"adminUsername" binding:"required"`
	AdminPassword string `json:"adminPassword" binding:"required"`

	// AI configuration (optional)
	AIApiURL string `json:"aiApiUrl"`
	AIApiKey string `json:"aiApiKey"`
	AIModel  string `json:"aiModel"`

	// Initial board (optional)
	FirstBoardSlug string `json:"firstBoardSlug"`
	FirstBoardName string `json:"firstBoardName"`
	FirstBoardIcon string `json:"firstBoardIcon"`
}

// SiteSettings represents the site configuration
type SiteSettings struct {
	SiteTitle             string `json:"siteTitle"`
	SiteDescription       string `json:"siteDescription"`
	SiteFooter            string `json:"siteFooter"`
	SiteLogoURL           string `json:"siteLogoUrl"`
	SiteLicense           string `json:"siteLicense"`
	SiteLicenseURL        string `json:"siteLicenseUrl"`
	AIApiURL              string `json:"aiApiUrl,omitempty"`
	AIApiKey              string `json:"aiApiKey,omitempty"` // masked in response
	AIModel               string `json:"aiModel,omitempty"`
	SetupCompleted        string `json:"setupCompleted,omitempty"`
	GlobalCSS             string `json:"globalCSS"`
	CustomCSSEnabledTypes string `json:"customCSSEnabledTypes"`
	FrontendDomain        string `json:"frontendDomain"`
	AdminDomain           string `json:"adminDomain"`
	FrontendSSLEnabled    string `json:"frontendSslEnabled"`
	FrontendSSLHasCert    bool   `json:"frontendSslHasCert"`
	FrontendSSLMode       string `json:"frontendSslMode"`  // "off", "manual", "auto"
	AdminSSLEnabled       string `json:"adminSslEnabled"`
	AdminSSLHasCert       bool   `json:"adminSslHasCert"`
	AdminSSLMode          string `json:"adminSslMode"`     // "off", "manual", "auto"
	AcmeEmail             string `json:"acmeEmail"`
}

// SSLConfigRequest is the body for PUT /api/admin/ssl
type SSLConfigRequest struct {
	Target  string `json:"target" binding:"required"` // "frontend" or "admin"
	CertPEM string `json:"certPEM" binding:"required"`
	KeyPEM  string `json:"keyPEM" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// UpdateSettingsRequest is the body for PUT /api/admin/settings
type UpdateSettingsRequest struct {
	SiteTitle             *string `json:"siteTitle"`
	SiteDescription       *string `json:"siteDescription"`
	SiteFooter            *string `json:"siteFooter"`
	SiteLogoURL           *string `json:"siteLogoUrl"`
	SiteLicense           *string `json:"siteLicense"`
	SiteLicenseURL        *string `json:"siteLicenseUrl"`
	AIApiURL              *string `json:"aiApiUrl"`
	AIApiKey              *string `json:"aiApiKey"`
	AIModel               *string `json:"aiModel"`
	GlobalCSS             *string `json:"globalCSS"`
	CustomCSSEnabledTypes *string `json:"customCSSEnabledTypes"`
	FrontendDomain        *string `json:"frontendDomain"`
	AdminDomain           *string `json:"adminDomain"`
	FrontendSSLMode       *string `json:"frontendSslMode"`
	AdminSSLMode          *string `json:"adminSslMode"`
	AcmeEmail             *string `json:"acmeEmail"`
}
