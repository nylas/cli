package rpc

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/rpcserver"
	otpapp "github.com/nylas/cli/internal/app/otp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

const (
	envWSAddr   = "NYLAS_WS_ADDR"
	defaultAddr = "127.0.0.1:7369"

	envPollFast     = "NYLAS_WS_POLL_FAST"
	envPollIdle     = "NYLAS_WS_POLL_IDLE"
	envPollContacts = "NYLAS_WS_POLL_CONTACTS"
)

// pollInterval reads a positive Go duration from env (e.g. "2s", "1m"),
// falling back to def when unset or invalid.
func pollInterval(getenv func(string) string, key string, def time.Duration) time.Duration {
	if v := getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the JSON-RPC WebSocket server",
		RunE:  runServe,
	}

	cmd.Flags().String("addr", "", "address to bind (or NYLAS_WS_ADDR)")
	cmd.Flags().Bool("allow-remote", false, "allow binding to a non-loopback address")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	addr, err := cmd.Flags().GetString("addr")
	if err != nil {
		return fmt.Errorf("read --addr: %w", err)
	}
	if addr == "" {
		addr = os.Getenv(envWSAddr)
	}
	if addr == "" {
		addr = defaultAddr
	}

	allowRemote, err := cmd.Flags().GetBool("allow-remote")
	if err != nil {
		return fmt.Errorf("read --allow-remote: %w", err)
	}

	loopback, err := rpcserver.IsLoopback(addr)
	if err != nil {
		return err
	}
	if !loopback && !allowRemote {
		return fmt.Errorf("refusing to bind credential-holding RPC socket to non-loopback address %q without --allow-remote", addr)
	}
	if !loopback {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: binding credential-holding RPC socket to non-loopback address %s\n", addr)
	}

	client, err := common.GetNylasClient()
	if err != nil {
		return err
	}
	grantID, _ := common.GetGrantID(nil)

	store, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return fmt.Errorf("open secret store: %w", err)
	}
	token, err := rpcserver.ResolveToken(store, os.Getenv)
	if err != nil {
		return err
	}

	d := rpcserver.NewDispatcher()
	d.LogError = func(err error) { _, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rpc handler error: %v\n", err) }
	rpcserver.RegisterEmailHandlers(d, client, grantID)
	rpcserver.RegisterThreadHandlers(d, client, grantID)
	rpcserver.RegisterCalendarHandlers(d, client, grantID)
	rpcserver.RegisterContactHandlers(d, client, grantID)
	rpcserver.RegisterAgentHandlers(d, client)
	cfgStore := config.NewDefaultFileStore()
	rpcserver.RegisterConfigHandlers(d, cfgStore)
	grantStore, gerr := common.NewDefaultGrantStore()
	if gerr == nil {
		rpcserver.RegisterGrantHandlers(d, grantStore)
	} else {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "grant/otp handlers disabled: %v\n", gerr)
	}

	// Phase 2 writes (the client confirms before calling; the server executes immediately).
	rpcserver.RegisterEmailWriteHandlers(d, client, grantID)
	rpcserver.RegisterEmailExtHandlers(d, client, grantID)
	rpcserver.RegisterThreadWriteHandlers(d, client, grantID)
	rpcserver.RegisterCalendarWriteHandlers(d, client, grantID)
	rpcserver.RegisterCalendarExtHandlers(d, client, grantID)
	rpcserver.RegisterContactWriteHandlers(d, client, grantID)
	rpcserver.RegisterContactExtHandlers(d, client, grantID)

	// Extended domains.
	rpcserver.RegisterDraftHandlers(d, client, grantID)
	rpcserver.RegisterNotetakerHandlers(d, client, grantID)
	rpcserver.RegisterSchedulerHandlers(d, client, grantID)
	rpcserver.RegisterTemplateWorkflowHandlers(d, client, grantID)
	rpcserver.RegisterAdminHandlers(d, client)
	rpcserver.RegisterAuthHandlers(d, client, grantID)
	if auditStore, aerr := audit.NewFileStore(""); aerr == nil {
		rpcserver.RegisterAuditHandlers(d, auditStore)
	} else {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "audit handlers disabled: %v\n", aerr)
	}
	if gerr == nil {
		rpcserver.RegisterOTPHandlers(d, otpapp.NewService(client, grantStore, cfgStore))
	}

	srv := rpcserver.NewServer(rpcserver.Config{
		Addr:  addr,
		Token: token,
	}, d)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fast := pollInterval(os.Getenv, envPollFast, 5*time.Second)
	idle := pollInterval(os.Getenv, envPollIdle, 30*time.Second)
	contactInterval := pollInterval(os.Getenv, envPollContacts, 60*time.Second)

	ctrl := rpcserver.NewIntervalController(fast, idle)
	contactCtrl := rpcserver.NewIntervalController(contactInterval, contactInterval)
	rpcserver.RegisterFocusHandler(d, ctrl)
	rpcserver.RegisterPollConfigHandler(d, ctrl, contactCtrl)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		cancel()
	}()

	if grantID == "" {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "no default grant configured; live notifications disabled — set a default grant to enable them")
	} else {
		since := time.Now().Unix()
		onErr := func(err error) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rpc poll error: %v\n", err)
		}
		startPoller := func(name string, run func() error) {
			go func() {
				if err := run(); err != nil && ctx.Err() == nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rpc %s poller stopped: %v\n", name, err)
				}
			}()
		}

		mp := rpcserver.NewMessagePoller(client, grantID, since, srv.Broadcast)
		startPoller("message", func() error { return rpcserver.RunAdaptive(ctx, ctrl, onErr, mp.PollOnce) })

		tp := rpcserver.NewThreadPoller(client, grantID, since, srv.Broadcast)
		startPoller("thread", func() error { return rpcserver.RunAdaptive(ctx, ctrl, onErr, tp.PollOnce) })

		calendarIDs := []string{"primary"}
		calCtx, calCancel := context.WithTimeout(ctx, 10*time.Second)
		cals, cerr := client.GetCalendars(calCtx, grantID)
		calCancel()
		if cerr != nil {
			onErr(fmt.Errorf("list calendars for event pollers: %w", cerr))
		} else if len(cals) > 0 {
			calendarIDs = calendarIDs[:0]
			for _, cal := range cals {
				if cal.ID != "" {
					calendarIDs = append(calendarIDs, cal.ID)
				}
			}
			if len(calendarIDs) == 0 {
				calendarIDs = []string{"primary"}
			}
		}

		// ponytail: per-calendar polling scales API calls with calendar count; webhooks are the upgrade path.
		for _, calendarID := range calendarIDs {
			ep := rpcserver.NewEventPoller(client, grantID, calendarID, since, srv.Broadcast)
			startPoller("event", func() error { return rpcserver.RunAdaptive(ctx, ctrl, onErr, ep.PollOnce) })
		}

		// ponytail: contacts have no server-side time filter — refetch+diff on a slow cadence.
		cp := rpcserver.NewContactPoller(client, grantID, srv.Broadcast)
		startPoller("contact", func() error { return rpcserver.RunAdaptive(ctx, contactCtrl, onErr, cp.PollOnce) })
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Nylas RPC WebSocket listening on %s\n", addr)
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Authenticate with Authorization: Bearer <token> or ?token=<token>.")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "The token is stored in the keyring or read from NYLAS_WS_TOKEN.")

	return srv.Serve(ctx)
}
