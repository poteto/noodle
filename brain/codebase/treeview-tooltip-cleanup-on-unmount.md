# Treeview Tooltip Cleanup On Unmount

- Symptom: topology node tooltip remained stuck on screen after leaving the tree route.
- Root cause: tooltip DOM is appended to `document.body` and cleanup only ran on node `mouseleave`, which may never fire during route navigation/unmount.
- Fix: centralize tooltip teardown and call it on both node `mouseleave` and `TreeView` effect cleanup (unmount/snapshot re-render path).
- Regression test: `ui/src/components/TreeView.interaction.test.tsx` (`cleans up tooltip nodes when tree view unmounts`).

See also [[principles/experience-first]], [[principles/fix-root-causes]]
