import { createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";

function createAppRouter() {
  return createRouter({ routeTree });
}

let router: ReturnType<typeof createAppRouter> | undefined;

export function getRouter() {
  if (!router) {
    router = createAppRouter();
  }
  return router;
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof createAppRouter>;
  }
}
