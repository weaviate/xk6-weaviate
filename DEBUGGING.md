# Debugging xk6-weaviate

This document explains how to set up and use debugging for the xk6-weaviate extension using VSCode or Cursor.

## Prerequisites

- [Go toolchain](https://go101.org/article/go-toolchain.html) installed
- [VSCode](https://code.visualstudio.com/) or [Cursor](https://cursor.sh/) with the Go extension installed
- Git

## Debugging Setup

### Step 1: Build the Debug Version

First, you need to build a debug version of k6 with the xk6-weaviate extension. This version includes debug symbols that allow the debugger to work properly.

```bash
# Build the debug version using the make command
make build-debug
```

This command will:
1. Use xk6 to build a custom k6 binary
2. Include debug symbols via the `-gcflags='all=-N -l'` flags
3. Output the binary as `k6-debug` in your project root

### Step 2: Configure VSCode/Cursor for Debugging

Create or update the `.vscode/launch.json` file in your project with the following configuration:

```json
{
    "version": "0.2.0",
    "configurations": [
      {
        "name": "Debug xk6-weaviate",
        "type": "go",
        "request": "launch",
        "mode": "exec",
        "program": "${workspaceFolder}/k6-debug",
        "args": ["run", "examples/object-operations.js"],
        "buildFlags": "-tags=debugger"
      }
    ]
}
```

You can modify the `args` array to specify different test scripts as needed. For example:
- `["run", "examples/batch-operations.js"]` to debug batch operations
- `["run", "examples/tenant-operations.js"]` to debug tenant operations

### Step 3: Set Breakpoints

1. Open the Go source files you want to debug
2. Click in the gutter (the space to the left of the line numbers) to set breakpoints
3. Breakpoints will appear as red dots

### Step 4: Start Debugging

There are several ways to start the debugging session:

- Press `F5` on your keyboard
- Click the "Run and Debug" button in the sidebar and then click the green play button
- Use the "Run" menu and select "Start Debugging"

### Step 5: Debug Controls

Once debugging starts and hits a breakpoint, you can:

- **Continue (F5)**: Resume execution until the next breakpoint
- **Step Over (F10)**: Execute the current line and move to the next line
- **Step Into (F11)**: Step into a function call
- **Step Out (Shift+F11)**: Step out of the current function
- **Restart (Ctrl+Shift+F5)**: Restart the debugging session
- **Stop (Shift+F5)**: End the debugging session

## Debugging Tips

1. **Watch Variables**: In the debug sidebar, you can add variables to the watch panel to monitor their values
2. **Call Stack**: View the call stack to understand the execution path
3. **Breakpoint Conditions**: Right-click on a breakpoint to add conditions that must be true for the breakpoint to trigger
4. **Debug Console**: Use the debug console to evaluate expressions during a debugging session

## Troubleshooting

- **No Symbols Loaded**: Ensure you've built with `make build-debug` and not the regular build
- **Breakpoints Not Hitting**: Verify that the code being executed matches the code where you set breakpoints
- **Debugger Not Attaching**: Check that the `k6-debug` binary exists in your project root

## Example Debugging Session

1. Build the debug version: `make build-debug`
2. Set a breakpoint in `weaviate.go` or another source file
3. Press F5 to start debugging
4. When execution hits your breakpoint, inspect variables and step through the code
5. Use the debug controls to navigate through the execution

## Additional Resources

- [Go Debugging in VSCode](https://github.com/golang/vscode-go/blob/master/docs/debugging.md)
- [Delve Debugger Documentation](https://github.com/go-delve/delve/tree/master/Documentation) 