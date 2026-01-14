## Admin Commands

Administrative commands for managing Nylas platform resources at an organizational level. These commands require API key authentication.

### Applications

Manage Nylas applications in your organization.

```bash
# List applications
nylas admin applications list
nylas admin apps list              # Alias
nylas admin app list --json        # Output as JSON

# Show application details
nylas admin applications show <app-id>
nylas admin app show <app-id> --json

# Create application
nylas admin applications create --name "My App" --region us
nylas admin app create --name "My App" --region eu \
  --branding-name "MyApp" \
  --website-url "https://myapp.com" \
  --callback-uris "https://myapp.com/oauth/callback,https://myapp.com/oauth/redirect"

# Update application
nylas admin applications update <app-id> --name "Updated Name"
nylas admin app update <app-id> --branding-name "NewBrand" --website-url "https://new.com"

# Delete application
nylas admin applications delete <app-id>
nylas admin app delete <app-id> --yes  # Skip confirmation
```

**Example: List applications**
```bash
$ nylas admin applications list

Found 2 application(s):

APP ID              REGION    ENVIRONMENT
myapp-prod-123      us        production
myapp-dev-456       us        development
```

**Example: Show application details**
```bash
$ nylas admin applications show myapp-prod-123

Application Details
  ID: app_abc123
  Application ID: myapp-prod-123
  Organization ID: org_xyz789
  Region: us
  Environment: production

Branding:
  Name: MyApp
  Website: https://myapp.com

Callback URIs (2):
  1. https://myapp.com/oauth/callback
  2. https://myapp.com/oauth/redirect
```

### Connectors

Manage email provider connectors (Google, Microsoft, IMAP, etc.).

```bash
# List connectors
nylas admin connectors list
nylas admin conn list              # Alias
nylas admin connectors list --json

# Show connector details
nylas admin connectors show <connector-id>
nylas admin conn show <connector-id> --json

# Create OAuth connector (Google/Microsoft)
nylas admin connectors create --name "Gmail" --provider google \
  --client-id "xxx.apps.googleusercontent.com" \
  --scopes "https://www.googleapis.com/auth/gmail.readonly,https://www.googleapis.com/auth/calendar"

# Create IMAP connector
nylas admin connectors create --name "Custom IMAP" --provider imap \
  --imap-host "imap.example.com" --imap-port 993 \
  --smtp-host "smtp.example.com" --smtp-port 587

# Update connector
nylas admin connectors update <connector-id> --name "Updated Name"
nylas admin conn update <connector-id> --scopes "new,scopes,list"

# Delete connector
nylas admin connectors delete <connector-id>
nylas admin conn delete <connector-id> --yes
```

**Example: List connectors**
```bash
$ nylas admin connectors list

Found 3 connector(s):

NAME                ID                    PROVIDER      SCOPES
Gmail               conn_google_123       google        3
Microsoft 365       conn_ms365_456        microsoft     4
Custom IMAP         conn_imap_789         imap          0
```

**Example: Show connector details**
```bash
$ nylas admin connectors show conn_google_123

Connector: Gmail
  ID: conn_google_123
  Provider: google

Scopes (3):
  1. https://www.googleapis.com/auth/gmail.readonly
  2. https://www.googleapis.com/auth/calendar
  3. https://www.googleapis.com/auth/contacts.readonly

Settings:
  Client ID: 123456789.apps.googleusercontent.com
```

### Credentials

Manage authentication credentials for connectors.

```bash
# List credentials
nylas admin credentials list
nylas admin creds list             # Alias
nylas admin credentials list --json

# Show credential details
nylas admin credentials show <credential-id>
nylas admin cred show <credential-id> --json

# Create credential
nylas admin credentials create --connector-id <connector-id> \
  --name "Production Credentials" \
  --credential-type oauth

# Create credential with data
nylas admin cred create --connector-id <connector-id> \
  --name "Service Account" \
  --credential-type service_account \
  --credential-data '{"private_key":"..."}'

# Update credential
nylas admin credentials update <credential-id> --name "Updated Name"

# Delete credential
nylas admin credentials delete <credential-id>
nylas admin cred delete <credential-id> --yes
```

**Example: List credentials**
```bash
$ nylas admin credentials list

Found 2 credential(s):

NAME                    ID                    CONNECTOR          TYPE
Production OAuth        cred_oauth_123        conn_google_123    oauth
Service Account         cred_sa_456           conn_google_123    service_account
```

**Example: Show credential details**
```bash
$ nylas admin credentials show cred_oauth_123

Credential: Production OAuth
  ID: cred_oauth_123
  Connector ID: conn_google_123
  Name: Production OAuth
  Type: oauth

Created: Dec 1, 2024 10:00 AM
Updated: Dec 15, 2024 2:30 PM
```

### Grants

View and manage grants across all applications.

```bash
# List grants
nylas admin grants list
nylas admin grant list             # Alias
nylas admin grants list --limit 100 --offset 0
nylas admin grants list --connector-id <connector-id>
nylas admin grants list --status valid
nylas admin grants list --status invalid
nylas admin grants list --json

# Show grant statistics
nylas admin grants stats
nylas admin grants stats --json
```

**Example: List grants**
```bash
$ nylas admin grants list --limit 5

Found 5 grant(s):

EMAIL                   ID                    PROVIDER    STATUS
user@gmail.com          grant_abc123          google      valid
work@company.com        grant_def456          microsoft   valid
john@example.com        grant_ghi789          google      invalid
-                       grant_jkl012          imap        valid
alice@startup.io        grant_mno345          google      valid
```

**Example: Grant statistics**
```bash
$ nylas admin grants stats

Grant Statistics
  Total Grants: 150
  Valid: 142
  Invalid: 8

By Provider:
PROVIDER          COUNT
google            95
microsoft         42
imap              13

By Status:
STATUS            COUNT
valid             142
invalid           8
```

**Filter options:**
- `--offset` - Offset for pagination (default: 0)
- `--connector-id` - Filter by connector ID
- `--status` - Filter by status (valid, invalid)

**Common flags:** `--limit N` (default: 50), `--json` (see [Global Flags](#global-flags))

---

