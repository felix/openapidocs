package openapidocs

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"html/template"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Document struct {
	Slug        string
	Title       string
	Description template.HTML
	Version     string
	Servers     []Server
	Tags        map[string]Tag
	Endpoints   []Endpoint
	TagGroups   map[string][]Endpoint
	Standalone  bool

	StyleContent template.HTML
}

type Endpoint struct {
	Method      string
	Path        string
	Summary     string
	Description template.HTML
	Tags        []string
	Parameters  map[string][]Parameter
	RequestBody *RequestBody
	Responses   map[string]Response
	Deprecated  bool
	Slug        string
}

func DocumentFromSpec(spec *Spec) (*Document, error) {
	processSpec(spec)
	doc := &Document{
		Title:       spec.Info.Title,
		Description: spec.Info.DescriptionHTML(),
		Version:     spec.Info.Version,
		Servers:     spec.Servers,
		Tags:        make(map[string]Tag, len(spec.Tags)),
		TagGroups:   make(map[string][]Endpoint),
	}

	for _, t := range spec.Tags {
		doc.Tags[t.Name] = t
	}

	paths := make([]string, 0, len(spec.Paths))
	for path := range spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		item := spec.Paths[path]
		methods := []struct {
			name string
			op   *Operation
		}{
			{"GET", item.Get},
			{"POST", item.Post},
			{"PUT", item.Put},
			{"DELETE", item.Delete},
			{"PATCH", item.Patch},
			{"OPTIONS", item.Options},
			{"HEAD", item.Head},
		}

		for _, m := range methods {
			if m.op == nil {
				continue
			}
			params := groupParameters(append(
				item.Parameters,
				m.op.Parameters...))

			slug := buildSlug(m.name, path)
			endpoint := Endpoint{
				Method:      m.name,
				Path:        path,
				Summary:     m.op.Summary,
				Description: m.op.DescriptionHTML(),
				Tags:        m.op.Tags,
				Parameters:  params,
				RequestBody: m.op.RequestBody,
				Responses:   m.op.Responses,
				Deprecated:  m.op.Deprecated,
				Slug:        slug,
			}
			doc.Endpoints = append(doc.Endpoints, endpoint)

			if len(endpoint.Tags) == 0 {
				doc.TagGroups["default"] = append(doc.TagGroups["default"], endpoint)
			} else {
				for _, tag := range endpoint.Tags {
					doc.TagGroups[tag] = append(doc.TagGroups[tag], endpoint)
				}
			}
		}
	}

	return doc, nil
}

func processSpec(spec *Spec) {
	if spec.Components == nil {
		return
	}
	cache := make(map[string]*Schema)

	// Resolve shared parameters
	for _, param := range spec.Components.Parameters {
		resolveSchemaRefs(param.Schema, spec.Components, cache)
	}
	// Resolve shared schemas
	for _, schema := range spec.Components.Schemas {
		resolveSchemaRefs(schema, spec.Components, cache)
	}

	// Process each operation
	for _, pathItem := range spec.Paths {
		for i, p := range pathItem.Parameters {
			if p.Ref == "" {
				continue
			}
			resolveParamRefs(&p, spec.Components, cache)
			pathItem.Parameters[i] = p
		}
		ops := []*Operation{
			pathItem.Get,
			pathItem.Post,
			pathItem.Put,
			pathItem.Delete,
			pathItem.Patch,
			pathItem.Options,
			pathItem.Head,
		}
		for _, op := range ops {
			if op == nil {
				continue
			}
			resolveOperationRefs(op, spec.Components, cache)
			renderOperationExamples(op)
		}
	}
}

var slugRE = regexp.MustCompile("[{}]")

func buildSlug(method, path string) string {
	path = slugRE.ReplaceAllString(path, "")
	path = strings.ReplaceAll(path, "/", "-")
	return strings.ToLower(method) + "-" + strings.Trim(path, "-")
}

func groupParameters(in []Parameter) map[string][]Parameter {
	out := make(map[string][]Parameter)
	var group string
	for _, p := range in {
		group = cmp.Or(p.In, "body")
		out[group] = append(out[group], p)
	}
	return out
}

func resolveOperationRefs(op *Operation, components *Components, cache map[string]*Schema) {
	for i, p := range op.Parameters {
		if p.Ref != "" {
			resolveParamRefs(&p, components, cache)
			op.Parameters[i] = p
		}
	}

	if op.RequestBody != nil {
		for ct, media := range op.RequestBody.Content {
			if media.Schema != nil {
				resolveSchemaRefs(media.Schema, components, cache)
				op.RequestBody.Content[ct] = media
			}
		}
	}

	for code, resp := range op.Responses {
		for ct, media := range resp.Content {
			if media.Schema != nil {
				resolveSchemaRefs(media.Schema, components, cache)
				resp.Content[ct] = media
			}
		}
		op.Responses[code] = resp
	}
}

func resolveParamRefs(param *Parameter, components *Components, cache map[string]*Schema) {
	if param == nil || param.Ref == "" {
		return
	}
	_, refName, _ := strings.CutLast(param.Ref, "/")
	if refName == "" {
		return
	}
	resolved, ok := components.Parameters[refName]
	if ok {
		resolveSchemaRefs(resolved.Schema, components, cache)
		param.In = cmp.Or(param.In, resolved.In)
		param.Name = cmp.Or(param.Name, resolved.Name)
		param.Description = cmp.Or(param.Description, resolved.Description)
		param.Schema = resolved.Schema
	}
}

func resolveSchemaRefs(schema *Schema, components *Components, cache map[string]*Schema) {
	if schema == nil {
		return
	}
	var cacheKey string
	if schema.Ref != "" {
		cached, ok := cache[schema.Ref]
		if ok {
			schema = cached
		} else {
			cacheKey = schema.Ref
			_, refName, _ := strings.CutLast(schema.Ref, "/")
			if refName == "" {
				return
			}
			resolved, ok := components.Schemas[refName]
			if ok {
				// Recursively resolve the component schema first
				resolveSchemaRefs(resolved, components, cache)
				// Inline the resolved schema, keeping the ref name for display
				schema.RefName = refName
				schema.Ref = ""
				schema.Type = cmp.Or(schema.Type, resolved.Type)
				schema.Description = cmp.Or(schema.Description, resolved.Description)
				if schema.Properties == nil {
					schema.Properties = resolved.Properties
				}
				schema.Items = cmp.Or(schema.Items, resolved.Items)
				if schema.Required == nil {
					schema.Required = resolved.Required
				}
				if schema.Enum == nil {
					schema.Enum = resolved.Enum
				}
				schema.Format = cmp.Or(schema.Format, resolved.Format)
				if schema.AllOf == nil {
					schema.AllOf = resolved.AllOf
				}
				if schema.OneOf == nil {
					schema.OneOf = resolved.OneOf
				}
				if schema.AnyOf == nil {
					schema.AnyOf = resolved.AnyOf
				}
			}
		}
	}

	// Resolve allOf by merging properties
	if len(schema.AllOf) > 0 {
		for _, sub := range schema.AllOf {
			resolveSchemaRefs(sub, components, cache)
		}
		merged := mergeAllOf(schema.AllOf)
		if schema.Type == "" {
			schema.Type = merged.Type
		}
		if schema.Properties == nil {
			schema.Properties = merged.Properties
		} else {
			for k, v := range merged.Properties {
				if _, exists := schema.Properties[k]; !exists {
					schema.Properties[k] = v
				}
			}
		}
		if schema.Required == nil {
			schema.Required = merged.Required
		}
		schema.AllOf = nil
	}

	// Resolve nested
	for _, prop := range schema.Properties {
		resolveSchemaRefs(prop, components, cache)
	}
	if schema.Items != nil {
		resolveSchemaRefs(schema.Items, components, cache)
	}
	for _, sub := range schema.OneOf {
		resolveSchemaRefs(sub, components, cache)
	}
	for _, sub := range schema.AnyOf {
		resolveSchemaRefs(sub, components, cache)
	}
	if cacheKey != "" {
		cache[schema.Ref] = schema
	}
}

func mergeAllOf(schemas []*Schema) *Schema {
	merged := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}
	for _, s := range schemas {
		if s.Type != "" {
			merged.Type = s.Type
		}
		maps.Copy(merged.Properties, s.Properties)
		merged.Required = append(merged.Required, s.Required...)
	}
	return merged
}

func renderOperationExamples(op *Operation) {
	if op == nil {
		return
	}
	if op.RequestBody != nil {
		renderExample(op.RequestBody.Content)
	}
	for _, res := range op.Responses {
		renderExample(res.Content)
	}

}

func renderExample(content map[string]Content) {
	marshal := func(in any, ct string) any {
		var out any
		switch ct {
		case "application/json":
			b, _ := json.Marshal(in, json.Deterministic(true))
			v := jsontext.Value(b)
			_ = v.Indent(jsontext.EscapeForHTML(true), jsontext.WithIndent("  "))
			out = string(v)
		default:
			out = in
		}
		return out
	}
	for ct, c := range content {
		if c.Example != nil {
			continue
		}
		if len(c.Examples) > 0 {
			for n, ex := range c.Examples {
				ex.Rendered = marshal(ex.Value, ct)
				c.Examples[n] = ex
			}
			continue
		}
		ex := generateExampleSchema(c.Schema, ct)
		c.Example = marshal(ex, ct)
		content[ct] = c
	}
}

var (
	then = time.Date(2006, 1, 2, 3, 4, 5, 0, time.UTC)
	uuid = "01a18a20-c665-763d-ad69-d5b623fa65ec"
)

func generateExampleSchema(schema *Schema, name string) any {
	if schema == nil {
		return "example"
	}
	if schema.Example != nil {
		return schema.Example
	}

	if len(schema.Properties) == 0 {
		if len(schema.AllOf) > 0 {
			merged := map[string]any{}
			for _, sub := range schema.AllOf {
				if v, ok := generateExampleSchema(sub, name).(map[string]any); ok {
					maps.Copy(merged, v)
				}
			}
			if len(merged) > 0 {
				return merged
			}
		}
		if schema.Type == "" {
			if len(schema.OneOf) > 0 {
				return generateExampleSchema(schema.OneOf[0], name)
			}
			if len(schema.AnyOf) > 0 {
				return generateExampleSchema(schema.AnyOf[0], name)
			}
		}
	}

	return objectExample(schema, name)
}

func objectExample(schema *Schema, name string) any {
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}
	name = strings.ToLower(name)
	switch schema.Type {
	case "string":
		switch {
		case schema.Format == "date-time":
			return then
		case schema.Format == "email":
			return "user@example.com"
		case schema.Format == "uuid":
			return uuid
		case strings.Contains(name, "id"):
			return uuid
		default:
			return "string"
		}
	case "boolean":
		return true
	case "integer", "number":
		return cmp.Or(schema.Minimum, schema.Maximum, 1)
	case "array":
		return []any{objectExample(schema.Items, name)}
	case "object":
		m := make(map[string]any, len(schema.Properties))
		for k, v := range schema.Properties {
			m[k] = objectExample(v, k)
		}
		return m
	default:
		panic(schema.Type)
	}
}
