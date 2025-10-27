# Tyler Boyd's Portfolio

A simple portfolio website built with Go and Echo framework, featuring newsletter subscription functionality for "The Paper" daily tech digest.

## Features

- 📄 **Personal Portfolio**: Simple, clean portfolio page
- 📧 **Newsletter Subscription**: Subscribe to "The Paper" daily digest
- 🔓 **Easy Unsubscribe**: One-click unsubscribe via email links
- 🗄️ **Database Integration**: PostgreSQL for subscriber management
- 🎨 **Minimal Design**: Dark theme with JetBrains Mono font

## Setup

### Prerequisites

- Go 1.22+
- PostgreSQL database (shared with thepaper project)
- Environment variables configured

### Installation

1. **Clone and navigate to the project:**
```bash
cd go_projects/tyler-portfolio
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Configure environment variables:**

Create a `.env` file based on `.env.example`:

```bash
# Database connection (same database as thepaper)
DB_CONNECT_STRING=postgresql://user:password@localhost:5432/thepaper?sslmode=disable

# Portfolio URL (used in unsubscribe links in emails)
PORTFOLIO_URL=http://localhost:4040
```

**Important:** 
- Use the same `DB_CONNECT_STRING` as your thepaper project
- In production, set `PORTFOLIO_URL` to your actual domain (e.g., `https://tyboyd.dev`)

4. **Run the application:**
```bash
go run main.go
```

The server will start on `http://localhost:4040`

## Routes

### Pages

- `GET /` - Homepage with portfolio information
- `GET /subscribe` - Newsletter subscription page
- `GET /unsubscribe?token=<token>` - Unsubscribe confirmation (called from email links)

### API Endpoints

- `POST /api/subscribe` - Subscribe to newsletter
  - Request body: `{"email": "user@example.com"}`
  - Returns: Success/error message
  
- `POST /api/unsubscribe` - Unsubscribe from newsletter
  - Request body: `{"token": "unsubscribe_token"}`
  - Returns: Success/error message

## How It Works

### Subscription Flow

1. User visits `/subscribe` page
2. Enters their email address
3. Form submits to `/api/subscribe` endpoint
4. System checks if email already exists:
   - If new: Creates user with unique unsubscribe token
   - If exists and subscribed: Returns "already subscribed" message
   - If exists but unsubscribed: Reactivates subscription
5. User receives confirmation message

### Unsubscribe Flow

1. User clicks unsubscribe link in email newsletter
2. Link format: `https://yourdomain.com/unsubscribe?token=<unique_token>`
3. System validates token and unsubscribes user
4. Displays unsubscribe confirmation page with option to resubscribe

## Integration with thepaper

This portfolio serves as the subscriber management interface for the [thepaper](../../thepaper) newsletter system.

**Key integration points:**

1. **Shared Database**: Both projects use the same PostgreSQL database and `users` table
2. **Unsubscribe Links**: thepaper emails include unsubscribe links pointing to this portfolio
3. **Token Management**: Unique unsubscribe tokens are generated during subscription

**Environment variable setup in thepaper:**

In your thepaper project's `.env` file, add:
```bash
PORTFOLIO_URL=http://localhost:4040  # or your production URL
```

This tells thepaper where to point unsubscribe links.

## Database Schema

The application uses the `users` table with the following structure:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    subscribed BOOLEAN DEFAULT true,
    unsubscribe_token VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

**Note:** Only `email` is required for subscription. The `name` field is optional.

## Project Structure

```
tyler-portfolio/
├── main.go                 # Main application with routes and handlers
├── template/
│   └── template.go         # Template rendering setup
├── public/
│   ├── index.html          # Homepage template
│   ├── subscribe.html      # Subscription page template
│   └── unsubscribe.html    # Unsubscribe confirmation template
├── go.mod                  # Go dependencies
├── go.sum                  # Dependency checksums
├── .env.example            # Example environment variables
└── README.md               # This file
```

## API Response Format

All API endpoints return JSON responses with the following structure:

```json
{
  "success": true,
  "message": "Success message here"
}
```

Or on error:

```json
{
  "success": false,
  "error": "Error message here"
}
```

## Development

### Running Locally

```bash
go run main.go
```

### Building for Production

```bash
go build -o portfolio
./portfolio
```

### Testing the Subscribe Flow

1. Start the server
2. Visit `http://localhost:4040/subscribe`
3. Enter an email address
4. Check the database to verify the user was created:

```sql
SELECT email, subscribed, created_at FROM users WHERE email = 'test@example.com';
```

### Testing the Unsubscribe Flow

1. Get a user's unsubscribe token from the database:
```sql
SELECT unsubscribe_token FROM users WHERE email = 'test@example.com';
```

2. Visit: `http://localhost:4040/unsubscribe?token=<token>`
3. Verify the user was unsubscribed:
```sql
SELECT email, subscribed FROM users WHERE email = 'test@example.com';
```

## Security Considerations

- Unsubscribe tokens are 64-character cryptographically secure random strings
- Database connections use SSL in production (configure in connection string)
- Email validation on the frontend and backend
- CORS middleware enabled for API endpoints
- No authentication required for subscription (by design)

## Deployment

When deploying to production:

1. Update `PORTFOLIO_URL` in both projects' `.env` files to your production domain
2. Ensure PostgreSQL is accessible from both applications
3. Use SSL for database connections (`sslmode=require` in connection string)
4. Consider rate limiting for API endpoints
5. Set up proper logging and monitoring

## License

MIT