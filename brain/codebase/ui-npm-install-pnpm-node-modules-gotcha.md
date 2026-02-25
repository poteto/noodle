# UI npm Install Fails with pnpm Node Modules

- Symptom: running `npm install ...` in `ui/` can fail with `Unsupported URL Type "workspace:": workspace:*`.
- Context: `ui/` currently uses pnpm-managed `node_modules` + `pnpm-lock.yaml`.
- Practical fix: add deps with `pnpm add ...` in `ui/`, then keep using `npm run ...` scripts for verification/build.
