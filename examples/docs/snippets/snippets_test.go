package snippets

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// Each Example below runs one documented fragment and checks what it
// produced. The fragments are compared with the docs by docs_test.go in the
// repository root.

// greetingScript is the README Quick Start script, used as the script
// variable that README fragments refer to.
const greetingScript = `
	let name = ctx.get("name")

	let p = "."
	if (ctx.get("excited")) {
		p = "!"
	}

	let message = "Hello, " + name + p

	{
		"greeting": message,
		"length": len(message)
	}
`

// check prints a fragment's error, so that a failure shows up in the
// Example's output.
func check(err error) {
	if err != nil {
		fmt.Println("error:", err)
	}
}

func Example_fromFile() {
	scriptPath, err := filepath.Abs("testdata/hello.risor")
	check(err)
	check(fromFile(scriptPath))
	// Output:
	// map[greeting:Hello from disk]
}

func Example_fromHTTP() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer my-api-token" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		_, err := fmt.Fprint(w, `{"greeting": "Hello from HTTP"}`)
		check(err)
	}))
	defer srv.Close()

	check(fromHTTP(srv.URL + "/greet.risor"))
	// Output:
	// map[greeting:Hello from HTTP]
}

func Example_captureLogs() {
	check(captureLogs(greetingScript, map[string]any{"name": "World"}))
	// Output:
	// map[greeting:Hello, World. length:13]
}

func Example_staticProvider() {
	check(staticProvider(greetingScript))
	// Output:
	// map[greeting:Hello, cats! length:12]
}

func Example_contextProvider() {
	check(contextProvider(greetingScript))
	// Output:
	// map[greeting:Hello, Billie Jean. length:19]
}

func Example_derivedContexts() {
	provider := data.NewContextProvider(constants.EvalData)
	check(derivedContexts(
		context.Background(),
		provider,
		map[string]any{"name": "Robert"},
		map[string]any{"excited": true},
	))
	// Output:
	// map[excited:true name:Robert]
}

func Example_staticAndDynamic() {
	check(staticAndDynamic(greetingScript))
	// Output:
	// map[greeting:Hello, Robert! length:14]
}

func Example_starlarkEngine() {
	check(starlarkEngine())
	// Output:
	// map[greeting:Hello, World! length:13]
}

func Example_directRisor() {
	check(directRisor())
	// Output:
	// map[name:World]
}

func Example_directExtism() {
	check(directExtism(wasmdata.TestModule))
	// Output:
	// map[greeting:Hello, World!]
}

func Example_lifetime() {
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	check(lifetime(appCtx, wasmdata.TestModule))
	// Output:
	// map[greeting:Hello, World!]
}

func Example_godocNew() {
	check(godocNew(context.Background(), `"Hello, World"`))
	// Output:
	// Hello, World
}

func Example_godocStaticData() {
	check(godocStaticData(context.Background(), greetingScript))
	// Output:
	// map[greeting:Hello, World. length:13]
}

func Example_godocExtism() {
	check(godocExtism(context.Background(), wasmdata.TestModule))
	// Output:
	// map[greeting:Hello, World!]
}

func Example_godocTypeArgument() {
	check(godocTypeArgument(context.Background(), greetingScript, map[string]any{"name": "World"}))
	// Output:
	// map[greeting:Hello, World. length:13]
}

func Example_godocConstructorRisor() {
	check(godocConstructorRisor(context.Background(), `"Hello, World"`))
	// Output:
	// Hello, World
}

func Example_godocConstructorExtism() {
	check(godocConstructorExtism(context.Background(), wasmdata.TestModule))
	// Output:
	// map[greeting:Hello, World!]
}

func Example_godocRisorQuickStart() {
	check(godocRisorQuickStart(context.Background()))
	// Output:
	// hello
}

func Example_godocRisorOptions() {
	ldr, err := loader.NewFromString(`ctx["version"]`)
	check(err)
	check(godocRisorOptions(context.Background(), ldr))
	// Output:
	// 1.0.0
}

func Example_godocStarlarkQuickStart() {
	check(godocStarlarkQuickStart(context.Background()))
	// Output:
	// hello
}

func Example_godocStarlarkOptions() {
	ldr, err := loader.NewFromString(`_ = ctx["version"]`)
	check(err)
	check(godocStarlarkOptions(context.Background(), ldr))
	// Output:
	// 1.0.0
}

func Example_godocExtismQuickStart() {
	check(godocExtismQuickStart(context.Background(), wasmdata.TestModule))
	// Output:
	// map[greeting:Hello, World!]
}

func Example_godocExtismOptions() {
	ldr, err := loader.NewFromBytes(wasmdata.TestModule)
	check(err)
	check(godocExtismOptions(context.Background(), ldr))
	// Output:
	// map[greeting:Hello, World!]
}

// basicAuthServer returns a test server that serves body to requests with
// the user and pass Basic credentials the platform fragments use.
func basicAuthServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "user" || pass != "pass" {
			http.Error(w, "missing basic auth", http.StatusUnauthorized)
			return
		}
		_, err := fmt.Fprint(w, body)
		check(err)
	}))
}

func Example_godocCompositeAddData() {
	check(godocCompositeAddData(map[string]any{"debug": true}, map[string]any{"name": "World"}))
	// Output:
	// map[config:map[debug:true] name:World]
}

func Example_godocSetterAddData() {
	ctx := context.Background()
	evaluator, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(`ctx["greeting"]`))
	check(err)
	request := httptest.NewRequest(http.MethodGet, "/greet", nil)
	check(godocSetterAddData(ctx, evaluator, request))
	// Output:
	// Hello, World!
}

func Example_godocHTTPOptions() {
	srv := basicAuthServer(`"Hello from HTTPOptions"`)
	defer srv.Close()

	check(godocHTTPOptions(context.Background(), srv.URL+"/script.risor"))
	// Output:
	// "Hello from HTTPOptions"
}

func Example_godocNewFromHTTP() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprint(w, `"Hello from NewFromHTTP"`)
		check(err)
	}))
	defer srv.Close()

	check(godocNewFromHTTP(context.Background(), srv.URL+"/script.risor"))
	// Output:
	// "Hello from NewFromHTTP"
}

func Example_godocNewFromHTTPWithOptions() {
	srv := basicAuthServer(`"Hello from NewFromHTTPWithOptions"`)
	defer srv.Close()

	check(godocNewFromHTTPWithOptions(context.Background(), srv.URL+"/script.risor"))
	// Output:
	// "Hello from NewFromHTTPWithOptions"
}
