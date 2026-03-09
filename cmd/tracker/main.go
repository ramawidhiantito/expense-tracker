package main

import (
	"context"
	"expense-tracking/internal/application/services"
	"expense-tracking/internal/core/ports"
	"expense-tracking/internal/infrastructure/db"
	"expense-tracking/internal/infrastructure/gmail"
	"expense-tracking/internal/infrastructure/parser"
	"expense-tracking/internal/infrastructure/sheets"
	"expense-tracking/internal/infrastructure/web"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	backfill := flag.Bool("backfill", false, "Run in manual backfill mode instead of cron mode")
	after := flag.String("after", "", "Start date for backfill in YYYY/MM/DD format (e.g., 2023/10/01)")
	before := flag.String("before", "", "End date for backfill in YYYY/MM/DD format (e.g., 2023/10/31)")
	flag.Parse()
	// Load configuration
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	pgConnStr := os.Getenv("DATABASE_URL")
	gmailCreds := os.Getenv("GMAIL_CREDENTIALS_JSON")
	gmailToken := os.Getenv("GMAIL_TOKEN_JSON")
	sheetsCreds := os.Getenv("SHEETS_CREDENTIALS_JSON")
	targetSheetID := os.Getenv("TARGET_SHEET_ID")
	targetSheetRange := os.Getenv("TARGET_SHEET_RANGE")
	targetSender := os.Getenv("TARGET_BANK_SENDER")
	if pgConnStr == "" || targetSheetID == "" || targetSender == "" {
		log.Fatal("Missing required environment variables (DATABASE_URL, TARGET_SHEET_ID, TARGET_BANK_SENDER)")
	}
	ctx := context.Background()
	// 1. Initialize DB State Manager
	stateManager, err := db.NewPostgresStateManager(ctx, pgConnStr)
	if err != nil {
		log.Fatalf("Failed to initialize state manager: %v", err)
	}
	defer stateManager.Close()
	if err := stateManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start state manager/migrations: %v", err)
	}
	// 2. Initialize Gmail Client
	gmailSrv, err := gmail.NewGmailService(ctx, gmailCreds, gmailToken)
	if err != nil {
		log.Fatalf("Failed to initialize Gmail client: %v", err)
	}
	emailProvider := gmail.NewGmailProvider(gmailSrv)
	// 3. Initialize Google Sheets Client
	sheetProvider, err := sheets.NewSheetsProvider(ctx, sheetsCreds)
	if err != nil {
		log.Fatalf("Failed to initialize Sheets client: %v", err)
	}
	// 4. Initialize Parsers
	var parsers []ports.Parser
	// Configuring specific parsers for banks
	parsers = append(parsers, parser.NewBCAParser())
	parsers = append(parsers, parser.NewBluParser())
	// 5. Initialize Sync Service
	syncService := services.NewSyncService(emailProvider, sheetProvider, stateManager, parsers, targetSheetID, targetSheetRange)

	// 5b. Initialize Web Server
	webPort := os.Getenv("WEB_PORT")
	if webPort == "" {
		webPort = ":8080"
	}
	webServer := web.NewServer(stateManager, webPort)
	syncService.OnNewExpense = webServer.BroadcastNewExpense
	
	go webServer.Start()

	// Build base email query to support multiple comma-separated senders
	senders := strings.Split(targetSender, ",")
	var senderQueries []string
	for _, s := range senders {
		senderQueries = append(senderQueries, fmt.Sprintf("from:%s", strings.TrimSpace(s)))
	}
	baseQuery := "(" + strings.Join(senderQueries, " OR ") + ")"
	if *backfill {
		log.Println("Starting expense tracking in BACKFILL mode...")
		query := baseQuery
		if *after != "" {
			query += fmt.Sprintf(" after:%s", strings.ReplaceAll(*after, "-", "/")) // Gmail expects YYYY/MM/DD format
		}
		if *before != "" {
			query += fmt.Sprintf(" before:%s", strings.ReplaceAll(*before, "-", "/"))
		}
		runSync(ctx, syncService, query, false)
		log.Println("Backfill complete. Web server is still running. Press Ctrl+C to exit.")
	} else {
		// 6. Run Application in Cron Mode
		log.Println("Starting expense tracking worker in CRON mode...")
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		// Run once immediately in a goroutine so it doesn't block the main thread's signal listener
		go func() {
			runSync(ctx, syncService, baseQuery, true)
			for range ticker.C {
				runSync(ctx, syncService, baseQuery, true)
			}
		}()
	}

	// Wait for interrupt signal to gracefully shut down the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down...")
}
func runSync(ctx context.Context, srv *services.SyncService, query string, unreadOnly bool) {
	log.Println("=== Starting Sync Cycle ===")
	count, err := srv.Run(ctx, query, unreadOnly)
	if err != nil {
		log.Printf("Error during sync cycle: %v\n", err)
		return
	}
	log.Printf("=== Sync Cycle Complete. Processed %d expenses. ===\n", count)
}
