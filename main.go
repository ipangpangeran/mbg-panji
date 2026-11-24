// MBG Ordering System — lightweight food ordering app.
// Single-file backend: SQLite (modernc, pure Go), Gin REST API, Excel export.
package main

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // embed tz database so Asia/Jakarta works on minimal alpine images
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Settings & globals
// ---------------------------------------------------------------------------

const (
	timeLayout    = "2006-01-02 15:04:05"
	maxBodyBytes  = 3 << 20 // 3 MB (allows base64 QR image in config)
	maxNameLen    = 60
	maxMenuLen    = 150
	maxNoteLen    = 200
	maxMenuItems  = 50
	maxItemQty    = 100
	maxLoginFails = 5
	lockDuration  = 5 * time.Minute
)

var (
	db         *sql.DB
	loc        *time.Location
	orderMu    sync.Mutex // serialises quota check + insert
	pinRegex         = regexp.MustCompile(`^\d{6}$`)
	parenRegex       = regexp.MustCompile(`\(([^()]+)\)\s*$`)
	priceRegex       = regexp.MustCompile(`(?i)^([+-])?\s*(?:rp\.?\s*)?(\d+(?:[.,]\d+)?)\s*(k|rb|ribu)?$`)
	barePriceSuffix  = regexp.MustCompile(`(?i)(?:^|\s)((?:[+-]\s*)?(?:rp\.?\s*\d+(?:[.,]\d+)?(?:\s*(?:k|rb|ribu))?|\d+(?:[.,]\d+)?\s*(?:k|rb|ribu)|[+-]\s*\d{3,}|\d{4,}))\s*$`)
	unitRegex        = regexp.MustCompile(`(?i)^\d+\s*(k|rb|ribu|pcs|gr|gram|ml|l|bks|porsi)$`)
	guard            = &loginGuard{fails: map[string]*failRec{}}
)

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

type AppConfig struct {
	IsOpen        bool     `json:"is_open"`
	GlobalQuota   int      `json:"global_quota"`
	MainMenus     []string `json:"main_menus"`
	AlacarteMenus []string `json:"alacarte_menus"`
	SideMenus     []string `json:"side_menus"`
	CustomOptions []string `json:"custom_options"`
	QRURL         string   `json:"qr_url"`
	AdminPIN      string   `json:"-"`
}

type OrderItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

type Order struct {
	ID            int64       `json:"id"`
	Timestamp     string      `json:"timestamp"`
	Name          string      `json:"name"`
	MainMenus     []OrderItem `json:"main_menus"`
	MainMenu      string      `json:"main_menu"`
	AlacarteMenus []OrderItem `json:"alacarte_menus"`
	SideMenus     []OrderItem `json:"side_menus"`
	CustomItems   []OrderItem `json:"custom_items"`
	CustomOptions []string    `json:"custom_options"` // formatted labels for backward compatibility
	CustomNote    string      `json:"custom_note"`
	QuotaUsed     int         `json:"quota_used"` // MainMenus qty + Lauk Ala Carte qty
	TotalPrice    int         `json:"total_price"`
	PaymentMethod string      `json:"payment_method"`
	Status        string      `json:"status"`
}

type CountItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Summary struct {
	Total          int         `json:"total"`          // Total quota slots used (Paket + Lauk Ala Carte)
	PackageTotal   int         `json:"package_total"`  // Total Menu Paket portions
	LaukAlacarte   int         `json:"lauk_alacarte"`  // Total Lauk Ala Carte portions (quota-consuming)
	OrderCount     int         `json:"order_count"`    // Total customer transactions
	TotalRevenue   int         `json:"total_revenue"`  // Sum of TotalPrice across orders
	Quota          int         `json:"quota"`
	Remaining      int         `json:"remaining"`
	IsOpen         bool        `json:"is_open"`
	ByMainMenu     []CountItem `json:"by_main_menu"`
	ByAlacarteMenu []CountItem `json:"by_alacarte_menu"`
	BySideMenu     []CountItem `json:"by_side_menu"`
	ByCustomOption []CountItem `json:"by_custom_option"`
	CustomOrderCnt int         `json:"custom_order_count"`
	ByPayment      []CountItem `json:"by_payment"`
	GeneratedAt    string      `json:"generated_at"`
}

// ---------------------------------------------------------------------------
// Price, Quota & Title-Case Helpers
// ---------------------------------------------------------------------------

// isLaukAlacarte determines whether an Ala Carte item counts toward the daily Lauk/Paket quota.
// True when the item title contains "Lauk" (case-insensitive) OR does NOT start with non-lauk staples
// like Nasi, Sayur, Tumis, Krupuk/Kerupuk, Sambal, Kuah (so e.g. "Ayam Rendang (8k)" also counts if admin omits "Lauk").
var nonLaukPrefix = regexp.MustCompile(`(?i)^(nasi|sayur|tumis|capcay|sop|sup|bening|krupuk|kerupuk|emping|peyek|sambal|sambel|kuah|lalap|lalapan)\b`)
var laukWord = regexp.MustCompile(`(?i)\blauk\b`)

func isLaukAlacarte(name string) bool {
	title := strings.TrimSpace(shortMenuName(name))
	if title == "" {
		return false
	}
	if laukWord.MatchString(name) {
		return true
	}
	if nonLaukPrefix.MatchString(title) {
		return false
	}
	// Protein dish written directly without "Lauk" prefix (e.g. "Ayam Rendang (8k)", "Bandeng Presto (8k)")
	return true
}

func calcQuotaUsed(mains, alacartes []OrderItem) int {
	q := 0
	for _, m := range mains {
		q += m.Qty
	}
	for _, a := range alacartes {
		if isLaukAlacarte(a.Name) {
			q += a.Qty
		}
	}
	return q
}

// extractPriceParts splits a menu string into (bodyWithoutPrice, normalizedTag, priceInIDR).
// Supports both parenthesized "(5k)", "(-4k)", "(Rp 15.000)" AND bare trailing "5k", "10k", "-4k", "Rp 15.000", "15000".
func extractPriceParts(raw string) (body string, tag string, price int) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", 0
	}
	parseToken := func(tok string) (string, int, bool) {
		pm := priceRegex.FindStringSubmatch(strings.TrimSpace(tok))
		if len(pm) < 4 {
			return "", 0, false
		}
		sign := 1
		if pm[1] == "-" {
			sign = -1
		}
		numStr := strings.ReplaceAll(pm[2], ".", "")
		numStr = strings.ReplaceAll(numStr, ",", ".")
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return "", 0, false
		}
		if pm[3] != "" {
			val *= 1000
		}
		absIDR := int(math.Round(val))
		// Format canonical compact tag e.g. "(15k)" or "(-4k)"
		prefix := ""
		if sign < 0 {
			prefix = "-"
		} else if pm[1] == "+" {
			prefix = "+"
		}
		var compact string
		if absIDR >= 1000 && absIDR%1000 == 0 {
			compact = fmt.Sprintf("(%s%dk)", prefix, absIDR/1000)
		} else if absIDR >= 1000 && absIDR%100 == 0 {
			compact = fmt.Sprintf("(%s%.1fk)", prefix, float64(absIDR)/1000.0)
		} else {
			compact = fmt.Sprintf("(%s%d)", prefix, absIDR)
		}
		return compact, sign * absIDR, true
	}

	// 1. Check parenthesized trailing price e.g. "Nasi Putih (5k)"
	if locIdx := parenRegex.FindStringSubmatchIndex(s); locIdx != nil {
		inside := s[locIdx[2]:locIdx[3]]
		if normTag, p, ok := parseToken(inside); ok {
			return strings.TrimSpace(s[:locIdx[0]]), normTag, p
		}
	}
	// 2. Check bare trailing price e.g. "nasi putih 5k", "lauk ayam 10k", "tanpa nasi -4k"
	if locIdx := barePriceSuffix.FindStringSubmatchIndex(s); locIdx != nil {
		b := strings.TrimSpace(s[:locIdx[0]])
		tok := s[locIdx[2]:locIdx[3]]
		if b != "" {
			if normTag, p, ok := parseToken(tok); ok {
				return b, normTag, p
			}
		}
	}
	return s, "", 0
}

func parseItemPrice(raw string) int {
	_, _, p := extractPriceParts(raw)
	return p
}

func calcOrderTotal(mains, alacartes, sides, customs []OrderItem) int {
	total := 0
	for _, m := range mains {
		total += m.Qty * parseItemPrice(m.Name)
	}
	for _, a := range alacartes {
		total += a.Qty * parseItemPrice(a.Name)
	}
	for _, s := range sides {
		total += s.Qty * parseItemPrice(s.Name)
	}
	for _, c := range customs {
		total += c.Qty * parseItemPrice(c.Name)
	}
	if total < 0 {
		return 0
	}
	return total
}

func formatIDR(n int) string {
	if n <= 0 {
		return "Rp 0"
	}
	s := strconv.Itoa(n)
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}
	return "Rp " + string(out)
}

// titleCaseMenu capitalizes each word and auto-normalizes trailing prices like "nasi 5k" -> "Nasi (5k)".
func titleCaseMenu(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	body, normTag, _ := extractPriceParts(s)
	suffix := ""
	if normTag != "" {
		suffix = " " + normTag
		s = body
	}

	words := strings.Split(s, " ")
	for i, w := range words {
		if w == "" || w == "-" || w == "+" || w == "/" || w == "&" {
			continue
		}
		if unitRegex.MatchString(w) {
			words[i] = strings.ToLower(w)
			continue
		}
		runes := []rune(w)
		capNext := true
		for j, r := range runes {
			if unicode.IsLetter(r) {
				if capNext {
					runes[j] = unicode.ToUpper(r)
					capNext = false
				}
			} else if r == '/' || r == '(' || r == '-' {
				capNext = true
			} else {
				capNext = false
			}
		}
		words[i] = string(runes)
	}
	return strings.TrimSpace(strings.Join(words, " ")) + suffix
}

// ---------------------------------------------------------------------------
// Database
// ---------------------------------------------------------------------------

func openDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1)
	if err := d.Ping(); err != nil {
		return nil, err
	}
	return d, nil
}

func migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS Config (
			id    INTEGER PRIMARY KEY,
			key   TEXT UNIQUE,
			value TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS Orders (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp           DATETIME,
			name                TEXT,
			main_menu           TEXT,
			main_menus_json     TEXT,
			alacarte_menus_json TEXT,
			side_menus_json     TEXT,
			custom_options_json TEXT,
			custom_note         TEXT,
			total_price         INTEGER DEFAULT 0,
			payment_method      TEXT,
			status              TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_ts ON Orders(timestamp)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	alterCols := []string{
		`ALTER TABLE Orders ADD COLUMN main_menus_json TEXT DEFAULT ''`,
		`ALTER TABLE Orders ADD COLUMN alacarte_menus_json TEXT DEFAULT '[]'`,
		`ALTER TABLE Orders ADD COLUMN custom_options_json TEXT DEFAULT '[]'`,
		`ALTER TABLE Orders ADD COLUMN custom_note TEXT DEFAULT ''`,
		`ALTER TABLE Orders ADD COLUMN total_price INTEGER DEFAULT 0`,
	}
	for _, s := range alterCols {
		_, _ = db.Exec(s) // ignore duplicate column error
	}
	return nil
}

// seedConfig inserts default values for any missing key (idempotent).
func seedConfig() error {
	pin := env("ADMIN_PIN", "123456")
	if !pinRegex.MatchString(pin) {
		log.Printf("WARNING: ADMIN_PIN env is not 6 digits, falling back to 123456")
		pin = "123456"
	}
	defaultMain := []string{
		"Paket Ayam - Nasi + Ayam Rendang + Tumis Buncis Labu Siam + Krupuk (15k)",
		"Paket Bandeng - Nasi + Bandeng Presto + Tumis Buncis Labu Siam + Krupuk (15k)",
	}
	// Ala Carte: pecahan menu paket (Lauk ikut memotong kuota harian)
	defaultAlacarte := []string{
		"Lauk Ayam Rendang (8k)",
		"Lauk Bandeng Presto (8k)",
		"Nasi Putih (5k)",
		"Sayur Tumis Buncis Labu Siam (3k)",
		"Krupuk (1k)",
	}
	// Add-ons: gorengan & menu pendamping di luar isi paket
	defaultSides := []string{
		"Gorengan Martabak Tahu 2pcs (3k)",
		"Gorengan Martabak Tahu 4pcs (5k)",
		"Orek Tempe (3k)",
		"Terong Balado (3k)",
		"Mie Goreng (3k)",
	}
	defaultCustom := []string{
		"Tanpa Nasi (-4k)",
		"Nasi Setengah",
		"Tanpa Kerupuk",
		"Tanpa Sayur/Tumis",
	}
	mainMenus, _ := json.Marshal(defaultMain)
	alacarteMenus, _ := json.Marshal(defaultAlacarte)
	sideMenus, _ := json.Marshal(defaultSides)
	customOpts, _ := json.Marshal(defaultCustom)

	defaults := [][2]string{
		{"is_open", "true"},
		{"global_quota", "50"},
		{"admin_pin", pin},
		{"main_menus_json", string(mainMenus)},
		{"alacarte_menus_json", string(alacarteMenus)},
		{"side_menus_json", string(sideMenus)},
		{"custom_options_json", string(customOpts)},
		{"qr_url", "/qris.jpg"},
	}
	for _, kv := range defaults {
		if _, err := db.Exec(`INSERT OR IGNORE INTO Config(key, value) VALUES(?, ?)`, kv[0], kv[1]); err != nil {
			return err
		}
	}
	_, _ = db.Exec(`UPDATE Config SET value = '/qris.jpg' WHERE key = 'qr_url' AND (value IS NULL OR value = '')`)

	var curCustom string
	if err := db.QueryRow(`SELECT value FROM Config WHERE key = 'custom_options_json'`).Scan(&curCustom); err == nil {
		if curCustom == `["Tanpa Nasi","Tanpa Kerupuk","Tanpa Sayur/Tumis","Nasi Setengah"]` {
			_, _ = db.Exec(`UPDATE Config SET value = ? WHERE key = 'custom_options_json'`, string(customOpts))
		}
	}
	return nil
}

func loadConfig() (AppConfig, error) {
	rows, err := db.Query(`SELECT key, value FROM Config`)
	if err != nil {
		return AppConfig{}, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k string
		var v sql.NullString
		if err := rows.Scan(&k, &v); err != nil {
			return AppConfig{}, err
		}
		m[k] = v.String
	}
	if err := rows.Err(); err != nil {
		return AppConfig{}, err
	}

	cfg := AppConfig{
		IsOpen:   m["is_open"] == "true",
		QRURL:    m["qr_url"],
		AdminPIN: m["admin_pin"],
	}
	cfg.GlobalQuota, _ = strconv.Atoi(m["global_quota"])
	_ = json.Unmarshal([]byte(m["main_menus_json"]), &cfg.MainMenus)
	_ = json.Unmarshal([]byte(m["alacarte_menus_json"]), &cfg.AlacarteMenus)
	_ = json.Unmarshal([]byte(m["side_menus_json"]), &cfg.SideMenus)
	_ = json.Unmarshal([]byte(m["custom_options_json"]), &cfg.CustomOptions)
	if cfg.MainMenus == nil {
		cfg.MainMenus = []string{}
	}
	if cfg.AlacarteMenus == nil {
		cfg.AlacarteMenus = []string{}
	}
	if cfg.SideMenus == nil {
		cfg.SideMenus = []string{}
	}
	if cfg.CustomOptions == nil {
		cfg.CustomOptions = []string{}
	}
	return cfg, nil
}

func parseOrderItems(raw string) []OrderItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []OrderItem{}
	}
	var items []OrderItem
	if err := json.Unmarshal([]byte(raw), &items); err == nil && items != nil {
		out := []OrderItem{}
		for _, it := range items {
			if it.Name != "" && it.Qty > 0 {
				out = append(out, it)
			}
		}
		return out
	}
	var strs []string
	if err := json.Unmarshal([]byte(raw), &strs); err == nil {
		out := []OrderItem{}
		for _, s := range strs {
			if s != "" {
				out = append(out, OrderItem{Name: s, Qty: 1})
			}
		}
		return out
	}
	return []OrderItem{}
}

func shortMenuName(s string) string {
	body, _, _ := extractPriceParts(s)
	if idx := strings.Index(body, " - "); idx > 0 {
		return strings.TrimSpace(body[:idx])
	}
	if body != "" {
		return body
	}
	return s
}

func formatItems(items []OrderItem) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		if it.Qty > 0 && it.Name != "" {
			parts = append(parts, fmt.Sprintf("%dx %s", it.Qty, shortMenuName(it.Name)))
		}
	}
	return strings.Join(parts, ", ")
}

func formatCustomItems(items []OrderItem, note string) string {
	parts := make([]string, 0, len(items)+1)
	for _, it := range items {
		if it.Qty <= 0 || it.Name == "" {
			continue
		}
		if it.Qty > 1 {
			parts = append(parts, fmt.Sprintf("%dx %s", it.Qty, shortMenuName(it.Name)))
		} else {
			parts = append(parts, shortMenuName(it.Name))
		}
	}
	if strings.TrimSpace(note) != "" {
		parts = append(parts, strings.TrimSpace(note))
	}
	return strings.Join(parts, ", ")
}

func countUsedQuota() (int, error) {
	orders, err := listOrders()
	if err != nil {
		return 0, err
	}
	total := 0
	for _, o := range orders {
		total += o.QuotaUsed
	}
	return total, nil
}

func listOrders() ([]Order, error) {
	rows, err := db.Query(`SELECT id, timestamp, name, main_menu, main_menus_json, alacarte_menus_json, side_menus_json, custom_options_json, custom_note, total_price, payment_method, status
		FROM Orders ORDER BY timestamp DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var o Order
		var ts, name, mainMenu, mainsJSON, alacarteJSON, sidesJSON, customOptsJSON, customNote, pay, status sql.NullString
		var totalPrice sql.NullInt64
		if err := rows.Scan(&o.ID, &ts, &name, &mainMenu, &mainsJSON, &alacarteJSON, &sidesJSON, &customOptsJSON, &customNote, &totalPrice, &pay, &status); err != nil {
			return nil, err
		}
		o.Timestamp = normalizeTS(ts.String)
		o.Name = name.String
		o.MainMenus = parseOrderItems(mainsJSON.String)
		o.AlacarteMenus = parseOrderItems(alacarteJSON.String)
		if len(o.MainMenus) == 0 && len(o.AlacarteMenus) == 0 && strings.TrimSpace(mainMenu.String) != "" && mainMenu.String != "-" {
			o.MainMenus = []OrderItem{{Name: mainMenu.String, Qty: 1}}
		}
		o.MainMenu = formatItems(o.MainMenus)
		o.SideMenus = parseOrderItems(sidesJSON.String)
		o.CustomItems = parseOrderItems(customOptsJSON.String)
		o.CustomOptions = make([]string, 0, len(o.CustomItems))
		for _, ci := range o.CustomItems {
			if ci.Qty > 1 {
				o.CustomOptions = append(o.CustomOptions, fmt.Sprintf("%dx %s", ci.Qty, shortMenuName(ci.Name)))
			} else {
				o.CustomOptions = append(o.CustomOptions, shortMenuName(ci.Name))
			}
		}
		o.CustomNote = customNote.String
		o.QuotaUsed = calcQuotaUsed(o.MainMenus, o.AlacarteMenus)
		o.TotalPrice = int(totalPrice.Int64)
		if o.TotalPrice == 0 {
			o.TotalPrice = calcOrderTotal(o.MainMenus, o.AlacarteMenus, o.SideMenus, o.CustomItems)
		}
		o.PaymentMethod = pay.String
		o.Status = status.String
		out = append(out, o)
	}
	return out, rows.Err()
}

func normalizeTS(s string) string {
	for _, layout := range []string{timeLayout, time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05 -0700 MST"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format(timeLayout)
		}
	}
	return s
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func cleanList(in []string, autoCap bool) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range in {
		s = strings.Join(strings.Fields(s), " ")
		if s == "" {
			continue
		}
		if r := []rune(s); len(r) > maxMenuLen {
			s = string(r[:maxMenuLen])
		}
		if autoCap {
			s = titleCaseMenu(s)
		}
		key := strings.ToLower(s)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
		if len(out) >= maxMenuItems {
			break
		}
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func paymentLabel(p string) string {
	switch p {
	case "qris":
		return "QRIS"
	case "cash":
		return "Cash"
	}
	return p
}

func buildSummary(cfg AppConfig, orders []Order) Summary {
	mainCount := map[string]int{}
	alacarteCount := map[string]int{}
	sideCount := map[string]int{}
	customCount := map[string]int{}
	payCount := map[string]int{}
	packagePortions := 0
	laukAlacarte := 0
	totalQuotaUsed := 0
	totalRevenue := 0
	customOrderCnt := 0

	for _, o := range orders {
		for _, m := range o.MainMenus {
			mainCount[m.Name] += m.Qty
			packagePortions += m.Qty
		}
		for _, a := range o.AlacarteMenus {
			alacarteCount[a.Name] += a.Qty
			if isLaukAlacarte(a.Name) {
				laukAlacarte += a.Qty
			}
		}
		for _, s := range o.SideMenus {
			sideCount[s.Name] += s.Qty
		}
		hasCustom := len(o.CustomItems) > 0 || strings.TrimSpace(o.CustomNote) != ""
		if hasCustom {
			customOrderCnt++
		}
		for _, c := range o.CustomItems {
			customCount[c.Name] += c.Qty
		}
		totalQuotaUsed += o.QuotaUsed
		totalRevenue += o.TotalPrice
		payCount[o.PaymentMethod]++
	}

	ordered := func(conf []string, counts map[string]int) []CountItem {
		res := []CountItem{}
		used := map[string]bool{}
		for _, name := range conf {
			res = append(res, CountItem{Name: name, Count: counts[name]})
			used[name] = true
		}
		extra := []string{}
		for name := range counts {
			if !used[name] {
				extra = append(extra, name)
			}
		}
		sort.Strings(extra)
		for _, name := range extra {
			res = append(res, CountItem{Name: name, Count: counts[name]})
		}
		return res
	}
	remaining := cfg.GlobalQuota - totalQuotaUsed
	if remaining < 0 {
		remaining = 0
	}
	return Summary{
		Total:          totalQuotaUsed,
		PackageTotal:   packagePortions,
		LaukAlacarte:   laukAlacarte,
		OrderCount:     len(orders),
		TotalRevenue:   totalRevenue,
		Quota:          cfg.GlobalQuota,
		Remaining:      remaining,
		IsOpen:         cfg.IsOpen,
		ByMainMenu:     ordered(cfg.MainMenus, mainCount),
		ByAlacarteMenu: ordered(cfg.AlacarteMenus, alacarteCount),
		BySideMenu:     ordered(cfg.SideMenus, sideCount),
		ByCustomOption: ordered(cfg.CustomOptions, customCount),
		CustomOrderCnt: customOrderCnt,
		ByPayment: []CountItem{
			{Name: "Cash", Count: payCount["cash"]},
			{Name: "QRIS", Count: payCount["qris"]},
		},
		GeneratedAt: time.Now().In(loc).Format(timeLayout),
	}
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"ok": false, "error": msg})
}

// ---------------------------------------------------------------------------
// Admin auth (PIN in X-Admin-Pin header) + brute-force guard
// ---------------------------------------------------------------------------

type failRec struct {
	count int
	until time.Time
}

type loginGuard struct {
	mu    sync.Mutex
	fails map[string]*failRec
}

func (g *loginGuard) locked(ip string) (bool, time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if r, ok := g.fails[ip]; ok && time.Now().Before(r.until) {
		return true, time.Until(r.until)
	}
	return false, 0
}

func (g *loginGuard) fail(ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.fails[ip]
	if !ok || (!r.until.IsZero() && time.Now().After(r.until)) {
		r = &failRec{}
		g.fails[ip] = r
	}
	r.count++
	if r.count >= maxLoginFails {
		r.until = time.Now().Add(lockDuration)
		r.count = 0
	}
}

func (g *loginGuard) reset(ip string) {
	g.mu.Lock()
	delete(g.fails, ip)
	g.mu.Unlock()
}

func checkPin(c *gin.Context, pin string) (int, string) {
	ip := c.ClientIP()
	if l, d := guard.locked(ip); l {
		return http.StatusTooManyRequests, fmt.Sprintf("Terlalu banyak percobaan. Coba lagi dalam %d menit.", int(d.Minutes())+1)
	}
	cfg, err := loadConfig()
	if err != nil {
		return http.StatusInternalServerError, "Gagal membaca konfigurasi"
	}
	if !pinRegex.MatchString(pin) || subtle.ConstantTimeCompare([]byte(pin), []byte(cfg.AdminPIN)) != 1 {
		guard.fail(ip)
		return http.StatusUnauthorized, "PIN salah"
	}
	guard.reset(ip)
	return 0, ""
}

func adminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if code, msg := checkPin(c, c.GetHeader("X-Admin-Pin")); code != 0 {
			fail(c, code, msg)
			return
		}
		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// GET /api/config (public — never exposes the PIN)
func handleGetConfig(c *gin.Context) {
	cfg, err := loadConfig()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca konfigurasi")
		return
	}
	used, err := countUsedQuota()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal menghitung pesanan")
		return
	}
	remaining := cfg.GlobalQuota - used
	if remaining < 0 {
		remaining = 0
	}
	// Tell frontend which alacarte items consume Lauk quota so both sides stay 100% consistent
	quotaAlacartes := []string{}
	for _, a := range cfg.AlacarteMenus {
		if isLaukAlacarte(a) {
			quotaAlacartes = append(quotaAlacartes, a)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"is_open":          cfg.IsOpen,
		"global_quota":     cfg.GlobalQuota,
		"main_menus":       cfg.MainMenus,
		"alacarte_menus":   cfg.AlacarteMenus,
		"quota_alacartes":  quotaAlacartes,
		"side_menus":       cfg.SideMenus,
		"custom_options":   cfg.CustomOptions,
		"qr_url":           cfg.QRURL,
		"used":             used,
		"remaining":        remaining,
	})
}

// POST /api/login
func handleLogin(c *gin.Context) {
	var req struct {
		PIN string `json:"pin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "Format permintaan tidak valid")
		return
	}
	if code, msg := checkPin(c, strings.TrimSpace(req.PIN)); code != 0 {
		fail(c, code, msg)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/config (admin) — partial update, auto-capitalizes menu & custom option names.
func handleUpdateConfig(c *gin.Context) {
	var req struct {
		IsOpen        *bool     `json:"is_open"`
		GlobalQuota   *int      `json:"global_quota"`
		MainMenus     *[]string `json:"main_menus"`
		AlacarteMenus *[]string `json:"alacarte_menus"`
		SideMenus     *[]string `json:"side_menus"`
		CustomOptions *[]string `json:"custom_options"`
		QRURL         *string   `json:"qr_url"`
		AdminPIN      *string   `json:"admin_pin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "Format permintaan tidak valid")
		return
	}

	updates := map[string]string{}
	if req.IsOpen != nil {
		updates["is_open"] = strconv.FormatBool(*req.IsOpen)
	}
	if req.GlobalQuota != nil {
		if *req.GlobalQuota < 0 || *req.GlobalQuota > 100000 {
			fail(c, http.StatusBadRequest, "Kuota harus antara 0 dan 100000")
			return
		}
		updates["global_quota"] = strconv.Itoa(*req.GlobalQuota)
	}
	if req.MainMenus != nil {
		list := cleanList(*req.MainMenus, true)
		if len(list) == 0 {
			fail(c, http.StatusBadRequest, "Minimal harus ada 1 menu paket")
			return
		}
		b, _ := json.Marshal(list)
		updates["main_menus_json"] = string(b)
	}
	if req.AlacarteMenus != nil {
		b, _ := json.Marshal(cleanList(*req.AlacarteMenus, true))
		updates["alacarte_menus_json"] = string(b)
	}
	if req.SideMenus != nil {
		b, _ := json.Marshal(cleanList(*req.SideMenus, true))
		updates["side_menus_json"] = string(b)
	}
	if req.CustomOptions != nil {
		b, _ := json.Marshal(cleanList(*req.CustomOptions, true))
		updates["custom_options_json"] = string(b)
	}
	if req.QRURL != nil {
		u := strings.TrimSpace(*req.QRURL)
		if u != "" && !(strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") ||
			strings.HasPrefix(u, "data:image/") || strings.HasPrefix(u, "/")) {
			fail(c, http.StatusBadRequest, "URL QRIS harus diawali http(s)://, / atau data:image/")
			return
		}
		updates["qr_url"] = u
	}
	if req.AdminPIN != nil && strings.TrimSpace(*req.AdminPIN) != "" {
		p := strings.TrimSpace(*req.AdminPIN)
		if !pinRegex.MatchString(p) {
			fail(c, http.StatusBadRequest, "PIN harus 6 digit angka")
			return
		}
		updates["admin_pin"] = p
	}
	if len(updates) == 0 {
		fail(c, http.StatusBadRequest, "Tidak ada perubahan")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal menyimpan konfigurasi")
		return
	}
	for k, v := range updates {
		if _, err := tx.Exec(`INSERT INTO Config(key, value) VALUES(?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			_ = tx.Rollback()
			fail(c, http.StatusInternalServerError, "Gagal menyimpan konfigurasi")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, "Gagal menyimpan konfigurasi")
		return
	}
	_, pinChanged := updates["admin_pin"]
	c.JSON(http.StatusOK, gin.H{"ok": true, "pin_changed": pinChanged})
}

// POST /api/order (public) — accepts Menu Paket, Ala Carte (standalone or combined), and Add-ons.
func handleCreateOrder(c *gin.Context) {
	var req struct {
		Name             string          `json:"name"`
		MainMenu         string          `json:"main_menu"`
		MainMenus        []OrderItem     `json:"main_menus"`
		AlacarteMenus    []OrderItem     `json:"alacarte_menus"`
		SideMenusRaw     json.RawMessage `json:"side_menus"`
		CustomItems      []OrderItem     `json:"custom_items"`
		CustomOptionsRaw json.RawMessage `json:"custom_options"`
		CustomNote       string          `json:"custom_note"`
		PaymentMethod    string          `json:"payment_method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "Format permintaan tidak valid")
		return
	}
	name := strings.Join(strings.Fields(req.Name), " ")
	if name == "" {
		fail(c, http.StatusBadRequest, "Nama wajib diisi")
		return
	}
	if len([]rune(name)) > maxNameLen {
		fail(c, http.StatusBadRequest, fmt.Sprintf("Nama maksimal %d karakter", maxNameLen))
		return
	}
	pay := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if pay != "cash" && pay != "qris" {
		fail(c, http.StatusBadRequest, "Metode pembayaran tidak valid")
		return
	}

	note := strings.Join(strings.Fields(req.CustomNote), " ")
	if r := []rune(note); len(r) > maxNoteLen {
		note = string(r[:maxNoteLen])
	}

	orderMu.Lock()
	defer orderMu.Unlock()

	cfg, err := loadConfig()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca konfigurasi")
		return
	}
	if !cfg.IsOpen {
		fail(c, http.StatusForbidden, "Maaf, pemesanan sedang tutup")
		return
	}

	// 1. Normalize main_menus (Menu Paket)
	mainsInput := req.MainMenus
	if len(mainsInput) == 0 && strings.TrimSpace(req.MainMenu) != "" {
		mainsInput = []OrderItem{{Name: strings.TrimSpace(req.MainMenu), Qty: 1}}
	}
	mainsMap := map[string]int{}
	mainsOrder := []string{}
	packagePortions := 0
	for _, it := range mainsInput {
		if it.Qty <= 0 {
			continue
		}
		if it.Qty > maxItemQty {
			fail(c, http.StatusBadRequest, fmt.Sprintf("Jumlah porsi maksimal %d per menu", maxItemQty))
			return
		}
		if !contains(cfg.MainMenus, it.Name) {
			fail(c, http.StatusBadRequest, "Menu paket tidak tersedia: "+it.Name)
			return
		}
		if _, seen := mainsMap[it.Name]; !seen {
			mainsOrder = append(mainsOrder, it.Name)
		}
		mainsMap[it.Name] += it.Qty
		packagePortions += it.Qty
	}
	mains := make([]OrderItem, 0, len(mainsOrder))
	for _, mName := range mainsOrder {
		mains = append(mains, OrderItem{Name: mName, Qty: mainsMap[mName]})
	}

	// 2. Normalize alacarte_menus (Menu Ala Carte / Pecahan Paket)
	alaMap := map[string]int{}
	alaOrder := []string{}
	alaTotalQty := 0
	for _, it := range req.AlacarteMenus {
		if it.Qty <= 0 {
			continue
		}
		if it.Qty > maxItemQty {
			fail(c, http.StatusBadRequest, fmt.Sprintf("Jumlah ala carte maksimal %d per item", maxItemQty))
			return
		}
		if !contains(cfg.AlacarteMenus, it.Name) {
			fail(c, http.StatusBadRequest, "Menu ala carte tidak tersedia: "+it.Name)
			return
		}
		if _, seen := alaMap[it.Name]; !seen {
			alaOrder = append(alaOrder, it.Name)
		}
		alaMap[it.Name] += it.Qty
		alaTotalQty += it.Qty
	}
	alacartes := make([]OrderItem, 0, len(alaOrder))
	for _, aName := range alaOrder {
		alacartes = append(alacartes, OrderItem{Name: aName, Qty: alaMap[aName]})
	}

	// Customer must select at least 1 Menu Paket OR 1 Menu Ala Carte
	if packagePortions <= 0 && alaTotalQty <= 0 {
		fail(c, http.StatusBadRequest, "Pilih minimal 1 Menu Paket atau Menu Ala Carte")
		return
	}

	// 3. Normalize side_menus (Add-ons)
	sidesParsed := parseOrderItems(string(req.SideMenusRaw))
	sidesMap := map[string]int{}
	sidesOrder := []string{}
	for _, it := range sidesParsed {
		if it.Qty <= 0 {
			continue
		}
		if it.Qty > maxItemQty {
			fail(c, http.StatusBadRequest, fmt.Sprintf("Jumlah add-on maksimal %d per item", maxItemQty))
			return
		}
		if !contains(cfg.SideMenus, it.Name) {
			fail(c, http.StatusBadRequest, "Add-on tidak tersedia: "+it.Name)
			return
		}
		if _, seen := sidesMap[it.Name]; !seen {
			sidesOrder = append(sidesOrder, it.Name)
		}
		sidesMap[it.Name] += it.Qty
	}
	sides := make([]OrderItem, 0, len(sidesOrder))
	for _, sName := range sidesOrder {
		sides = append(sides, OrderItem{Name: sName, Qty: sidesMap[sName]})
	}

	// 4. Normalize custom_items (only applies to Menu Paket portions)
	customsInput := req.CustomItems
	if len(customsInput) == 0 && len(req.CustomOptionsRaw) > 0 {
		customsInput = parseOrderItems(string(req.CustomOptionsRaw))
	}
	customsMap := map[string]int{}
	customsOrder := []string{}
	if packagePortions > 0 {
		for _, it := range customsInput {
			if it.Qty <= 0 {
				continue
			}
			if !contains(cfg.CustomOptions, it.Name) {
				fail(c, http.StatusBadRequest, "Opsi custom tidak tersedia: "+it.Name)
				return
			}
			if _, seen := customsMap[it.Name]; !seen {
				customsOrder = append(customsOrder, it.Name)
			}
			customsMap[it.Name] += it.Qty
			if customsMap[it.Name] > packagePortions {
				customsMap[it.Name] = packagePortions
			}
		}
	}
	customs := make([]OrderItem, 0, len(customsOrder))
	customLabels := make([]string, 0, len(customsOrder))
	for _, cName := range customsOrder {
		q := customsMap[cName]
		customs = append(customs, OrderItem{Name: cName, Qty: q})
		if q > 1 {
			customLabels = append(customLabels, fmt.Sprintf("%dx %s", q, shortMenuName(cName)))
		} else {
			customLabels = append(customLabels, shortMenuName(cName))
		}
	}

	// 5. Quota validation: Menu Paket portions + Lauk Ala Carte portions consume daily quota
	quotaNeeded := calcQuotaUsed(mains, alacartes)
	used, err := countUsedQuota()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal menghitung pesanan")
		return
	}
	remaining := cfg.GlobalQuota - used
	if remaining < 0 {
		remaining = 0
	}
	if quotaNeeded > 0 {
		if remaining <= 0 {
			fail(c, http.StatusConflict, "Maaf, kuota lauk/paket hari ini sudah habis")
			return
		}
		if quotaNeeded > remaining {
			fail(c, http.StatusConflict, fmt.Sprintf("Maaf, sisa kuota lauk/paket hanya %d porsi (kamu memilih %d porsi)", remaining, quotaNeeded))
			return
		}
	}

	totalPrice := calcOrderTotal(mains, alacartes, sides, customs)
	mainsJSON, _ := json.Marshal(mains)
	alacartesJSON, _ := json.Marshal(alacartes)
	sidesJSON, _ := json.Marshal(sides)
	customsJSON, _ := json.Marshal(customs)
	mainMenuSummary := formatItems(mains)
	ts := time.Now().In(loc).Format(timeLayout)

	res, err := db.Exec(`INSERT INTO Orders(timestamp, name, main_menu, main_menus_json, alacarte_menus_json, side_menus_json, custom_options_json, custom_note, total_price, payment_method, status)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ts, name, mainMenuSummary, string(mainsJSON), string(alacartesJSON), string(sidesJSON), string(customsJSON), note, totalPrice, pay, "baru")
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal menyimpan pesanan")
		return
	}
	id, _ := res.LastInsertId()
	newRemaining := remaining - quotaNeeded
	if newRemaining < 0 {
		newRemaining = 0
	}
	c.JSON(http.StatusCreated, gin.H{
		"ok":      true,
		"message": "Terima kasih, " + name + "!",
		"order": Order{
			ID:            id,
			Timestamp:     ts,
			Name:          name,
			MainMenus:     mains,
			MainMenu:      mainMenuSummary,
			AlacarteMenus: alacartes,
			SideMenus:     sides,
			CustomItems:   customs,
			CustomOptions: customLabels,
			CustomNote:    note,
			QuotaUsed:     quotaNeeded,
			TotalPrice:    totalPrice,
			PaymentMethod: pay,
			Status:        "baru",
		},
		"remaining": newRemaining,
	})
}

// GET /api/orders (admin)
func handleListOrders(c *gin.Context) {
	orders, err := listOrders()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca pesanan")
		return
	}
	c.JSON(http.StatusOK, orders)
}

// DELETE /api/orders (admin)
func handleResetOrders(c *gin.Context) {
	orderMu.Lock()
	defer orderMu.Unlock()
	tx, err := db.Begin()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal mereset data")
		return
	}
	if _, err := tx.Exec(`DELETE FROM Orders`); err != nil {
		_ = tx.Rollback()
		fail(c, http.StatusInternalServerError, "Gagal mereset data")
		return
	}
	_, _ = tx.Exec(`DELETE FROM sqlite_sequence WHERE name = 'Orders'`)
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, "Gagal mereset data")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/summary (admin)
func handleSummary(c *gin.Context) {
	cfg, err := loadConfig()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca konfigurasi")
		return
	}
	orders, err := listOrders()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca pesanan")
		return
	}
	c.JSON(http.StatusOK, buildSummary(cfg, orders))
}

// GET /api/export/excel (admin)
func handleExportExcel(c *gin.Context) {
	cfg, err := loadConfig()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca konfigurasi")
		return
	}
	orders, err := listOrders()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Gagal membaca pesanan")
		return
	}
	f, err := buildExcel(cfg, orders)
	if err != nil {
		log.Printf("excel: %v", err)
		fail(c, http.StatusInternalServerError, "Gagal membuat file Excel")
		return
	}
	defer f.Close()

	filename := "mbg-orders-" + time.Now().In(loc).Format("20060102-1504") + ".xlsx"
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if err := f.Write(c.Writer); err != nil {
		log.Printf("excel write: %v", err)
	}
}

func buildExcel(cfg AppConfig, orders []Order) (*excelize.File, error) {
	f := excelize.NewFile()
	const sOrders, sSummary = "Pesanan", "Ringkasan"
	if err := f.SetSheetName("Sheet1", sOrders); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(sSummary); err != nil {
		return nil, err
	}

	header, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Family: "Calibri", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"10B981"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borders("0E9F6E"),
	})
	cell, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    borders("E2E8F0"),
	})
	currency, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    borders("E2E8F0"),
		NumFmt:    3,
	})
	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14, Color: "006C49"}})
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Border: borders("E2E8F0")})
	section, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Color: "00422B"},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"D1FAE5"}, Pattern: 1},
		Border: borders("A7F3D0"),
	})

	// --- Sheet: Pesanan ---
	cols := []interface{}{"No", "Waktu", "Nama & Lantai", "Menu Paket (Qty)", "Menu Ala Carte (Qty)", "Slot Kuota", "Add-ons (Qty)", "Custom / Catatan", "Total Tagihan (Rp)", "Pembayaran", "Status"}
	_ = f.SetSheetRow(sOrders, "A1", &cols)
	_ = f.SetCellStyle(sOrders, "A1", "K1", header)
	_ = f.SetRowHeight(sOrders, 1, 22)
	for i := range orders {
		o := orders[len(orders)-1-i]
		row := i + 2
		mains := o.MainMenu
		if mains == "" {
			mains = "-"
		}
		alacartes := formatItems(o.AlacarteMenus)
		if alacartes == "" {
			alacartes = "-"
		}
		sides := formatItems(o.SideMenus)
		if sides == "" {
			sides = "-"
		}
		custom := formatCustomItems(o.CustomItems, o.CustomNote)
		if custom == "" {
			custom = "-"
		}
		vals := []interface{}{i + 1, o.Timestamp, o.Name, mains, alacartes, o.QuotaUsed, sides, custom, o.TotalPrice, paymentLabel(o.PaymentMethod), o.Status}
		ref, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow(sOrders, ref, &vals)
	}
	if n := len(orders); n > 0 {
		last, _ := excelize.CoordinatesToCellName(11, n+1)
		_ = f.SetCellStyle(sOrders, "A2", last, cell)
		priceEnd, _ := excelize.CoordinatesToCellName(9, n+1)
		_ = f.SetCellStyle(sOrders, "I2", priceEnd, currency)
		_ = f.AutoFilter(sOrders, "A1:"+last, nil)
	}
	for col, w := range map[string]float64{"A": 6, "B": 20, "C": 24, "D": 30, "E": 30, "F": 11, "G": 28, "H": 28, "I": 18, "J": 13, "K": 10} {
		_ = f.SetColWidth(sOrders, col, col, w)
	}
	_ = f.SetPanes(sOrders, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	// --- Sheet: Ringkasan ---
	sum := buildSummary(cfg, orders)
	r := 1
	put := func(a, b interface{}, style int) {
		ca, _ := excelize.CoordinatesToCellName(1, r)
		cb, _ := excelize.CoordinatesToCellName(2, r)
		_ = f.SetCellValue(sSummary, ca, a)
		if b != nil {
			_ = f.SetCellValue(sSummary, cb, b)
		}
		if style != 0 {
			_ = f.SetCellStyle(sSummary, ca, cb, style)
		}
		r++
	}
	put("Rekap Pesanan MBG Ordering System", nil, 0)
	_ = f.SetCellStyle(sSummary, "A1", "A1", title)
	put("Dibuat", sum.GeneratedAt, 0)
	r++
	put("Total Slot Kuota Terpakai (Paket + Lauk)", sum.Total, bold)
	put("  - Porsi Menu Paket", sum.PackageTotal, cell)
	put("  - Porsi Lauk Ala Carte", sum.LaukAlacarte, cell)
	put("Total Estimasi Omzet", formatIDR(sum.TotalRevenue), bold)
	put("Jumlah Pemesan", sum.OrderCount, cell)
	put("Pesanan Custom", sum.CustomOrderCnt, cell)
	put("Kuota Global", sum.Quota, cell)
	put("Sisa Slot Kuota", sum.Remaining, cell)
	r++
	block := func(label string, items []CountItem) {
		put(label, "Jumlah", section)
		for _, it := range items {
			put(it.Name, it.Count, cell)
		}
		r++
	}
	block("Menu Paket (Porsi)", sum.ByMainMenu)
	block("Menu Ala Carte (Qty)", sum.ByAlacarteMenu)
	block("Add-ons (Qty)", sum.BySideMenu)
	block("Kategori Custom (Porsi)", sum.ByCustomOption)
	block("Pembayaran (Transaksi)", sum.ByPayment)
	_ = f.SetColWidth(sSummary, "A", "A", 54)
	_ = f.SetColWidth(sSummary, "B", "B", 22)

	f.SetActiveSheet(0)
	return f, nil
}

func borders(color string) []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: color, Style: 1},
		{Type: "right", Color: color, Style: 1},
		{Type: "top", Color: color, Style: 1},
		{Type: "bottom", Color: color, Style: 1},
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	var err error
	loc, err = time.LoadLocation(env("TZ", "Asia/Jakarta"))
	if err != nil {
		log.Printf("WARNING: invalid TZ, using UTC: %v", err)
		loc = time.UTC
	}

	dbPath := env("DB_PATH", filepath.Join("data", "app.db"))
	db, err = openDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := seedConfig(); err != nil {
		log.Fatalf("seed: %v", err)
	}

	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Next()
	})

	staticDir := env("STATIC_DIR", ".")
	indexPath := filepath.Join(staticDir, "index.html")
	qrisPath := filepath.Join(staticDir, "qris.jpg")
	logoPath := filepath.Join(staticDir, "logo.jpg")
	serveIndex := func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File(indexPath)
	}
	serveLogo := func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=86400")
		c.File(logoPath)
	}
	r.GET("/", serveIndex)
	r.GET("/index.html", serveIndex)
	r.GET("/logo.jpg", serveLogo)
	r.GET("/favicon.ico", serveLogo)
	r.GET("/qris.jpg", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=3600")
		c.File(qrisPath)
	})
	r.GET("/healthz", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	api := r.Group("/api")
	api.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() })
	{
		api.GET("/config", handleGetConfig)
		api.POST("/login", handleLogin)
		api.POST("/order", handleCreateOrder)

		admin := api.Group("", adminAuth())
		admin.POST("/config", handleUpdateConfig)
		admin.GET("/orders", handleListOrders)
		admin.DELETE("/orders", handleResetOrders)
		admin.GET("/summary", handleSummary)
		admin.GET("/export/excel", handleExportExcel)
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			fail(c, http.StatusNotFound, "Endpoint tidak ditemukan")
			return
		}
		serveIndex(c)
	})

	addr := ":" + env("PORT", "8080")
	log.Printf("MBG Ordering System listening on %s (db: %s, tz: %s)", addr, dbPath, loc)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}
