# Machine Implementations

This package contains virtual machine implementations for executing scripts in various languages through a consistent interface. While each supported VM has its own unique characteristics, they all follow a standardized flow pattern.

## Design Philosophy

1. **Common Interface**: All VMs present the same interface (`evaluationEvaluator`) regardless of underlying implementation
2. **Separation of Concerns**: Compilation, data preparation, and execution are distinct phases
3. **Thread-safe Evaluation**: Each VM is designed to allow concurrent execution of scripts
3. **Context-Based Data Flow**:  Runtime data is accessed with a `context.Context` object (saved/loaded with a `data.Provider`) 
4. **Execution Results**: All VMs return the same `evaluation.EvaluatorResponse` object, which contains the execution result and metadata

## Dataflow & Architecture

1. **Compilation Instantiation**
   - Each VM has a `NewCompiler` function that returns a compiler instance that implements the `script.Compiler` interface
   - The `NewCompiler` function may have some VM-specific options
   - The `Compiler` object includes a `Compile` method that takes a `loader.Loader` implementation
   - `loader.Loader` is a generic way to load script content from various sources
   - Compile-time errors are captured and returned to the caller
   - A `script.ExecutableContent` is returned by `Compile`

2. **Executable Creation Stage**
   - The `script.ExecutableUnit` is a wrapper around the `script.ExecutableContent`
   - `NewExecutableUnit` receives a `Compiler` and several other objects
   - Calls the `script.Compiler` to compile the script, storing the result in the `ExecutableContent`
   - The `ExecutableUnit` is responsible for managing the lifecycle of the script execution

3. **Evaluator Creation**
   - `NewEvaluator` takes a `script.ExecutableUnit` and returns an object that implements `evaluationEvaluator`
   - At this point it can be called with `.Eval(ctx)`, however input data is required it must be prepared

4. **Data Preparation Stage**
   - This phase is optional, and must happen prior to evaluation when runtime input data is used
   - The `Evaluator` implements the `evaluationEvaluator` interface, which has a `PrepareContext` method
   - The `PrepareContext` method takes a `context.Context` and a variadic list of `any`
   - `PrepareContext` calls the `data.Provider` to convert and store the data, somewhere accessible to the Evaluator
   - The conversion is fairly opinionated, and handled by the `data.Provider`
   - For example, it converts an `http.Request` into a `map[string]any` using the schema in `helper.RequestToMap`
   - The `PrepareContext` method returns a new context with the data stored or linked in it

5. **Execution Stage**
   - When `Eval(ctx)` is called, the `data.Provider` first loads the input data into the VM
   - The VM executes the script and returns an `evaluation.EvaluatorResponse`

6. **Result Processing**
   - The process for building the `evaluation.EvaluatorResponse` is different for each VM
   - There are several type conversions, and the result is accessible with the `Interface()` method
   - The `evaluation.EvaluatorResponse` also contains metadata about the execution
