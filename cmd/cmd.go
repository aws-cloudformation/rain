package cmd

// Labels used as annotations on commands to provide grouping
const (
	groupAnnotationLabel = "Group"
	stackGroup           = "Stack commands"
	templateGroup        = "Template commands"
)

var (
	groups = []string{
		stackGroup,
		templateGroup,
	}
	stackAnnotation    = map[string]string{groupAnnotationLabel: stackGroup}
	templateAnnotation = map[string]string{groupAnnotationLabel: templateGroup}
)

// Fmt is exposed to be available to the cfn-format command
var Fmt = fmtCmd

// Build is exposed to be available to the cfn-skeleton command
var Build = buildCmd
