package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"stress-test/internal/loadtest"
)

func newRootCmd() *cobra.Command {
	var cfg loadtest.Config

	cmd := &cobra.Command{
		Use:   "stress-test",
		Short: "Teste de carga para serviços web",
		Long:  "Dispara um número total de requisições HTTP GET contra uma URL, com um nível de concorrência definido, e imprime um relatório ao final.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validate(cfg); err != nil {
				return err
			}
			cmd.SilenceUsage = true

			if cfg.Concurrency > cfg.Requests {
				cfg.Concurrency = cfg.Requests
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			fmt.Fprintf(cmd.OutOrStdout(), "Testando %s: %d requests, %d simultâneas...\n\n", cfg.URL, cfg.Requests, cfg.Concurrency)
			loadtest.Run(ctx, cfg).Print(cmd.OutOrStdout())
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.URL, "url", "", "URL do serviço a ser testado (http:// ou https://)")
	cmd.Flags().IntVar(&cfg.Requests, "requests", 0, "número total de requests a realizar")
	cmd.Flags().IntVar(&cfg.Concurrency, "concurrency", 0, "número de chamadas simultâneas")
	for _, name := range []string{"url", "requests", "concurrency"} {
		_ = cmd.MarkFlagRequired(name)
	}
	return cmd
}

func validate(cfg loadtest.Config) error {
	u, err := url.Parse(cfg.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("--url inválida %q: use uma URL completa, ex.: http://exemplo.com", cfg.URL)
	}
	if cfg.Requests <= 0 {
		return fmt.Errorf("--requests deve ser maior que zero")
	}
	if cfg.Concurrency <= 0 {
		return fmt.Errorf("--concurrency deve ser maior que zero")
	}
	return nil
}

// Execute runs the CLI and exits with a non-zero status on error.
func Execute() {
	if err := newRootCmd().ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
