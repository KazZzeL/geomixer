package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/geoip"
	"github.com/KazZzeL/geomixer/internal/geosite"
	"github.com/KazZzeL/geomixer/internal/utils/httpclient"
)

var (
	mixOutputDir   string
	mixGeositeOnly bool
	mixGeoipOnly   bool
	mixTLSMin      string
	mixHTTPTimeout time.Duration
)

var mixCmd = &cobra.Command{
	Use:   "mix",
	Short: "mix geofiles",
	Long:  "mix geofiles",
	RunE:  runMix,
}

func init() {
	mixCmd.Flags().StringVarP(&mixOutputDir, "output", "o", "./output", "Generated geofiles dir (default: ./output)")
	mixCmd.Flags().BoolVarP(&mixGeositeOnly, "geosite-only", "s", false, "Generage only geosites")
	mixCmd.Flags().BoolVarP(&mixGeoipOnly, "geoip-only", "i", false, "Generage only geoips")
	mixCmd.Flags().StringVar(&mixTLSMin, "min-tls", "1.3", "TLS min version")
	mixCmd.Flags().DurationVar(&mixHTTPTimeout, "http-timeout", 15*time.Second, "HTTP requests timeout")

	// Отмечаем конфликтующие флаги
	mixCmd.MarkFlagsMutuallyExclusive("geosite-only", "geoip-only")
}

func runMix(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	configFile := args[0]

	cfg, err := config.Parse(configFile)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hc := httpclient.NewClient(mixTLSMin)

	eg, egCtx := errgroup.WithContext(ctx)

	if !mixGeoipOnly {
		eg.Go(func() error {
			if err := geosite.NewRunner(cfg.Geosite, mixOutputDir).Run(egCtx, hc, mixHTTPTimeout); err != nil {
				return fmt.Errorf("geosite: %w", err)
			}

			return nil
		})
	}

	if !mixGeositeOnly {
		eg.Go(func() error {
			if err := geoip.NewRunner(cfg.Geoip, mixOutputDir).Run(egCtx, hc, mixHTTPTimeout); err != nil {
				return fmt.Errorf("geoip: %w", err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
