#  Expense Tracker

An automated expense tracking system built with Go. It monitors bank notification emails (supported banks right now: BCA, Blu), parses transaction details, and synchronizes them to a Google Sheet for easy management.

##  Features

- **Automated Sync**: Periodically fetches new emails from Gmail.
- **Multiple Bank Support**: 
    - **BCA**: Parses transaction notifications from BCA.
    - **Blu**: Parses digital banking notifications from Blu by BCA Digital.
- **Google Sheets Integration**: Automatically appends parsed transactions to a designated spreadsheet.
- **State Management**: Uses PostgreSQL to track processed emails and avoid duplicates.
- **Dual Modes**: 
    - **Cron Mode**: Runs every 15 minutes to process new transactions.
    - **Backfill Mode**: Manually process historical emails within a specific date range.

##  Project Structure

The project follows Domain-Driven Design (DDD) principles:

```text
├── cmd/
│   └── tracker/          # Application entry point
├── internal/
│   ├── application/      # Use cases and services (Sync Service)
│   ├── core/             # Domain entities and port definitions
│   └── infrastructure/   # Adapters (Gmail, Sheets, Parsers, DB)
└── go.mod                # Go dependencies
```

##  Configuration

Create a `.env` file in the root directory using the `.env example` as a template:

```env
DATABASE_URL=postgres://user:password@localhost:5432/expense-tracker
GMAIL_CREDENTIALS_JSON=credential_oauth.json
GMAIL_TOKEN_JSON=token.json
SHEETS_CREDENTIALS_JSON=service-account.json
TARGET_SHEET_ID=your_spreadsheet_id
TARGET_SHEET_RANGE=Sheet1!A:E
TARGET_BANK_SENDER=bca@bca.co.id,receipts@blubybcadigital.id
```

### Required Credentials
1. **Gmail API**: Place your OAuth2 credentials in `credential_oauth.json` and generate `token.json`.
2. **Google Sheets API**: Place your Service Account credentials in `service-account.json`.

##  Getting Started

### Prerequisites
- Go 1.25+
- PostgreSQL
- Google Cloud Project with Gmail and Sheets APIs enabled.

### Run in Cron Mode
Processes unread emails every 15 minutes.
```bash
go run ./cmd/tracker/main.go
```

### Run in Backfill Mode
Processes emails within a specific date range.
```bash
go run ./cmd/tracker/main.go -backfill -after "2023/10/01" -before "2023/10/31"
```

##  Testing

The project includes unit tests for parsers and services.
```bash
go test ./...
```
