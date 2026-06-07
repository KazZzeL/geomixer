package geoip

import (
	"context"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go4.org/netipx"

	"github.com/KazZzeL/geomixer/internal/config"
)

func TestInput_IPSets_NilInput(t *testing.T) {
	var i *Input
	_, _, err := i.IPSets(nil, nil)
	require.ErrorIs(t, err, ErrInputIsNil)
}

func TestInput_IPSets_NoCategories(t *testing.T) {
	i := &Input{name: "test", categories: []*InputCategory{}}
	_, _, err := i.IPSets(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracted 0 ip sets")
}

func TestInput_addCIDR_IPv4(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("10.0.0.0/8"))
	require.NoError(t, c.buildIPSets())
	require.NotNil(t, c.ipv4Set)
}

func TestInput_addCIDR_IPv6(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("2001:db8::/32"))
	require.NoError(t, c.buildIPSets())
	require.NotNil(t, c.ipv6Set)
}

func TestInput_addCIDR_SingleIP(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("192.168.1.1"))
	require.NoError(t, c.buildIPSets())
	prefixes := c.ipv4Set.Prefixes()
	require.Len(t, prefixes, 1)
	assert.Equal(t, 32, prefixes[0].Bits())
}

func TestInput_addCIDR_IPv6Single(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("::1"))
	require.NoError(t, c.buildIPSets())
	prefixes := c.ipv6Set.Prefixes()
	require.Len(t, prefixes, 1)
	assert.Equal(t, 128, prefixes[0].Bits())
}

func TestInput_addCIDR_CommentHash(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("10.0.0.0/8 # private network"))
	require.NoError(t, c.buildIPSets())
	require.NotNil(t, c.ipv4Set)
	prefixes := c.ipv4Set.Prefixes()
	require.Len(t, prefixes, 1)
	assert.Equal(t, "10.0.0.0/8", prefixes[0].String())
}

func TestInput_addCIDR_CommentDoubleSlash(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("10.0.0.0/8 // private network"))
	require.NoError(t, c.buildIPSets())
	require.NotNil(t, c.ipv4Set)
	prefixes := c.ipv4Set.Prefixes()
	require.Len(t, prefixes, 1)
	assert.Equal(t, "10.0.0.0/8", prefixes[0].String())
}

func TestInput_addCIDR_CommentBlock(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("10.0.0.0/8 /* private */"))
	require.NoError(t, c.buildIPSets())
	prefixes := c.ipv4Set.Prefixes()
	require.Len(t, prefixes, 1)
	assert.Equal(t, "10.0.0.0/8", prefixes[0].String())
}

func TestInput_addCIDR_OnlyCommentLine(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("# just a comment"))
}

func TestInput_addCIDR_InvalidCIDR(t *testing.T) {
	c := &InputCategory{}
	require.Error(t, c.addCIDR("not-a-cidr"))
}

func TestInput_addCIDR_UnmapIPv4Normalized(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("10.0.0.0/8"))
	require.NoError(t, c.buildIPSets())
	require.NotNil(t, c.ipv4Set)
}

func TestNewInput_Parse_ListKind(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"10.0.0.0/8", "192.168.0.0/16"},
	}
	i := NewInput(cfg, nil, time.Second)
	require.NoError(t, i.Parse(context.Background()))
	require.Len(t, i.categories, 1)
	assert.Equal(t, "all", i.categories[0].name)
}

func TestNewInput_Parse_ListKind_WithComments(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"10.0.0.0/8", "# comment", "192.168.0.0/16", "// another comment"},
	}
	i := NewInput(cfg, nil, time.Second)
	require.NoError(t, i.Parse(context.Background()))
	_, _, err := i.IPSets(nil, nil)
	require.NoError(t, err)
}

func TestNewInput_IPSets_FilterInclude(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"10.0.0.0/8"},
	}
	i := NewInput(cfg, nil, time.Second)
	require.NoError(t, i.Parse(context.Background()))
	_, _, err := i.IPSets([]string{"nonexistent"}, nil)
	require.Error(t, err)
	assert.Equal(t, "extracted 0 ip sets from input: test", err.Error())
}

func TestNewInput_IPSets_FilterExclude(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
		List: []string{"10.0.0.0/8"},
	}
	i := NewInput(cfg, nil, time.Second)
	require.NoError(t, i.Parse(context.Background()))
	_, _, err := i.IPSets(nil, []string{"all"})
	require.Error(t, err)
	assert.Equal(t, "extracted 0 ip sets from input: test", err.Error())
}

func TestNewInput_IPSets_SeparateV4V6(t *testing.T) {
	i := &Input{
		name: "test",
		categories: []*InputCategory{
			{
				name:    "all",
				ipv4Set: mustParseIPSet("10.0.0.0/8"),
				ipv6Set: mustParseIPSet("2001:db8::/32"),
			},
		},
	}
	v4, v6, err := i.IPSets(nil, nil)
	require.NoError(t, err)
	require.Len(t, v4, 1)
	require.Len(t, v6, 1)
	require.NotNil(t, v4[0])
	require.NotNil(t, v6[0])
}

func TestNewInput_IPSets_MultipleCategories(t *testing.T) {
	i := &Input{
		name: "test",
		categories: []*InputCategory{
			{
				name:    "cn",
				ipv4Set: mustParseIPSet("10.0.0.0/8"),
			},
			{
				name:    "us",
				ipv4Set: mustParseIPSet("192.168.0.0/16"),
			},
		},
	}
	v4, v6, err := i.IPSets(nil, nil)
	require.NoError(t, err)
	require.Len(t, v4, 2)
	require.Len(t, v6, 2)
}

func TestParse_NilInput(t *testing.T) {
	var i *Input
	err := i.Parse(context.Background())
	require.ErrorIs(t, err, ErrInputIsNil)
}

func TestInput_String_Nil(t *testing.T) {
	var i *Input
	assert.Empty(t, i.String())
}

func TestInput_String(t *testing.T) {
	i := &Input{name: "test"}
	assert.Equal(t, "test", i.String())
}

func TestParseCIDR_IPv4MappedIPv6(t *testing.T) {
	c := &InputCategory{
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}
	require.NoError(t, c.addCIDR("10.0.0.0/8"))
	require.NoError(t, c.buildIPSets())
	prefixes := c.ipv4Set.Prefixes()
	found := false
	for _, p := range prefixes {
		if p.String() == "10.0.0.0/8" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 10.0.0.0/8 in prefixes, got %v", prefixes)
}

func mustParseIPSet(cidrs ...string) *netipx.IPSet {
	b := new(netipx.IPSetBuilder)
	for _, c := range cidrs {
		prefix := netip.MustParsePrefix(c)
		b.AddPrefix(prefix)
	}
	set, err := b.IPSet()
	if err != nil {
		panic(err)
	}

	return set
}

func TestInput_Parse_EmptyListKind(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindLst,
	}
	i := NewInput(cfg, nil, time.Second)
	require.NoError(t, i.Parse(context.Background()))
	_, _, err := i.IPSets(nil, nil)
	require.NoError(t, err)
}

func TestInput_Parse_GeoKindNoData(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindGeo,
		Path: "nonexistent.dat",
	}
	i := NewInput(cfg, &http.Client{}, time.Second)
	require.Error(t, i.Parse(context.Background()))
}

func TestInput_Parse_TxtKindNoData(t *testing.T) {
	cfg := &config.Input{
		Name: "test",
		Kind: config.InputKindTxt,
		Path: "nonexistent.txt",
	}
	i := NewInput(cfg, &http.Client{}, time.Second)
	require.Error(t, i.Parse(context.Background()))
}
