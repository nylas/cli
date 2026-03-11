package provider

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"golang.org/x/term"
)

// lineReader abstracts reading a line of text (for testing).
type lineReader interface {
	ReadString(delim byte) (string, error)
}

func trimInput(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// lookPathFunc allows overriding exec.LookPath for testing.
var lookPathFunc = exec.LookPath

// runGcloudLoginFunc allows overriding the gcloud login command for testing.
var runGcloudLoginFunc = runGcloudLogin

func runGcloudLogin(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "application-default", "login",
		"--scopes=https://www.googleapis.com/auth/cloud-platform")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// checkPrerequisites verifies gcloud CLI and ADC are available.
func checkPrerequisites(ctx context.Context, gcpClient ports.GCPClient) (string, error) {
	// Check gcloud CLI
	if _, err := lookPathFunc("gcloud"); err != nil {
		return "", common.NewUserError(
			"gcloud CLI not found",
			"Install from: https://cloud.google.com/sdk/docs/install",
		)
	}
	common.PrintSuccess("gcloud CLI found")

	// Check ADC authentication
	email, err := gcpClient.CheckAuth(ctx)
	if err != nil {
		fmt.Println("  Application Default Credentials not configured. Logging in...")
		if loginErr := runGcloudLoginFunc(ctx); loginErr != nil {
			return "", fmt.Errorf("gcloud auth failed: %w", loginErr)
		}
		// Retry auth check
		email, err = gcpClient.CheckAuth(ctx)
		if err != nil {
			return "", fmt.Errorf("authentication still failing after login: %w", err)
		}
	}
	common.PrintSuccess("Authenticated as %s", email)
	return email, nil
}

// promptProjectSelection lets the user pick an existing project or create a new one.
func promptProjectSelection(ctx context.Context, gcpClient ports.GCPClient, reader lineReader, flagProjectID string) (projectID, displayName string, isNew bool, err error) {
	if flagProjectID != "" {
		return flagProjectID, "", false, nil
	}

	projects, err := gcpClient.ListProjects(ctx)
	if err != nil {
		return "", "", false, fmt.Errorf("failed to list projects: %w", err)
	}

	fmt.Println("\n  Your GCP Projects:")
	for i, p := range projects {
		fmt.Printf("  [%d] %s (%s)\n", i+1, p.ProjectID, p.DisplayName)
	}
	createIdx := len(projects) + 1
	fmt.Printf("  [%d] Create a new project\n", createIdx)
	fmt.Printf("\n  Select a project (1-%d): ", createIdx)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}

	var selected int
	if _, err := fmt.Sscanf(strings.TrimSpace(input), "%d", &selected); err != nil || selected < 1 || selected > createIdx {
		return "", "", false, common.NewInputError(fmt.Sprintf("invalid selection: %s", strings.TrimSpace(input)))
	}

	if selected == createIdx {
		return promptNewProject(reader)
	}

	p := projects[selected-1]
	return p.ProjectID, p.DisplayName, false, nil
}

func promptNewProject(reader lineReader) (projectID, displayName string, isNew bool, err error) {
	fmt.Print("  Enter project name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", false, common.NewInputError("project name cannot be empty")
	}

	suggested := generateProjectID(name)
	fmt.Printf("  Generated project ID: %s (ok? Y/n/custom): ", suggested)
	choice, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}

	choice = trimInput(choice)
	switch choice {
	case "", "y", "yes":
		return suggested, name, true, nil
	case "n", "no", "custom":
		fmt.Print("  Enter custom project ID: ")
		custom, err := reader.ReadString('\n')
		if err != nil {
			return "", "", false, err
		}
		custom = strings.TrimSpace(custom)
		if custom == "" {
			return "", "", false, common.NewInputError("project ID cannot be empty")
		}
		return custom, name, true, nil
	default:
		// Treat as custom ID
		return choice, name, true, nil
	}
}

// promptFeatureSelection lets the user choose which features to set up.
func promptFeatureSelection(reader lineReader) ([]string, error) {
	allFeatures := []string{domain.FeatureEmail, domain.FeatureCalendar, domain.FeatureContacts, domain.FeaturePubSub}

	fmt.Println("\n  Which features will you use?")
	for i, f := range allFeatures {
		fmt.Printf("  [%d] %s\n", i+1, featureLabel(f))
	}
	fmt.Print("  Select (1-4, comma-separated, or 'all'): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = trimInput(input)
	if input == "all" || input == "" {
		return allFeatures, nil
	}

	var features []string
	seen := map[string]bool{}
	for _, part := range strings.Split(input, ",") {
		var idx int
		if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &idx); err != nil || idx < 1 || idx > len(allFeatures) {
			return nil, common.NewInputError(fmt.Sprintf("invalid selection: %s", strings.TrimSpace(part)))
		}
		f := allFeatures[idx-1]
		if !seen[f] {
			features = append(features, f)
			seen[f] = true
		}
	}

	if len(features) == 0 {
		return nil, common.NewInputError("at least one feature must be selected")
	}
	return features, nil
}

// promptRegion asks for the Nylas region.
func promptRegion(reader lineReader) (string, error) {
	fmt.Print("\n  Which Nylas region? (us/eu): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	region := trimInput(input)
	if region == "" {
		region = "us"
	}
	if region != "us" && region != "eu" {
		return "", common.NewInputError(fmt.Sprintf("invalid region: %s (must be 'us' or 'eu')", region))
	}
	return region, nil
}

// createGCPProject creates a new GCP project if user chose to.
func createGCPProject(ctx context.Context, gcpClient ports.GCPClient, cfg *domain.GoogleSetupConfig) error {
	if !cfg.IsNewProject {
		return nil
	}

	spinner := common.NewSpinner(fmt.Sprintf("Creating GCP project \"%s\"...", cfg.ProjectID))
	spinner.Start()
	err := gcpClient.CreateProject(ctx, cfg.ProjectID, cfg.DisplayName)
	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	common.PrintSuccess("Project created")
	return nil
}

// enableAPIs enables the required Google APIs.
func enableAPIs(ctx context.Context, gcpClient ports.GCPClient, cfg *domain.GoogleSetupConfig) error {
	apis := featureToAPIs(cfg.Features)
	if len(apis) == 0 {
		return nil
	}

	spinner := common.NewSpinner("Enabling APIs...")
	spinner.Start()
	err := gcpClient.BatchEnableAPIs(ctx, cfg.ProjectID, apis)
	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to enable APIs: %w", err)
	}
	common.PrintSuccess("%d APIs enabled", len(apis))
	return nil
}

// addIAMOwner adds support@nylas.com as project owner.
func addIAMOwner(ctx context.Context, gcpClient ports.GCPClient, cfg *domain.GoogleSetupConfig, reader lineReader) error {
	if !cfg.SkipConfirmations {
		fmt.Printf("\n  Add %s as project owner? (Y/n) ", domain.NylasSupportEmail)
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = trimInput(input)
		if input == "n" || input == "no" {
			_, _ = common.Yellow.Println("  Skipped IAM owner setup")
			return nil
		}
	}

	policy, err := gcpClient.GetIAMPolicy(ctx, cfg.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get IAM policy: %w", err)
	}

	member := "user:" + domain.NylasSupportEmail
	if policy.HasMemberInRole("roles/owner", member) {
		_, _ = common.Yellow.Printf("  %s is already an owner\n", domain.NylasSupportEmail)
		return nil
	}

	policy.AddBinding("roles/owner", member)

	spinner := common.NewSpinner("Updating IAM policy...")
	spinner.Start()
	err = gcpClient.SetIAMPolicy(ctx, cfg.ProjectID, policy)
	spinner.Stop()

	if err != nil {
		return fmt.Errorf("failed to set IAM policy: %w", err)
	}
	common.PrintSuccess("IAM policy updated")
	return nil
}

// setupPubSub creates the Pub/Sub topic, service account, and grants publisher role.
func setupPubSub(ctx context.Context, gcpClient ports.GCPClient, cfg *domain.GoogleSetupConfig, state *domain.SetupState, configDir string) error {
	if !cfg.HasFeature(domain.FeaturePubSub) {
		return nil
	}

	// Create topic
	if !state.IsStepCompleted(domain.StepPubSubTopic) {
		spinner := common.NewSpinner("Creating Pub/Sub topic...")
		spinner.Start()
		err := gcpClient.CreateTopic(ctx, cfg.ProjectID, domain.NylasPubSubTopicName)
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("failed to create Pub/Sub topic: %w", err)
		}
		common.PrintSuccess("Topic created: %s", domain.NylasPubSubTopicName)
		state.CompleteStep(domain.StepPubSubTopic)
		_ = saveState(configDir, state)
	}

	// Create service account
	var saEmail string
	if !state.IsStepCompleted(domain.StepServiceAccount) {
		spinner := common.NewSpinner("Creating service account...")
		spinner.Start()
		email, err := gcpClient.CreateServiceAccount(ctx, cfg.ProjectID, domain.NylasPubSubServiceAccount, "Nylas Gmail Realtime")
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("failed to create service account: %w", err)
		}
		saEmail = email
		common.PrintSuccess("Service account created: %s", email)
		state.CompleteStep(domain.StepServiceAccount)
		_ = saveState(configDir, state)
	} else {
		saEmail = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", domain.NylasPubSubServiceAccount, cfg.ProjectID)
	}

	// Grant publisher role
	if !state.IsStepCompleted(domain.StepPubSubPublish) {
		spinner := common.NewSpinner("Granting publisher role...")
		spinner.Start()
		err := gcpClient.SetTopicIAMPolicy(ctx, cfg.ProjectID, domain.NylasPubSubTopicName,
			"serviceAccount:"+saEmail, "roles/pubsub.publisher")
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("failed to grant publisher role: %w", err)
		}
		common.PrintSuccess("Publisher role granted")
		state.CompleteStep(domain.StepPubSubPublish)
		_ = saveState(configDir, state)
	}

	return nil
}

// guideBrowserSteps walks the user through the two manual browser steps.
func guideBrowserSteps(browser ports.Browser, reader lineReader, cfg *domain.GoogleSetupConfig) error {
	// Step 1: OAuth Consent Screen
	fmt.Println("\n  Step 1/2: OAuth Consent Screen")
	url := consentScreenURL(cfg.ProjectID)
	fmt.Printf("  Opening: %s\n\n", url)
	_ = browser.Open(url)

	fmt.Println("  Instructions:")
	fmt.Println("    1. Select 'External' user type → Create")
	fmt.Println("    2. Fill in App name and User support email")
	fmt.Println("    3. Add your developer contact email")
	fmt.Println("    4. Click 'Save and Continue' through remaining steps")
	fmt.Println("    5. On the Summary page, click 'Back to Dashboard'")
	fmt.Print("\n  Press Enter when done... ")
	_, _ = reader.ReadString('\n')

	// Step 2: OAuth Credentials
	fmt.Println("\n  Step 2/2: OAuth Credentials")
	url = credentialsURL(cfg.ProjectID)
	fmt.Printf("  Opening: %s\n\n", url)
	_ = browser.Open(url)

	callbackURI := redirectURI(cfg.Region)
	fmt.Println("  Instructions:")
	fmt.Println("    1. Select 'Web application' as the application type")
	fmt.Println("    2. Give it a name (e.g., 'Nylas Integration')")
	fmt.Printf("    3. Add authorized redirect URI: %s\n", callbackURI)
	fmt.Println("    4. Click 'Create'")
	fmt.Println("    5. Copy the Client ID and Client Secret below")

	return nil
}

// promptOAuthCredentials reads OAuth client ID and secret from the user.
func promptOAuthCredentials(reader lineReader) (clientID, clientSecret string, err error) {
	fmt.Print("\n  Paste Client ID: ")
	id, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	clientID = strings.TrimSpace(id)
	if clientID == "" {
		return "", "", common.NewInputError("client ID cannot be empty")
	}

	fmt.Print("  Paste Client Secret (hidden): ")
	secretBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to regular input if terminal not available
		secret, readErr := reader.ReadString('\n')
		if readErr != nil {
			return "", "", readErr
		}
		clientSecret = strings.TrimSpace(secret)
	} else {
		fmt.Println()
		clientSecret = strings.TrimSpace(string(secretBytes))
	}

	if clientSecret == "" {
		return "", "", common.NewInputError("client secret cannot be empty")
	}
	return clientID, clientSecret, nil
}

// createNylasConnector creates the Google connector in Nylas.
func createNylasConnector(ctx context.Context, nylasClient ports.NylasClient, cfg *domain.GoogleSetupConfig) (*domain.Connector, error) {
	scopes := featureToScopes(cfg.Features)

	settings := &domain.ConnectorSettings{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
	}

	if cfg.HasFeature(domain.FeaturePubSub) {
		settings.TopicName = fmt.Sprintf("projects/%s/topics/%s", cfg.ProjectID, domain.NylasPubSubTopicName)
	}

	req := &domain.CreateConnectorRequest{
		Name:     "Google",
		Provider: "google",
		Settings: settings,
		Scopes:   scopes,
	}

	spinner := common.NewSpinner("Creating Google connector...")
	spinner.Start()
	connector, err := nylasClient.CreateConnector(ctx, req)
	spinner.Stop()

	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %w", err)
	}
	common.PrintSuccess("Connector created!")
	return connector, nil
}

// validateSetup verifies the connector was created successfully.
func validateSetup(ctx context.Context, nylasClient ports.NylasClient, connectorID string) {
	connector, err := nylasClient.GetConnector(ctx, connectorID)
	if err != nil {
		common.PrintWarning("Could not verify connector: %v", err)
		return
	}

	fmt.Printf("\n  Connector ID: %s\n", connector.ID)
	if len(connector.Scopes) > 0 {
		fmt.Printf("  Scopes: %s\n", strings.Join(connector.Scopes, ", "))
	}
}

// printSummary prints the post-setup summary and next steps.
func printSummary(cfg *domain.GoogleSetupConfig) {
	fmt.Println("\n  Setup complete! Next steps:")
	fmt.Println("    1. Run 'nylas auth login' to authenticate a Google account")
	fmt.Println("    2. Run 'nylas email list' to verify email access")
	if cfg.HasFeature(domain.FeatureCalendar) {
		fmt.Println("    3. Run 'nylas calendar list' to verify calendar access")
	}
	fmt.Printf("\n  Dashboard: https://dashboard.nylas.com\n")
}

// newStdinReader creates a buffered reader from stdin.
func newStdinReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}
