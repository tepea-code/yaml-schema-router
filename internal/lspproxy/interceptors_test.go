package lspproxy

import (
	"encoding/json"
	"testing"
)

func TestInterceptWorkspaceConfiguration(t *testing.T) {
	t.Run("preserves existing editor schemas when no schemas detected", func(t *testing.T) {
		p := &Proxy{
			schemaState: make(map[string]string),
		}

		editorYAML := map[string]any{
			"hover":      true,
			"completion": true,
			"validation": true,
			"schemas": map[string]any{
				"https://json.schemastore.org/github-workflow.json": []any{".github/workflows/*.yaml"},
				"https://example.com/custom-schema.json":            []any{"configs/*.yaml"},
			},
		}
		rawResult, err := json.Marshal([]any{editorYAML})
		if err != nil {
			t.Fatal(err)
		}

		msg := &BaseRPC{
			JSONRPC: "2.0",
			ID:      1,
			Result:  rawResult,
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}

		modified := p.interceptWorkspaceConfiguration(msg, payload)

		var resMsg BaseRPC
		if err := json.Unmarshal(modified, &resMsg); err != nil {
			t.Fatalf("failed to unmarshal modified message: %v", err)
		}

		var res []map[string]any
		if err := json.Unmarshal(resMsg.Result, &res); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		schemas, ok := res[0]["schemas"].(map[string]any)
		if !ok {
			t.Fatalf("expected schemas map in result, got: %v", res[0]["schemas"])
		}

		if _, exists := schemas["https://json.schemastore.org/github-workflow.json"]; !exists {
			t.Errorf("expected github-workflow schema to be preserved, got: %v", schemas)
		}
		if _, exists := schemas["https://example.com/custom-schema.json"]; !exists {
			t.Errorf("expected custom-schema to be preserved, got: %v", schemas)
		}
	})

	t.Run("merges router-detected schemas with existing editor schemas", func(t *testing.T) {
		p := &Proxy{
			schemaState: map[string]string{
				"file:///workspace/deploy.yaml": "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/v1.30.0/all.json",
			},
		}

		editorYAML := map[string]any{
			"hover":      true,
			"completion": true,
			"validation": true,
			"schemas": map[string]any{
				"https://json.schemastore.org/github-workflow.json": []any{".github/workflows/*.yaml"},
			},
		}
		rawResult, err := json.Marshal([]any{editorYAML})
		if err != nil {
			t.Fatal(err)
		}

		msg := &BaseRPC{
			JSONRPC: "2.0",
			ID:      1,
			Result:  rawResult,
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}

		modified := p.interceptWorkspaceConfiguration(msg, payload)

		var resMsg BaseRPC
		if err := json.Unmarshal(modified, &resMsg); err != nil {
			t.Fatalf("failed to unmarshal modified message: %v", err)
		}

		var res []map[string]any
		if err := json.Unmarshal(resMsg.Result, &res); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		schemas, ok := res[0]["schemas"].(map[string]any)
		if !ok {
			t.Fatalf("expected schemas map in result, got: %v", res[0]["schemas"])
		}

		if _, exists := schemas["https://json.schemastore.org/github-workflow.json"]; !exists {
			t.Errorf("expected editor schema to be preserved, got: %v", schemas)
		}
		if _, exists := schemas["https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/v1.30.0/all.json"]; !exists {
			t.Errorf("expected detected k8s schema to be present, got: %v", schemas)
		}
	})

	t.Run("injects feature defaults and schemas when editor returns nil", func(t *testing.T) {
		p := &Proxy{
			schemaState: map[string]string{
				"file:///workspace/deploy.yaml": "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/v1.30.0/all.json",
			},
		}

		rawResult, err := json.Marshal([]any{nil})
		if err != nil {
			t.Fatal(err)
		}

		msg := &BaseRPC{
			JSONRPC: "2.0",
			ID:      1,
			Result:  rawResult,
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}

		modified := p.interceptWorkspaceConfiguration(msg, payload)

		var resMsg BaseRPC
		if err := json.Unmarshal(modified, &resMsg); err != nil {
			t.Fatalf("failed to unmarshal modified message: %v", err)
		}

		var res []map[string]any
		if err := json.Unmarshal(resMsg.Result, &res); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if res[0]["hover"] != true || res[0]["completion"] != true || res[0]["validation"] != true {
			t.Errorf("expected feature defaults to be injected, got: %v", res[0])
		}

		schemas, ok := res[0]["schemas"].(map[string]any)
		if !ok || len(schemas) != 1 {
			t.Errorf("expected 1 injected schema, got: %v", res[0]["schemas"])
		}
	})
}
