package geosite

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	sync "sync"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/utils/filer"
)

type (
	OutputDomain struct {
		domain *Domain
		plain  string
	}

	OutputStepOptions struct {
		resetAttributes  bool
		deleteAttributes []string
		appendAttributes []string
	}

	OutputStep struct {
		action  config.StepAction
		input   *Input
		include []string
		exclude []string
		options *OutputStepOptions
	}

	OutputCategory struct {
		name  string
		steps []*OutputStep
	}

	Output struct {
		name       string
		dir        string
		categories []*OutputCategory
		mtx        sync.Mutex
	}
)

var (
	ErrOutputIsNil        = errors.New("output object is nil")
	ErrNoDomainsCollected = errors.New("no domains collected")
)

func NewOutput(cfg *config.Output, defaultDir string, inputs map[string]*Input) *Output {
	categories := make([]*OutputCategory, 0, len(cfg.Categories))

	for _, c := range cfg.Categories {
		steps := make([]*OutputStep, 0, len(c.Steps))

		for _, s := range c.Steps {
			options := &OutputStepOptions{}
			if s.Options != nil {
				options.resetAttributes = s.Options.ResetAttributes
				options.deleteAttributes = s.Options.DeleteAttributes
				options.appendAttributes = s.Options.AppendAttributes
			}

			steps = append(steps, &OutputStep{
				action:  s.Action,
				input:   inputs[s.Input],
				include: s.Include,
				exclude: s.Exclude,
				options: options,
			})
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

		categories = append(categories, &OutputCategory{
			name:  c.Name,
			steps: steps,
		})
	}

	dir := defaultDir
	if cfg.Dir != nil {
		dir = *cfg.Dir
	}

	return &Output{
		name:       cfg.Name,
		dir:        dir,
		categories: categories,
	}
}

func (o *Output) String() string {
	if o == nil {
		return ""
	}

	return o.name
}

func (o *Output) Generate(ctx context.Context) error {
	if o == nil {
		return ErrOutputIsNil
	}

	geosite, err := o.buildGeoSite(ctx)
	if err != nil {
		return fmt.Errorf("build geosite (%s): %w", o.name, err)
	}

	data, err := proto.Marshal(geosite)
	if err != nil {
		return fmt.Errorf("marshall geosite: %w", err)
	}

	if err := filer.WriteFile(o.dir, o.name, data); err != nil {
		return fmt.Errorf("write geosite: %w", err)
	}

	log.Printf("✅ [geosite] %s --> %s", o.name, o.dir)

	return nil
}

func (o *Output) buildGeoSite(ctx context.Context) (*GeoSite, error) {
	geosite := new(GeoSite)
	geosite.Categories = make([]*Category, 0, len(o.categories))

	eg, egCtx := errgroup.WithContext(ctx)
	for _, c := range o.categories {
		eg.Go(func() error {
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			default:
				domains, err := c.buildDomains()
				if err != nil {
					return fmt.Errorf("build domains (%s): %w", c.name, err)
				}

				if len(domains) == 0 {
					return fmt.Errorf("check domains (%s): %w", c.name, ErrNoDomainsCollected)
				}

				o.mtx.Lock()
				geosite.Categories = append(geosite.Categories, &Category{
					Name:    c.name,
					Domains: domains,
				})
				o.mtx.Unlock()
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("build categories: %w", err)
	}

	slices.SortFunc(geosite.GetCategories(), func(i, j *Category) int {
		return strings.Compare(i.GetName(), j.GetName())
	})

	return geosite, nil
}

func (c *OutputCategory) buildDomains() ([]*Domain, error) {
	iDomains, err := c.collectDomains()
	if err != nil {
		return nil, fmt.Errorf("collect domains: %w", err)
	}

	final := make([]*OutputDomain, 0, len(iDomains))
	queue := make([]*OutputDomain, 0, len(iDomains))
	check := make(map[string]struct{})

	for _, d := range iDomains {
		switch d.domain.GetType() {
		case Domain_regex, Domain_substr:
			final = append(final, d)
		case Domain_domain, Domain_full:
			if d.domain.GetType() == Domain_domain {
				check[d.domain.GetValue()] = struct{}{}
			}
			if len(d.domain.GetAttributes()) == 0 {
				queue = append(queue, d)
			} else {
				final = append(final, d)
			}
		}
	}

	for _, q := range queue {
		hasParent := false
		value := q.domain.GetValue()
		if q.domain.GetType() == Domain_full {
			value = "." + value
		}

		for {
			var found bool
			_, value, found = strings.Cut(value, ".")
			if !found {
				break
			}
			if _, ok := check[value]; ok {
				hasParent = true
				break
			}
		}

		if !hasParent {
			final = append(final, q)
		}
	}

	slices.SortFunc(final, func(a, b *OutputDomain) int {
		return strings.Compare(a.plain, b.plain)
	})

	result := make([]*Domain, 0, len(final))
	for i := range final {
		result = append(result, final[i].domain)
	}

	return result, nil
}

func (c *OutputCategory) collectDomains() (map[string]*OutputDomain, error) {
	result := make(map[string]*OutputDomain, defaultDomainsLen)

	for _, s := range c.steps {
		domains, err := s.input.Domains(s.include, s.exclude)
		if err != nil {
			return nil, fmt.Errorf("build input domains: %w", err)
		}

		for _, d := range domains {
			od := d.toOutputDomain(s.options)

			switch s.action {
			case config.StepActionAdd:
				result[od.plain] = od
			case config.StepActionDel:
				// delete if exact match
				delete(result, od.plain)
			}
		}
	}

	return result, nil
}

func (d *Domain) toOutputDomain(options *OutputStepOptions) *OutputDomain {
	if options == nil {
		return &OutputDomain{
			domain: d,
			plain:  plainKey(d),
		}
	}

	attrs := applyAttrOptions(d.GetAttributes(), options)

	nd := &Domain{
		Type:       d.GetType(),
		Value:      d.GetValue(),
		Attributes: attrs,
	}

	return &OutputDomain{
		domain: nd,
		plain:  plainKey(nd),
	}
}

func applyAttrOptions(attrs []*Domain_Attribute, opts *OutputStepOptions) []*Domain_Attribute {
	if !opts.resetAttributes && len(opts.deleteAttributes) == 0 && len(opts.appendAttributes) == 0 {
		return attrs
	}

	result := make([]*Domain_Attribute, 0, len(attrs)+len(opts.appendAttributes))

	if !opts.resetAttributes {
		for _, a := range attrs {
			if slices.Contains(opts.deleteAttributes, a.GetKey()) {
				continue
			}
			result = append(result, a)
		}
	}

	for _, key := range opts.appendAttributes {
		result = append(result, &Domain_Attribute{
			Key:        key,
			TypedValue: &Domain_Attribute_BoolValue{BoolValue: true},
		})
	}

	return result
}

func plainKey(d *Domain) string {
	typ := d.GetType().String()
	val := d.GetValue()
	attrs := d.GetAttributes()

	if len(attrs) == 0 {
		return typ + ":" + val
	}

	var plain strings.Builder
	plain.Grow(len(typ) + 1 + len(val) + 2*len(attrs)*8)

	plain.WriteString(typ)
	plain.WriteByte(':')
	plain.WriteString(val)

	for i, a := range attrs {
		if i == 0 {
			plain.WriteByte(':')
		} else {
			plain.WriteByte(',')
		}
		plain.WriteByte('@')
		plain.WriteString(a.GetKey())
	}

	return plain.String()
}
