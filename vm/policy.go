// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pb "github.com/kubearmor/KubeArmor/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/yaml"
)

const (
	// KubeArmorPolicy is the Kind used for KubeArmor container policies
	KubeArmorPolicy = "KubeArmorPolicy"
	// KubeArmorHostPolicy is the Kind used for KubeArmor host policies
	KubeArmorHostPolicy = "KubeArmorHostPolicy"
	// KubeArmorNetworkPolicy is the Kind used for KubeArmor network policies
	KubeArmorNetworkPolicy = "KubeArmorNetworkPolicy"
)

// PolicyOptions are optional configuration for kArmor vm policy
type PolicyOptions struct {
	GRPC string
}

func sendPolicyOverGRPC(o PolicyOptions, policyEventData []byte, kind string) error {
	var (
		gRPC = ""
		resp *pb.Response
		err  error
	)

	if o.GRPC != "" {
		gRPC = o.GRPC
	} else {
		if val, ok := os.LookupEnv("KUBEARMOR_SERVICE"); ok {
			gRPC = val
		} else {
			gRPC = "localhost:32767"
		}
	}

	conn, err := grpc.NewClient(gRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewPolicyServiceClient(conn)

	req := pb.Policy{
		Policy: policyEventData,
	}

	switch kind {
	case KubeArmorPolicy:
		resp, err = client.ContainerPolicy(context.Background(), &req)
	case KubeArmorHostPolicy:
		resp, err = client.HostPolicy(context.Background(), &req)
	case KubeArmorNetworkPolicy:
		resp, err = client.NetworkPolicy(context.Background(), &req)
	}

	if err != nil {
		return fmt.Errorf("failed to send policy")
	}

	fmt.Printf("Policy %s \n", resp.Status)
	return nil
}

// PolicyHandling Function recives path to YAML file with the type of event and emits an Host Policy Event to KubeArmor gRPC/HTTP Server
func PolicyHandling(t string, path string, o PolicyOptions) error {
	fmt.Println("ADDING POLICY")
	var k struct {
		Kind string `json:"kind"`
	}

	policyFile, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}

	policies := strings.Split(string(policyFile), "---")

	for _, policy := range policies {
		re := regexp.MustCompile(`^\\s*$`)
		if matched := re.MatchString(policy); matched {
			continue
		}

		js, err := yaml.YAMLToJSON([]byte(policy))
		if err != nil {
			return err
		}

		err = json.Unmarshal(js, &k)
		if err != nil {
			return err
		}

		// Instead of unmarshaling into typed structs (which drops unknown fields
		// like matchPackages if compiled against an older types.go), we wrap the
		// raw JSON directly into the policy event envelope.
		var rawSpec json.RawMessage = js
		policyEvent := struct {
			Type   string          `json:"type"`
			Object json.RawMessage `json:"object"`
		}{
			Type:   t,
			Object: rawSpec,
		}

		policyEventData, err := json.Marshal(policyEvent)
		if err != nil {
			return err
		}

		// Systemd mode, hence send policy over gRPC
		if err = sendPolicyOverGRPC(o, policyEventData, k.Kind); err != nil {
			return err
		}

	}

	return nil
}
