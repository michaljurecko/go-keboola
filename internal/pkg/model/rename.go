package model

type RenamedPath struct {
	ObjectState ObjectState
	OldPath     string
	RenameFrom  string // old path with renamed parents dirs
	NewPath     string
}

type RenameAction struct {
	Record      Record
	OldPath     string
	RenameFrom  string // old path with renamed parents dirs
	NewPath     string
	Description string
}

func (a *RenameAction) String() string {
	return a.Description
}
