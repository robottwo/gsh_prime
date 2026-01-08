# Native Aider Integration for GSH

## Context
`gsh` (Generative Shell) supports subagents (specialized AI assistants) invoked via `@`. Currently, subagents are defined via configuration files (Claude/Roo style). We want to add a **native** integration for [Aider](https://aider.chat), an AI pair programming tool, accessible via the command `@aider`.

## Goal
Implement `@aider` as a first-class citizen in `gsh`, allowing users to invoke Aider directly from the shell prompt.

## Requirements

1.  **Command Handling (`@aider`)**:
    -   Intercept `@aider` in `SubagentIntegration.HandleCommand`.
    -   Support two modes:
        -   **Interactive Mode**: `@aider` (no arguments) -> Launches `aider` in interactive mode, taking over the terminal.
        -   **Message Mode**: `@aider <instruction>` -> Executes `aider --message "<instruction>"`.
    -   Bypass the standard `SubagentExecutor` (which uses OpenAI API) and instead execute the `aider` CLI binary directly.

2.  **Implementation Details**:
    -   **File**: `internal/subagent/integration.go`
    -   **Detection**: Before execution, check if `aider` is installed (available in PATH). If not, return a helpful error message.
    -   **Execution (`runAider`)**:
        -   Implement a `runAider(prompt string) error` method.
        -   Use `exec.Command` to run `aider`.
        -   For **Interactive Mode**:
            -   Connect `Stdin`, `Stdout`, `Stderr` of the command to `os.Stdin`, `os.Stdout`, `os.Stderr`.
            -   This ensures the user can interact with Aider's TUI.
        -   For **Message Mode**:
            -   Arguments: `aider`, `--message`, `prompt`.
            -   Connect `Stdout` and `Stderr` to `os.Stdout`/`os.Stderr` so the user sees progress.
            -   Alternatively, capture output and stream it to the chat channel, but direct output is preferred to preserve Aider's rich styling/colors.
    -   **Return Values**:
        -   `HandleCommand` expects `(bool, <-chan string, *Subagent, error)`.
        -   Return `true` (handled).
        -   Return a "virtual" `Subagent` struct for Aider (ID: "aider", Name: "Aider", Description: "AI Pair Programmer").
        -   Return a closed/empty channel (or a channel with a single "Done" message) since Aider handles its own output directly to the terminal.

3.  **Completions**:
    -   **File**: `internal/subagent/completion_adapter.go`
    -   Modify `GetAllSubagents` and `GetSubagent` to include the virtual "aider" subagent if the binary is found.
    -   This ensures `@aider` shows up in tab completions.

4.  **Configuration**:
    -   Respect `AIDER_` environment variables (handled automatically by `aider` process).
    -   No additional config file needed for this native integration.

## Step-by-Step Implementation Plan

1.  **Modify `internal/subagent/integration.go`**:
    -   Add constant `AiderSubagentID = "aider"`.
    -   In `HandleCommand`, add logic to check for `subagentID == AiderSubagentID`.
    -   Implement `runAider(prompt string)`:
        -   Check `exec.LookPath("aider")`.
        -   Construct `exec.Command`.
        -   Set `Cmd.Stdin = os.Stdin`, `Cmd.Stdout = os.Stdout`, `Cmd.Stderr = os.Stderr`.
        -   Run and wait.
    -   In `HandleCommand`:
        -   If `@aider` matches:
            -   Call `runAider` in a goroutine?
            -   **Issue**: `HandleCommand` is synchronous but returns a channel for *async* execution output.
            -   **Solution**: `runAider` should be called *inside* the goroutine that normally handles `executor.Chat`.
            -   Wait, `executor.Chat` spawns its own goroutine.
            -   So `HandleCommand` should return a channel, and spawn a goroutine that calls `runAider` and then closes the channel.
            -   Since `runAider` uses `os.Stdout`, `RunInteractiveShell` loop (which consumes the channel) will be blocked waiting for channel close, while `aider` prints to screen.
            -   Note: `RunInteractiveShell` prints "Asking <agent>..." then iterates channel.
            -   Ensure coordination so `aider` output appears cleanly.

2.  **Modify `internal/subagent/completion_adapter.go`**:
    -   In `GetAllSubagents`, check `exec.LookPath("aider")`.
    -   If found, append `aider` info to the result map.
    -   Similarly for `GetSubagent`.

## Verification
-   Run `@aider` to verify interactive mode.
-   Run `@aider update README` to verify message mode.
-   Check tab completion for `@`.
