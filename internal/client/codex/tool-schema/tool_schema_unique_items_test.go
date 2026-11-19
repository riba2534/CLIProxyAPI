package toolschema

import (
	"testing"

	"github.com/MoeForge/moego/ojson"
)

func TestNormalizeCodexToolSchemasStripsUniqueItems(t *testing.T) {
	input := []byte(`{
		"model": "gpt-6.2-astra",
		"tools": [{
			"type": "function",
			"name": "ask_user",
			"parameters": {
				"type": "object",
				"properties": {
					"question": {"type": "string"},
					"options": {
						"type": "array",
						"items": {"type": "string"},
						"uniqueItems": true
					}
				},
				"required": ["question"]
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	params := ojson.GetBytesNoCopy(out, "tools.0.parameters")

	if params.Get("properties.options.uniqueItems").Exists() {
		t.Fatalf("expected options.uniqueItems to be removed, got: %s", params.Get("properties.options").Raw)
	}
	if got := params.Get("properties.options.type").String(); got != "array" {
		t.Errorf("options.type = %q, want array", got)
	}
	if got := params.Get("properties.options.items.type").String(); got != "string" {
		t.Errorf("options.items.type = %q, want string", got)
	}
	if got := params.Get("properties.question.type").String(); got != "string" {
		t.Errorf("question.type = %q, want string", got)
	}
	if required := params.Get("required").Array(); len(required) != 1 || required[0].String() != "question" {
		t.Errorf("required = %v, want [question]", required)
	}
}

func TestNormalizeCodexToolSchemasStripsNestedUniqueItems(t *testing.T) {
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "nested",
			"parameters": {
				"type": "object",
				"uniqueItems": false,
				"properties": {
					"outer": {
						"type": "object",
						"properties": {
							"inner": {
								"type": "array",
								"uniqueItems": true,
								"items": {
									"type": "array",
									"uniqueItems": true,
									"items": {"type": "string"}
								}
							}
						}
					},
					"choice": {
						"anyOf": [
							{"type": "array", "uniqueItems": true, "items": {"type": "number"}},
							{"type": "null"}
						]
					},
					"referenced": {"$ref": "#/$defs/tag_list"}
				},
				"$defs": {
					"tag_list": {"type": "array", "uniqueItems": true, "items": {"type": "string"}}
				}
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	params := ojson.GetBytesNoCopy(out, "tools.0.parameters")

	paths := []string{
		"uniqueItems",
		"properties.outer.properties.inner.uniqueItems",
		"properties.outer.properties.inner.items.uniqueItems",
		"properties.choice.anyOf.0.uniqueItems",
		"$defs.tag_list.uniqueItems",
	}
	for _, path := range paths {
		if params.Get(path).Exists() {
			t.Errorf("expected %s to be removed, got: %s", path, params.Get(path).Raw)
		}
	}

	if got := params.Get("properties.outer.properties.inner.items.items.type").String(); got != "string" {
		t.Errorf("inner.items.items.type = %q, want string", got)
	}
	if got := params.Get("properties.choice.anyOf.1.type").String(); got != "null" {
		t.Errorf("choice.anyOf.1.type = %q, want null", got)
	}
	if got := params.Get("$defs.tag_list.items.type").String(); got != "string" {
		t.Errorf("$defs.tag_list.items.type = %q, want string", got)
	}
	if got := params.Get("properties.referenced.$ref").String(); got != "#/$defs/tag_list" {
		t.Errorf("referenced.$ref = %q, want #/$defs/tag_list", got)
	}
}

func TestNormalizeCodexToolSchemasKeepsUniqueItemsProperty(t *testing.T) {
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "user_data",
			"parameters": {
				"type": "object",
				"properties": {
					"uniqueItems": {
						"type": "array",
						"description": "A user field that happens to be named uniqueItems.",
						"items": {"type": "string"}
					}
				}
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	params := ojson.GetBytesNoCopy(out, "tools.0.parameters")

	prop := params.Get("properties.uniqueItems")
	if !prop.Exists() || !prop.IsObject() {
		t.Fatalf("expected the uniqueItems property to be preserved, got: %s", params.Raw)
	}
	if got := prop.Get("type").String(); got != "array" {
		t.Errorf("properties.uniqueItems.type = %q, want array", got)
	}
	if got := prop.Get("items.type").String(); got != "string" {
		t.Errorf("properties.uniqueItems.items.type = %q, want string", got)
	}

	doc := ojson.NewDoc(input)
	if NormalizeCodexToolSchemasDoc(doc) {
		t.Error("expected no changes for a schema whose only uniqueItems is a property name")
	}
}

func TestNormalizeCodexToolSchemasKeepsUniqueItemsPropertyWithOwnKeyword(t *testing.T) {
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "user_data",
			"parameters": {
				"type": "object",
				"properties": {
					"uniqueItems": {
						"type": "array",
						"items": {"type": "string"},
						"uniqueItems": true
					}
				}
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	params := ojson.GetBytesNoCopy(out, "tools.0.parameters")

	prop := params.Get("properties.uniqueItems")
	if !prop.Exists() || !prop.IsObject() {
		t.Fatalf("expected the uniqueItems property to be preserved, got: %s", params.Raw)
	}
	if prop.Get("uniqueItems").Exists() {
		t.Errorf("expected the uniqueItems keyword inside the property schema to be removed, got: %s", prop.Raw)
	}
	if got := prop.Get("items.type").String(); got != "string" {
		t.Errorf("properties.uniqueItems.items.type = %q, want string", got)
	}
}

func TestNormalizeCodexToolSchemasUniqueItemsWithExistingSanitization(t *testing.T) {
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "mixed",
			"parameters": {
				"type": "object",
				"properties": {
					"tags": {
						"type": "array",
						"uniqueItems": true,
						"items": {"type": "string", "pattern": "^[^\\p{Cc}]+$"}
					},
					"asset_id": {"type": "string", "pattern": "^[0-9a-f]{32}$"},
					"huge": {"type": "integer", "maximum": 9223372036854775808},
					"bounded": {"type": "integer", "minimum": 0, "maximum": 100}
				}
			}
		}]
	}`)

	out := NormalizeCodexToolSchemas(input)
	params := ojson.GetBytesNoCopy(out, "tools.0.parameters")

	if params.Get("properties.tags.uniqueItems").Exists() {
		t.Errorf("expected tags.uniqueItems to be removed, got: %s", params.Get("properties.tags").Raw)
	}
	if params.Get("properties.tags.items.pattern").Exists() {
		t.Errorf("expected tags.items.pattern to be removed, got: %s", params.Get("properties.tags.items").Raw)
	}
	if params.Get("properties.huge.maximum").Exists() {
		t.Errorf("expected huge.maximum to be removed, got: %s", params.Get("properties.huge").Raw)
	}
	if got := params.Get("properties.asset_id.pattern").String(); got != "^[0-9a-f]{32}$" {
		t.Errorf("asset_id.pattern = %q, want ^[0-9a-f]{32}$", got)
	}
	if got := params.Get("properties.bounded.minimum").Int(); got != 0 {
		t.Errorf("bounded.minimum = %d, want 0", got)
	}
	if got := params.Get("properties.bounded.maximum").Int(); got != 100 {
		t.Errorf("bounded.maximum = %d, want 100", got)
	}
}

func TestNormalizeCodexToolSchemasLeavesCleanSchemaUntouched(t *testing.T) {
	input := []byte(`{
		"tools": [{
			"type": "function",
			"name": "clean",
			"parameters": {
				"type": "object",
				"properties": {
					"options": {"type": "array", "items": {"type": "string"}}
				}
			}
		}]
	}`)

	doc := ojson.NewDoc(input)
	if NormalizeCodexToolSchemasDoc(doc) {
		t.Fatalf("expected no changes for a schema without unsupported keywords, got: %s", doc.Bytes())
	}
}
