package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_JSON(t *testing.T) {
	t.Parallel()

	cfg, err := Parse("testdata/valid.json")
	require.NoError(t, err)
	require.NotNil(t, cfg.Geosite)
	require.NotNil(t, cfg.Geoip)
}

func TestParse_YAML(t *testing.T) {
	t.Parallel()

	cfg, err := Parse("testdata/valid.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg.Geosite)
}

func TestParse_EmptyConfig(t *testing.T) {
	t.Parallel()

	_, err := Parse("testdata/empty.json")
	require.Error(t, err)
}

func TestValidate_EmptyInputs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{},
			Outputs: []*Output{
				{
					Name:       "out",
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_EmptyOutputs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs:  []*Input{{Name: "in", Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_MissingInputName(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{
				{
					Name:       "out",
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_MissingURLAndPath(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindGeo}},
			Outputs: []*Output{
				{
					Name:       "out",
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_MissingList(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindLst}},
			Outputs: []*Output{
				{
					Name:       "out",
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_UnknownKind(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKind("unknown"), URL: "http://example.com"}},
			Outputs: []*Output{
				{
					Name:       "out",
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_UnknownAction(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{
				{
					Name: "out",
					Categories: []*Category{
						{Name: "cat", Steps: []*Step{{Action: StepAction("unknown"), Input: "in"}}},
					},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_MissingStepInput(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{
				{
					Name: "out",
					Categories: []*Category{
						{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "nonexistent"}}},
					},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_AmbiguousIncludeExclude(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindGeo, URL: "http://example.com"}},
			Outputs: []*Output{
				{
					Name: "out",
					Categories: []*Category{
						{
							Name: "cat",
							Steps: []*Step{
								{
									Action:  StepActionAdd,
									Input:   "in",
									Include: []string{"common"},
									Exclude: []string{"common"},
								},
							},
						},
					},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_NonGeoIncludeExcludeCleared(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindTxt, URL: "http://example.com"}},
			Outputs: []*Output{
				{
					Name: "out",
					Categories: []*Category{
						{
							Name: "cat",
							Steps: []*Step{
								{Action: StepActionAdd, Input: "in", Include: []string{"x"}, Exclude: []string{"y"}},
							},
						},
					},
				},
			},
		},
	}
	require.NoError(t, cfg.validate())
}

func TestValidate_DefaultOptions(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{
				{
					Name:       "out",
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.NoError(t, cfg.validate())
}

func TestValidate_IgnoredAllIPTypes(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{
				{
					Name: "out",
					Categories: []*Category{
						{
							Name: "cat",
							Steps: []*Step{
								{
									Action:  StepActionAdd,
									Input:   "in",
									Options: &Options{IgnoreIPv4: true, IgnoreIPv6: true},
								},
							},
						},
					},
				},
			},
		},
	}
	require.Error(t, cfg.validate())
}

func TestValidate_OutputDir(t *testing.T) {
	dir := "custom-dir"
	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{{Name: "in", Kind: InputKindLst, List: []string{"example.com"}}},
			Outputs: []*Output{
				{
					Name:       "out",
					Dir:        &dir,
					Categories: []*Category{{Name: "cat", Steps: []*Step{{Action: StepActionAdd, Input: "in"}}}},
				},
			},
		},
	}
	require.NoError(t, cfg.validate())
	assert.Equal(t, "custom-dir", *cfg.Geosite.Outputs[0].Dir)
}
