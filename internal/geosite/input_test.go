package geosite

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KazZzeL/geomixer/internal/config"
)

func TestInput_addCategory_UpperCase(t *testing.T) {
	i := &Input{name: "test"}

	c := i.addCategory("lowercase")

	assert.Equal(t, "LOWERCASE", c.name)
}

func TestInput_addCategory_TrimSpace(t *testing.T) {
	i := &Input{name: "test"}

	c := i.addCategory("  spaced  ")

	assert.Equal(t, "SPACED", c.name)
}

func TestInput_Domains_NilInput(t *testing.T) {
	var i *Input

	_, err := i.Domains(nil, nil)

	require.ErrorIs(t, err, ErrInputIsNil)
}

func TestInput_Domains_NoCategories(t *testing.T) {
	i := &Input{name: "test", categories: []*InputCategory{}}

	_, err := i.Domains(nil, nil)

	require.Error(t, err)
}

func TestInput_Domains_FilterInclude(t *testing.T) {
	i := &Input{
		name: "test",
		categories: []*InputCategory{
			{name: "cn", domains: []*Domain{{Type: Domain_domain, Value: "google.com"}}},
			{name: "us", domains: []*Domain{{Type: Domain_domain, Value: "example.com"}}},
		},
	}

	domains, err := i.Domains([]string{"cn"}, nil)

	require.NoError(t, err)
	require.Len(t, domains, 1)
	assert.Equal(t, "google.com", domains[0].GetValue())
}

func TestInput_Domains_FilterExclude(t *testing.T) {
	i := &Input{
		name: "test",
		categories: []*InputCategory{
			{name: "cn", domains: []*Domain{{Type: Domain_domain, Value: "google.com"}}},
			{name: "us", domains: []*Domain{{Type: Domain_domain, Value: "example.com"}}},
		},
	}

	domains, err := i.Domains(nil, []string{"cn"})

	require.NoError(t, err)
	require.Len(t, domains, 1)
	assert.Equal(t, "example.com", domains[0].GetValue())
}

func TestInput_Parse_NilInput(t *testing.T) {
	var i *Input

	err := i.Parse(t.Context())

	require.ErrorIs(t, err, ErrInputIsNil)
}

func TestInput_String(t *testing.T) {
	i := &Input{name: "test"}

	assert.Equal(t, "test", i.String())
}

func TestInput_String_Nil(t *testing.T) {
	var i *Input

	assert.Empty(t, i.String())
}

func TestNewInput_Parse_ListKind(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{
			"domain:google.com",
			"keyword:example",
			"regexp:^test\\.com$",
			"full:exact.com",
		},
	}

	i := NewInput(cfg, nil, time.Second)

	require.NoError(t, i.Parse(t.Context()))
	require.Len(t, i.categories, 1)
	assert.Equal(t, "ALL", i.categories[0].name)
	require.Len(t, i.categories[0].domains, 4)
}

func TestNewInput_Parse_ListKind_DefaultType(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"google.com"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.NoError(t, i.Parse(t.Context()))
	require.Len(t, i.categories[0].domains, 1)
	assert.Equal(t, Domain_domain, i.categories[0].domains[0].GetType())
}

func TestNewInput_Parse_ListKind_WithAttributes(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"google.com @ads @tracking"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.NoError(t, i.Parse(t.Context()))

	domains := i.categories[0].domains

	require.Len(t, domains, 1)
	require.Len(t, domains[0].GetAttributes(), 2)
	assert.Equal(t, "ads", domains[0].GetAttributes()[0].GetKey())
}

func TestNewInput_Parse_ListKind_Comments(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{
			"google.com # this is a comment",
			"example.com",
		},
	}

	i := NewInput(cfg, nil, time.Second)

	require.NoError(t, i.Parse(t.Context()))
	require.Len(t, i.categories[0].domains, 2)
}

func TestNewInput_Parse_ListKind_OnlyComment(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"# just a comment"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.NoError(t, i.Parse(t.Context()))
	assert.Empty(t, i.categories[0].domains)
}

func TestNewInput_Parse_ListKind_InvalidRegexp(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"regexp:[invalid"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.Error(t, i.Parse(t.Context()))
}

func TestNewInput_Parse_ListKind_InvalidDomainChars(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"domain:test@domain"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.Error(t, i.Parse(t.Context()))
}

func TestNewInput_Parse_ListKind_EmptyRule(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"domain:"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.Error(t, i.Parse(t.Context()))
}

func TestNewInput_Parse_ListKind_InvalidAttribute(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"google.com @invalid@attr"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.Error(t, i.Parse(t.Context()))
}

func TestNewInput_Domains_WithFilter(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"google.com", "example.com"},
	}

	i := NewInput(cfg, nil, time.Second)

	require.NoError(t, i.Parse(t.Context()))

	domains, err := i.Domains(nil, nil)

	require.NoError(t, err)
	require.Len(t, domains, 2)
}

func TestParseDomain_Keyword(t *testing.T) {
	c := &InputCategory{}

	d, err := c.parseDomain("keyword:example")

	require.NoError(t, err)
	assert.Equal(t, Domain_substr, d.GetType())
	assert.Equal(t, "example", d.GetValue())
}

func TestParseDomain_Regexp(t *testing.T) {
	c := &InputCategory{}

	d, err := c.parseDomain("regexp:^google\\.com$")

	require.NoError(t, err)
	assert.Equal(t, Domain_regex, d.GetType())
}

func TestParseDomain_Full(t *testing.T) {
	c := &InputCategory{}

	d, err := c.parseDomain("full:example.com")

	require.NoError(t, err)
	assert.Equal(t, Domain_full, d.GetType())
}

func TestParseDomain_Domain(t *testing.T) {
	c := &InputCategory{}

	d, err := c.parseDomain("domain:Example.COM")

	require.NoError(t, err)
	assert.Equal(t, "example.com", d.GetValue())
}

func TestParseDomain_UnknownType(t *testing.T) {
	c := &InputCategory{}

	_, err := c.parseDomain("unknown:value")

	require.Error(t, err)
}

func TestParseDomain_CommentOnly(t *testing.T) {
	c := &InputCategory{}

	_, err := c.parseDomain("# just a comment")

	require.ErrorIs(t, err, ErrCommentLine)
}

func TestParseDomain_Keyword_Invalid(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{"at", "keyword:test@domain"},
		{"underscore", "keyword:domain_with_underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &InputCategory{}

			_, err := c.parseDomain(tt.rule)

			require.ErrorIs(t, err, ErrInvalidKeyword)
		})
	}
}

func TestParseDomain_Domain_Invalid(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{"leading hyphen label", "domain:-example.com"},
		{"trailing hyphen label", "domain:example-.com"},
		{"empty label", "domain:example..com"},
		{"leading dot", "domain:.example.com"},
		{"trailing dot", "domain:example.com."},
		{"label too long", "domain:" + strings.Repeat("a", maxLabelLen+1) + ".com"},
		{
			"domain too long",
			"domain:" + strings.Repeat(
				"a",
				maxLabelLen,
			) + "." + strings.Repeat(
				"b",
				maxLabelLen,
			) + "." + strings.Repeat(
				"c",
				maxLabelLen,
			) + "." + strings.Repeat(
				"d",
				maxLabelLen,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &InputCategory{}

			_, err := c.parseDomain(tt.rule)

			require.ErrorIs(t, err, ErrInvalidDomain)
		})
	}
}

func TestParseDomain_Full_Invalid(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{"leading hyphen label", "full:-example.com"},
		{"trailing hyphen label", "full:example-.com"},
		{"empty label", "full:example..com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &InputCategory{}

			_, err := c.parseDomain(tt.rule)

			require.ErrorIs(t, err, ErrInvalidDomain)
		})
	}
}

func TestParseDomain_Keyword_WithDotAllowed(t *testing.T) {
	c := &InputCategory{}

	d, err := c.parseDomain("keyword:example.com")

	require.NoError(t, err)
	assert.Equal(t, "example.com", d.GetValue())
}

func TestValidateDomainName(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"example.com", true},
		{"test-domain.co.uk", true},
		{"a.b.c", true},
		{"xn--bcher-kva.example", true},
		{strings.Repeat("a", maxLabelLen) + ".com", true},
		{
			strings.Repeat(
				"a",
				maxLabelLen,
			) + "." + strings.Repeat(
				"b",
				maxLabelLen,
			) + "." + strings.Repeat(
				"c",
				maxLabelLen,
			) + "." + strings.Repeat(
				"d",
				maxLabelLen-2,
			),
			true,
		},
		{"", false},
		{"example..com", false},
		{".example.com", false},
		{"example.com.", false},
		{"-example.com", false},
		{"example-.com", false},
		{"UPPERCASE", false},
		{"test@domain", false},
		{"domain_with_underscore", false},
		{strings.Repeat("a", maxLabelLen+1) + ".com", false},
		{
			strings.Repeat(
				"a",
				maxLabelLen,
			) + "." + strings.Repeat(
				"b",
				maxLabelLen,
			) + "." + strings.Repeat(
				"c",
				maxLabelLen,
			) + "." + strings.Repeat(
				"d",
				maxLabelLen,
			),
			false,
		},
	}

	for _, tt := range tests {
		got := validateDomainName(tt.input)
		assert.Equal(t, tt.valid, got, "validateDomainName(%q)", tt.input)
	}
}

func TestValidateDomainChars(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"example.com", true},
		{"test-domain.co.uk", true},
		{"", false},
		{"test@domain", false},
		{"domain_with_underscore", false},
		{"UPPERCASE", false},
	}

	for _, tt := range tests {
		got := validateDomainChars(tt.input)
		assert.Equal(t, tt.valid, got, "validateDomainChars(%q)", tt.input)
	}
}

func TestValidateAttrChars(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"ads", true},
		{"tracking123", true},
		{"", false},
		{"ads!", true},
		{"ads@", false},
	}

	for _, tt := range tests {
		got := validateAttrChars(tt.input)
		assert.Equal(t, tt.valid, got, "validateAttrChars(%q)", tt.input)
	}
}

func TestNewInput_Parse_GeofileKind_Nonexistent(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindGeo,
		Path: "nonexistent.dat",
	}

	i := NewInput(cfg, nil, time.Second)

	require.Error(t, i.Parse(t.Context()))
}

func TestNewInput_Parse_TxtKind_Nonexistent(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindTxt,
		Path: "nonexistent.txt",
	}

	i := NewInput(cfg, nil, time.Second)

	require.Error(t, i.Parse(t.Context()))
}
