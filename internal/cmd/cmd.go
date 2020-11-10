package cmd

// Labels used as annotations on commands to provide grouping
const (
	GroupAnnotationLabel = "Group"
	StackGroup           = "Stack commands"
	TemplateGroup        = "Template commands"
)

// Groups represents the ordering of commands in Rain's help text
var Groups = []string{
	StackGroup,
	TemplateGroup,
}

// StackAnnotation is used by rain sub-commands to
// register as commands in the stack group
var StackAnnotation = map[string]string{GroupAnnotationLabel: StackGroup}

// TemplateAnnotation is used by rain sub-commands to
// register as commands in the template group
var TemplateAnnotation = map[string]string{GroupAnnotationLabel: TemplateGroup}
