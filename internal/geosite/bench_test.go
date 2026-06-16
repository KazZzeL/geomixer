package geosite

import (
	"context"
	"testing"

	"github.com/KazZzeL/geomixer/internal/config"
)

func BenchmarkParseDomain(b *testing.B) {
	c := &InputCategory{}
	cases := []string{
		"example.com",
		"domain:google.com",
		"keyword:example",
		"regexp:^test\\.com$",
		"full:exact.com",
		"google.com @ads @tracking",
		"example.com # comment",
	}

	b.ResetTimer()
	for b.Loop() {
		for _, p := range cases {
			_, _ = c.parseDomain(p)
		}
	}
}

func BenchmarkInputParseList(b *testing.B) {
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "domain:example.com"
	}

	cfg := &config.Input{
		Name: "bench",
		Kind: config.InputKindLst,
		List: lines,
	}

	b.ResetTimer()
	for b.Loop() {
		in := NewInput(cfg, nil, 0)
		if err := in.Parse(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDomains(b *testing.B) {
	domains := make([]*Domain, 100)
	for i := range domains {
		domains[i] = &Domain{
			Type:  Domain_domain,
			Value: "example.com",
		}
	}

	in := &Input{
		name: "bench",
		categories: []*InputCategory{
			{name: "a", domains: domains},
			{name: "b", domains: domains},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := in.Domains([]string{"a"}, []string{"b"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToOutputDomain(b *testing.B) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
		Attributes: []*Domain_Attribute{
			{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
			{Key: "tracking", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
		},
	}
	opts := &OutputStepOptions{
		deleteAttrs: []string{"ads"},
		appendAttrs: []string{"newattr"},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = opts.newOutputDomain(d)
	}
}

func BenchmarkCollectDomains(b *testing.B) {
	domains := make([]*Domain, 100)
	for i := range domains {
		domains[i] = &Domain{
			Type:  Domain_domain,
			Value: "example.com",
		}
	}

	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", domains: domains},
		},
	}
	cat := &OutputCategory{
		name: "bench",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := cat.collectDomains()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildDomains(b *testing.B) {
	domains := make([]*Domain, 500)
	for i := range domains {
		domains[i] = &Domain{
			Type:  Domain_domain,
			Value: "example.com",
		}
	}
	domains[0].Value = "google.com"

	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", domains: domains},
		},
	}
	cat := &OutputCategory{
		name: "bench",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := cat.buildDomains()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildGeoSite(b *testing.B) {
	domains := make([]*Domain, 100)
	for i := range domains {
		domains[i] = &Domain{
			Type:  Domain_domain,
			Value: "example.com",
		}
	}

	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", domains: domains},
		},
	}
	o := &Output{
		name: "bench.dat",
		categories: []*OutputCategory{
			{
				name: "CN",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
				},
			},
			{
				name: "GLOBAL",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
				},
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := o.buildGeoSite(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPlainKey(b *testing.B) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
		Attributes: []*Domain_Attribute{
			{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
			{Key: "tracking", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
			{Key: "malware", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = plainKey(d, false)
	}
}
