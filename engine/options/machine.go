package options

// IMPORTANT: Machine-specific options have been moved to their respective packages.
// Do not use engine.Option for machine-specific configuration.
//
// Use machine options directly instead:
// - extism.WithEntryPoint() - NOT options.WithExtismEntryPoint()
// - risor.WithGlobals() - NOT options.WithRisorGlobals()
// - risor.WithCtxGlobal() - NO wrapper exists
// - starlark.WithGlobals() - NOT options.WithStarlarkGlobals()
// - starlark.WithCtxGlobal() - NO wrapper exists
//
// All of these options can be passed directly to the machine-specific evaluator creators:
// - NewExtismEvaluator(options.WithLoader(l), extism.WithEntryPoint("main"))
// - NewRisorEvaluator(options.WithLoader(l), risor.WithCtxGlobal())
// - NewStarlarkEvaluator(options.WithLoader(l), starlark.WithCtxGlobal())
