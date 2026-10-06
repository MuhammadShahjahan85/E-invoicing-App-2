// Command einvoice is the E-Invoicing Suite PK server.
//
//	einvoice serve [--data DIR] [--listen ADDR]   run the server (default command)
//	einvoice service install|uninstall|start|stop  manage the Windows service
//	einvoice mock-fbr [--listen ADDR]              run the offline FBR DI simulator
//	einvoice backup [--data DIR] [--out FILE]      write a database backup
//	einvoice reset-password --user NAME --password PW [--data DIR]
//	einvoice version
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // Asia/Karachi on Windows without a zoneinfo database

	"einvoicing/internal/app"
	"einvoicing/internal/brand"
	"einvoicing/internal/config"
	"einvoicing/internal/db"
	"einvoicing/internal/fbrmock"
	"einvoicing/internal/service"
	"einvoicing/internal/winsvc"
)

func main() {
	if winsvc.IsService() {
		data := config.DefaultDataDir()
		for i, a := range os.Args {
			if a == "--data" && i+1 < len(os.Args) {
				data = os.Args[i+1]
			}
		}
		if err := winsvc.Run(func(ctx context.Context) error { return serve(ctx, data, "", false) }); err != nil {
			log.Fatal(err)
		}
		return
	}
	cmd := "serve"
	args := os.Args[1:]
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "serve":
		err = cmdServe(args)
	case "service":
		err = cmdService(args)
	case "mock-fbr":
		err = cmdMock(args)
	case "backup":
		err = cmdBackup(args)
	case "reset-password":
		err = cmdResetPassword(args)
	case "version":
		fmt.Printf("%s %s (built %s)\n", brand.ProductName, brand.Version, brand.BuildDate)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf(`%s %s

Usage:
  einvoice serve [--data DIR] [--listen ADDR]     run the server (default)
  einvoice service install [--data DIR]           install as a Windows service (run as Administrator)
  einvoice service uninstall|start|stop           manage the Windows service
  einvoice mock-fbr [--listen 127.0.0.1:9090]     run the offline FBR DI simulator for ERP testing
  einvoice backup [--data DIR] [--out FILE]       write a consistent database backup
  einvoice reset-password --user NAME --password NEW [--data DIR]
  einvoice version
`, brand.ProductName, brand.Version)
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	data := fs.String("data", config.DefaultDataDir(), "data directory")
	listen := fs.String("listen", "", "listen address (overrides config.json), e.g. 0.0.0.0:8443")
	noTLS := fs.Bool("no-tls", false, "serve plain HTTP (only for localhost use)")
	_ = fs.Parse(args)
	ctx, cancel := signalContext()
	defer cancel()
	return serve(ctx, *data, *listen, *noTLS)
}

func serve(ctx context.Context, data, listen string, noTLS bool) error {
	a, err := app.Open(data, true)
	if err != nil {
		return err
	}
	defer a.Close()
	if listen != "" {
		a.Cfg.Listen = listen
	}
	if noTLS {
		a.Cfg.TLS.Enabled = false
	}
	return a.Serve(ctx)
}

func cmdService(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service: expected install, uninstall, start or stop")
	}
	switch args[0] {
	case "install":
		fs := flag.NewFlagSet("install", flag.ExitOnError)
		data := fs.String("data", config.DefaultDataDir(), "data directory")
		_ = fs.Parse(args[1:])
		if err := os.MkdirAll(*data, 0o750); err != nil {
			return err
		}
		if err := winsvc.Install(*data); err != nil {
			return err
		}
		fmt.Println("Service installed. Start it with: einvoice service start")
		return nil
	case "uninstall":
		return winsvc.Uninstall()
	case "start":
		return winsvc.Start()
	case "stop":
		return winsvc.Stop()
	}
	return fmt.Errorf("unknown service command %q", args[0])
}

func cmdMock(args []string) error {
	fs := flag.NewFlagSet("mock-fbr", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:9090", "listen address")
	_ = fs.Parse(args)
	sim := fbrmock.New()
	sim.AcceptAnyToken = true
	srv := &http.Server{Addr: *listen, Handler: sim.Handler(), ReadHeaderTimeout: 10 * time.Second}
	fmt.Printf("FBR DI simulator listening on http://%s (any bearer token accepted)\n", *listen)
	fmt.Println("Endpoints: /di_data/v1/di/postinvoicedata[_sb], /di_data/v1/di/validateinvoicedata[_sb], /pdi/v1/*, /pdi/v2/*, /dist/v1/*")
	fmt.Println("Fault injection: POST /mock/fault {\"kind\":\"timeout|drop|server_error|unavailable|unauthorized\",\"count\":1}")
	ctx, cancel := signalContext()
	defer cancel()
	go func() { <-ctx.Done(); _ = srv.Close() }()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func cmdBackup(args []string) error {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	data := fs.String("data", config.DefaultDataDir(), "data directory")
	out := fs.String("out", "", "output file (default: <data>/backups/...)")
	_ = fs.Parse(args)
	a, err := app.Open(*data, false)
	if err != nil {
		return err
	}
	defer a.Close()
	if *out != "" {
		if err := db.Backup(context.Background(), a.Svc.Store.DB, *out); err != nil {
			return err
		}
		fmt.Println("Backup written to", *out)
		return nil
	}
	b, err := a.Svc.Backup(context.Background(), service.System, "manual")
	if err != nil {
		return err
	}
	fmt.Println("Backup written:", b.Name, "in", a.Svc.BackupDir())
	fmt.Println("Remember: keep a separate, secure copy of master.key from the data directory.")
	return nil
}

func cmdResetPassword(args []string) error {
	fs := flag.NewFlagSet("reset-password", flag.ExitOnError)
	data := fs.String("data", config.DefaultDataDir(), "data directory")
	user := fs.String("user", "", "username")
	pw := fs.String("password", "", "new password (8+ characters with letters and digits)")
	_ = fs.Parse(args)
	if *user == "" || *pw == "" {
		return fmt.Errorf("--user and --password are required")
	}
	a, err := app.Open(*data, false)
	if err != nil {
		return err
	}
	defer a.Close()
	if err := a.Svc.ResetAdminPassword(context.Background(), *user, *pw); err != nil {
		return err
	}
	fmt.Println("Password reset. The user must change it at next login.")
	return nil
}
