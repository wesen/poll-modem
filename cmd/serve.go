package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-go-golems/poll-modem/internal/modem"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type serveSnapshot struct {
	LastSuccess time.Time        `json:"last_success"`
	LastError   string           `json:"last_error,omitempty"`
	Current     *modem.ModemInfo `json:"current,omitempty"`
}

type modemCollector struct {
	client    *modem.Client
	db        *modem.Database
	sessionID int64

	mu       sync.RWMutex
	snapshot serveSnapshot
}

func newModemCollector(baseURL, username, password string) (*modemCollector, error) {
	client := modem.NewClient(baseURL)
	client.SetCredentials(username, password)

	db, err := modem.NewDatabase()
	if err != nil {
		return nil, err
	}

	sessionID, err := db.StartSession()
	if err != nil {
		db.Close()
		return nil, err
	}

	return &modemCollector{client: client, db: db, sessionID: sessionID}, nil
}

func (c *modemCollector) Close() error {
	if c.db != nil {
		if err := c.db.EndSession(c.sessionID); err != nil {
			log.Error().Err(err).Msg("failed to end session")
		}
		return c.db.Close()
	}
	return nil
}

func (c *modemCollector) Poll(ctx context.Context) error {
	start := time.Now()
	info, err := c.client.LoginAndFetch(ctx)
	if err != nil {
		pollMetrics.observeFailure(time.Since(start))
		c.mu.Lock()
		c.snapshot.LastError = err.Error()
		c.mu.Unlock()
		return err
	}

	if err := c.db.StoreModemInfo(c.sessionID, info); err != nil {
		pollMetrics.observeFailure(time.Since(start))
		c.mu.Lock()
		c.snapshot.LastError = err.Error()
		c.mu.Unlock()
		return err
	}

	pollMetrics.observeSuccess(time.Since(start), info)

	c.mu.Lock()
	c.snapshot.LastSuccess = time.Now()
	c.snapshot.LastError = ""
	c.snapshot.Current = info
	c.mu.Unlock()

	return nil
}

func (c *modemCollector) Snapshot() serveSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshot
}

var listenAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run collector and HTTP dashboard",
	Long:  `Polls the modem continuously, stores results in SQLite, and serves a small HTTP dashboard.`,
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&listenAddr, "listen", ":8080", "HTTP listen address")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	if username == "" {
		username = os.Getenv("MODEM_USERNAME")
	}
	if password == "" {
		password = os.Getenv("MODEM_PASSWORD")
	}

	baseURL := url
	if baseURL == "" {
		baseURL = "http://192.168.0.1"
	}

	collector, err := newModemCollector(baseURL, username, password)
	if err != nil {
		return fmt.Errorf("failed to initialize collector: %w", err)
	}
	defer collector.Close()

	ctx := cmd.Context()
	go func() {
		if err := collector.Poll(ctx); err != nil {
			log.Error().Err(err).Msg("initial poll failed")
		}

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := collector.Poll(ctx); err != nil {
					log.Error().Err(err).Msg("poll failed")
				}
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(collector.Snapshot())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snap := collector.Snapshot()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = dashboardTemplate.Execute(w, snap)
	})

	srv := &http.Server{Addr: listenAddr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	log.Info().Str("listen", listenAddr).Msg("serving modem dashboard")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>poll-modem</title>
  <style>
    body { font-family: sans-serif; margin: 2rem; background: #111; color: #eee; }
    .muted { color: #aaa; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 1.5rem; }
    th, td { border-bottom: 1px solid #333; padding: 0.4rem 0.6rem; text-align: left; }
    h1, h2 { color: #fff; }
    .error { color: #f88; }
    .ok { color: #8f8; }
  </style>
</head>
<body>
  <h1>poll-modem</h1>
  <p class="muted">Last success: {{if .LastSuccess.IsZero}}never{{else}}{{.LastSuccess}}{{end}}</p>
  {{if .LastError}}<p class="error">Last error: {{.LastError}}</p>{{else}}<p class="ok">Collector healthy</p>{{end}}
  {{if .Current}}
    <h2>Cable modem</h2>
    <table>
      <tr><th>Model</th><td>{{.Current.CableModem.Model}}</td></tr>
      <tr><th>Vendor</th><td>{{.Current.CableModem.Vendor}}</td></tr>
      <tr><th>Boot</th><td>{{.Current.CableModem.BOOTVersion}}</td></tr>
      <tr><th>Core</th><td>{{.Current.CableModem.CoreVersion}}</td></tr>
      <tr><th>Product</th><td>{{.Current.CableModem.ProductType}}</td></tr>
    </table>

    <h2>Downstream</h2>
    <table>
      <tr><th>ID</th><th>Status</th><th>Freq</th><th>SNR</th><th>Power</th><th>Modulation</th></tr>
      {{range .Current.Downstream}}<tr><td>{{.ChannelID}}</td><td>{{.LockStatus}}</td><td>{{.Frequency}}</td><td>{{.SNR}}</td><td>{{.PowerLevel}}</td><td>{{.Modulation}}</td></tr>{{end}}
    </table>

    <h2>Upstream</h2>
    <table>
      <tr><th>ID</th><th>Status</th><th>Freq</th><th>Symbol</th><th>Power</th><th>Modulation</th><th>Type</th></tr>
      {{range .Current.Upstream}}<tr><td>{{.ChannelID}}</td><td>{{.LockStatus}}</td><td>{{.Frequency}}</td><td>{{.SymbolRate}}</td><td>{{.PowerLevel}}</td><td>{{.Modulation}}</td><td>{{.ChannelType}}</td></tr>{{end}}
    </table>

    <h2>Error codewords</h2>
    <table>
      <tr><th>ID</th><th>Unerrored</th><th>Correctable</th><th>Uncorrectable</th></tr>
      {{range .Current.ErrorCodewords}}<tr><td>{{.ChannelID}}</td><td>{{.UnerroredCodewords}}</td><td>{{.CorrectableCodewords}}</td><td>{{.UncorrectableCodewords}}</td></tr>{{end}}
    </table>
  {{else}}
    <p>No data collected yet.</p>
  {{end}}
</body>
</html>`))
