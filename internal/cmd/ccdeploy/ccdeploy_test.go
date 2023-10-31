package ccdeploy

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/internal/config"
)

func TestReady(t *testing.T) {

	config.Debug = true

	g := graph.Empty()

	a := graph.Node{Name: "a", Type: "Resources"}
	b := graph.Node{Name: "b", Type: "Resources"}

	g.Link(a, b)

	ar := NewResource(a.Name, "AWS::S3::Bucket", Waiting, nil)
	br := NewResource(b.Name, "AWS::S3::Bucket", Waiting, nil)

	if ready(ar, &g) {
		t.Errorf("ar should not be ready")
	}

	if !ready(br, &g) {
		t.Errorf("br should be ready")
	}

	c := graph.Node{Name: "c", Type: "Resources"}
	ar.State = Waiting
	br.State = Deployed
	cr := NewResource(c.Name, "AWS::S3::Bucket", Waiting, nil)
	g.Link(b, c)

	if !ready(cr, &g) {
		t.Errorf("cr should be ready")
	}

	if ready(ar, &g) {
		t.Errorf("ar should not be ready after adding c")
	}

}

func TestVerifyDeletes(t *testing.T) {

	g := graph.Empty()

	a := graph.Node{Name: "A", Type: "Resources"}
	b := graph.Node{Name: "B", Type: "Resources"}

	g.Link(a, b) // A depends on B

	resourceMap := make(map[string]*Resource)
	ar := NewResource(a.Name, "AWS::S3::Bucket", Waiting, nil)
	ar.Action = diff.Update
	resourceMap[a.Name] = ar
	br := NewResource(b.Name, "AWS::S3::Bucket", Waiting, nil)
	br.Action = diff.Delete
	resourceMap[b.Name] = br

	err := verifyDeletes([]*Resource{br}, &g, resourceMap)
	if err == nil {
		t.Fatalf("Should have failed: A depends on B, which is being deleted")
	}

	br.Action = diff.Update
	c := graph.Node{Name: "C", Type: "Resources"}
	g.Link(b, c) // B depends on C
	cr := NewResource(c.Name, "AWS::S3::Bucket", Waiting, nil)
	cr.Action = diff.Delete
	resourceMap[c.Name] = cr

	err = verifyDeletes([]*Resource{cr}, &g, resourceMap)
	if err == nil {
		t.Fatalf("Should have failed: A depends on B->C, which is being deleted")
	}

	ar.Action = diff.Delete
	br.Action = diff.Delete
	err = verifyDeletes([]*Resource{ar, br, cr}, &g, resourceMap)
	if err != nil {
		t.Fatalf("Should not have failed: %v", err)
	}

}
