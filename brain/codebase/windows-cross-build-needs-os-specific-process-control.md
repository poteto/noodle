# Windows Cross-Build Needs OS-Specific Process Control

- Shared process-management code cannot reference Unix-only syscall APIs (`SysProcAttr.Setpgid`, `Getpgid`, `Kill`) in files compiled for all targets.
- GoReleaser cross-compiling for `windows/*` fails at compile time if those symbols exist in common files.
- Keep process-group signaling in `//go:build !windows` files and provide Windows-specific fallbacks in `//go:build windows` files.
- For PID liveness and shutdown on Windows, use Windows-compatible paths (`OpenProcess/GetExitCodeProcess`, `os.FindProcess`, `Process.Signal/Process.Kill`) instead of Unix signal semantics.
