package main

import (
	"bufio"
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	stripe "github.com/stripe/stripe-go/v86"
)

//go:embed templates/*.html
var tmplFS embed.FS

//go:embed static/*
var staticFS embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"money": money,
}).ParseFS(tmplFS, "templates/*.html"))

// baseCurrency is what the "converted" view rolls everything into.
var baseCurrency = "USD"

func main() {
	loadDotenv(".env")

	baseCurrency = strings.ToUpper(envOr("SCROOGE_BASE_CURRENCY", "USD"))
	addr := envOr("SCROOGE_ADDR", ":8080")
	key := os.Getenv("STRIPE_API_KEY")
	demo := key == "" || os.Getenv("SCROOGE_DEMO") == "1"

	var sc *stripe.Client
	if !demo {
		sc = stripe.NewClient(key)
	}

	http.Handle("/static/", http.FileServer(http.FS(staticFS)))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handleCalendar(w, r, sc, demo)
	})

	if demo {
		log.Printf("scrooge: DEMO MODE (no STRIPE_API_KEY), serving sample data on %s", addr)
	} else {
		log.Printf("scrooge: live Stripe data on %s", addr)
	}
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleCalendar(w http.ResponseWriter, r *http.Request, sc *stripe.Client, demo bool) {
	now := time.Now()

	// Which month are we looking at? ?m=YYYY-MM, default this month.
	anchor := now
	if m := r.URL.Query().Get("m"); m != "" {
		if t, err := time.ParseInLocation("2006-01", m, now.Location()); err == nil {
			anchor = t
		}
	}

	// Fetch a window generous enough to cover every grid cell.
	from := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -7)
	to := from.AddDate(0, 1, 0).AddDate(0, 0, 21)

	var charges []Charge
	if demo {
		charges = demoCharges(anchor)
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		var err error
		charges, err = FetchCharges(ctx, sc, from, to)
		if err != nil {
			log.Printf("stripe fetch failed: %v", err)
			http.Error(w, "Could not reach Stripe. Check STRIPE_API_KEY and try again.", http.StatusBadGateway)
			return
		}
	}

	// Currency view: "split" (per currency, default) or "convert" (one base currency).
	convert := r.URL.Query().Get("view") == "convert"
	fxErr := false
	if convert {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		rates, err := fxRates(ctx, baseCurrency)
		cancel()
		if err != nil {
			log.Printf("fx rates unavailable: %v", err)
			convert, fxErr = false, true // fall back to per-currency view
		} else {
			charges = convertCharges(charges, baseCurrency, rates)
		}
	}

	month := BuildMonth(anchor, now, charges)

	data := struct {
		Month
		Demo    bool
		Convert bool
		Base    string
		FXErr   bool
		Now     time.Time
	}{month, demo, convert, baseCurrency, fxErr, now}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("render failed: %v", err)
	}
}

// loadDotenv reads KEY=VALUE lines from path into the environment. Missing file
// is fine. Real environment variables take precedence over the file.
func loadDotenv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, v)
		}
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
