package test

import (
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/definition"
	"github.com/keboola/keboola-as-code/internal/pkg/service/buffer/definition/key"
)

func NewSource(k key.SourceKey) definition.Source {
	return definition.Source{
		SourceKey:   k,
		Type:        definition.SourceTypeHTTP,
		Name:        "My Source",
		Description: "My Description",
		HTTP:        &definition.HTTPSource{Secret: "012345678901234567890123456789012345678912345678"},
	}
}
