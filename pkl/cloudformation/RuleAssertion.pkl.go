// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type RuleAssertion interface {
	GetAssert() map[any]any

	GetAssertDescription() *any
}

var _ RuleAssertion = (*RuleAssertionImpl)(nil)

// A Rule assertion, which is an element of a Rule
type RuleAssertionImpl struct {
	Assert map[any]any `pkl:"Assert"`

	AssertDescription *any `pkl:"AssertDescription"`
}

func (rcv *RuleAssertionImpl) GetAssert() map[any]any {
	return rcv.Assert
}

func (rcv *RuleAssertionImpl) GetAssertDescription() *any {
	return rcv.AssertDescription
}
