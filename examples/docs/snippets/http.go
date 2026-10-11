package snippets

// snippet:begin http
import (
	"context"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// snippet:end

// fromHTTP is the README "From an HTTP endpoint" fragment.
func fromHTTP(scriptURL string) error {
	// snippet:begin http
	ctx := context.Background()

	httpOpts := loader.DefaultHTTPOptions().
		WithTimeout(10 * time.Second).
		WithBearerAuth("my-api-token")

	// scriptURL is the script's location, such as "https://scripts.example.com/greet.risor"
	httpLoader, err := loader.NewFromHTTPWithOptions(scriptURL, httpOpts)
	if err != nil {
		return err
	}

	evaluator, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromLoader(httpLoader))
	// snippet:end
	return evalAndPrint(ctx, evaluator, err)
}
