package dialog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/keboola/keboola-as-code/internal/pkg/cli/options"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/template"
	"github.com/keboola/keboola-as-code/internal/pkg/template/input"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/orderedmap"
)

// askTemplateInputs - dialog to define user inputs for a new template.
// Used in AskCreateTemplateOpts.
func (p *Dialogs) askTemplateInputs(opts *options.Options, branch *model.Branch, configs []*model.ConfigWithRows) (objectInputsMap, *template.Inputs, error) {
	// Create empty inputs map
	inputs := newInputsMap()

	// Select which config/row fields will be replaced by user input.
	objectInputs, err := newInputsSelectDialog(p.Prompt, opts, branch, configs, inputs).ask()
	if err != nil {
		return objectInputs, inputs.all(), err
	}

	// Define name/description for each user input.
	if err := newInputsDetailsDialog(p.Prompt, inputs).ask(); err != nil {
		return objectInputs, inputs.all(), err
	}

	return objectInputs, inputs.all(), nil
}

type inputFields map[string]inputField

func (f inputFields) Write(out *strings.Builder) {
	var table []inputFieldLine
	var inputIdMaxLength int
	var fieldPathMaxLength int

	// Convert and get max lengths for padding
	for _, field := range f {
		line := field.Line()
		table = append(table, line)

		if len(line.inputId) > inputIdMaxLength {
			inputIdMaxLength = len(line.inputId)
		}

		if len(line.fieldPath) > fieldPathMaxLength {
			fieldPathMaxLength = len(line.fieldPath)
		}
	}

	// Sort by field path
	sort.SliceStable(table, func(i, j int) bool {
		return table[i].fieldPath < table[j].fieldPath
	})

	// Format with padding
	format := fmt.Sprintf("%%s %%-%ds  %%-%ds %%s", inputIdMaxLength, fieldPathMaxLength)

	// Write
	for _, line := range table {
		example := ""
		if len(line.example) > 0 {
			example = "<!-- " + line.example + " -->"
		}
		out.WriteString(strings.TrimSpace(fmt.Sprintf(format, line.mark, line.inputId, line.fieldPath, example)))
		out.WriteString("\n")
	}
}

type inputField struct {
	path     orderedmap.Key
	example  string
	input    input.Input
	selected bool
}

func (f inputField) Line() inputFieldLine {
	mark := "[ ]"
	if f.selected {
		mark = "[x]"
	}

	return inputFieldLine{
		mark:      mark,
		inputId:   f.input.Id,
		fieldPath: f.path.String(),
		example:   f.example,
	}
}

type inputFieldLine struct {
	mark      string
	inputId   string
	fieldPath string
	example   string
}

// objectInputsMap - map of inputs used in an object.
type objectInputsMap map[model.Key][]template.InputDef

func (v objectInputsMap) add(objectKey model.Key, inputDef template.InputDef) {
	v[objectKey] = append(v[objectKey], inputDef)
}

func (v objectInputsMap) setTo(configs []template.ConfigDef) {
	for i := range configs {
		c := &configs[i]
		c.Inputs = v[c.Key]
		for j := range c.Rows {
			r := &c.Rows[j]
			r.Inputs = v[r.Key]
		}
	}
}

func newInputsMap() inputsMap {
	return inputsMap{data: orderedmap.New()}
}

// inputsMap - map of all Inputs by Input.Id.
type inputsMap struct {
	data *orderedmap.OrderedMap
}

func (v inputsMap) add(input template.Input) {
	v.data.Set(input.Id, input)
}

func (v inputsMap) get(inputId string) (template.Input, bool) {
	value, found := v.data.Get(inputId)
	if !found {
		return template.Input{}, false
	}
	return value.(template.Input), true
}

func (v inputsMap) all() *template.Inputs {
	out := make([]template.Input, v.data.Len())
	i := 0
	for _, key := range v.data.Keys() {
		item, _ := v.data.Get(key)
		out[i] = item.(template.Input)
		i++
	}
	return template.NewInputs().Set(out)
}
