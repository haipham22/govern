// Package jobs provides the claim-fanout abstraction shared by Temporal job
// workers: a poll workflow registers one Spec per claimable job type and runs
// the registry — replacing the repeated claim + child-start blocks with one
// data-driven loop. Claim order is the registry order (workflow replay
// determinism needs a stable sequence); version gates live on the Spec.
package jobs
