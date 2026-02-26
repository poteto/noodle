import { createContext, useContext } from "react";
import type { ControlCommand } from "~/client";

type SendFn = (command: ControlCommand) => void;

const ControlContext = createContext<SendFn>(() => {
  /* noop default */
});

export { ControlContext };

export function useControl(): SendFn {
  return useContext(ControlContext);
}
