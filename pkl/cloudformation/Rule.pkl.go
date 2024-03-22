// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type Rule interface {
	GetRuleCondition() *map[any]any

	GetAssertions() []RuleAssertion
}

var _ Rule = (*RuleImpl)(nil)

// A Rule, which is a set of assertions that can be made on Parameters
type RuleImpl struct {
	RuleCondition *map[any]any `pkl:"RuleCondition"`

	Assertions []RuleAssertion `pkl:"Assertions"`
}

func (rcv *RuleImpl) GetRuleCondition() *map[any]any {
	return rcv.RuleCondition
}

func (rcv *RuleImpl) GetAssertions() []RuleAssertion {
	return rcv.Assertions
}
