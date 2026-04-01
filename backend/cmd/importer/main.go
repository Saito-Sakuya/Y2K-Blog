package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

// BoardConfig matches /content/boards/*.yaml
type BoardConfig struct {
	Name   string  `yaml:"name"`
	Slug   string  `yaml:"slug"`
	Color  string  `yaml:"color"`
	Icon   string  `yaml:"icon"`
	Order  int     `yaml:"order"`
	Parent *string `yaml:"parent"` // parent slug or null
}

// PostFrontmatter matches any content type's YAML frontmatter
type PostFrontmatter struct {
	Title        string       `yaml:"title"`
	Type         string       `yaml:"type"`
	Date         string       `yaml:"date"`
	Tags         []string     `yaml:"tags"`
	Boards       []string     `yaml:"boards"`
	Excerpt      string       `yaml:"excerpt"`
	ReadTime     int          `yaml:"readTime"`
	CustomFooter string       `yaml:"customFooter"`
	Icon         string       `yaml:"icon"`
	Order        int          `yaml:"order"`
	ShowInMenu   bool         `yaml:"showInMenu"`
	Slug         string       `yaml:"slug"`
	Cover        string       `yaml:"cover"`
	Summary      string       `yaml:"summary"`
	RadarCharts  []RadarChart `yaml:"radarCharts"`
	Pages        []PhotoPage  `yaml:"pages"`
}

type RadarChart struct {
	Name string     `yaml:"name" json:"name"`
	Axes []RadarAxis `yaml:"axes" json:"axes"`
}

type RadarAxis struct {
	Label string  `yaml:"label" json:"label"`
	Score float64 `yaml:"score" json:"score"`
}

type PhotoPage struct {
	Image string `yaml:"image" json:"image"`
	Text  string `yaml:"text" json:"text"`
}

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://blog:blog@localhost:5432/y2k_blog?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("DB connect error: %v", err)
	}
	defer db.Close()

	contentDir := os.Getenv("CONTENT_DIR")
	if contentDir == "" {
		contentDir = "../content"
	}

	// 1. Import boards
	log.Println("📁 Importing boards...")
	importBoards(db, filepath.Join(contentDir, "boards"))

	// 2. Import posts (all types)
	log.Println("📝 Importing posts...")
	importPosts(db, filepath.Join(contentDir, "posts"))

	log.Println("✅ Import complete!")
}

func importBoards(db *sql.DB, dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		log.Printf("Warning: could not read boards dir: %v", err)
		return
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			log.Printf("  ❌ Error reading %s: %v", f, err)
			continue
		}

		var board BoardConfig
		if err := yaml.Unmarshal(data, &board); err != nil {
			log.Printf("  ❌ Error parsing %s: %v", f, err)
			continue
		}

		_, err = db.Exec(`
			INSERT INTO boards (slug, name, color, icon, sort_order, parent_id)
			VALUES ($1, $2, $3, $4, $5, (SELECT id FROM boards WHERE slug = $6))
			ON CONFLICT (slug) DO UPDATE SET
				name = EXCLUDED.name,
				color = EXCLUDED.color,
				icon = EXCLUDED.icon,
				sort_order = EXCLUDED.sort_order
		`, board.Slug, board.Name, board.Color, board.Icon, board.Order, board.Parent)

		if err != nil {
			log.Printf("  ❌ Error inserting board %s: %v", board.Slug, err)
		} else {
			log.Printf("  ✅ Board: %s", board.Name)
		}
	}
}

func importPosts(db *sql.DB, dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("  ❌ Error reading %s: %v", path, err)
			return nil
		}

		// Parse frontmatter + content
		fm, content, err := parseFrontmatter(string(data))
		if err != nil {
			log.Printf("  ❌ Error parsing frontmatter %s: %v", path, err)
			return nil
		}

		// Derive slug from filename if not in frontmatter
		slug := fm.Slug
		if slug == "" {
			slug = strings.TrimSuffix(filepath.Base(path), ".md")
		}

		tags := strings.Join(fm.Tags, ",")
		wordCount := len([]rune(content))
		readTime := fm.ReadTime
		if readTime == 0 {
			readTime = (wordCount + 399) / 400 // 400 chars/min
		}

		// Serialize JSON fields
		var radarJSON, pagesJSON []byte
		if len(fm.RadarCharts) > 0 {
			radarJSON, _ = json.Marshal(fm.RadarCharts)
		}
		if len(fm.Pages) > 0 {
			pagesJSON, _ = json.Marshal(fm.Pages)
		}

		// Upsert post
		var postID int
		err = db.QueryRow(`
			INSERT INTO posts (slug, title, type, date, tags, excerpt,
				content_raw, content_html, custom_footer, read_time, word_count,
				icon, sort_order, show_in_menu, cover, summary,
				radar_charts, photo_pages)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16, $17, $18)
			ON CONFLICT (slug) DO UPDATE SET
				title = EXCLUDED.title,
				type = EXCLUDED.type,
				date = EXCLUDED.date,
				tags = EXCLUDED.tags,
				excerpt = EXCLUDED.excerpt,
				content_raw = EXCLUDED.content_raw,
				content_html = EXCLUDED.content_html,
				custom_footer = EXCLUDED.custom_footer,
				read_time = EXCLUDED.read_time,
				word_count = EXCLUDED.word_count,
				icon = EXCLUDED.icon,
				sort_order = EXCLUDED.sort_order,
				show_in_menu = EXCLUDED.show_in_menu,
				cover = EXCLUDED.cover,
				summary = EXCLUDED.summary,
				radar_charts = EXCLUDED.radar_charts,
				photo_pages = EXCLUDED.photo_pages
			RETURNING id
		`, slug, fm.Title, fm.Type, fm.Date, tags, fm.Excerpt,
			content, content, fm.CustomFooter, readTime, wordCount,
			fm.Icon, fm.Order, fm.ShowInMenu, fm.Cover, fm.Summary,
			radarJSON, pagesJSON,
		).Scan(&postID)

		if err != nil {
			log.Printf("  ❌ Error inserting post %s: %v", slug, err)
			return nil
		}

		// Link post to boards
		for _, boardSlug := range fm.Boards {
			_, err = db.Exec(`
				INSERT INTO post_boards (post_id, board_id)
				SELECT $1, id FROM boards WHERE slug = $2
				ON CONFLICT DO NOTHING
			`, postID, boardSlug)
			if err != nil {
				log.Printf("    ⚠️ Error linking %s to board %s: %v", slug, boardSlug, err)
			}
		}

		log.Printf("  ✅ [%s] %s", fm.Type, fm.Title)
		return nil
	})

	if err != nil {
		log.Printf("Error walking posts directory: %v", err)
	}
}

// parseFrontmatter splits "---\nyaml\n---\ncontent" into frontmatter struct + markdown body
func parseFrontmatter(raw string) (*PostFrontmatter, string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return nil, "", fmt.Errorf("missing frontmatter delimiter")
	}

	// Find closing ---
	rest := raw[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	yamlStr := rest[:idx]
	content := strings.TrimSpace(rest[idx+4:])

	var fm PostFrontmatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, "", fmt.Errorf("yaml parse error: %w", err)
	}

	return &fm, content, nil
}
