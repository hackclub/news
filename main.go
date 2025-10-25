// main.go
package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

/*
Design goals
- Headless CMS: stable shapes, no PII, suitable for static site generation.
- Only include emails actually sent to a mailing list.
- Include mailing list meta (name, description, color) and subscriber counts.
- Include per-email aggregate stats + contents (html/markdown/structured json).
- Intermediate cache to reduce DB load; plus HTTP caching headers.
- Simple, dependency-light server with strong defaults.
*/

// ---------- Models (JSON response shapes) ----------

type MailingList struct {
	ID              string     `json:"id"`
	Slug            string     `json:"slug"` // derived from friendly_name (for CMS URLs)
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Color           string     `json:"color"`
	IsPublic        bool       `json:"is_public"`
	SubscriberCount int64      `json:"subscriber_count"`
	LastUpdatedAt   *time.Time `json:"last_updated_at,omitempty"`
	LastSentAt      *time.Time `json:"last_sent_at,omitempty"`
	SentEmailCount  int64      `json:"sent_email_count"`
}

type EmailStats struct {
	Opens  int64 `json:"opens"`
	Clicks int64 `json:"clicks"`
}

type Email struct {
	ID             string     `json:"id"`
	Slug           string     `json:"slug"` // derived from subject or name
	Subject        string     `json:"subject"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	MailingListID  string     `json:"mailing_list_id"`
	MailingListRef ListRef    `json:"mailing_list"`
	Stats          EmailStats `json:"stats"`
	HTML           *string    `json:"html,omitempty"`
	Markdown       *string    `json:"markdown,omitempty"`
	PreviewText    *string    `json:"preview_text,omitempty"` // first ~200 chars for listing cards
}

type ListRef struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type Paginated[T any] struct {
	Items []T   `json:"items"`
	Next  *int  `json:"next_offset,omitempty"`
	Count *int  `json:"count,omitempty"`
}

// ---------- Utilities ----------

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "&", " and ")
	s = strings.ReplaceAll(s, " + ", " and ")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		h := sha1.Sum([]byte(s))
		return hex.EncodeToString(h[:6])
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func weakETag(payload []byte) string {
	sum := sha1.Sum(payload)
	return `W/"` + hex.EncodeToString(sum[:]) + `"`
}

// ---------- Very small TTL cache ----------

type cacheItem struct {
	val       []byte
	expiresAt time.Time
	etag      string
}

type TTLCache struct {
	mu    sync.RWMutex
	store map[string]cacheItem
	ttl   time.Duration
	max   int
}

func NewTTLCache(ttl time.Duration, max int) *TTLCache {
	return &TTLCache{store: make(map[string]cacheItem), ttl: ttl, max: max}
}

func (c *TTLCache) Get(key string) (val []byte, etag string, ok bool) {
	c.mu.RLock()
	it, ok := c.store[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(it.expiresAt) {
		return nil, "", false
	}
	return it.val, it.etag, true
}

func (c *TTLCache) Set(key string, val []byte) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.store) >= c.max {
		c.store = make(map[string]cacheItem)
	}
	etag := weakETag(val)
	c.store[key] = cacheItem{val: val, etag: etag, expiresAt: time.Now().Add(c.ttl)}
	return etag
}

func cacheKey(r *http.Request) string {
	return r.Method + " " + r.URL.Path + "?" + r.URL.RawQuery
}

// ---------- Database layer ----------

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.MaxConnLifetime = 55 * time.Minute
	cfg.MaxConnIdleTime = 10 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx2); err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) ListMailingLists(ctx context.Context, limit, offset int) ([]MailingList, *int, error) {
	q := `
WITH sent_counts AS (
  SELECT mailing_list_id, COUNT(*) AS sent_email_count, MAX(sent_at) as last_sent_at
  FROM loops.campaigns
  WHERE status = 'Sent' AND mailing_list_id IS NOT NULL AND ai_publishable = true
  GROUP BY mailing_list_id
),
sub_counts AS (
  SELECT mailing_list_id, COUNT(*)::bigint AS subscriber_count
  FROM loops.audience_mailing_lists
  GROUP BY mailing_list_id
)
SELECT ml.id,
       ml.friendly_name,
       ml.description,
       COALESCE(ml.is_public, false) AS is_public,
       COALESCE(ml.color_scheme, '#000000') AS color_scheme,
       ml.last_updated_at,
       COALESCE(sc.subscriber_count, 0) AS subscriber_count,
       COALESCE(se.sent_email_count, 0) AS sent_email_count,
       se.last_sent_at
FROM loops.mailing_lists ml
LEFT JOIN sub_counts sc ON sc.mailing_list_id = ml.id
LEFT JOIN sent_counts se ON se.mailing_list_id = ml.id
WHERE COALESCE(sc.subscriber_count, 0) > 0 AND COALESCE(se.sent_email_count, 0) > 0
ORDER BY (se.last_sent_at IS NULL) ASC, se.last_sent_at DESC NULLS LAST, ml.friendly_name ASC
LIMIT $1 OFFSET $2;
`
	rows, err := s.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	out := make([]MailingList, 0, limit)
	for rows.Next() {
		var ml MailingList
		var name, desc, color string
		var isPublic bool
		var lastUpdated *time.Time
		var lastSent *time.Time
		var subCount, sentCount int64
		var id string
		if err := rows.Scan(&id, &name, &desc, &isPublic, &color, &lastUpdated, &subCount, &sentCount, &lastSent); err != nil {
			return nil, nil, err
		}
		ml.ID = id
		ml.Name = name
		ml.Description = desc
		ml.Color = color
		ml.IsPublic = isPublic
		ml.LastUpdatedAt = lastUpdated
		ml.LastSentAt = lastSent
		ml.SubscriberCount = subCount
		ml.SentEmailCount = sentCount
		ml.Slug = slugify(name)
		out = append(out, ml)
	}
	var next *int
	if len(out) == limit {
		n := offset + limit
		next = &n
	}
	return out, next, rows.Err()
}

func (s *Store) ListEmails(ctx context.Context, mailingListID *string, limit, offset int) ([]Email, *int, error) {
	args := []any{}
	where := "WHERE c.status = 'Sent' AND c.mailing_list_id IS NOT NULL AND c.ai_publishable = true"
	if mailingListID != nil && *mailingListID != "" {
		where += " AND c.mailing_list_id = $1"
		args = append(args, *mailingListID)
	}
	q := fmt.Sprintf(`
SELECT
  c.id,
  c.subject,
  c.sent_at,
  c.mailing_list_id,
  ml.friendly_name,
  ml.description,
  COALESCE(ml.color_scheme, '#000000'),
  COALESCE(c.opens, 0)::bigint,
  COALESCE(c.clicks, 0)::bigint,
  c.ai_publishable_content_html,
  c.ai_publishable_content_markdown,
  c.ai_publishable_slug
FROM loops.campaigns c
JOIN loops.mailing_lists ml ON ml.id = c.mailing_list_id
%s
ORDER BY c.sent_at DESC NULLS LAST, c.created_at DESC
LIMIT %s OFFSET %s;
`, where,
		fmt.Sprintf("$%d", len(args)+1),
		fmt.Sprintf("$%d", len(args)+2),
	)
	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	out := make([]Email, 0, limit)
	for rows.Next() {
		var e Email
		var sentAt *time.Time
		var mlName, mlDesc, mlColor string
		var opens, clicks int64
		var html, md *string
		var aiSlug *string
		if err := rows.Scan(
			&e.ID, &e.Subject, &sentAt, &e.MailingListID,
			&mlName, &mlDesc, &mlColor,
			&opens, &clicks,
			&html, &md, &aiSlug,
		); err != nil {
			return nil, nil, err
		}
		e.SentAt = sentAt
		e.MailingListRef = ListRef{
			ID:          e.MailingListID,
			Slug:        slugify(mlName),
			Name:        mlName,
			Description: mlDesc,
			Color:       mlColor,
		}
		e.Stats = EmailStats{
			Opens:  opens,
			Clicks: clicks,
		}
		e.HTML = html
		e.Markdown = md
		if aiSlug != nil && *aiSlug != "" {
			e.Slug = *aiSlug
		} else {
			e.Slug = slugify(e.Subject)
			if e.Slug == "" {
				e.Slug = e.ID
			}
		}

		if e.Markdown != nil && *e.Markdown != "" {
			preview := strings.TrimSpace(*e.Markdown)
			if len(preview) > 200 {
				preview = preview[:200]
			}
			e.PreviewText = &preview
		} else if e.HTML != nil && *e.HTML != "" {
			preview := stripTags(*e.HTML)
			if len(preview) > 200 {
				preview = preview[:200]
			}
			e.PreviewText = &preview
		}

		out = append(out, e)
	}
	var next *int
	if len(out) == limit {
		n := offset + limit
		next = &n
	}
	return out, next, rows.Err()
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// ---------- HTTP Handlers ----------

type Server struct {
	store *Store
	cache *TTLCache
}

func NewServer(store *Store) *Server {
	return &Server{
		store: store,
		cache: NewTTLCache(30*time.Second, 512),
	}
}

func (s *Server) jsonCached(w http.ResponseWriter, r *http.Request, build func() (any, error)) {
	key := cacheKey(r)
	if body, etag, ok := s.cache.Get(key); ok {
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=30, stale-while-revalidate=60")
		w.Header().Set("ETag", etag)
		_, _ = w.Write(body)
		return
	}

	v, err := build()
	if err != nil {
		httpError(w, err)
		return
	}
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		httpError(w, err)
		return
	}
	etag := s.cache.Set(key, body)

	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=30, stale-while-revalidate=60")
	w.Header().Set("ETag", etag)
	_, _ = w.Write(body)
}

func parseLimitOffset(r *http.Request, defLimit int) (limit, offset int) {
	limit = defLimit
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

func (s *Server) handleMailingLists(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 50)
	s.jsonCached(w, r, func() (any, error) {
		lists, next, err := s.store.ListMailingLists(r.Context(), limit, offset)
		if err != nil {
			return nil, err
		}
		return Paginated[MailingList]{Items: lists, Next: next}, nil
	})
}

func (s *Server) handleEmails(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 50)
	var mlid *string
	if v := r.URL.Query().Get("mailing_list_id"); v != "" {
		mlid = &v
	}
	s.jsonCached(w, r, func() (any, error) {
		emails, next, err := s.store.ListEmails(r.Context(), mlid, limit, offset)
		if err != nil {
			return nil, err
		}
		return Paginated[Email]{Items: emails, Next: next}, nil
	})
}

type GroupedEmails struct {
	MailingList MailingList `json:"mailing_list"`
	Emails      []Email     `json:"emails"`
}

func (s *Server) handleMailingListsEmails(w http.ResponseWriter, r *http.Request) {
	groupAll := r.URL.Query().Get("group_all") == "true"
	limitPerList := 1
	if v := r.URL.Query().Get("limit_per_list"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limitPerList = n
		}
	}
	s.jsonCached(w, r, func() (any, error) {
		lists, _, err := s.store.ListMailingLists(r.Context(), 1000, 0)
		if err != nil {
			return nil, err
		}
		out := make([]GroupedEmails, 0, len(lists))
		for _, ml := range lists {
			mlid := ml.ID
			emails, _, err := s.store.ListEmails(r.Context(), &mlid, limitPerList, 0)
			if err != nil {
				return nil, err
			}
			if len(emails) == 0 {
				continue
			}
			out = append(out, GroupedEmails{
				MailingList: ml,
				Emails: func() []Email {
					if groupAll {
						return emails
					}
					return emails[:1]
				}(),
			})
		}
		return out, nil
	})
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(apiDocsMarkdown))
}

// ---------- Errors ----------

type apiErr struct {
	Message string `json:"message"`
}

func httpError(w http.ResponseWriter, err error) {
	var status = http.StatusInternalServerError
	msg := http.StatusText(status)
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
		msg = "upstream timed out"
	} else if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		status = http.StatusGatewayTimeout
		msg = "network timeout"
	} else if err != nil {
		msg = err.Error()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErr{Message: msg})
}

// ---------- Main ----------

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	_ = godotenv.Load()
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	store, err := NewStore(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer store.pool.Close()

	srv := NewServer(store)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(securityHeaders())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/docs", http.StatusFound) })
	r.Get("/docs", srv.handleDocs)
	r.Get("/mailing_lists", srv.handleMailingLists)
	r.Get("/emails", srv.handleEmails)
	r.Get("/mailing_lists/emails", srv.handleMailingListsEmails)

	addr := ":" + env("PORT", "8080")
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func securityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src * data:; style-src 'unsafe-inline';")
			next.ServeHTTP(w, r)
		})
	}
}

// ---------- API Docs (Markdown) ----------
const apiDocsMarkdown = `
# Hack Club Email CMS API

A read-only, PII-safe headless CMS for generating a blog from mailing lists and sent emails.

Base URL: ` + "`/`" + `

## Authentication
None (read-only). You should front this behind your CDN or add your own layer if needed.

## Data guarantees
- **No PII**: We never expose recipient emails, names, or per-user data.
- **Sent-only**: ` + "`/emails`" + ` and ` + "`/mailing_lists/emails`" + ` only include campaigns with ` + "`status = \"Sent\"`" + `.
- **Stability**: Fields are chosen for static site generation (SSG) and caching.

## Caching
- Server-side in-memory TTL cache (30s).
- HTTP cache headers: ` + "`Cache-Control: public, max-age=30, stale-while-revalidate=60`" + ` and ` + "`ETag`" + `.
- Respect ` + "`If-None-Match`" + ` to avoid bytes over the wire.

---

## GET /mailing_lists

List mailing lists with metadata and aggregate counts.

### Query Params
- ` + "`limit`" + ` (int, default 50, max 200)
- ` + "`offset`" + ` (int, default 0)

### Response
` + "```json" + `
{
  "items": [
    {
      "id": "clzvjqcvk00kq0ll4a8qu4qzz",
      "slug": "hcb-newsletter",
      "name": "HCB Newsletter",
      "description": "Occasional emails about new features on HCB! hackclub.com/fiscal-sponsorship",
      "color": "#c87ae4",
      "is_public": false,
      "subscriber_count": 12345,
      "last_updated_at": "2025-10-24T16:31:26.469823Z",
      "last_sent_at": "2025-10-10T03:47:14.357Z",
      "sent_email_count": 12
    }
  ],
  "next_offset": 50
}
` + "```" + `

---

## GET /emails

List **sent** emails. Returns content + stats and a compact reference to the mailing list.

### Query Params
- ` + "`limit`" + ` (int, default 50, max 200)
- ` + "`offset`" + ` (int, default 0)
- ` + "`mailing_list_id`" + ` (string, optional) — filter to a specific list.

### Response
` + "```json" + `
{
  "items": [
    {
      "id": "cmgkb2b058ngw210ij7jpskf4",
      "slug": "hack-club-events-fellowship-apply-today",
      "subject": "Hack Club Events Fellowship: apply today",
      "internal_title": "RM Outreach > Counterspell",
      "emoji": null,
      "sent_at": "2025-10-10T03:47:14.357Z",
      "mailing_list_id": "cm1fqxdc900qn0ll9fd5m3wdv",
      "mailing_list": {
        "id": "cm1fqxdc900qn0ll9fd5m3wdv",
        "slug": "counterspell",
        "name": "Counterspell",
        "description": "World's biggest game jam in 2026?",
        "color": "#ec3750"
      },
      "stats": {
        "sent_count": 3152,
        "opens": 815,
        "clicks": 82,
        "unsubscribes": 2,
        "hard_bounces": 0,
        "soft_bounces": 88,
        "open_rate": 0.2586,
        "click_rate": 0.0260
      },
      "html": "<!doctype html> ...",
      "markdown": "Hey there, ...",
      "content_json": { "root": { "...": "..." } },
      "preview_text": "Hey there, My name is..."
    }
  ],
  "next_offset": 50
}
` + "```" + `

**Notes**
- ` + "`internal_title`" + ` is the campaigns' internal ` + "`name`" + ` (useful for CMS + editor context).
- We do **not** expose ` + "`from_email`" + `, ` + "`reply_to_email`" + `, or any per-recipient stats.

---

## GET /mailing_lists/emails

Convenience endpoint for building index pages.

### Modes
1. **Default (latest)**: returns one latest sent email per list.
2. **Grouped (all)**: pass ` + "`group_all=true`" + ` to return up to ` + "`limit_per_list`" + ` sent emails per list.

### Query Params
- ` + "`group_all`" + ` (bool, default ` + "`false`" + `)
- ` + "`limit_per_list`" + ` (int, default 1, max 50)

### Response (default)
` + "```json" + `
[
  {
    "mailing_list": {
      "id": "clzvo8z3g01dr0ll749nohxz6",
      "slug": "arcade",
      "name": "Arcade",
      "description": "Spend your summer coding projects, get prizes! hackclub.com/arcade",
      "color": "#ff8a00",
      "is_public": false,
      "subscriber_count": 9999,
      "last_sent_at": "2025-10-10T03:45:58.073Z",
      "sent_email_count": 3
    },
    "emails": [ { "...": "latest email object" } ]
  }
]
` + "```" + `

---

## Sorting & Pagination
- ` + "`/mailing_lists`" + ` is ordered by most recently sent email (desc), then by name.
- ` + "`/emails`" + ` is ordered by ` + "`sent_at`" + ` (desc).
- Use ` + "`limit`" + ` + ` + "`offset`" + `. If ` + "`next_offset`" + ` is present, more results exist.

## Content fields
We expose **email_html**, **email_markdown**, and **email_content_json** straight from your Loops sync so you can render rich blog posts. If you want to sanitize/transform, do it at build time in your SSG.

## Privacy
- Endpoint never returns audience emails or per-recipient events.
- If you later ingest anything recipient-specific, keep it out of this surface.

## Status & Health
- ` + "`/healthz`" + ` returns 200 OK when the server is alive.

---
`
