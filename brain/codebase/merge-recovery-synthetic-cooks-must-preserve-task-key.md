# Merge Recovery Synthetic Cooks Must Preserve Task Key

- Crash-recovery paths in `loop/reconcile.go` synthesize `cookHandle` values for already-merged, requeued-merge, and terminal-failure cases.
- Those synthetic cooks must copy `state.StageNode.TaskKey`, not `Skill`.
- `TaskKey` and `Skill` are separate canonical fields. Recovery code that rewrites `TaskKey` to `Skill` silently corrupts downstream metadata.
- Immediate symptoms show up in recovered `stage.completed` loop events and scheduler forwarding messages, which read `cook.stage.TaskKey`.
- Regression test: `TestHandleAlreadyMergedStagePreservesTaskKeyInStageCompletedEvent`.
