package jobs

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// ChildOptions decorates the poll context before starting a child workflow
// (per-type timeouts, task queue, memo — nil = passthrough).
type ChildOptions func(workflow.Context) workflow.Context

// Spec is one claimable job type in a poll registry.
type Spec struct {
	// Type is the job type string (must be on the web queue whitelist).
	Type string
	// Child is the child workflow function (must be registered on the worker).
	Child any
	// ChildCtx decorates the context for this type's children (nil = passthrough).
	ChildCtx ChildOptions
	// Gate optionally version-gates the spec (workflow.GetVersion inside —
	// in-flight poll histories replay the old branch deterministically).
	// Nil = always claimed.
	Gate func(workflow.Context) bool
}

// Registry is an ordered set of claimable job types. Order IS the claim
// order (workflow replay determinism needs a stable sequence).
type Registry struct {
	specs []Spec
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register appends a spec (claim order = registration order); chainable.
func (r *Registry) Register(spec Spec) *Registry {
	r.specs = append(r.specs, spec)
	return r
}

// ClaimFanOut claims up to limit jobs of spec.Type via claimActivity, then
// starts one spec.Child child workflow per claimed row (spec.ChildCtx
// decorates the child context). It waits for each child's START only — a
// child that fails to start fails the poll loudly, completion is
// fire-and-forget so the poll returns in seconds. Errors carry the job type.
func ClaimFanOut[T any](ctx workflow.Context, claimActivity any, spec Spec, limit int) error {
	var claimed []T
	if err := workflow.ExecuteActivity(ctx, claimActivity, spec.Type, limit).Get(ctx, &claimed); err != nil {
		return fmt.Errorf("claim %s jobs: %w", spec.Type, err)
	}
	childCtx := ctx
	if spec.ChildCtx != nil {
		childCtx = spec.ChildCtx(ctx)
	}
	for _, job := range claimed {
		future := workflow.ExecuteChildWorkflow(childCtx, spec.Child, job)
		if err := future.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
			return fmt.Errorf("start %s child: %w", spec.Type, err)
		}
	}
	return nil
}

// Run walks the registry in order: gate-false specs are skipped, every other
// spec goes through ClaimFanOut. The first failure aborts the whole poll.
func Run[T any](r *Registry, ctx workflow.Context, claimActivity any, limit int) error {
	for _, spec := range r.specs {
		if spec.Gate != nil && !spec.Gate(ctx) {
			continue
		}
		if err := ClaimFanOut[T](ctx, claimActivity, spec, limit); err != nil {
			return err
		}
	}
	return nil
}
