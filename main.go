// main.go
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
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
	Clicks int64 `json:"clicks"`
	Views  int64 `json:"views"`
}

type Email struct {
	ID             string     `json:"id"`
	Slug           string     `json:"slug"` // derived from subject or name
	Subject        string     `json:"subject"`
	Excerpt        *string    `json:"excerpt,omitempty"`
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

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
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
		var oldestKey string
		oldestTime := time.Now()
		for k, v := range c.store {
			if v.expiresAt.Before(oldestTime) {
				oldestTime = v.expiresAt
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(c.store, oldestKey)
		}
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
	pool        *pgxpool.Pool
	metricsPool *pgxpool.Pool
}

func NewStore(ctx context.Context, url string, metricsURL string) (*Store, error) {
	if os.Getenv("ALLOW_DB_INSECURE") != "1" && !strings.Contains(url, "sslmode=") {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		url += sep + "sslmode=require"
	}
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

	var metricsPool *pgxpool.Pool
	if metricsURL != "" {
		metricsCfg, err := pgxpool.ParseConfig(metricsURL)
		if err != nil {
			return nil, fmt.Errorf("metrics db config: %w", err)
		}
		metricsCfg.MaxConns = 5
		metricsCfg.MinConns = 1
		metricsPool, err = pgxpool.NewWithConfig(ctx, metricsCfg)
		if err != nil {
			return nil, fmt.Errorf("metrics db connect: %w", err)
		}
		if err := metricsPool.Ping(ctx2); err != nil {
			return nil, fmt.Errorf("metrics db ping: %w", err)
		}
	}

	return &Store{pool: pool, metricsPool: metricsPool}, nil
}

func (s *Store) RunMetricsMigrations(ctx context.Context) error {
	if s.metricsPool == nil {
		log.Println("metrics database not configured, skipping migrations")
		return nil
	}

	log.Println("running metrics database migrations...")

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS email_views (
			time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			session_id TEXT NOT NULL,
			email_id TEXT NOT NULL
		)`,
		
		`SELECT create_hypertable('email_views', 'time', if_not_exists => TRUE)`,
		
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_email_views_dedup 
		ON email_views (session_id, email_id, time_bucket('5 minutes', time), time)`,
		
		`CREATE MATERIALIZED VIEW IF NOT EXISTS email_view_counts
		WITH (timescaledb.continuous) AS
		SELECT 
			time_bucket('1 hour', time) as bucket,
			email_id,
			COUNT(DISTINCT session_id) as view_count
		FROM email_views
		GROUP BY bucket, email_id
		WITH NO DATA`,
		
		`SELECT add_continuous_aggregate_policy('email_view_counts',
			start_offset => INTERVAL '1 day',
			end_offset => INTERVAL '1 hour',
			schedule_interval => INTERVAL '1 hour',
			if_not_exists => TRUE)`,
		
		`CREATE INDEX IF NOT EXISTS idx_email_views_email_id ON email_views(email_id, time DESC)`,
		
		`CREATE TABLE IF NOT EXISTS email_link_clicks (
			time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			session_id TEXT NOT NULL,
			email_id TEXT NOT NULL,
			link_url TEXT NOT NULL,
			link_index INT NOT NULL
		)`,
		
		`SELECT create_hypertable('email_link_clicks', 'time', if_not_exists => TRUE)`,
		
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_email_link_clicks_dedup 
		ON email_link_clicks (session_id, email_id, link_index, time_bucket('5 minutes', time), time)`,
		
		`CREATE INDEX IF NOT EXISTS idx_email_link_clicks_email_id ON email_link_clicks(email_id, time DESC)`,
	}

	for i, migration := range migrations {
		_, err := s.metricsPool.Exec(ctx, migration)
		if err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	log.Println("metrics database migrations completed successfully")
	return nil
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
WHERE COALESCE(se.sent_email_count, 0) > 0
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

func (s *Store) ListEmails(ctx context.Context, r *http.Request, mailingListID *string, limit, offset int) ([]Email, *int, error) {
	args := []any{}
	where := "WHERE c.status = 'Sent' AND c.mailing_list_id IS NOT NULL AND c.ai_publishable = true"
	if mailingListID != nil && *mailingListID != "" {
		where += " AND c.mailing_list_id = $1"
		args = append(args, *mailingListID)
	}
	q := fmt.Sprintf(`
SELECT
  c.id,
  c.ai_publishable_response_json->>'title',
  c.sent_at,
  c.mailing_list_id,
  ml.friendly_name,
  ml.description,
  COALESCE(ml.color_scheme, '#000000'),
  COALESCE(c.clicks, 0)::bigint,
  COALESCE(c.opens, 0)::bigint,
  c.ai_publishable_content_html,
  c.ai_publishable_content_markdown,
  c.ai_publishable_slug,
  c.ai_publishable_response_json->>'excerpt'
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
		var clicks, warehouseOpens int64
		var html, md *string
		var aiSlug, excerpt *string
		if err := rows.Scan(
			&e.ID, &e.Subject, &sentAt, &e.MailingListID,
			&mlName, &mlDesc, &mlColor,
			&clicks, &warehouseOpens,
			&html, &md, &aiSlug, &excerpt,
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
		
		metricsViews, _ := s.GetMetricsViewCount(ctx, e.ID)
		
		metricsClicks, _ := s.GetMetricsClickCount(ctx, e.ID)
		
		e.Stats = EmailStats{
			Clicks: clicks + metricsClicks,
			Views:  warehouseOpens + metricsViews,
		}
		
		if html != nil && *html != "" {
			rewritten, err := rewriteEmailLinks(r, e.ID, *html)
			if err == nil {
				e.HTML = &rewritten
			} else {
				e.HTML = html
			}
		} else {
			e.HTML = html
		}
		e.Markdown = md
		e.Excerpt = excerpt
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

var scriptStyleRegex = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)

func rewriteEmailLinks(r *http.Request, emailID string, html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html, err
	}
	
	// Determine scheme (http or https)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header (for reverse proxies)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	
	// Get host from request
	host := r.Host
	
	// Build base URL
	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	
	linkIndex := 0
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		
		if strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "tel:") {
			return
		}
		
		newURL := fmt.Sprintf("%s/emails/%s/click/%d?url=%s", baseURL, emailID, linkIndex, url.QueryEscape(href))
		s.SetAttr("href", newURL)
		linkIndex++
	})
	
	rewritten, err := doc.Html()
	if err != nil {
		return html, err
	}
	return rewritten, nil
}

func stripTags(s string) string {
	s = scriptStyleRegex.ReplaceAllString(s, "")
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

func (s *Store) TrackEmailView(ctx context.Context, sessionID, emailID string) error {
	if s.metricsPool == nil {
		return nil
	}
	
	// Check if this session already viewed this email in the last 5 minutes
	var exists bool
	err := s.metricsPool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM email_views
			WHERE session_id = $1
			  AND email_id = $2
			  AND time > NOW() - INTERVAL '5 minutes'
			LIMIT 1
		)
	`, sessionID, emailID).Scan(&exists)
	
	if err != nil {
		return err
	}
	
	// Only insert if not already viewed in last 5 minutes
	if !exists {
		_, err = s.metricsPool.Exec(ctx, `
			INSERT INTO email_views (session_id, email_id)
			VALUES ($1, $2)
		`, sessionID, emailID)
		return err
	}
	
	return nil
}

func (s *Store) TrackLinkClick(ctx context.Context, sessionID, emailID, linkURL string, linkIndex int) error {
	if s.metricsPool == nil {
		return nil
	}
	
	// Check if this session already clicked this link in the last 5 minutes
	var exists bool
	err := s.metricsPool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM email_link_clicks
			WHERE session_id = $1
			  AND email_id = $2
			  AND link_index = $3
			  AND time > NOW() - INTERVAL '5 minutes'
			LIMIT 1
		)
	`, sessionID, emailID, linkIndex).Scan(&exists)
	
	if err != nil {
		return err
	}
	
	// Only insert if not already clicked in last 5 minutes
	if !exists {
		_, err = s.metricsPool.Exec(ctx, `
			INSERT INTO email_link_clicks (session_id, email_id, link_url, link_index)
			VALUES ($1, $2, $3, $4)
		`, sessionID, emailID, linkURL, linkIndex)
		return err
	}
	
	return nil
}

func (s *Store) GetMetricsViewCount(ctx context.Context, emailID string) (int64, error) {
	if s.metricsPool == nil {
		return 0, nil
	}
	
	var count int64
	err := s.metricsPool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT session_id)
		FROM email_views
		WHERE email_id = $1
	`, emailID).Scan(&count)
	
	if err != nil && err.Error() != "no rows in result set" {
		return 0, nil
	}
	
	return count, nil
}

func (s *Store) GetMetricsClickCount(ctx context.Context, emailID string) (int64, error) {
	if s.metricsPool == nil {
		return 0, nil
	}
	
	var count int64
	err := s.metricsPool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT (session_id, link_index))
		FROM email_link_clicks
		WHERE email_id = $1
	`, emailID).Scan(&count)
	
	if err != nil && err.Error() != "no rows in result set" {
		return 0, nil
	}
	
	return count, nil
}

func (s *Store) GetEmailViewCount(ctx context.Context, emailID string) (int64, error) {
	metricsCount, _ := s.GetMetricsViewCount(ctx, emailID)
	
	var warehouseOpens int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(opens, 0)
		FROM loops.campaigns
		WHERE id = $1
	`, emailID).Scan(&warehouseOpens)
	
	if err != nil {
		if !strings.Contains(err.Error(), "does not exist") && err.Error() != "no rows in result set" {
			log.Printf("warehouse opens error: %v", err)
		}
		warehouseOpens = 0
	}
	
	return metricsCount + warehouseOpens, nil
}

// ---------- View Notifier ----------

type ViewNotifier struct {
	mu          sync.RWMutex
	subscribers map[string][]chan struct{}
}

func NewViewNotifier() *ViewNotifier {
	return &ViewNotifier{
		subscribers: make(map[string][]chan struct{}),
	}
}

func (vn *ViewNotifier) Subscribe(emailID string) chan struct{} {
	vn.mu.Lock()
	defer vn.mu.Unlock()
	ch := make(chan struct{}, 10)
	vn.subscribers[emailID] = append(vn.subscribers[emailID], ch)
	return ch
}

func (vn *ViewNotifier) Unsubscribe(emailID string, ch chan struct{}) {
	vn.mu.Lock()
	defer vn.mu.Unlock()
	subs := vn.subscribers[emailID]
	for i, sub := range subs {
		if sub == ch {
			vn.subscribers[emailID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
	if len(vn.subscribers[emailID]) == 0 {
		delete(vn.subscribers, emailID)
	}
}

func (vn *ViewNotifier) Notify(emailID string) {
	vn.mu.RLock()
	defer vn.mu.RUnlock()
	for _, ch := range vn.subscribers[emailID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ---------- Click Tracker Rate Limiter ----------

type ClickTracker struct {
	mu       sync.RWMutex
	clicks   map[string]time.Time // key: IP address
	cleanupC chan struct{}
}

func NewClickTracker() *ClickTracker {
	ct := &ClickTracker{
		clicks:   make(map[string]time.Time),
		cleanupC: make(chan struct{}),
	}
	
	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ct.cleanup()
			case <-ct.cleanupC:
				return
			}
		}
	}()
	
	return ct
}

func (ct *ClickTracker) cleanup() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	cutoff := time.Now().Add(-1 * time.Minute)
	for ip, lastClick := range ct.clicks {
		if lastClick.Before(cutoff) {
			delete(ct.clicks, ip)
		}
	}
}

// ShouldTrack returns true if this IP should be tracked (max 10 clicks/second)
func (ct *ClickTracker) ShouldTrack(ip string) bool {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	now := time.Now()
	lastClick, exists := ct.clicks[ip]
	
	// Allow if never seen or last click was >100ms ago (10/sec max)
	if !exists || now.Sub(lastClick) > 100*time.Millisecond {
		ct.clicks[ip] = now
		return true
	}
	
	return false
}

// ---------- HTTP Handlers ----------

type Server struct {
	store        *Store
	cache        *TTLCache
	viewNotifier *ViewNotifier
	clickTracker *ClickTracker
}

func NewServer(store *Store) *Server {
	return &Server{
		store:        store,
		cache:        NewTTLCache(30*time.Second, 512),
		viewNotifier: NewViewNotifier(),
		clickTracker: NewClickTracker(),
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
		emails, next, err := s.store.ListEmails(r.Context(), r, mlid, limit, offset)
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

func getOrCreateSession(w http.ResponseWriter, r *http.Request) *http.Cookie {
	cookie, err := r.Cookie("_track")
	if err != nil {
		sessionID := generateSessionID()
		cookie = &http.Cookie{
			Name:     "_track",
			Value:    sessionID,
			MaxAge:   30 * 24 * 60 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
			Path:     "/",
		}
		http.SetCookie(w, cookie)
	}
	return cookie
}

func (s *Server) handleEmailView(w http.ResponseWriter, r *http.Request) {
	emailID := chi.URLParam(r, "id")
	if emailID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiErr{Message: "missing email id"})
		return
	}

	cookie := getOrCreateSession(w, r)

	if err := s.store.TrackEmailView(r.Context(), cookie.Value, emailID); err != nil {
		log.Printf("track view error: %v", err)
	} else {
		s.viewNotifier.Notify(emailID)
	}

	viewCount, err := s.store.GetEmailViewCount(r.Context(), emailID)
	if err != nil {
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]int64{"views": viewCount})
}

func (s *Server) handleLinkClick(w http.ResponseWriter, r *http.Request) {
	emailID := chi.URLParam(r, "id")
	linkIndexStr := chi.URLParam(r, "index")
	targetURL := r.URL.Query().Get("url")
	
	if emailID == "" || linkIndexStr == "" || targetURL == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}
	
	linkIndex, err := strconv.Atoi(linkIndexStr)
	if err != nil {
		http.Error(w, "invalid link index", http.StatusBadRequest)
		return
	}
	
	// Always get/set session cookie
	cookie := getOrCreateSession(w, r)
	
	// Rate limit tracking (not redirect) - max 10 clicks/sec per IP
	clientIP := r.RemoteAddr
	if shouldTrack := s.clickTracker.ShouldTrack(clientIP); shouldTrack {
		if err := s.store.TrackLinkClick(r.Context(), cookie.Value, emailID, targetURL, linkIndex); err != nil {
			log.Printf("track click error: %v", err)
		} else {
			s.viewNotifier.Notify(emailID)
		}
	}
	// If rate limited, we skip tracking but still redirect
	
	// ALWAYS redirect regardless of tracking
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func (s *Server) handleEmailStatsStream(w http.ResponseWriter, r *http.Request) {
	emailID := chi.URLParam(r, "id")
	if emailID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	notifyCh := s.viewNotifier.Subscribe(emailID)
	defer s.viewNotifier.Unsubscribe(emailID, notifyCh)

	throttle := time.NewTicker(333 * time.Millisecond)
	defer throttle.Stop()

	sendUpdate := func() {
		viewCount, err := s.store.GetEmailViewCount(r.Context(), emailID)
		if err != nil {
			log.Printf("stream view count error: %v", err)
			return
		}
		
		metricsClicks, _ := s.store.GetMetricsClickCount(r.Context(), emailID)
		var warehouseClicks int64
		_ = s.store.pool.QueryRow(r.Context(), `
			SELECT COALESCE(clicks, 0)
			FROM loops.campaigns
			WHERE id = $1
		`, emailID).Scan(&warehouseClicks)
		clickCount := metricsClicks + warehouseClicks
		
		stats := map[string]int64{
			"views":  viewCount,
			"clicks": clickCount,
		}
		data, _ := json.Marshal(stats)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sendUpdate()

	var pending bool
	for {
		select {
		case <-notifyCh:
			pending = true
		case <-throttle.C:
			if pending {
				sendUpdate()
				pending = false
			}
		case <-r.Context().Done():
			return
		}
	}
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
			emails, _, err := s.store.ListEmails(r.Context(), r, &mlid, limitPerList, 0)
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
	status := http.StatusInternalServerError
	public := "internal server error"

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
		public = "upstream timed out"
	case func() bool { nerr, ok := err.(net.Error); return ok && nerr.Timeout() }():
		status = http.StatusGatewayTimeout
		public = "network timeout"
	}

	log.Printf("error: %v", err)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiErr{Message: public})
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
	metricsDBURL := os.Getenv("METRICS_DATABASE_URL")
	
	store, err := NewStore(ctx, dbURL, metricsDBURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer store.pool.Close()
	if store.metricsPool != nil {
		defer store.metricsPool.Close()
	}

	if err := store.RunMetricsMigrations(ctx); err != nil {
		log.Fatalf("metrics migrations failed: %v", err)
	}

	srv := NewServer(store)

	var trustedCIDRs []*net.IPNet
	if cidrStr := os.Getenv("TRUSTED_PROXY_CIDRS"); cidrStr != "" {
		for _, cidr := range strings.Split(cidrStr, ",") {
			_, n, err := net.ParseCIDR(strings.TrimSpace(cidr))
			if err != nil {
				log.Printf("warning: invalid CIDR %q: %v", cidr, err)
				continue
			}
			trustedCIDRs = append(trustedCIDRs, n)
		}
	}

	var allowedOrigins []string
	if originsStr := os.Getenv("CORS_ALLOWED_ORIGINS"); originsStr != "" {
		for _, origin := range strings.Split(originsStr, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
		}
		log.Printf("CORS allowed origins: %v", allowedOrigins)
	}

	r := chi.NewRouter()
	r.Use(trustProxyRealIP(trustedCIDRs))
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))
	r.Use(middleware.Timeout(30 * time.Second))
	if len(allowedOrigins) > 0 {
		r.Use(corsMiddleware(allowedOrigins))
	}
	r.Use(securityHeaders())

	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(30, 1*time.Second))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/docs", http.StatusFound) })
		r.Get("/docs", srv.handleDocs)
		r.Get("/mailing_lists", srv.handleMailingLists)
		r.Get("/emails", srv.handleEmails)
		r.Get("/emails/{id}/view", srv.handleEmailView)
		r.Get("/mailing_lists/emails", srv.handleMailingListsEmails)
	})

	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(100, 1*time.Second))
		r.Get("/emails/{id}/stats/stream", srv.handleEmailStatsStream)
	})

	// Link clicks: ALWAYS redirect, but rate limit tracking
	r.Get("/emails/{id}/click/{index}", srv.handleLinkClick)

	addr := env("HOST", "127.0.0.1") + ":" + env("PORT", "8080")
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func trustProxyRealIP(trustedCIDRs []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			trusted := false
			for _, n := range trustedCIDRs {
				if n.Contains(ip) {
					trusted = true
					break
				}
			}
			if !trusted {
				r.Header.Del("X-Forwarded-For")
				r.Header.Del("X-Real-IP")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	localhostRegex := regexp.MustCompile(`^https?://localhost(:\d+)?$|^https?://127\.0\.0\.1(:\d+)?$|^https?://\[::1\](:\d+)?$`)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			if origin != "" {
				allowed := false
				
				if localhostRegex.MatchString(origin) {
					allowed = true
				} else if len(allowedOrigins) > 0 {
					for _, allowedOrigin := range allowedOrigins {
						if origin == allowedOrigin || allowedOrigin == "*" {
							allowed = true
							break
						}
					}
				}
				
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
			}
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none';")
			if os.Getenv("ENABLE_HSTS") == "1" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}
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
        "clicks": 82,
        "views": 1234
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
- ` + "`stats.views`" + ` = real-time TimescaleDB views + warehouse opens (email opens from Loops).
- ` + "`stats.clicks`" + ` = real-time TimescaleDB link clicks + warehouse clicks from Loops.
- ` + "`html`" + ` field contains **rewritten links** for click tracking (see Link Click Tracking below).
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

## GET /emails/{id}/view

Track a page view for an email and return the total view count.

### Behavior
- **Automatic tracking**: Sets a ` + "`_track`" + ` cookie (30-day session ID) and records the view.
- **Deduplication**: Same session + email + 5-minute window = counted once.
- **Privacy-first**: Only tracks anonymous session IDs, no PII.
- **Combined counts**: Returns views from both TimescaleDB (real-time) + warehouse analytics.

### Response
` + "```json" + `
{
  "views": 1234
}
` + "```" + `

### Cookie
The server sets ` + "`_track`" + ` cookie automatically:
- ` + "`HttpOnly`" + `, ` + "`SameSite=Lax`" + `, ` + "`Secure`" + ` (on HTTPS)
- ` + "`Max-Age: 2592000`" + ` (30 days)
- Path: ` + "`/`" + `

---

## Link Click Tracking

All links in email HTML are automatically rewritten to track clicks while preserving the user experience.

### How It Works

1. **Automatic Link Rewriting**: When you fetch email HTML from ` + "`/emails`" + `, all ` + "`<a href>`" + ` tags are rewritten:
   - Original: ` + "`<a href=\"https://example.com\">Click here</a>`" + `
   - Rewritten: ` + "`<a href=\"/emails/{id}/click/0?url=https%3A%2F%2Fexample.com\">Click here</a>`" + `

2. **Link Indexing**: Each link gets a sequential index (0, 1, 2...) for tracking which specific links are clicked.

3. **Preserved Links**: ` + "`mailto:`" + `, ` + "`tel:`" + `, and ` + "`#`" + ` anchor links are **not** rewritten.

4. **Click Tracking**: When a user clicks a rewritten link:
   - Session is tracked via ` + "`_track`" + ` cookie (same as view tracking)
   - Click is recorded in TimescaleDB
   - User is redirected to the original URL (302 redirect)

5. **Deduplication**: Same session clicking the same link = counted once (per email).

---

## GET /emails/{id}/click/{index}?url={url}

Track a link click and redirect to the original URL.

### Parameters
- ` + "`id`" + ` - Email ID
- ` + "`index`" + ` - Link index (0-based, from HTML rewriting)
- ` + "`url`" + ` - URL-encoded original destination

### Behavior
- Sets ` + "`_track`" + ` cookie if not present (30-day session)
- Records click in TimescaleDB with deduplication
- Emits real-time event to SSE subscribers
- Returns 302 redirect to original URL

### Example
` + "```" + `
GET /emails/abc123/click/0?url=https%3A%2F%2Fexample.com
→ 302 Redirect to https://example.com
→ Click tracked in database
→ SSE subscribers notified
` + "```" + `

---

## GET /emails/{id}/stats/stream

Real-time Server-Sent Events (SSE) stream of **both** view and click count updates.

### Behavior
- Streams stats updates whenever views OR clicks are tracked
- Throttled to max 3 updates/second to prevent flooding
- Auto-closes when client disconnects
- Sends initial stats immediately on connection

### Response Format
` + "```" + `
data: {"views":1234,"clicks":82}

data: {"views":1235,"clicks":82}

data: {"views":1235,"clicks":83}
` + "```" + `

Each message is a JSON object with both view and click counts.

### Frontend Example
` + "```javascript" + `
const es = new EventSource('/emails/abc123/stats/stream');
es.onmessage = e => {
  const stats = JSON.parse(e.data);
  document.getElementById('views').textContent = stats.views;
  document.getElementById('clicks').textContent = stats.clicks;
};
` + "```" + `

### Update Triggers
Events are emitted when:
- A view is tracked (` + "`/emails/{id}/view`" + `)
- A link click is tracked (` + "`/emails/{id}/click/{index}`" + `)
- Updates are throttled: rapid events are batched into periodic updates (333ms interval)

---

## Click Analytics

### Counting Method
- **Database**: Stores all click events with session_id, email_id, link_index, link_url, timestamp
- **Deduplication**: Uses ` + "`COUNT(DISTINCT (session_id, link_index))`" + ` to count unique clicks
- **Combined Total**: TimescaleDB tracked clicks + warehouse clicks from Loops

### Privacy & Session Tracking
- Same ` + "`_track`" + ` cookie used for both views and clicks
- Anonymous session IDs only (no PII)
- HttpOnly, SameSite=Lax, Secure (on HTTPS)
- 30-day cookie lifetime

### Deduplication Rules
- Same session + same link = 1 click (counted once)
- Same session + different links = multiple clicks
- Different sessions + same link = multiple clicks
- Multiple clicks within 5-min window stored but counted as one

---
`
