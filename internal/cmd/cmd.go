package cmd

// Labels used as annotations on commands to provide grouping
const (
	groupAnnotationLabel = "Group"
	stackGroup           = "Stack commands"
	templateGroup        = "Template commands"
)

// Groups represents the ordering of commands in Rain's help text
var Groups = []string{
	stackGroup,
	templateGroup,
}

// StackAnnotation is used by rain sub-commands to
// register as commands in the stack group
var StackAnnotation = map[string]string{groupAnnotationLabel: stackGroup}

// TemplateAnnotation is used by rain sub-commands to
// register as commands in the template group
var TemplateAnnotation = map[string]string{groupAnnotationLabel: templateGroup}
