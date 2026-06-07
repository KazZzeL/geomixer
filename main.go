package main

//go:generate go run . schema -o ./jsonschema -f geomixer.schema.json

import "github.com/KazZzeL/geomixer/cmd"

func main() {
	cmd.Execute()
}
