package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/geoip"
	"github.com/KazZzeL/geomixer/internal/geosite"
	"github.com/KazZzeL/geomixer/internal/utils/httpclient"
)

var testDataDir = "./input/geofiles-testdata"

func BenchmarkE2EGeoIP(b *testing.B) {
	cfg := &config.Config{
		Geoip: &config.Runner{
			Inputs: []*config.Input{
				{
					Name: "kazzzel",
					Kind: config.InputKindGeo,
					Path: testDataDir + "/allow-subnets.dat",
				},
			},
			Outputs: []*config.Output{
				{
					Name: "geoip",
					Categories: []*config.Category{
						{
							Name: "included-direct",
							Steps: []*config.Step{
								{Action: config.StepActionAdd, Input: "kazzzel", Options: &config.Options{}},
							},
						},
					},
				},
			},
		},
	}

	hc := &http.Client{Timeout: 10 * time.Second}

	b.ResetTimer()
	for b.Loop() {
		runner := geoip.NewRunner(cfg.Geoip, b.TempDir())
		if err := runner.Run(context.Background(), hc, 30*time.Second); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkE2EGeoSite(b *testing.B) {
	cfg := &config.Config{
		Geosite: &config.Runner{
			Inputs: []*config.Input{
				{
					Name: "kazzzel",
					Kind: config.InputKindGeo,
					Path: testDataDir + "/allow-domains.dat",
				},
			},
			Outputs: []*config.Output{
				{
					Name: "geosite",
					Categories: []*config.Category{
						{
							Name: "included-direct",
							Steps: []*config.Step{
								{Action: config.StepActionAdd, Input: "kazzzel", Options: &config.Options{}},
							},
						},
					},
				},
			},
		},
	}

	hc := &http.Client{Timeout: 10 * time.Second}

	b.ResetTimer()
	for b.Loop() {
		runner := geosite.NewRunner(cfg.Geosite, b.TempDir())
		if err := runner.Run(context.Background(), hc, 30*time.Second); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkE2ERealV2Fly(b *testing.B) {
	geoipCfg := &config.Config{
		Geoip: &config.Runner{
			Inputs: []*config.Input{
				{
					Name: "v2fly",
					Kind: config.InputKindGeo,
					Path: testDataDir + "/geoip.dat",
				},
			},
			Outputs: []*config.Output{
				{
					Name: "geoip",
					Categories: []*config.Category{
						{
							Name: "private",
							Steps: []*config.Step{
								{Action: config.StepActionAdd, Input: "v2fly", Options: &config.Options{}},
							},
						},
					},
				},
			},
		},
	}
	geositeCfg := &config.Config{
		Geosite: &config.Runner{
			Inputs: []*config.Input{
				{
					Name: "v2fly",
					Kind: config.InputKindGeo,
					Path: testDataDir + "/geosite.dat",
				},
			},
			Outputs: []*config.Output{
				{
					Name: "geosite",
					Categories: []*config.Category{
						{
							Name: "geosite-cn",
							Steps: []*config.Step{
								{Action: config.StepActionAdd, Input: "v2fly", Options: &config.Options{}},
							},
						},
					},
				},
			},
		},
	}

	httpClient := httpclient.NewClient("1.3")

	b.ResetTimer()
	for b.Loop() {
		r1 := geoip.NewRunner(geoipCfg.Geoip, b.TempDir())
		if err := r1.Run(context.Background(), httpClient, 30*time.Second); err != nil {
			b.Fatal(err)
		}
		r2 := geosite.NewRunner(geositeCfg.Geosite, b.TempDir())
		if err := r2.Run(context.Background(), httpClient, 30*time.Second); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkE2ECombined(b *testing.B) {
	geoipCfg := &config.Config{
		Geoip: &config.Runner{
			Inputs: []*config.Input{
				{
					Name: "kazzzel",
					Kind: config.InputKindGeo,
					Path: testDataDir + "/allow-subnets.dat",
				},
			},
			Outputs: []*config.Output{
				{
					Name: "geoip",
					Categories: []*config.Category{
						{
							Name: "included-direct",
							Steps: []*config.Step{
								{Action: config.StepActionAdd, Input: "kazzzel", Options: &config.Options{}},
							},
						},
					},
				},
			},
		},
	}
	geositeCfg := &config.Config{
		Geosite: &config.Runner{
			Inputs: []*config.Input{
				{
					Name: "kazzzel",
					Kind: config.InputKindGeo,
					Path: testDataDir + "/allow-domains.dat",
				},
			},
			Outputs: []*config.Output{
				{
					Name: "geosite",
					Categories: []*config.Category{
						{
							Name: "included-direct",
							Steps: []*config.Step{
								{Action: config.StepActionAdd, Input: "kazzzel", Options: &config.Options{}},
							},
						},
					},
				},
			},
		},
	}

	httpClient := httpclient.NewClient("1.3")

	b.ResetTimer()
	for b.Loop() {
		r1 := geoip.NewRunner(geoipCfg.Geoip, b.TempDir())
		if err := r1.Run(context.Background(), httpClient, 30*time.Second); err != nil {
			b.Fatal(err)
		}
		r2 := geosite.NewRunner(geositeCfg.Geosite, b.TempDir())
		if err := r2.Run(context.Background(), httpClient, 30*time.Second); err != nil {
			b.Fatal(err)
		}
	}
}
