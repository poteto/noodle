---
name: deploy
description: Deploys the project to the target environment.
schedule: "After test stages pass, to ship verified changes"
---

# Deploy

Deploy the current state to the target environment.

## Steps

1. Confirm that the previous test stage passed.
2. Run the deploy script or command for the project.
3. Verify the deployment succeeded.
4. Report the deployment status.
