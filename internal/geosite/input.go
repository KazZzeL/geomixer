package geosite

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	unsafe "unsafe"

	"google.golang.org/protobuf/proto"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/utils/fetcher"
)

type InputCategory struct {
	name    string
	domains []*Domain
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

const (
	defaultCategoryName string = "all"

	defaultCategoriesLen int = 128
	defaultDomainsLen    int = 1024
)

var (
	ErrInputIsNil         = errors.New("input object is nil")
	ErrCommentLine        = errors.New("comment line")
	ErrUnknownRuleType    = errors.New("unknown rule type")
	ErrEmptyRule          = errors.New("empty rule")
	ErrInvalidDomain      = errors.New("invalid domain")
	ErrInvalidDotless     = errors.New("substr in dotless rule should not contain a dot")
	ErrInvalidAttribute   = errors.New("invalid attribute")
	ErrExtractedNoDomains = errors.New("extracted 0 domains from input")
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

func (i *Input) Domains(include, exclude []string) ([]*Domain, error) {
	if i == nil {
		return nil, ErrInputIsNil
	}

	domains := make([]*Domain, 0, len(i.categories)*defaultDomainsLen)

	for _, c := range i.categories {
		if len(include) != 0 && !slices.Contains(include, c.name) {
			continue
		}
		if len(exclude) != 0 && slices.Contains(exclude, c.name) {
			continue
		}

		domains = append(domains, c.domains...)
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrExtractedNoDomains, i.name)
	}

	return domains, nil
}

func (i *Input) addCategory(name string) *InputCategory {
	c := &InputCategory{
		name:    strings.ToUpper(strings.TrimSpace(name)),
		domains: make([]*Domain, 0, defaultDomainsLen),
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

	var geosite GeoSite
	if err := proto.Unmarshal(data, &geosite); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	wg := sync.WaitGroup{}

	for _, category := range geosite.GetCategories() {
		wg.Go(func() {
			c := i.addCategory(category.GetName())
			c.domains = category.GetDomains()
		})
	}

	wg.Wait()

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
			if err := c.addPlain(i.list[ind]); err != nil {
				return fmt.Errorf("add: %w", err)
			}
		}
	}

	return nil
}

func (c *InputCategory) addPlain(p string) error {
	d, err := c.parseDomain(p)
	if err != nil {
		if errors.Is(err, ErrCommentLine) {
			return nil
		}

		return fmt.Errorf("rule: %w", err)
	}

	c.domains = append(c.domains, d)

	return nil
}

func (c *InputCategory) parseDomain(rule string) (*Domain, error) {
	rule, _, _ = strings.Cut(rule, "#")
	rule, _, _ = strings.Cut(rule, "//")
	rule, _, _ = strings.Cut(rule, "/*")
	rule = strings.TrimSpace(rule)
	if len(rule) == 0 {
		return nil, ErrCommentLine
	}

	domain := new(Domain)

	switch {
	case strings.HasPrefix(rule, "domain:"):
		domain.Type = Domain_domain
		rule = rule[7:]
	case strings.HasPrefix(rule, "regexp:"):
		domain.Type = Domain_regex
		rule = rule[7:]
	case strings.HasPrefix(rule, "full:"):
		domain.Type = Domain_full
		rule = rule[5:]
	case strings.HasPrefix(rule, "keyword:"):
		domain.Type = Domain_substr
		rule = rule[8:]
	case strings.HasPrefix(rule, "dotless:"):
		domain.Type = Domain_regex
		switch substr := rule[8:]; {
		case substr == "":
			rule = "^[^.]*$"
		case !strings.Contains(substr, "."):
			rule = "^[^.]*" + substr + "[^.]*$"
		default:
			return nil, ErrInvalidDotless
		}
	default:
		domain.Type = Domain_domain
	}

	parts := strings.Fields(rule)
	if len(parts) == 0 {
		return nil, ErrEmptyRule
	}

	switch domain.GetType() {
	case Domain_regex:
		if _, err := regexp.Compile(parts[0]); err != nil {
			return nil, fmt.Errorf("invalid regexp '%s': %w", parts[0], err)
		}

		domain.Value = parts[0]
	case Domain_substr, Domain_domain, Domain_full:
		value := strings.ToLower(parts[0])

		if !validateDomainChars(value) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidDomain, parts[0])
		}

		domain.Value = value
	}

	domain.Attributes = make([]*Domain_Attribute, 0, len(parts[1:]))
	for _, part := range parts[1:] {
		attr := part[1:]

		if part[0] != '@' || !validateAttrChars(attr) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidAttribute, part)
		}

		domain.Attributes = append(domain.Attributes, &Domain_Attribute{
			Key:        attr,
			TypedValue: &Domain_Attribute_BoolValue{BoolValue: true},
		})
	}

	return domain, nil
}

func validateDomainChars(domain string) bool {
	if domain == "" {
		return false
	}
	for i := range domain {
		c := domain[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '-' {
			continue
		}

		return false
	}

	return true
}

func validateAttrChars(attr string) bool {
	if attr == "" {
		return false
	}
	for i := range attr {
		c := attr[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '!' {
			continue
		}

		return false
	}

	return true
}
