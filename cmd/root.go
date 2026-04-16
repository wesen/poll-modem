package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/poll-modem/internal/modem"
	"github.com/go-go-golems/poll-modem/internal/tui"
)

var (
	url          string
	pollInterval time.Duration
	username     string
	password     string

	rootCmd = &cobra.Command{
		Use:   "poll-modem",
		Short: "Cable modem monitoring TUI",
		Long: `A Terminal User Interface (TUI) application that polls a cable modem's 
network setup page and displays the channel information in a nice table format.

The application continuously polls the modem endpoint and displays:
- Cable modem hardware information
- Downstream channel details (frequency, SNR, power levels, etc.)
- Upstream channel details
- Error codeword statistics

Use tab/shift+tab to navigate between different views.

If the modem requires authentication, provide username and password flags.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
		RunE: runTUI,
	}

	collectCmd = &cobra.Command{
		Use:   "collect",
		Short: "Headless collector mode",
		Long:  `Continuously polls the modem and stores data in SQLite. Designed to run as a container.`,
		RunE:  runCollect,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	err := clay.InitGlazed("poll-modem", rootCmd)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Viper")
	}

	// Add application-specific flags
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "http://192.168.0.1", "Modem base URL (e.g., http://192.168.0.1)")
	rootCmd.PersistentFlags().DurationVarP(&pollInterval, "interval", "i", 30*time.Second, "Poll interval (e.g., 30s, 1m, 5m)")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "n", "", "Modem username for authentication (or MODEM_USERNAME env)")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Modem password for authentication (or MODEM_PASSWORD env)")

	rootCmd.AddCommand(collectCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	log.Info().Msg("Running TUI")

	// Use the provided URL as base URL
	baseURL := url
	if baseURL == "" {
		baseURL = "http://192.168.0.1"
	}

	app := tui.NewApp(baseURL, pollInterval, username, password)
	defer app.Cleanup() // Ensure cleanup is called when function exits

	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()

	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func runCollect(cmd *cobra.Command, args []string) error {
	// Allow env vars as fallback for credentials
	if username == "" {
		username = os.Getenv("MODEM_USERNAME")
	}
	if password == "" {
		password = os.Getenv("MODEM_PASSWORD")
	}

	log.Info().Str("url", url).Dur("interval", pollInterval).Msg("Starting headless collector")

	client := modem.NewClient(url)
	client.SetCredentials(username, password)

	db, err := modem.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	sessionID, err := db.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	log.Info().Int64("session_id", sessionID).Msg("Session started")

	ctx := cmd.Context()

	// Poll loop
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Fetch immediately on start
	if err := collectOnce(ctx, client, db, sessionID); err != nil {
		log.Error().Err(err).Msg("Initial collection failed")
	}

	for range ticker.C {
		if err := collectOnce(ctx, client, db, sessionID); err != nil {
			log.Error().Err(err).Msg("Collection failed")
		}
	}

	return nil
}

func collectOnce(ctx context.Context, client *modem.Client, db *modem.Database, sessionID int64) error {
	log.Info().Msg("Polling modem...")

	info, err := client.LoginAndFetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch modem info: %w", err)
	}

	if err := db.StoreModemInfo(sessionID, info); err != nil {
		return fmt.Errorf("failed to store modem info: %w", err)
	}

	log.Info().
		Int("downstream", len(info.Downstream)).
		Int("upstream", len(info.Upstream)).
		Int("errors", len(info.ErrorCodewords)).
		Msg("Collected and stored")

	return nil
}
