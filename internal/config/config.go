package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type InputKind string
type StepAction string

const (
	InputKindGeo InputKind = "geofile"
	InputKindTxt InputKind = "plain"
	InputKindLst InputKind = "list"

	StepActionAdd StepAction = "add"
	StepActionDel StepAction = "del"
)

var (
	ErrParsing           = errors.New("config parsing error")
	ErrEmptyConfig       = errors.New("no domains or subnets provided")
	ErrEmptyBlock        = errors.New("no configuration provided")
	ErrMissingParam      = errors.New("parameter is not provided")
	ErrMissingStepInput  = errors.New("missing step input in inputs")
	ErrUnknownInputKind  = errors.New("unknown input kind")
	ErrUnknownStepAction = errors.New("unknown step action")
	ErrAmbiguousStepInEx = errors.New("exclude and include have common categories")
	ErrIgnoredAllIPTypes = errors.New("cannot ignore all types of ip")
)

type (
	// Input describes a named data source.
	Input struct {
		Name string    `json:"name"           yaml:"name" jsonschema:"description=Unique input name referenced by steps"`
		Kind InputKind `json:"kind"           yaml:"kind" jsonschema:"description=Soure type,enum=geofile,enum=plain,enum=list"`
		URL  string    `json:"url,omitempty"  yaml:"url"  jsonschema:"description=Remote URL to fetch (required for geofile/plain),example=https://example.com/data.dat"`
		Path string    `json:"path,omitempty" yaml:"path" jsonschema:"description=Local filesystem path (alternative to url),example=./data.dat"`
		List []string  `json:"list,omitempty" yaml:"list" jsonschema:"description=Inline list of CIDRs or domain rules (required for list kind),example=10.0.0.0/8,example=example.com"`
	}

	// Options defines per-step transformations.
	Options struct {
		IgnoreIPv4 bool `json:"ignoreIPv4,omitempty" yaml:"ignoreIPv4" jsonschema:"description=Skip IPv4 CIDRs from geoip input"`
		IgnoreIPv6 bool `json:"ignoreIPv6,omitempty" yaml:"ignoreIPv6" jsonschema:"description=Skip IPv6 CIDRs from geoip input"`

		ResetAttributes  bool     `json:"resetAttributes,omitempty"  yaml:"resetAttributes"  jsonschema:"description=Clear all existing domain attributes before appending"`
		DeleteAttributes []string `json:"deleteAttributes,omitempty" yaml:"deleteAttributes" jsonschema:"description=Remove specific attributes by key,example=ads"`
		AppendAttributes []string `json:"appendAttributes,omitempty" yaml:"appendAttributes" jsonschema:"description=Add new boolean attributes,example=tracking"`
	}

	// Step is a single transformation: add or remove domains/CIDRs from an input.
	Step struct {
		Action  StepAction `json:"action"            yaml:"action"  jsonschema:"description=Operation to perform,enum=add,enum=del"`
		Input   string     `json:"input"             yaml:"input"   jsonschema:"description=Name of the input to use"`
		Include []string   `json:"include,omitempty" yaml:"include" jsonschema:"description=Only process categories in this list,example=CN"`
		Exclude []string   `json:"exclude,omitempty" yaml:"exclude" jsonschema:"description=Skip categories in this list,example=RU"`
		Options *Options   `json:"options,omitempty" yaml:"options" jsonschema:"description=Additional transformation options"`
	}

	// Category groups a set of steps into a named output category.
	Category struct {
		Name  string  `json:"name"  yaml:"name"  jsonschema:"description=Category name,example=CN"`
		Steps []*Step `json:"steps" yaml:"steps" jsonschema:"description=Ordered list of transformation steps"`
	}

	// Output defines a single output file and its categories.
	Output struct {
		Name       string      `json:"name"          yaml:"name"       jsonschema:"description=Output filename,example=geosite.dat"`
		Dir        *string     `json:"dir,omitempty" yaml:"dir"        jsonschema:"description=Output directory override,example=./dist"`
		Categories []*Category `json:"categories"    yaml:"categories" jsonschema:"description=List of categories to build"`
	}

	// Runner groups inputs and outputs for a single data type (geosite or geoip).
	Runner struct {
		Inputs  []*Input  `json:"inputs"  yaml:"inputs"  jsonschema:"description=Named data sources"`
		Outputs []*Output `json:"outputs" yaml:"outputs" jsonschema:"description=Output files to generate"`
	}

	// Config is the top-level configuration.
	Config struct {
		Geosite *Runner `json:"geosite,omitempty" yaml:"geosite" jsonschema:"description=Domain (geosite) rules configuration"`
		Geoip   *Runner `json:"geoip,omitempty"   yaml:"geoip"   jsonschema:"description=IP (geoip) rules configuration"`
	}
)

func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config open: %w", err)
	}

	parser := json.Unmarshal
	if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
		parser = yaml.Unmarshal
	}

	cfg := &Config{}
	if err := parser(data, &cfg); err != nil {
		return nil, fmt.Errorf("config parsing: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	// log.Println(*cfg.Geoip.Outputs[0].Categories[0].Steps[0])

	return cfg, nil
}

func (cfg *Config) validate() error {
	if cfg.Geosite == nil && cfg.Geoip == nil {
		return ErrEmptyConfig
	}

	check := func(r *Runner) error {
		if r == nil {
			return nil
		}

		if len(r.Inputs) == 0 {
			return fmt.Errorf("inputs: %w", ErrEmptyBlock)
		}

		if len(r.Outputs) == 0 {
			return fmt.Errorf("outputs: %w", ErrEmptyBlock)
		}

		inputs, err := validateInputs(r.Inputs)
		if err != nil {
			return fmt.Errorf("inputs: %w", err)
		}

		if err := validateOutputs(r.Outputs, inputs); err != nil {
			return fmt.Errorf("outputs: %w", err)
		}

		return nil
	}

	if err := check(cfg.Geosite); err != nil {
		return fmt.Errorf("geosite: %w", err)
	}

	if err := check(cfg.Geoip); err != nil {
		return fmt.Errorf("geoip: %w", err)
	}

	return nil
}

func validateInputs(inputs []*Input) (map[string]InputKind, error) {
	kinds := make(map[string]InputKind, len(inputs))

	for i, input := range inputs {
		if input.Name == "" {
			return nil, fmt.Errorf("[%d]: %w: param = name", i, ErrMissingParam)
		}

		switch input.Kind {
		case InputKindGeo, InputKindTxt:
			if input.URL == "" && input.Path == "" {
				return nil, fmt.Errorf("[%d]: %w: param = url & path", i, ErrMissingParam)
			}
		case InputKindLst:
			if len(input.List) == 0 {
				return nil, fmt.Errorf("[%d]: %w: param = list", i, ErrMissingParam)
			}
		default:
			return nil, fmt.Errorf("[%d]: %w: kind = %s", i, ErrUnknownInputKind, input.Kind)
		}

		kinds[input.Name] = input.Kind
	}

	return kinds, nil
}

func validateOutputs(outputs []*Output, inputs map[string]InputKind) error {
	for i, output := range outputs {
		if output.Name == "" {
			return fmt.Errorf("[%d]: %w: param = name", i, ErrMissingParam)
		}

		if len(output.Categories) == 0 {
			return fmt.Errorf("[%d]: %w: param = categories", i, ErrMissingParam)
		}

		for ii, category := range output.Categories {
			if category.Name == "" {
				return fmt.Errorf("[%d:%d]: %w: param = name", i, ii, ErrMissingParam)
			}

			if len(category.Steps) == 0 {
				return fmt.Errorf("[%d:%d]: %w: param = steps", i, ii, ErrMissingParam)
			}

			for iii, step := range category.Steps {
				switch step.Action {
				case StepActionAdd, StepActionDel:
				default:
					return fmt.Errorf(
						"[%d:%d:%d]: %w: action = %s",
						i,
						ii,
						iii,
						ErrUnknownStepAction,
						step.Action,
					)
				}

				kind, ok := inputs[step.Input]
				if !ok {
					return fmt.Errorf("step(%d:%d:%d): %w: input = %s", i, ii, iii, ErrMissingStepInput, step.Input)
				}

				if kind == InputKindGeo {
					for i := range step.Include {
						step.Include[i] = strings.ToLower(step.Include[i])
					}
					for i := range step.Exclude {
						step.Exclude[i] = strings.ToLower(step.Exclude[i])
					}

					if len(step.Include) != 0 && len(step.Exclude) != 0 {
						for _, v := range step.Include {
							if slices.Contains(step.Exclude, v) {
								return fmt.Errorf("step(%d:%d:%d): %w", i, ii, iii, ErrAmbiguousStepInEx)
							}
						}
					}
				} else {
					step.Include = nil
					step.Exclude = nil
				}

				if step.Options != nil && step.Options.IgnoreIPv4 && step.Options.IgnoreIPv6 {
					return fmt.Errorf("step(%d:%d:%d): %w", i, ii, iii, ErrIgnoredAllIPTypes)
				}
			}
		}
	}

	return nil
}
