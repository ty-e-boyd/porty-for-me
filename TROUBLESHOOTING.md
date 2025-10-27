# Troubleshooting Guide - Tyler Portfolio

## Common Issues and Solutions

### Issue: CreatedAt and UpdatedAt are NULL for new users

**Symptom:** When checking the database, `created_at` and `updated_at` fields are NULL for newly created users.

**Root Cause:** The User struct was missing the `CreatedAt` and `UpdatedAt` fields that GORM needs to auto-populate timestamps.

**Solution:** This has been fixed in the code. The User struct now includes:
```go
type User struct {
    ID               uint      `gorm:"primaryKey" json:"id"`
    Email            string    `gorm:"uniqueIndex;not null" json:"email"`
    Name             string    `json:"name"`
    Subscribed       bool      `gorm:"default:true" json:"subscribed"`
    UnsubscribeToken string    `gorm:"uniqueIndex;not null" json:"-"`
    CreatedAt        time.Time `json:"created_at"`  // ← Added
    UpdatedAt        time.Time `json:"updated_at"`  // ← Added
}
```

**How GORM Auto-Timestamps Work:**
- When you have fields named `CreatedAt` or `UpdatedAt` with type `time.Time`, GORM automatically:
  - Sets `CreatedAt` to current time on INSERT
  - Sets `UpdatedAt` to current time on INSERT and UPDATE
- No additional configuration needed!

**Verify the Fix:**
1. Rebuild the application: `go build`
2. Restart the server: `go run main.go`
3. Subscribe a new user
4. Check the database:
```sql
SELECT email, created_at, updated_at FROM users WHERE email = 'test@example.com';
```
5. Both timestamps should now be populated

**Fix Existing Records:**
If you have users with NULL timestamps, update them:

```sql
-- Set timestamps for all users with NULL values
UPDATE users 
SET 
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW())
WHERE created_at IS NULL OR updated_at IS NULL;
```

Or for a specific user:
```sql
UPDATE users 
SET 
    created_at = NOW(),
    updated_at = NOW()
WHERE email = 'user@example.com';
```

---

### Issue: Database connection fails

**Symptom:** Error message: "DB_CONNECT_STRING environment variable is required"

**Solutions:**

1. **Create .env file:**
```bash
cp .env.example .env
```

2. **Edit .env with your database credentials:**
```bash
DB_CONNECT_STRING=postgresql://username:password@localhost:5432/thepaper
```

3. **Verify PostgreSQL is running:**
```bash
psql -h localhost -U username -d thepaper -c "SELECT 1;"
```

4. **Test connection string format:**
```
postgresql://[user]:[password]@[host]:[port]/[database]?[options]
```

---

### Issue: User already exists error

**Symptom:** Cannot subscribe because email already exists in database

**Cause:** Email has unique constraint - user might have unsubscribed previously

**Solution:** The API handles this automatically:
- If user exists and is subscribed → "Already subscribed" message
- If user exists but unsubscribed → Automatically resubscribes them

**Manual check:**
```sql
SELECT email, subscribed, created_at FROM users WHERE email = 'user@example.com';
```

**Manual resubscribe:**
```sql
UPDATE users SET subscribed = true WHERE email = 'user@example.com';
```

---

### Issue: Unsubscribe token not working

**Symptom:** Clicking unsubscribe link shows "Invalid token" error

**Possible Causes:**

1. **Token doesn't match database:**
```sql
-- Verify token exists
SELECT email, unsubscribe_token FROM users WHERE unsubscribe_token = 'your_token_here';
```

2. **URL encoding issues:**
- Tokens are 64 characters of hex (0-9, a-f)
- Should not need URL encoding
- Check for accidental truncation in email client

3. **Database mismatch:**
- Ensure portfolio and thepaper use the SAME database
- Check both .env files have identical `DB_CONNECT_STRING`

**Get a user's token:**
```sql
SELECT unsubscribe_token FROM users WHERE email = 'user@example.com';
```

**Test unsubscribe manually:**
```bash
curl "http://localhost:4040/unsubscribe?token=YOUR_TOKEN_HERE"
```

---

### Issue: Database migrations not running

**Symptom:** Table 'users' doesn't exist

**Solution:**

1. **Auto-migration runs on startup:**
The application automatically creates tables when it starts. Just run:
```bash
go run main.go
```

2. **Verify migration:**
Check if table was created:
```sql
\dt users
```

3. **Manual table creation (if needed):**
```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    subscribed BOOLEAN DEFAULT true,
    unsubscribe_token VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

---

### Issue: Port 4040 already in use

**Symptom:** "bind: address already in use"

**Solutions:**

1. **Find and kill the process:**
```bash
# macOS/Linux
lsof -ti:4040 | xargs kill -9

# Or
ps aux | grep portfolio
kill -9 [PID]
```

2. **Change the port:**
Edit `main.go`:
```go
e.Logger.Fatal(e.Start(":8080"))  // Use different port
```

---

### Issue: Cannot connect to database from both apps

**Symptom:** Portfolio works but thepaper can't connect (or vice versa)

**Cause:** Different connection strings in .env files

**Solution:**

1. **Check both .env files:**
```bash
# In thepaper directory
grep DB_CONNECT_STRING .env

# In tyler-portfolio directory
grep DB_CONNECT_STRING .env
```

2. **Ensure they match EXACTLY:**
```bash
DB_CONNECT_STRING=postgresql://user:password@localhost:5432/thepaper
```

3. **Test connectivity from both locations:**
```bash
# From thepaper directory
go run main.go --dry-run

# From portfolio directory
go run main.go
```

---

### Issue: Emails sent but no unsubscribe link appears

**Symptom:** Newsletter emails don't have unsubscribe link in footer

**Causes and Solutions:**

1. **Missing PORTFOLIO_URL in thepaper/.env:**
```bash
# Add this to thepaper/.env
PORTFOLIO_URL=http://localhost:4040
```

2. **User has empty unsubscribe_token:**
```sql
-- Check for empty tokens
SELECT email, unsubscribe_token FROM users WHERE unsubscribe_token = '' OR unsubscribe_token IS NULL;

-- Generate new token (use thepaper's add_user.go script)
```

3. **Using old BuildHTML() instead of BuildHTMLWithToken():**
Check `main.go` in thepaper - should be:
```go
htmlContent := email.BuildHTMLWithToken(selectedArticles, len(articles), len(uniqueSources), user.UnsubscribeToken)
```

---

### Issue: Timestamps update on every query

**Symptom:** `updated_at` changes even when no data is modified

**Cause:** This is normal GORM behavior - `UpdatedAt` updates on any Save() operation

**To prevent unnecessary updates:**
- Use `db.Model(&user).Update()` for specific fields
- Don't call `Save()` unless data actually changed

**Example:**
```go
// This updates updated_at
db.Save(&user)

// This only updates if subscribed changes
db.Model(&User{}).Where("id = ?", user.ID).Update("subscribed", false)
```

---

### Debugging Tips

**Enable verbose SQL logging:**

In `main.go`, change the logger config:
```go
db, err = gorm.Open(postgres.Open(dbConnectString), &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info), // Shows all SQL queries
})
```

**Check all users:**
```sql
SELECT id, email, subscribed, created_at, updated_at, unsubscribe_token 
FROM users 
ORDER BY created_at DESC;
```

**Count subscribers:**
```sql
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE subscribed = true) as active,
    COUNT(*) FILTER (WHERE subscribed = false) as unsubscribed
FROM users;
```

**Recent subscriptions:**
```sql
SELECT email, created_at 
FROM users 
WHERE created_at > NOW() - INTERVAL '7 days'
ORDER BY created_at DESC;
```

---

## Still Having Issues?

1. Check application logs for error messages
2. Verify all environment variables are set
3. Test database connectivity with `psql`
4. Ensure both apps use the same database
5. Check PostgreSQL logs: `/var/log/postgresql/`
6. Review the main README.md and SUBSCRIPTION_SETUP.md

## Getting Help

When reporting an issue, include:
- Error message (full stack trace)
- Output of: `SELECT * FROM users LIMIT 1;`
- Contents of `.env` file (remove sensitive values)
- Go version: `go version`
- PostgreSQL version: `psql --version`
