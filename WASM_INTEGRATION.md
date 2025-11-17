# Ferret WASM Integration Guide

## ✅ What's Done

1. **WASM Build**: Compiled Ferret to `website/public/ferret.wasm`
2. **WASM Support**: Copied Go's `wasm_exec.js` to `website/public/`
3. **Playground Integration**: Added WASM loader and initialization

## ⚠️ Required Compiler Modifications

Your Ferret compiler currently expects file paths as arguments. To work with WASM in the browser, you need to modify it to:

### Option 1: Accept Code via Stdin (Recommended)

Modify `main.go` to accept code from stdin when no file argument is provided:

```go
package main

import (
	"flag"
	"io"
	"os"
	"strings"
)

func main() {
	stdinFlag := flag.Bool("stdin", false, "Read code from stdin")
	flag.Parse()
	
	var code string
	var err error
	
	if *stdinFlag || len(flag.Args()) == 0 {
		// Read from stdin
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		code = string(input)
	} else {
		// Read from file (existing behavior)
		filePath := flag.Args()[0]
		input, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		code = string(input)
	}
	
	// Compile and run the code
	result := compileAndRun(code)
	fmt.Println(result)
}
```

### Option 2: Use Go's WASM Filesystem API

Use Go's virtual filesystem in WASM:

```go
//go:build js && wasm

package main

import (
	"syscall/js"
)

func compileFromBrowser(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "Expected 1 argument (code string)",
		}
	}
	
	code := args[0].String()
	
	// Compile the code
	result, err := compile(code)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}
	
	return map[string]interface{}{
		"success": true,
		"output":  result,
	}
}

func main() {
	c := make(chan struct{})
	
	// Register JavaScript function
	js.Global().Set("ferretCompile", js.FuncOf(compileFromBrowser))
	
	println("Ferret WASM Compiler Ready")
	<-c // Keep the program running
}
```

Then update the playground JavaScript:

```javascript
// Call the WASM compiler
const result = await new Promise((resolve) => {
  const res = ferretCompile(code);
  resolve(res);
});

if (result.success) {
  // Show output
  output.innerHTML = `<pre>${escapeHtml(result.output)}</pre>`;
} else {
  // Show error
  output.innerHTML = `<pre style="color: #ef4444;">${escapeHtml(result.error)}</pre>`;
}
```

## 🚀 Next Steps

1. **Choose an approach** (Option 2 recommended for browser integration)
2. **Modify your compiler** according to the chosen option
3. **Rebuild WASM**:
   ```powershell
   $env:GOOS="js"; $env:GOARCH="wasm"; go build -o website\public\ferret.wasm main.go
   ```
4. **Update playground** to call the WASM function
5. **Test** in browser

## 📦 File Structure

```
Ferret/
├── main.go (needs modification)
├── website/
│   ├── public/
│   │   ├── ferret.wasm (built)
│   │   └── wasm_exec.js (copied)
│   └── src/
│       └── components/
│           └── Playground.astro (updated with WASM loader)
```

## 🔍 Benefits of WASM Approach

- ✅ No backend server needed
- ✅ Faster execution (no network latency)
- ✅ Works offline
- ✅ Scales infinitely (runs on user's machine)
- ✅ Lower hosting costs

## ⚠️ Current Limitation

The playground will show an error message until the compiler is modified to work with WASM.

## 🛠️ Testing

After modifications, restart the dev server and visit the playground:
```powershell
cd website
npm run dev
```

The compiler will load automatically when the page loads (check browser console for "✅ Ferret WASM compiler loaded").
