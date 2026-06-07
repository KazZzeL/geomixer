package geoip

import (
	"context"
	"testing"

	"go4.org/netipx"

	"github.com/KazZzeL/geomixer/internal/config"
)

func BenchmarkParseCIDR(b *testing.B) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	cases := []string{
		"10.0.0.0/8",
		"2001:db8::/32",
		"192.168.1.1",
		"::1",
		"10.0.0.0/8 # comment",
		"10.0.0.0/8 // comment",
	}

	b.ResetTimer()
	for range b.N {
		for _, cidr := range cases {
			_ = c.addCIDR(cidr)
		}
	}
}

func BenchmarkBuildIPSets(b *testing.B) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	for range 1000 {
		_ = c.addCIDR("10.0.0.0/8")
	}

	b.ResetTimer()
	for range b.N {
		_ = c.buildIPSets()
	}
}

func BenchmarkInputParseList(b *testing.B) {
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "10.0.0.0/8"
	}

	cfg := &config.Input{
		Name: "bench",
		Kind: config.InputKindLst,
		List: lines,
	}

	b.ResetTimer()
	for range b.N {
		in := NewInput(cfg, nil, 0)
		if err := in.Parse(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIPSets(b *testing.B) {
	in := &Input{
		name: "bench",
		categories: []*InputCategory{
			{
				name:    "a",
				ipv4Set: mustParseIPSet("10.0.0.0/8"),
				ipv6Set: mustParseIPSet("2001:db8::/32"),
			},
			{
				name:    "b",
				ipv4Set: mustParseIPSet("192.168.0.0/16"),
				ipv6Set: mustParseIPSet("2001:db8:1::/48"),
			},
		},
	}

	b.ResetTimer()
	for range b.N {
		_, _, err := in.IPSets([]string{"a"}, []string{"b"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildPrefixes(b *testing.B) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", ipv4Set: mustParseIPSet("0.0.0.0/0")},
		},
	}
	cat := &OutputCategory{
		name: "bench",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input},
		},
	}

	b.ResetTimer()
	for range b.N {
		_, err := cat.buildPrefixes()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildCIDR(b *testing.B) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", ipv4Set: mustParseIPSet("10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12")},
		},
	}
	cat := &OutputCategory{
		name: "bench",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input},
		},
	}

	b.ResetTimer()
	for range b.N {
		_, err := cat.buildCIDR()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOutputBuildGeoIP(b *testing.B) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", ipv4Set: mustParseIPSet("10.0.0.0/8")},
			{name: "cn", ipv4Set: mustParseIPSet("192.168.0.0/16")},
		},
	}
	o := &Output{
		name: "bench.dat",
		categories: []*OutputCategory{
			{
				name: "CN",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: input},
				},
			},
			{
				name: "GLOBAL",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: input, include: []string{"all"}},
					{action: config.StepActionDel, input: input, include: []string{"cn"}},
				},
			},
		},
	}

	b.ResetTimer()
	for range b.N {
		_, err := o.buildGeoIP(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}
