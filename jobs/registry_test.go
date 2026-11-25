package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// testJob is the claimed-row shape the tests fan out over.
type testJob struct {
	ID   int
	Type string
}

// recorder records child-workflow starts across goroutines (testsuite runs
// children concurrently with the poll).
type recorder struct {
	mu    sync.Mutex
	order []string
}

func (r *recorder) add(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, event)
}

func (r *recorder) events() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.order...)
}

// fakeClaims is the claim activity: per-type claimed rows, error for unknown
// types. (Struct methods, not closures — the testsuite derives activity
// names from the func, and anonymous closures collide.)
type fakeClaims struct {
	rows map[string][]testJob
}

func (f *fakeClaims) Claim(ctx context.Context, jobType string, limit int) ([]testJob, error) {
	rows, ok := f.rows[jobType]
	if !ok {
		return nil, fmt.Errorf("no claim branch for %s", jobType)
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

// failAfter claims fine for okType, errors for every other type — simulates
// the claim activity failing on the Nth spec, aborting the registry run.
type failAfter struct {
	okType string
	err    error
}

func (f *failAfter) Claim(ctx context.Context, jobType string, limit int) ([]testJob, error) {
	if jobType == f.okType {
		return []testJob{{ID: 1, Type: jobType}}, nil
	}
	return nil, f.err
}

// testChildWorkflow is the single named child workflow the specs point at;
// the recorded event comes from the job's own Type (testsuite needs stable
// top-level function names — closures/method values resolve ambiguously).
func testChildWorkflow(ctx workflow.Context, job testJob) error {
	rec.add(fmt.Sprintf("%s:%d", job.Type, job.ID))
	return nil
}

// testPoll is the poll workflow under test: registry run over the given
// specs with the given claim activity — mirrors the production poll, which
// sets claim activity options on the poll context (no retry: a claim failure
// must surface immediately and abort Run).
type testPoll struct {
	claims func(context.Context, string, int) ([]testJob, error)
	specs  []Spec
}

func (p *testPoll) Poll(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	r := NewRegistry()
	for _, spec := range p.specs {
		r.Register(spec)
	}
	return Run[testJob](r, ctx, p.claims, 10)
}

// runPoll executes a poll over the given specs inside the in-memory env and
// returns the workflow error text ("" on success) plus the recorder.
func runPoll(t *testing.T, claims func(context.Context, string, int) ([]testJob, error), specs ...Spec) (string, *recorder) {
	t.Helper()
	// testChildWorkflow records into the package-level rec — reset it here
	// (tests run sequentially, no t.Parallel).
	rec.mu.Lock()
	rec.order = nil
	rec.mu.Unlock()
	poller := &testPoll{claims: claims, specs: specs}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(poller.Poll)
	env.RegisterActivity(claims)
	env.RegisterWorkflow(testChildWorkflow)
	env.ExecuteWorkflow(poller.Poll)
	var out string
	if wfErr := env.GetWorkflowError(); wfErr != nil {
		out = wfErr.Error()
	}
	env.AssertExpectations(t)
	return out, rec
}

func TestRunClaimsAndStartsChildrenInRegistryOrder(t *testing.T) {
	specs := []Spec{
		{Type: "download", Child: testChildWorkflow},
		{Type: "email", Child: testChildWorkflow},
	}
	claims := &fakeClaims{rows: map[string][]testJob{
		"download": {{ID: 1, Type: "download"}, {ID: 2, Type: "download"}},
		"email":    {{ID: 3, Type: "email"}},
	}}
	out, rec2 := runPoll(t, claims.Claim, specs...)
	if out != "" {
		t.Fatalf("poll failed: %s", out)
	}
	// Claim + child-start happen per spec in registry order, so all download
	// children start before any email child.
	got := rec2.events()
	want := []string{"download:1", "download:2", "email:3"}
	if len(got) != len(want) {
		t.Fatalf("started %v, want %v", got, want)
	}
	for _, e := range want {
		if got[0] != e {
			t.Fatalf("started %v, want %v in registry order", got, want)
		}
		got = got[1:]
	}
}

func TestRunSkipsGatedSpec(t *testing.T) {
	specs := []Spec{
		{Type: "gated", Child: testChildWorkflow, Gate: func(workflow.Context) bool { return false }},
		{Type: "free", Child: testChildWorkflow},
	}
	claims := &fakeClaims{rows: map[string][]testJob{
		"gated": {{ID: 1, Type: "gated"}},
		"free":  {{ID: 2, Type: "free"}},
	}}
	out, rec2 := runPoll(t, claims.Claim, specs...)
	if out != "" {
		t.Fatalf("poll failed: %s", out)
	}
	events := rec2.events()
	if len(events) != 1 || events[0] != "free:2" {
		t.Fatalf("children started %v, want only free:2 (gated spec must be skipped)", events)
	}
}

func TestRunAbortsOnClaimErrorWithJobType(t *testing.T) {
	specs := []Spec{
		{Type: "download", Child: testChildWorkflow},
		{Type: "render", Child: testChildWorkflow},
	}
	claims := &failAfter{okType: "download", err: errors.New("web queue down")}
	out, rec2 := runPoll(t, claims.Claim, specs...)
	if out == "" {
		t.Fatal("poll succeeded, want claim failure")
	}
	if !strings.Contains(out, "render") || !strings.Contains(out, "web queue down") {
		t.Fatalf("error %q must name the job type and wrap the cause", out)
	}
	if events := rec2.events(); len(events) != 1 || events[0] != "download:1" {
		t.Fatalf("children started %v, want only download:1 before abort", events)
	}
}

// rec is the shared child-start recorder for spec construction above (tests
// run sequentially — no t.Parallel).
var rec = &recorder{}
