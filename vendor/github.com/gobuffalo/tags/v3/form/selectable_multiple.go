package form

//SelectableMultiple allows any struct to add Selected option in the select tag.
type SelectableMultiple interface {
	IsSelected() bool
}

//SelectableMultiples is the plural for SelectableMultiple
type SelectableMultiples []SelectableMultiple
