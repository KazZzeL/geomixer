package geosite

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KazZzeL/geomixer/internal/config"
)

func TestNewOutput_WithDir(t *testing.T) {
	dir := "custom"
	cfg := &config.Output{
		Name: "geosite.dat",
		Dir:  &dir,
		Categories: []*config.Category{
			{Name: "CN", Steps: []*config.Step{
				{Action: config.StepActionAdd, Input: "in", Options: &config.Options{}},
			}},
		},
	}
	inputs := map[string]*Input{"in": {name: "in", categories: []*InputCategory{}}}
	o := NewOutput(cfg, "default", inputs)
	assert.Equal(t, "custom", o.dir)
}

func TestNewOutput_DefaultDir(t *testing.T) {
	cfg := &config.Output{
		Name: "geosite.dat",
		Categories: []*config.Category{
			{Name: "CN", Steps: []*config.Step{
				{Action: config.StepActionAdd, Input: "in", Options: &config.Options{}},
			}},
		},
	}
	inputs := map[string]*Input{"in": {name: "in", categories: []*InputCategory{}}}
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

func TestToOutputDomain_NoOptions(t *testing.T) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
		Attributes: []*Domain_Attribute{
			{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
		},
	}
	opts := &OutputStepOptions{}
	od := d.toOutputDomain(opts)
	assert.Equal(t, "domain:example.com:@ads", od.plain)
	require.Len(t, od.domain.GetAttributes(), 1)
}

func TestToOutputDomain_ResetAttributes(t *testing.T) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
		Attributes: []*Domain_Attribute{
			{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
		},
	}
	opts := &OutputStepOptions{resetAttributes: true}
	od := d.toOutputDomain(opts)
	assert.Empty(t, od.domain.GetAttributes())
	assert.Equal(t, "domain:example.com", od.plain)
}

func TestToOutputDomain_DeleteAttributes(t *testing.T) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
		Attributes: []*Domain_Attribute{
			{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
			{Key: "tracking", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
		},
	}
	opts := &OutputStepOptions{deleteAttributes: []string{"ads"}}
	od := d.toOutputDomain(opts)
	require.Len(t, od.domain.GetAttributes(), 1)
	assert.Equal(t, "tracking", od.domain.GetAttributes()[0].GetKey())
	assert.Equal(t, "domain:example.com:@tracking", od.plain)
}

func TestToOutputDomain_AppendAttributes(t *testing.T) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
	}
	opts := &OutputStepOptions{appendAttributes: []string{"newattr"}}
	od := d.toOutputDomain(opts)
	require.Len(t, od.domain.GetAttributes(), 1)
	assert.Equal(t, "newattr", od.domain.GetAttributes()[0].GetKey())
	assert.Equal(t, "domain:example.com:@newattr", od.plain)
}

func TestToOutputDomain_ResetThenAppend(t *testing.T) {
	d := &Domain{
		Type:  Domain_domain,
		Value: "example.com",
		Attributes: []*Domain_Attribute{
			{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
		},
	}
	opts := &OutputStepOptions{
		resetAttributes:  true,
		appendAttributes: []string{"newattr"},
	}
	od := d.toOutputDomain(opts)
	require.Len(t, od.domain.GetAttributes(), 1)
	assert.Equal(t, "newattr", od.domain.GetAttributes()[0].GetKey())
	assert.Equal(t, "domain:example.com:@newattr", od.plain)
}

func TestCollectDomains_Add(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{Type: Domain_domain, Value: "example.com"},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{
				action:  config.StepActionAdd,
				input:   input,
				options: &OutputStepOptions{},
			},
		},
	}
	domains, err := cat.collectDomains()
	require.NoError(t, err)
	require.Len(t, domains, 1)
}

func TestCollectDomains_AddThenDel(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{Type: Domain_domain, Value: "example.com"},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
			{action: config.StepActionDel, input: input, options: &OutputStepOptions{}},
		},
	}
	domains, err := cat.collectDomains()
	require.NoError(t, err)
	assert.Empty(t, domains)
}

func TestCollectDomains_DuplicateAdd(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{Type: Domain_domain, Value: "example.com"},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}
	domains, err := cat.collectDomains()
	require.NoError(t, err)
	require.Len(t, domains, 1)
}

func TestBuildDomains_SubdomainPruning(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{Type: Domain_domain, Value: "google.com"},
					{Type: Domain_domain, Value: "mail.google.com"},
					{Type: Domain_domain, Value: "example.com"},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}
	domains, err := cat.buildDomains()
	require.NoError(t, err)
	foundParent := false
	foundSub := false
	for _, d := range domains {
		switch d.GetValue() {
		case "google.com":
			foundParent = true
		case "mail.google.com":
			foundSub = true
		}
	}
	assert.True(t, foundParent, "expected 'google.com' to be present")
	assert.False(t, foundSub, "expected 'mail.google.com' to be pruned (subdomain of google.com)")
}

func TestBuildDomains_SubdomainPruningFull(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{Type: Domain_domain, Value: "google.com"},
					{Type: Domain_full, Value: "mail.google.com"},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}
	domains, err := cat.buildDomains()
	require.NoError(t, err)
	foundFull := false
	for _, d := range domains {
		if d.GetValue() == "mail.google.com" && d.GetType() == Domain_full {
			foundFull = true
		}
	}
	assert.False(t, foundFull, "expected 'full:mail.google.com' to be pruned (subdomain of domain:google.com)")
}

func TestBuildDomains_RegexSubstrNotPruned(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{Type: Domain_domain, Value: "google.com"},
					{Type: Domain_regex, Value: ".*\\.google\\.com"},
					{Type: Domain_substr, Value: "google"},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}
	domains, err := cat.buildDomains()
	require.NoError(t, err)

	hasRegex := false
	hasSubstr := false
	for _, d := range domains {
		switch d.GetType() {
		case Domain_regex:
			hasRegex = true
		case Domain_substr:
			hasSubstr = true
		case Domain_domain, Domain_full:
		}
	}
	assert.True(t, hasRegex, "expected regex domain to remain (not pruned)")
	assert.True(t, hasSubstr, "expected substr domain to remain (not pruned)")
}

func TestBuildDomains_AttrDomainsNotQueued(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{
				name: "all",
				domains: []*Domain{
					{
						Type:  Domain_domain,
						Value: "google.com",
						Attributes: []*Domain_Attribute{
							{Key: "ads", TypedValue: &Domain_Attribute_BoolValue{BoolValue: true}},
						},
					},
				},
			},
		},
	}
	cat := &OutputCategory{
		name: "test",
		steps: []*OutputStep{
			{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
		},
	}
	domains, err := cat.buildDomains()
	require.NoError(t, err)
	require.Len(t, domains, 1)
}

func TestBuildGeoSite(t *testing.T) {
	input := &Input{
		name: "in",
		categories: []*InputCategory{
			{name: "all", domains: []*Domain{{Type: Domain_domain, Value: "example.com"}}},
		},
	}
	o := &Output{
		name: "geosite.dat",
		categories: []*OutputCategory{
			{
				name: "CN",
				steps: []*OutputStep{
					{action: config.StepActionAdd, input: input, options: &OutputStepOptions{}},
				},
			},
		},
	}
	gs, err := o.buildGeoSite(context.Background())
	require.NoError(t, err)
	require.Len(t, gs.GetCategories(), 1)
	assert.Equal(t, "CN", gs.GetCategories()[0].GetName())
}

func TestBuildGeoSite_EmptyCategoryError(t *testing.T) {
	o := &Output{
		name: "geosite.dat",
		categories: []*OutputCategory{
			{
				name: "empty",
				steps: []*OutputStep{
					{
						action:  config.StepActionAdd,
						input:   &Input{name: "in", categories: []*InputCategory{}},
						options: &OutputStepOptions{},
					},
				},
			},
		},
	}
	_, err := o.buildGeoSite(context.Background())
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
