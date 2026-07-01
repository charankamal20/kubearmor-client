package main

import (
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
)

func main() {
	b, _ := os.ReadFile("D:\\policy-block-file.yaml")
	js, _ := yaml.YAMLToJSON(b)
	fmt.Printf("JSON: %s\n", js)
}
