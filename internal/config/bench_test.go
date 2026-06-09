package config

import (
	"testing"
)

func BenchmarkValidate(b *testing.B) {
	cfg := &Config{
		Geosite: &Runner{
			Inputs: []*Input{
				{Name: "in1", Kind: InputKindLst, List: []string{"example.com"}},
				{Name: "in2", Kind: InputKindLst, List: []string{"google.com"}},
			},
			Outputs: []*Output{
				{
					Name: "out.dat",
					Categories: []*Category{
						{
							Name: "CN",
							Steps: []*Step{
								{Action: StepActionAdd, Input: "in1"},
								{Action: StepActionDel, Input: "in2"},
							},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for range b.N {
		_ = validate(cfg)
	}
}

func BenchmarkValidateLarge(b *testing.B) {
	inputs := make([]*Input, 50)
	for i := range inputs {
		inputs[i] = &Input{
			Name: "in",
			Kind: InputKindLst,
			List: []string{"example.com"},
		}
	}
	inputs[0].Name = "in0"

	steps := make([]*Step, 100)
	for i := range steps {
		steps[i] = &Step{Action: StepActionAdd, Input: "in0"}
	}

	cfg := &Config{
		Geosite: &Runner{
			Inputs:  inputs,
			Outputs: []*Output{{Name: "out.dat", Categories: []*Category{{Name: "cat", Steps: steps}}}},
		},
	}

	b.ResetTimer()
	for range b.N {
		_ = validate(cfg)
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_, err := Parse("testdata/valid.json")
		if err != nil {
			b.Fatal(err)
		}
	}
}
