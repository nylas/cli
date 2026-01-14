## Contacts Management

Manage contacts and contact groups.

### List Contacts

```bash
nylas contacts list [grant-id]
nylas contacts list --limit 100
nylas contacts list --id                      # Show contact IDs
nylas contacts list --email "john@example.com"
nylas contacts list --source address_book
```

**Example output:**
```bash
$ nylas contacts list --limit 5

Found 5 contact(s):

NAME                EMAIL                      PHONE            COMPANY
Alice Johnson       alice@company.com          +1-555-0101      Acme Corp - Engineer
Bob Wilson          bob@example.com            +1-555-0102
Carol Davis         carol@startup.io           +1-555-0103      Startup Inc - CEO
David Brown         david@email.com                             Freelancer
Eve Martinez        eve@company.com            +1-555-0105      Acme Corp - Designer
```

### Show Contact

```bash
nylas contacts show <contact-id> [grant-id]
nylas contacts get <contact-id>  # Alias
```

**Example output:**
```bash
$ nylas contacts show contact_abc123

Alice Johnson

Work
  Job Title: Software Engineer
  Company: Acme Corporation
  Manager: John Smith

Email Addresses
  alice@company.com (work)
  alice.personal@gmail.com (personal)

Phone Numbers
  +1-555-0101 (mobile)
  +1-555-0102 (work)

Addresses
  (work)
    123 Main Street
    San Francisco, CA 94102
    United States

Web Pages
  https://linkedin.com/in/alice (linkedin)
  https://github.com/alice (profile)

Personal
  Nickname: Ali
  Birthday: 1990-05-15

Notes
  Met at the tech conference in 2023.

Details
  ID: contact_abc123
  Source: address_book
```

### Create Contact

```bash
nylas contacts create [grant-id]
nylas contacts create --first-name "John" --last-name "Doe" --email "john@example.com"
nylas contacts create --first-name "Jane" --last-name "Smith" \
  --email "jane@company.com" --phone "+1-555-123-4567" \
  --company "Acme Corp" --job-title "Engineer"
```

**Example output:**
```bash
$ nylas contacts create --first-name "John" --last-name "Doe" --email "john@example.com"

✓ Contact created successfully!

Name: John Doe
Email: john@example.com
ID: contact_new_123
```

### Update Contact

```bash
nylas contacts update <contact-id> [grant-id]
nylas contacts update <contact-id> --given-name "John" --surname "Smith"
nylas contacts update <contact-id> --company "Acme Inc" --job-title "Engineer"
nylas contacts update <contact-id> --email "new@example.com" --phone "+1-555-0123"
nylas contacts update <contact-id> --birthday "1990-05-15" --notes "Updated notes"
```

**Example output:**
```bash
$ nylas contacts update contact_abc123 --given-name "John" --surname "Smith"

✓ Contact updated successfully!

Name: John Smith
ID: contact_abc123
```

### Delete Contact

```bash
nylas contacts delete <contact-id> [grant-id]
nylas contacts delete <contact-id> --force   # Skip confirmation
```

### Contact Groups

Manage contact groups with full CRUD operations.

```bash
# List groups
nylas contacts groups list [grant-id]

# Show group details
nylas contacts groups show <group-id> [grant-id]

# Create group
nylas contacts groups create "VIP Clients" [grant-id]

# Update group
nylas contacts groups update <group-id> --name "Premium Clients"

# Delete group
nylas contacts groups delete <group-id> [grant-id]
nylas contacts groups delete <group-id> --force   # Skip confirmation
```

**Example output:**
```bash
$ nylas contacts groups list

Found 4 contact group(s):

NAME                ID                    PATH
Family              group_abc123          /Family
Work                group_def456          /Work
Friends             group_ghi789          /Friends
VIP                 group_jkl012          /VIP
```

**Example: Create a contact group**
```bash
$ nylas contacts groups create "VIP Clients"

✓ Contact group created successfully!

Name: VIP Clients
ID: group_new_123
```

### Advanced Contact Search

Search contacts with advanced filtering options including company name, email, phone, and more.

```bash
# Basic search
nylas contacts search [grant-id]

# Search by company name (partial match, case-insensitive)
nylas contacts search --company "Acme"

# Search by email address
nylas contacts search --email "john@example.com"

# Search by phone number
nylas contacts search --phone "+1-555-0101"

# Filter by contact source
nylas contacts search --source address_book
nylas contacts search --source inbox
nylas contacts search --source domain

# Only show contacts with email addresses
nylas contacts search --has-email

# Combine multiple filters
nylas contacts search --company "Corp" --has-email --limit 20

# Output as JSON
nylas contacts search --json
```

**Example output:**
```bash
$ nylas contacts search --company "Acme" --has-email

ID              Name              Email                   Company         Job Title
---             ----              -----                   -------         ---------
contact_001     Alice Johnson     alice@company.com       Acme Corp       Engineer
contact_002     Eve Martinez      eve@company.com         Acme Corp       Designer

Found 2 contacts
```

**Available filters:**
- `--company` - Filter by company name (partial match)
- `--email` - Filter by email address
- `--phone` - Filter by phone number
- `--source` - Filter by source (address_book, inbox, domain)
- `--group` - Filter by contact group ID
- `--has-email` - Only show contacts with email addresses

**Common flags:** `--limit N` (default: 50), `--json` (see [Global Flags](#global-flags))

### Profile Picture Management

Download and manage contact profile pictures.

#### Download Profile Picture

```bash
# Get Base64-encoded profile picture data
nylas contacts photo download <contact-id> [grant-id]

# Save profile picture to file (automatically decodes Base64)
nylas contacts photo download <contact-id> --output photo.jpg

# Get as JSON
nylas contacts photo download <contact-id> --json
```

**Example output:**
```bash
$ nylas contacts photo download contact_abc123 --output alice.jpg

Profile picture saved to: alice.jpg
Size: 15234 bytes
```

**Example output (Base64):**
```bash
$ nylas contacts photo download contact_abc123

Base64-encoded profile picture:
iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==

To save to a file, use the --output flag
```

#### Profile Picture Information

```bash
# View information about how profile pictures work in Nylas API v3
nylas contacts photo info
```

**Key points:**
- Profile pictures are retrieved using `?profile_picture=true` query parameter
- API returns Base64-encoded image data
- Images come directly from email provider (Gmail, Outlook, etc.)
- **Upload not supported** - pictures must be managed through provider
- Not all contacts have profile pictures
- Cache pictures locally if using frequently

### Contact Synchronization Info

View information about how contact synchronization works in Nylas API v3.

```bash
# View sync architecture and best practices
nylas contacts sync
```

**Key changes in v3:**
- **No more traditional sync model** - v3 eliminated local data storage
- **Direct provider access** - Requests forwarded to email providers
- **Provider-native IDs** - Contact IDs come from provider
- **Real-time data** - No stale cached data
- **No sync delays** - Instant access to new contacts

**Provider-specific behavior:**
- **Google/Gmail**: Real-time via Google Contacts API (5 min polling)
- **Microsoft/Outlook**: Real-time via Microsoft Graph
- **IMAP**: Depends on provider support
- **Virtual calendars**: Nylas-managed (no provider sync)

**Webhook events for change notifications:**
- `contact.created` - New contact added
- `contact.updated` - Contact modified
- `contact.deleted` - Contact removed

---

