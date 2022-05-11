package describe

import (
	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
)

type dependencies interface {
	Logger() log.Logger
}

func Run(tmpl *template.Template, d dependencies) (err error) {
	w := d.Logger().InfoWriter()

	w.Writef("Template ID:          %s", tmpl.TemplateRecord().Id)
	w.Writef("Name:                 %s", tmpl.TemplateRecord().Name)
	w.Writef("Description:          %s", tmpl.TemplateRecord().Description)
	w.Writef("")

	v := tmpl.VersionRecord()
	w.Writef("Version:              %s", v.Version.String())
	w.Writef("Stable:               %t", v.Stable)
	w.Writef("Description:          %s", v.Description)
	w.Writef("")

	// Groups
	for _, group := range tmpl.Inputs().ToExtended() {
		w.Writef("Group ID:             %s", group.Id)
		w.Writef("Description:          %s", group.Description)
		w.Writef("Required:             %s", string(group.Required))
		w.Writef("")

		// Steps
		for _, step := range group.Steps {
			w.Writef("  Step ID:            %s", step.Id)
			w.Writef("  Name:               %s", step.Name)
			w.Writef("  Description:        %s", step.Description)
			w.Writef("  Dialog Name:        %s", step.NameFoDialog())
			w.Writef("  Dialog Description: %s", step.DescriptionForDialog())
			w.Writef("")

			// Inputs
			for _, in := range step.Inputs {
				w.Writef("    Input ID:         %s", in.Id)
				w.Writef("    Name:             %s", in.Name)
				w.Writef("    Description:      %s", in.Description)
				w.Writef("    Type:             %s", in.Type)
				w.Writef("    Kind:             %s", string(in.Kind))
				if in.Default != nil {
					w.Writef("    Default:          %#v", in.DefaultOrEmpty())
				}
				if len(in.Options) > 0 {
					w.Writef("    Options:")
					for _, opt := range in.Options {
						w.Writef("      %s:  %s", opt.Value, opt.Label)
					}
				}
				w.Writef("")
			}
		}
	}

	return nil
}
