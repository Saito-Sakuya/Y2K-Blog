package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/acme/autocert"

	"github.com/y2k-pixel-blog/backend/internal/handler"
	"github.com/y2k-pixel-blog/backend/internal/middleware"
	"github.com/y2k-pixel-blog/backend/internal/repository"
	"github.com/y2k-pixel-blog/backend/internal/service"
)

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://blog:blog@localhost:5432/y2k_blog?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Database connected")

	// Initialize layers
	repo := repository.New(db)
	svc := service.New(repo)
	h := handler.New(svc)

	// Rate limiter for login
	rateLimiter := middleware.NewRateLimiter()

	// Gin router
	r := gin.Default()

	// --- CORS middleware ---
	var allowedOrigins []string

	sslCfg, _ := svc.GetSSLConfig()
	if sslCfg != nil {
		for _, d := range []string{sslCfg.FrontDomain, sslCfg.AdminDomain} {
			if d != "" {
				allowedOrigins = append(allowedOrigins, "https://"+d, "http://"+d)
			}
		}
	}

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = os.Getenv("CORS_ORIGIN")
	}
	if corsOrigins != "" && corsOrigins != "*" {
		for _, o := range strings.Split(corsOrigins, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(o))
		}
	}

	r.Use(func(c *gin.Context) {
		requestOrigin := c.GetHeader("Origin")
		allowOrigin := ""
		if len(allowedOrigins) == 0 {
			allowOrigin = "*"
		} else {
			for _, o := range allowedOrigins {
				if o == requestOrigin {
					allowOrigin = requestOrigin
					break
				}
			}
		}
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// --- Routes ---
	api := r.Group("/api")
	{
		api.GET("/boards", h.ListBoards)
		api.GET("/boards/:slug", h.GetBoardContent)
		api.GET("/posts/:slug", h.GetPost)
		api.GET("/search", h.Search)
		api.GET("/tags", h.ListTags)
		api.GET("/menu", h.GetMenu)
		api.GET("/css-config", h.GetCSSConfig)
		api.GET("/ai/summary", h.GetAISummary)
		api.GET("/site-config", h.GetSiteConfig)
	}

	r.POST("/api/admin/login", rateLimiter.LoginRateLimit(), h.Login)
	r.GET("/api/admin/captcha", rateLimiter.CaptchaHandler())
	r.GET("/api/admin/login/status", rateLimiter.LoginStatusHandler())

	r.GET("/api/setup/status", h.GetSetupStatus)
	r.POST("/api/setup/initialize", h.Initialize)

	admin := r.Group("/api/admin")
	admin.Use(middleware.JWTAuth())
	{
		admin.GET("/posts", h.ListPostsByStatus)
		admin.POST("/posts", h.CreatePost)
		admin.PUT("/posts/:slug", h.UpdatePost)
		admin.DELETE("/posts/:slug", h.DeletePost)
		admin.POST("/posts/:slug/trash", h.TrashPost)
		admin.POST("/posts/:slug/restore", h.RestorePost)
		admin.POST("/preview/:slug", h.GeneratePreviewToken)
		admin.DELETE("/trash", h.EmptyTrash)
		admin.POST("/boards", h.CreateBoard)
		admin.DELETE("/ai-cache/:slug", h.DeleteAICache)
		admin.PUT("/password", h.ChangePassword)
		admin.GET("/settings", h.GetSettings)
		admin.PUT("/settings", h.UpdateSettings)
		admin.PUT("/ssl", h.ConfigureSSL)
		admin.DELETE("/ssl", h.RemoveSSL)
	}

	r.GET("/api/preview/:token", h.GetPreview)

	// SEO routes (public, cacheable)
	r.GET("/feed.xml", h.RSSFeed)
	r.GET("/sitemap.xml", h.Sitemap)
	r.GET("/robots.txt", h.RobotsTxt)
	api.GET("/og/:slug", h.OpenGraph)

	// --- Start server ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sslCfg, sslErr := svc.GetSSLConfig()
	if sslErr != nil {
		sslCfg = nil
	}

	if sslCfg != nil {
		var (
			hasManualCert bool
			frontCert     tls.Certificate
			adminCert     tls.Certificate
			autoDomains   []string
		)

		// Manual certs
		if sslCfg.FrontMode == "manual" && sslCfg.FrontEnabled && sslCfg.FrontCert != "" {
			c, err := tls.X509KeyPair([]byte(sslCfg.FrontCert), []byte(sslCfg.FrontKey))
			if err != nil {
				log.Printf("⚠️ Frontend manual cert invalid: %v", err)
			} else {
				frontCert = c
				hasManualCert = true
			}
		}
		if sslCfg.AdminMode == "manual" && sslCfg.AdminEnabled && sslCfg.AdminCert != "" {
			c, err := tls.X509KeyPair([]byte(sslCfg.AdminCert), []byte(sslCfg.AdminKey))
			if err != nil {
				log.Printf("⚠️ Admin manual cert invalid: %v", err)
			} else {
				adminCert = c
				hasManualCert = true
			}
		}

		// Auto (Let's Encrypt) domains
		if sslCfg.FrontMode == "auto" && sslCfg.FrontDomain != "" {
			autoDomains = append(autoDomains, sslCfg.FrontDomain)
		}
		if sslCfg.AdminMode == "auto" && sslCfg.AdminDomain != "" {
			autoDomains = append(autoDomains, sslCfg.AdminDomain)
		}

		if hasManualCert || len(autoDomains) > 0 {
			// Setup autocert manager
			var acm *autocert.Manager
			if len(autoDomains) > 0 {
				cacheDir := os.Getenv("ACME_CACHE_DIR")
				if cacheDir == "" {
					cacheDir = "./certs"
				}
				os.MkdirAll(cacheDir, 0700)

				acm = &autocert.Manager{
					Prompt:     autocert.AcceptTOS,
					HostPolicy: autocert.HostWhitelist(autoDomains...),
					Cache:      autocert.DirCache(cacheDir),
					Email:      sslCfg.AcmeEmail,
				}
				log.Printf("🔐 Auto-cert (Let's Encrypt) for: %v", autoDomains)
			}

			// Combined GetCertificate
			tlsConfig := &tls.Config{
				GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					// Manual cert match
					if hello.ServerName == sslCfg.FrontDomain && frontCert.Certificate != nil {
						return &frontCert, nil
					}
					if hello.ServerName == sslCfg.AdminDomain && adminCert.Certificate != nil {
						return &adminCert, nil
					}
					// Autocert
					if acm != nil {
						return acm.GetCertificate(hello)
					}
					// Fallback
					if frontCert.Certificate != nil {
						return &frontCert, nil
					}
					if adminCert.Certificate != nil {
						return &adminCert, nil
					}
					return nil, fmt.Errorf("no certificate for %s", hello.ServerName)
				},
			}

			// HTTP :80 — ACME challenges + redirect
			go func() {
				mux := http.NewServeMux()
				if acm != nil {
					mux.Handle("/.well-known/acme-challenge/", acm.HTTPHandler(nil))
				}
				mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					target := "https://" + req.Host + req.URL.RequestURI()
					http.Redirect(w, req, target, http.StatusMovedPermanently)
				})
				log.Println("🔄 HTTP :80 (ACME + redirect)")
				if err := http.ListenAndServe(":80", mux); err != nil {
					log.Printf("⚠️ HTTP server error: %v", err)
				}
			}()

			// HTTPS :443
			httpsPort := os.Getenv("HTTPS_PORT")
			if httpsPort == "" {
				httpsPort = "443"
			}
			log.Printf("🔒 HTTPS on :%s", httpsPort)
			server := &http.Server{
				Addr:      ":" + httpsPort,
				Handler:   r,
				TLSConfig: tlsConfig,
			}
			if err := server.ListenAndServeTLS("", ""); err != nil {
				log.Fatalf("Failed to start HTTPS: %v", err)
			}
			return
		}
	}

	// Fallback: HTTP only
	log.Printf("🚀 Server starting on :%s (HTTP)", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
