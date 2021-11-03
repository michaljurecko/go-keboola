package state

import (
	"strings"

	"github.com/keboola/keboola-as-code/internal/pkg/fixtures"
	"github.com/keboola/keboola-as-code/internal/pkg/model"
	"github.com/keboola/keboola-as-code/internal/pkg/testproject"
	"github.com/keboola/keboola-as-code/internal/pkg/utils"
)

// NewProjectSnapshot - to validate final project state in tests.
func NewProjectSnapshot(s *State, testProject *testproject.Project) (*fixtures.ProjectSnapshot, error) {
	project := &fixtures.ProjectSnapshot{}

	branches := make(map[string]*fixtures.BranchWithConfigs)
	for _, bState := range s.Branches() {
		// Map branch
		branch := bState.Remote
		b := &fixtures.Branch{}
		b.Name = branch.Name
		b.Description = branch.Description
		b.IsDefault = branch.IsDefault
		branchConfigs := &fixtures.BranchWithConfigs{Branch: b, Configs: make([]*fixtures.Config, 0)}
		project.Branches = append(project.Branches, branchConfigs)
		branches[branch.String()] = branchConfigs
	}

	configs := make(map[string]*fixtures.Config)
	for _, cState := range s.Configs() {
		config := cState.Remote
		c := &fixtures.Config{Rows: make([]*fixtures.ConfigRow, 0)}
		c.ComponentId = config.ComponentId
		c.Name = config.Name
		c.Description = config.Description
		c.ChangeDescription = normalizeChangeDesc(config.ChangeDescription)
		c.Content = config.Content
		b := branches[config.BranchKey().String()]
		b.Configs = append(b.Configs, c)
		configs[config.String()] = c
	}

	for _, rState := range s.ConfigRows() {
		row := rState.Remote
		r := &fixtures.ConfigRow{}
		r.Name = row.Name
		r.Description = row.Description
		r.ChangeDescription = normalizeChangeDesc(row.ChangeDescription)
		r.IsDisabled = row.IsDisabled
		r.Content = row.Content
		c := configs[row.ConfigKey().String()]
		c.Rows = append(c.Rows, r)
	}

	schedules, err := testProject.SchedulerApi().ListSchedules()
	if err != nil {
		return nil, err
	}
	for _, schedule := range schedules {
		configKey := model.ConfigKey{BranchId: testProject.DefaultBranch().Id, ComponentId: model.SchedulerComponentId, Id: schedule.ConfigurationId}
		scheduleConfig := s.MustGet(configKey).(*model.ConfigState).Remote
		project.Schedules = append(project.Schedules, &fixtures.Schedule{Name: scheduleConfig.Name})
	}

	// Sort by name
	utils.SortByName(project.Branches)
	for _, b := range project.Branches {
		utils.SortByName(b.Configs)
		for _, c := range b.Configs {
			utils.SortByName(c.Rows)
		}
	}

	return project, nil
}

func normalizeChangeDesc(str string) string {
	// Default description if object has been created by test
	if str == "created by test" {
		return ""
	}

	// Default description if object has been created with a new branch
	if strings.HasPrefix(str, "Copied from ") {
		return ""
	}
	// Default description if rows has been deleted
	if strings.HasSuffix(str, " deleted") {
		return ""
	}

	return str
}
