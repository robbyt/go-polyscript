package snippets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// godocCompositeAddData is the fragment in the
// data.CompositeProvider.AddDataToContext doc.
func godocCompositeAddData(configData, userData map[string]any) error {
	// snippet:begin composite-add-data
	ctx := context.Background()
	staticProvider := data.NewStaticProvider(map[string]any{"config": configData})
	contextProvider := data.NewContextProvider(constants.EvalData)
	composite := data.NewCompositeProvider(staticProvider, contextProvider)
	ctx, err := composite.AddDataToContext(ctx, userData)
	// snippet:end
	if err != nil {
		return err
	}
	merged, err := composite.GetData(ctx)
	if err != nil {
		return err
	}
	fmt.Println(merged)
	return nil
}

// godocSetterAddData is the fragment in the data.Setter.AddDataToContext
// doc.
func godocSetterAddData(ctx context.Context, evaluator platform.Evaluator, request *http.Request) error {
	// snippet:begin setter-add-data
	scriptData := map[string]any{"greeting": "Hello, World!", "request": request}
	enrichedCtx, err := evaluator.AddDataToContext(ctx, scriptData)
	if err != nil {
		return err
	}
	result, err := evaluator.Eval(enrichedCtx)
	// snippet:end
	return printResult(result, err)
}

// godocHTTPOptions is the fragment in the loader.HTTPOptions doc.
func godocHTTPOptions(ctx context.Context, scriptURL string) error {
	// snippet:begin http-options
	options := loader.DefaultHTTPOptions()
	options.Timeout = 10 * time.Second
	options = options.WithBasicAuth("user", "pass")
	// snippet:end
	httpLoader, err := loader.NewFromHTTPWithOptions(scriptURL, options)
	if err != nil {
		return err
	}
	return printScript(ctx, httpLoader)
}

// godocNewFromHTTP is the fragment in the loader.NewFromHTTP doc.
func godocNewFromHTTP(ctx context.Context, scriptURL string) error {
	// snippet:begin new-from-http
	httpLoader, err := loader.NewFromHTTP(scriptURL)
	if err != nil {
		return err
	}
	// snippet:end
	return printScript(ctx, httpLoader)
}

// godocNewFromHTTPWithOptions is the fragment in the
// loader.NewFromHTTPWithOptions doc.
func godocNewFromHTTPWithOptions(ctx context.Context, scriptURL string) error {
	// snippet:begin new-from-http-with-options
	opts := loader.DefaultHTTPOptions().
		WithBasicAuth("user", "pass"). // or WithBearerAuth("token123")
		WithTimeout(10 * time.Second)
	httpLoader, err := loader.NewFromHTTPWithOptions(scriptURL, opts)
	if err != nil {
		return err
	}
	// snippet:end
	return printScript(ctx, httpLoader)
}

// printScript prints the script content a loader returns.
func printScript(ctx context.Context, l loader.Loader) (err error) {
	reader, err := l.GetReader(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	fmt.Println(string(content))
	return nil
}
