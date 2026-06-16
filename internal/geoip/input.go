package geoip

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"
	unsafe "unsafe"

	"go4.org/netipx"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/utils/fetcher"
)

type InputCategory struct {
	name        string
	ipv4Builder *netipx.IPSetBuilder
	ipv6Builder *netipx.IPSetBuilder
	ipv4Set     *netipx.IPSet
	ipv6Set     *netipx.IPSet
}

type Input struct {
	name       string
	kind       config.InputKind
	url        string
	path       string
	list       []string
	categories []*InputCategory
	mu         sync.Mutex
	hc         *http.Client
	ht         time.Duration
}

type IPType int

const (
	IPv4 IPType = iota
	IPv6
)

const (
	defaultCategoryName string = "all"

	defaultCategoriesLen int = 128
)

var (
	ErrInputIsNil        = errors.New("input object is nil")
	ErrInvalidIPType     = errors.New("invalid IP type")
	ErrInvalidIP         = errors.New("invalid IP address")
	ErrInvalidIPLength   = errors.New("invalid IP address length")
	ErrInvalidCIDR       = errors.New("invalid CIDR")
	ErrInvalidPrefix     = errors.New("invalid prefix")
	ErrExtractedNoIPSets = errors.New("extracted 0 ip sets from input")
	ErrCommentLine       = errors.New("comment line")
)

func NewInput(cfg *config.Input, hc *http.Client, ht time.Duration) *Input {
	categories := make([]*InputCategory, 0, defaultCategoriesLen)

	return &Input{
		name:       cfg.Name,
		kind:       cfg.Kind,
		url:        cfg.URL,
		path:       cfg.Path,
		list:       cfg.List,
		categories: categories,
		hc:         hc,
		ht:         ht,
	}
}

func (i *Input) String() string {
	if i == nil {
		return ""
	}

	return i.name
}

func (i *Input) Parse(ctx context.Context) error {
	if i == nil {
		return ErrInputIsNil
	}

	var err error
	switch i.kind {
	case config.InputKindGeo:
		err = i.parseGeo(ctx)
	case config.InputKindTxt:
		err = i.parseTxt(ctx)
	case config.InputKindLst:
		err = i.parseLst(ctx)
	}

	if err != nil {
		return fmt.Errorf("%s: %w", i.kind, err)
	}

	return nil
}

func (i *Input) IPSets(include, exclude []string) ([]*netipx.IPSet, []*netipx.IPSet, error) {
	if i == nil {
		return nil, nil, ErrInputIsNil
	}

	ipV4Sets := make([]*netipx.IPSet, 0, len(i.categories))
	ipV6Sets := make([]*netipx.IPSet, 0, len(i.categories))

	for _, c := range i.categories {
		if len(include) != 0 && !slices.Contains(include, c.name) {
			continue
		}
		if len(exclude) != 0 && slices.Contains(exclude, c.name) {
			continue
		}
		ipV4Sets = append(ipV4Sets, c.ipv4Set)
		ipV6Sets = append(ipV6Sets, c.ipv6Set)
	}

	if len(ipV4Sets) == 0 && len(ipV6Sets) == 0 {
		return nil, nil, fmt.Errorf("%w: %s", ErrExtractedNoIPSets, i.name)
	}

	return ipV4Sets, ipV6Sets, nil
}

func (i *Input) addCategory(name string) *InputCategory {
	c := &InputCategory{
		name:        strings.ToUpper(strings.TrimSpace(name)),
		ipv4Builder: new(netipx.IPSetBuilder),
		ipv6Builder: new(netipx.IPSetBuilder),
	}

	i.mu.Lock()
	i.categories = append(i.categories, c)
	i.mu.Unlock()

	return c
}

func (i *Input) parseGeo(ctx context.Context) error {
	var (
		data []byte
		err  error
	)

	switch {
	case i.path != "":
		data, err = fetcher.Path(i.path)
	case i.url != "":
		data, err = fetcher.URL(ctx, i.hc, i.ht, i.url)
	}
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	var geoip GeoIP
	if err := proto.Unmarshal(data, &geoip); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	eg, egCtx := errgroup.WithContext(ctx)

	for _, category := range geoip.GetCategories() {
		eg.Go(func() error {
			c := i.addCategory(category.GetName())

			for _, cidr := range category.GetCidr() {
				select {
				case <-egCtx.Done():
					return egCtx.Err()
				default:
					if err := c.addRawCIDR(cidr.GetIp(), int(cidr.GetPrefix())); err != nil {
						return fmt.Errorf("add cidr: %w", err)
					}
				}
			}

			if err := c.buildIPSets(); err != nil {
				return fmt.Errorf("build ipsets: %w", err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("parse categories: %w", err)
	}

	return nil
}

func (i *Input) parseTxt(ctx context.Context) error {
	var (
		data []byte
		err  error
	)

	switch {
	case i.path != "":
		data, err = fetcher.Path(i.path)
	case i.url != "":
		data, err = fetcher.URL(ctx, i.hc, i.ht, i.url)
	}
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	i.list = strings.Split(unsafe.String(unsafe.SliceData(data), len(data)), "\n")

	return i.parseLst(ctx)
}

func (i *Input) parseLst(ctx context.Context) error {
	c := i.addCategory(defaultCategoryName)

	for ind := range i.list {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.addCIDR(i.list[ind]); err != nil {
				return fmt.Errorf("add cidr: %w", err)
			}
		}
	}

	if err := c.buildIPSets(); err != nil {
		return fmt.Errorf("build ipsets: %w", err)
	}

	return nil
}

func (c *InputCategory) buildIPSets() error {
	if c.ipv4Builder != nil && c.ipv4Set == nil {
		ipv4set, err := c.ipv4Builder.IPSet()
		if err != nil {
			return fmt.Errorf("ipv4 set: %w", err)
		}
		c.ipv4Set = ipv4set
	}

	if c.ipv6Builder != nil && c.ipv6Set == nil {
		ipv6set, err := c.ipv6Builder.IPSet()
		if err != nil {
			return fmt.Errorf("ipv6 set: %w", err)
		}
		c.ipv6Set = ipv6set
	}

	return nil
}

func (c *InputCategory) addRawCIDR(rawIP []byte, prefixBits int) error {
	addr, ok := netip.AddrFromSlice(rawIP)
	if !ok {
		return ErrInvalidIP
	}
	addr = addr.Unmap()
	prefix := netip.PrefixFrom(addr, prefixBits)

	switch {
	case addr.Is4():
		c.ipv4Builder.AddPrefix(prefix)
	case addr.Is6():
		c.ipv6Builder.AddPrefix(prefix)
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (c *InputCategory) addCIDR(cidr string) error {
	prefix, ipType, err := c.parseCIDR(cidr)
	if err != nil {
		if errors.Is(err, ErrCommentLine) {
			return nil
		}

		return fmt.Errorf("parse cidr %s: %w", cidr, err)
	}

	switch ipType {
	case IPv4:
		c.ipv4Builder.AddPrefix(*prefix)
	case IPv6:
		c.ipv6Builder.AddPrefix(*prefix)
	default:
		return ErrInvalidIPType
	}

	return nil
}

func (*InputCategory) parseCIDR(src string) (prefix *netip.Prefix, ipType IPType, err error) {
	src, _, _ = strings.Cut(src, "#")
	src, _, _ = strings.Cut(src, "//")
	src, _, _ = strings.Cut(src, "/*")
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, IPv4, ErrCommentLine
	}

	if strings.Contains(src, "/") {
		p, err := netip.ParsePrefix(src)
		if err != nil {
			return nil, IPv4, ErrInvalidCIDR
		}
		addr := p.Addr().Unmap()
		if addr.Is4() && strings.Contains(src, "::") {
			return nil, IPv4, ErrInvalidCIDR
		}
		switch {
		case addr.Is4():
			return &p, IPv4, nil
		case addr.Is6():
			return &p, IPv6, nil
		default:
			return nil, IPv4, ErrInvalidIPLength
		}
	} else {
		ip, err := netip.ParseAddr(src)
		if err != nil {
			return nil, IPv4, ErrInvalidIP
		}
		ip = ip.Unmap()
		switch {
		case ip.Is4():
			prefix := netip.PrefixFrom(ip, 32)
			return &prefix, IPv4, nil
		case ip.Is6():
			prefix := netip.PrefixFrom(ip, 128)
			return &prefix, IPv6, nil
		default:
			return nil, IPv4, ErrInvalidIPLength
		}
	}
}
