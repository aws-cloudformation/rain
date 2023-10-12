package ccdeploy

import (
	"testing"

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
