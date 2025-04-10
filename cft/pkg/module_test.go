package pkg_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"gopkg.in/yaml.v3"
)

func TestModule(t *testing.T) {
	runTest("test", t)
}

func TestBucket(t *testing.T) {
	runTest("bucket", t)
}

func TestApi(t *testing.T) {
	runTest("api", t)
}

func TestSimple(t *testing.T) {
	runTest("simple", t)
}

func TestModInMod(t *testing.T) {
	runTest("modinmod", t)
}

func TestSub(t *testing.T) {
	runTest("sub", t)
}

func TestMany(t *testing.T) {
	runTest("many", t)
}

func TestRef(t *testing.T) {
	runTest("ref", t)
}

func TestMeta(t *testing.T) {
	runTest("meta", t)
}

func TestRefFalse(t *testing.T) {
	runTest("ref-false", t)
}

func TestOverride(t *testing.T) {
	runFailTest("override", t)
}

func TestPackageAlias(t *testing.T) {
	runTest("alias", t)
}

func TestIfPAram(t *testing.T) {
	runTest("ifparam", t)
}

func TestConstant(t *testing.T) {
	runTest("constant", t)
}

func TestDefault(t *testing.T) {
	runTest("default", t)
}

func TestSameMod(t *testing.T) {
	runTest("same-mod", t)
}

// TestAWSCLIModuleBasic tests the basic AWS CLI module functionality
func TestAWSCLIModuleBasic(t *testing.T) {
	runTest("awscli-modules/basic", t)
}

// TestAWSCLIModuleType tests the type AWS CLI module functionality
func TestAWSCLIModuleType(t *testing.T) {
	runTest("awscli-modules/type", t)
}

// TestAWSCLIModuleSub tests the sub AWS CLI module functionality
func TestAWSCLIModuleSub(t *testing.T) {
	runTest("awscli-modules/sub", t)
}

// TestAWSCLIModuleModInMod tests the module-in-module AWS CLI functionality
func TestAWSCLIModuleModInMod(t *testing.T) {
	runTest("awscli-modules/modinmod", t)
}

// TestAWSCLIModuleOutput tests the output AWS CLI module functionality
func TestAWSCLIModuleOutput(t *testing.T) {
	runTest("awscli-modules/output", t)
}

// TestAWSCLIModulePolicy tests the policy AWS CLI module functionality
func TestAWSCLIModulePolicy(t *testing.T) {
	runTest("awscli-modules/policy", t)
}

// TestAWSCLIModuleVPC tests the VPC AWS CLI module functionality
func TestAWSCLIModuleVPC(t *testing.T) {
	runTest("awscli-modules/vpc", t)
}

// TestAWSCLIModuleMap tests the map AWS CLI module functionality
func TestAWSCLIModuleMap(t *testing.T) {
	runTest("awscli-modules/map", t)
}

// TestAWSCLIModuleMapOut tests the map output AWS CLI module functionality
func TestAWSCLIModuleMapOut(t *testing.T) {
	runTest("awscli-modules/mapout", t)
}

// TestAWSCLIModuleConditional tests the conditional AWS CLI module functionality
func TestAWSCLIModuleConditional(t *testing.T) {
	runTest("awscli-modules/conditional", t)
}

// TestAWSCLIModuleCondIntrinsics tests the conditional intrinsics AWS CLI module functionality
func TestAWSCLIModuleCondIntrinsics(t *testing.T) {
	runTest("awscli-modules/cond-intrinsics", t)
}

// TestAWSCLIModuleExample tests the example AWS CLI module functionality
func TestAWSCLIModuleExample(t *testing.T) {
	runTest("awscli-modules/example", t)
}

// TestAWSCLIModuleGetAtt tests the GetAtt AWS CLI module functionality
func TestAWSCLIModuleGetAtt(t *testing.T) {
	runTest("awscli-modules/getatt", t)
}

// TestAWSCLIModuleConstant tests the constant AWS CLI module functionality
func TestAWSCLIModuleConstant(t *testing.T) {
	runTest("awscli-modules/constant", t)
}

// TestAWSCLIModulePropArray tests the property array AWS CLI module functionality
func TestAWSCLIModulePropArray(t *testing.T) {
	runTest("awscli-modules/proparray", t)
}

// TestAWSCLIModuleDepends tests the depends AWS CLI module functionality
func TestAWSCLIModuleDepends(t *testing.T) {
	runTest("awscli-modules/depends", t)
}

// TestAWSCLIModuleSelect tests the select AWS CLI module functionality
func TestAWSCLIModuleSelect(t *testing.T) {
	runTest("awscli-modules/select", t)
}

// TestAWSCLIModuleMerge tests the merge AWS CLI module functionality
func TestAWSCLIModuleMerge(t *testing.T) {
	runTest("awscli-modules/merge", t)
}

// TestAWSCLIModuleMergeTags tests the merge tags AWS CLI module functionality
func TestAWSCLIModuleMergeTags(t *testing.T) {
	runTest("awscli-modules/mergetags", t)
}

// TestAWSCLIModuleInsertFile tests the insert file AWS CLI module functionality
func TestAWSCLIModuleInsertFile(t *testing.T) {
	runTest("awscli-modules/insertfile", t)
}

// TestAWSCLIModuleOutSubList tests the output sub list AWS CLI module functionality
func TestAWSCLIModuleOutSubList(t *testing.T) {
	runTest("awscli-modules/outsublist", t)
}

// TestAWSCLIModuleOutJoin tests the output join AWS CLI module functionality
func TestAWSCLIModuleOutJoin(t *testing.T) {
	runTest("awscli-modules/outjoin", t)
}

// TestAWSCLIModuleInvoke tests the invoke AWS CLI module functionality
func TestAWSCLIModuleInvoke(t *testing.T) {
	runTest("awscli-modules/invoke", t)
}

// TestAWSCLIModuleZip tests the zip AWS CLI module functionality
func TestAWSCLIModuleZip(t *testing.T) {
	runTest("awscli-modules/zip", t)
}

// TestAWSCLIModuleDefault tests the default AWS CLI module functionality
func TestAWSCLIModuleDefault(t *testing.T) {
	runTest("awscli-modules/default", t)
}

// TestAWSCLIModuleSameMod tests the same-mod AWS CLI module functionality
func TestAWSCLIModuleSameMod(t *testing.T) {
	runTest("awscli-modules/same-mod", t)
}

// TestAWSCLIModuleCondUnRes tests the cond-unres AWS CLI module functionality
func TestAWSCLIModuleCondUnRes(t *testing.T) {
	runTest("awscli-modules/cond-unres", t)
}

// TestAWSCLIModuleCondConflict test a Condition conflict
func TestAWSCLIModuleCondConflict(t *testing.T) {
	runFailTest("awscli-modules/cond-conflict", t)
}

func runTest(test string, t *testing.T) {

	// There should be 3 files for each test, for example:
	// bucket-module.yaml, bucket-template.yaml, bucket-expect.yaml

	path := fmt.Sprintf("./tmpl/%v-expect.yaml", test)

	expectedTemplate, err := parse.File(path)
	if err != nil {
		t.Errorf("expected %s: %v", test, err)
		return
	}

	pkg.Experimental = true

	packaged, err := pkg.File(fmt.Sprintf("./tmpl/%v-template.yaml", test))
	if err != nil {
		t.Errorf("packaged %s: %v", test, err)
		return
	}

	//y := format.String(packaged, format.Options{
	//	JSON:     false,
	//	Unsorted: false,
	//})

	d := diff.New(packaged, expectedTemplate)
	if d.Mode() != "=" {
		t.Errorf("Module test %s failed: %v", test, d.Format(true))
	}
}

// runFailTest should fail to package
func runFailTest(test string, t *testing.T) {

	pkg.Experimental = true

	_, err := pkg.File(fmt.Sprintf("./tmpl/%v-template.yaml", test))
	if err == nil {
		t.Errorf("did not fail: packaged %s", test)
		return
	}

}

func TestCsvToSequence(t *testing.T) {
	csv := "A,B,C"
	seq := pkg.ConvertCsvToSequence(csv)
	if seq == nil || seq.Kind != yaml.SequenceNode {
		t.Errorf("expected a sequence node")
	}
	if seq.Content[0].Value != "A" ||
		seq.Content[1].Value != "B" ||
		seq.Content[2].Value != "C" {
		t.Errorf("Unexpected sequence")
	}
}

func init() {
	pkg.NoAnalytics = true
}
