package geoip

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"slices"
	"strings"
	sync "sync"

	"go4.org/netipx"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/utils/filer"
)

type (
	OutputStepOptions struct {
		ignoreIPv4 bool
		ignoreIPv6 bool
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
	ErrOutputIsNil     = errors.New("output object is nil")
	ErrNoCIDRCollected = errors.New("no cidr collected")
)

func NewOutput(cfg *config.Output, defaultDir string, inputs map[string]*Input) *Output {
	categories := make([]*OutputCategory, 0, len(cfg.Categories))

	for _, c := range cfg.Categories {
		steps := make([]*OutputStep, 0, len(c.Steps))

		for _, s := range c.Steps {
			options := &OutputStepOptions{}
			if s.Options != nil {
				options.ignoreIPv4 = s.Options.IgnoreIPv4
				options.ignoreIPv6 = s.Options.IgnoreIPv6
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

	geoip, err := o.buildGeoIP(ctx)
	if err != nil {
		return fmt.Errorf("build geoip (%s): %w", o.name, err)
	}

	data, err := proto.Marshal(geoip)
	if err != nil {
		return fmt.Errorf("marshall geoip: %w", err)
	}

	if err := filer.WriteFile(o.dir, o.name, data); err != nil {
		return fmt.Errorf("write geoip: %w", err)
	}

	log.Printf("✅ [geoip] %s --> %s", o.name, o.dir)

	return nil
}

func (o *Output) buildGeoIP(ctx context.Context) (*GeoIP, error) {
	geoip := new(GeoIP)
	geoip.Categories = make([]*Category, 0, len(o.categories))

	eg, egCtx := errgroup.WithContext(ctx)
	for _, c := range o.categories {
		eg.Go(func() error {
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			default:
				cidr, err := c.buildCIDR()
				if err != nil {
					return fmt.Errorf("build cidr (%s): %w", c.name, err)
				}

				if len(cidr) == 0 {
					return fmt.Errorf("check cidr (%s): %w", c.name, ErrNoCIDRCollected)
				}

				o.mtx.Lock()
				geoip.Categories = append(geoip.Categories, &Category{
					Name: c.name,
					Cidr: cidr,
				})
				o.mtx.Unlock()
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("build categories: %w", err)
	}

	slices.SortFunc(geoip.GetCategories(), func(i, j *Category) int {
		return strings.Compare(i.GetName(), j.GetName())
	})

	return geoip, nil
}

func (c *OutputCategory) buildCIDR() ([]*CIDR, error) {
	prefixes, err := c.buildPrefixes()
	if err != nil {
		return nil, fmt.Errorf("build prefixes: %w", err)
	}

	cidr := make([]*CIDR, 0, len(prefixes))

	for _, prefix := range prefixes {
		cidr = append(cidr, &CIDR{
			Ip:     prefix.Addr().AsSlice(),
			Prefix: uint32(prefix.Bits()),
		})
	}

	return cidr, nil
}

func (c *OutputCategory) buildPrefixes() ([]netip.Prefix, error) {
	ipSetBuilder := netipx.IPSetBuilder{}

	for _, s := range c.steps {
		ipV4Sets, ipV6Sets, err := s.input.IPSets(s.include, s.exclude)
		if err != nil {
			return nil, fmt.Errorf("build input ipsets: %w", err)
		}

		ignoreIPv4 := s.options != nil && s.options.ignoreIPv4
		ignoreIPv6 := s.options != nil && s.options.ignoreIPv6

		ipSets := make([]*netipx.IPSet, 0, len(ipV4Sets)+len(ipV6Sets))

		if !ignoreIPv4 {
			ipSets = append(ipSets, ipV4Sets...)
		}
		if !ignoreIPv6 {
			ipSets = append(ipSets, ipV6Sets...)
		}

		for _, ipSet := range ipSets {
			switch s.action {
			case config.StepActionAdd:
				ipSetBuilder.AddSet(ipSet)
			case config.StepActionDel:
				ipSetBuilder.RemoveSet(ipSet)
			}
		}
	}

	ipSet, err := ipSetBuilder.IPSet()
	if err != nil {
		return nil, fmt.Errorf("build output ipset %w", err)
	}

	return ipSet.Prefixes(), nil
}
