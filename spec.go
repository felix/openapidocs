package openapidocs

import (
	"encoding/json/v2"
	"html/template"
	"io"

	"github.com/russross/blackfriday/v2"
	"go.yaml.in/yaml/v4"
)

type Spec struct {
	OpenAPI    string               `json:"openapi" yaml:"openapi"`
	Info       Info                 `json:"info" yaml:"info"`
	Servers    []Server             `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths" yaml:"paths"`
	Tags       []Tag                `json:"tags,omitempty" yaml:"tags,omitempty"`
	Components *Components          `json:"components,omitempty" yaml:"components,omitempty"`
}

type Components struct {
	Schemas       map[string]*Schema      `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Parameters    map[string]*Parameter   `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBodies map[string]*RequestBody `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Responses     map[string]*Response    `json:"responses,omitempty" yaml:"responses,omitempty"`
}

type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type PathItem struct {
	Summary    string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Get        *Operation  `json:"get,omitempty" yaml:"get,omitempty"`
	Post       *Operation  `json:"post,omitempty" yaml:"post,omitempty"`
	Put        *Operation  `json:"put,omitempty" yaml:"put,omitempty"`
	Delete     *Operation  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch      *Operation  `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options    *Operation  `json:"options,omitempty" yaml:"options,omitempty"`
	Head       *Operation  `json:"head,omitempty" yaml:"head,omitempty"`
}

func (pi *PathItem) Methods() []string {
	var out []string
	if pi.Get != nil {
		out = append(out, "Get")
	}
	if pi.Head != nil {
		out = append(out, "Head")
	}
	if pi.Post != nil {
		out = append(out, "Post")
	}
	if pi.Put != nil {
		out = append(out, "Put")
	}
	if pi.Delete != nil {
		out = append(out, "Delete")
	}
	if pi.Patch != nil {
		out = append(out, "Patch")
	}
	if pi.Options != nil {
		out = append(out, "Options")
	}
	return out
}

type Operation struct {
	Summary     string              `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security    []SecurityReq       `json:"security,omitempty" yaml:"security,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Ref         string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`
}

type RequestBody struct {
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool               `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]Content `json:"content,omitempty" yaml:"content,omitempty"`
}

type Response struct {
	Description string             `json:"description" yaml:"description"`
	Content     map[string]Content `json:"content,omitempty" yaml:"content,omitempty"`
}

type Content struct {
	Schema   *Schema                  `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  any                      `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]ExampleObject `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// ExampleObject is an OpenAPI Example Object (the `value` is what we render).
type ExampleObject struct {
	Summary     string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Value       any    `json:"value,omitempty" yaml:"value,omitempty"`
	Rendered    any    `json:"-" yaml:"-"`
}

type Schema struct {
	Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string             `json:"format,omitempty" yaml:"format,omitempty"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Required    []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Enum        []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example     any                `json:"example,omitempty" yaml:"example,omitempty"`
	AllOf       []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf       []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf       []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	RefName     string             `json:"-" yaml:"-"`
	Minimum     int                `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     int                `json:"maximum,omitempty" yaml:"maximum,omitempty"`
}

type SecurityReq map[string][]string

func (i Info) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(i.Description)))
}
func (s Server) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(s.Description)))
}
func (o Tag) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(o.Description)))
}
func (b Operation) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(b.Description)))
}
func (p Parameter) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(p.Description)))
}
func (r RequestBody) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(r.Description)))
}
func (r Response) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(r.Description)))
}
func (x ExampleObject) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(x.Description)))
}
func (s Schema) DescriptionHTML() template.HTML {
	return template.HTML(blackfriday.Run([]byte(s.Description)))
}

func ReadSpecFromJSON(r io.Reader) (*Spec, error) {
	var out Spec
	err := json.UnmarshalRead(r, &out, json.RejectUnknownMembers(false))
	return &out, err
}

func ParseSpecFromJSON(b []byte) (*Spec, error) {
	var out Spec
	err := json.Unmarshal(b, &out, json.RejectUnknownMembers(false))
	return &out, err
}

func ReadSpecFromYAML(r io.Reader) (*Spec, error) {
	var out Spec
	l, err := yaml.NewLoader(r)
	if err != nil {
		return nil, err
	}
	err = l.Load(&out)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func ParseSpecFromYAML(b []byte) (*Spec, error) {
	var out Spec
	err := yaml.Load(b, &out, yaml.WithKnownFields(false))
	return &out, err
}
