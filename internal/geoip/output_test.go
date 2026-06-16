package geoip

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KazZzeL/geomixer/internal/config"
)

func TestNewOutput_CategoryNameUpperCase(t *testing.T) {
	dir := "custom"
	cfg := &config.Output{
		Name: "geoip.dat",
		Dir:  &dir,
		Categories: []*config.Category{
			{Name: "lowercase", Steps: []*config.Step{
				{Action: config.StepActionAdd, Input: "in", Options: &config.Options{}},
			}},
		},
	}
	inputs := map[string]*Input{
		"in": {name: "in", categories: []*InputCategory{}},
	}
	o := NewOutput(cfg, "default", inputs)
	require.Len(t, o.categories, 1)
	assert.Equal(t, "LOWERCASE", o.categories[0].name)
}

func TestNewOutput_WithDir(t *testing.T) {
	dir := "custom"
	cfg := &config.Output{
		Name: "geoip.dat",
		Dir:  &dir,
		Categories: []*config.Category{
			{Name: "CN", Steps: []*config.Step{
				{Action: config.StepActionAdd, Input: "in", Options: &config.Options{}},
			}},
		},
	}
	inputs := map[string]*Input{
		"in": {name: "in", categories: []*InputCategory{}},
	}
	o := NewOutput(cfg, "default", inputs)
	assert.Equal(t, "custom", o.dir)
}

func TestNewOutput_DefaultDir(t *testing.T) {
	cfg := &config.Output{
		Name: "geoip.dat",
		Categories: []*config.Category{
			{Name: "CN", Steps: []*config.Step{
				{Action: config.StepActionAdd, Input: "in", Options: &config.Options{}},
			}},
		},
	}
	inputs := map[string]*Input{
		"in": {name: "in", categories: []*InputCategory{}},
	}
	o := NewOutput(cfg, "default", inputs)
	assert.Equal(t, "default", o.dir)
}

func TestOutput_Generate_NilOutput(t *testing.T) {
	var o *Output
	err := o.Generate(context.Background())
	require.ErrorIs(t, err, ErrOutputIsNil)
}

func TestOutput_String(t *testing.T) {
	o := &Output{name: "test"}
	assert.Equal(t, "test", o.String())
}

func TestOutput_String_Nil(t *testing.T) {
	var o *Output
	assert.Empty(t, o.String())
}

func TestOutputCategory_BuildPrefixes_Add(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name:    "all",
				ipv4Set: mustParseIPSet("10.0.0.0/8", "192.168.0.0/16"),
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, include: []string{"all"}},
		},
	}
	prefixes, err := cat.buildPrefixes()
	require.NoError(t, err)
	require.NotEmpty(t, prefixes)
}

func TestOutputCategory_BuildPrefixes_AddDel(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name:    "all",
				ipv4Set: mustParseIPSet("0.0.0.0/0"),
			},
		},
	}
	inputDel := &Input{
		name: "del",
		categories: []*InputCategory{
			{
				name:    "all",
				ipv4Set: mustParseIPSet("10.0.0.0/8"),
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input},
			{action: config.StepActionDel, input: inputDel},
		},
	}
	prefixes, err := cat.buildPrefixes()
	require.NoError(t, err)
	require.NotEmpty(t, prefixes)
	for _, p := range prefixes {
		assert.NotEqual(t, "10.0.0.0/8", p.String(), "10.0.0.0/8 should have been removed by del step")
	}
}

func TestOutputCategory_BuildPrefixes_StepOrder(t *testing.T) {
	inputAll := &Input{
		name: "all",
		categories: []*InputCategory{
			{name: "all", ipv4Set: mustParseIPSet("0.0.0.0/0")},
		},
	}
	inputCN := &Input{
		name: "cn",
		categories: []*InputCategory{
			{name: "cn", ipv4Set: mustParseIPSet("10.0.0.0/8")},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: inputAll},
			{action: config.StepActionDel, input: inputCN},
		},
	}
	prefixes, err := cat.buildPrefixes()
	require.NoError(t, err)
	require.NotEmpty(t, prefixes)
}

func TestOutputCategory_BuildPrefixes_IgnoreIPv4(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name:    "all",
				ipv4Set: mustParseIPSet("10.0.0.0/8"),
				ipv6Set: mustParseIPSet("2001:db8::/32"),
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{
				action:  config.StepActionAdd,
				input:   input,
				options: &OutputStepOptions{ignoreIPv4: true},
			},
		},
	}
	prefixes, err := cat.buildPrefixes()
	require.NoError(t, err)
	for _, p := range prefixes {
		assert.False(t, p.Addr().Is4(), "unexpected IPv4 prefix: %s", p.String())
	}
}

func TestOutputCategory_BuildPrefixes_IgnoreIPv6(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name:    "all",
				ipv4Set: mustParseIPSet("10.0.0.0/8"),
				ipv6Set: mustParseIPSet("2001:db8::/32"),
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{
				action:  config.StepActionAdd,
				input:   input,
				options: &OutputStepOptions{ignoreIPv6: true},
			},
		},
	}
	prefixes, err := cat.buildPrefixes()
	require.NoError(t, err)
	for _, p := range prefixes {
		assert.True(t, p.Addr().Is4(), "expected only IPv4 prefixes but got %s", p.String())
	}
}

func TestOutputCategory_BuildCIDR(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", ipv4Set: mustParseIPSet("10.0.0.0/8")},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input},
		},
	}
	cidr, err := cat.buildCIDR()
	require.NoError(t, err)
	require.NotEmpty(t, cidr)
}

func TestBuildGeoIP(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", ipv4Set: mustParseIPSet("10.0.0.0/8")},
		},
	}
	o := &Output{
		name: "test.dat",
		categories: []*OutputCategory{
			{
				name: "CN",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: input},
				},
			},
		},
	}
	geoip, err := o.buildGeoIP(context.Background())
	require.NoError(t, err)
	require.Len(t, geoip.GetCategories(), 1)
	assert.Equal(t, "CN", geoip.GetCategories()[0].GetName())
}

func TestBuildGeoIP_EmptyCategoryError(t *testing.T) {
	o := &Output{
		name: "test.dat",
		categories: []*OutputCategory{
			{
				name: "empty",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: &Input{name: "in", categories: []*InputCategory{}}},
				},
			},
		},
	}
	_, err := o.buildGeoIP(context.Background())
	require.Error(t, err)
}

func TestStepOrdering(t *testing.T) {
	steps := []*OutputStep{
		{action: config.StepActionDel, input: &Input{name: "del"}},
		{action: config.StepActionAdd, input: &Input{name: "add1"}},
		{action: config.StepActionAdd, input: &Input{name: "add2"}},
	}
	slices.SortFunc(steps, func(i, j *OutputStep) int {
		switch i.action {
		case j.action:
			return 0
		case config.StepActionAdd:
			return -1
		case config.StepActionDel:
			return 1
		}

		return 0
	})
	assert.Equal(t, config.StepActionAdd, steps[0].action, "expected add step first")
	assert.Equal(t, config.StepActionDel, steps[len(steps)-1].action, "expected del step last")
}
