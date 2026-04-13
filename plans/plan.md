# telemetry-go plan

## Goals
- Opinionated OpenTelemetry integration for Go.
- Simple SDK bootstrap.
- Logger always on.
- Metrics conditional.
- Traces usable for correlation even when not exported.

## Packages
- `telemetry`
- `tlog`
- `terror`
- `thttpclient`
- `thttpserver`
- `tsql`
- `tcaller`

## SDK bootstrap
```go
sdk := telemetry.InitSDK(ctx, telemetry.SDK().ServiceName("svc"))
defer sdk.Shutdown(ctx)

if err := sdk.Err(); err != nil {
	tlog.Log(ctx, tlog.Opt().Message("failed to init sdk: %v", err))
}
```

Rules:
- `InitSDK` always returns non-nil SDK.
- Init errors are aggregated as degraded init error.
- Logger always active.
- Metrics enabled only if OTLP metrics endpoint is explicitly resolved.
- Traces provider always exists for correlation; exporter only enabled if OTLP traces endpoint is explicitly resolved.
- OTLP enablement must be explicit; do not silently rely on localhost exporter defaults.

## Option architecture
- API boundaries accept interfaces with unexported apply methods.
- Factories return mutable pointer builders, MongoDB-style.
- Merge order is positional; last write wins for singular fields, append for collection fields.

### SDK
```go
type SDKOption interface { applySDK(*sdkConfig) }
type SDKOptions struct{ /* internal */ }

func SDK() *SDKOptions
func InitSDK(ctx context.Context, opts ...SDKOption) *SDK
```

Planned methods:
- `ServiceName(string)`
- `ServiceVersion(string)`
- `DeploymentEnvironment(string)`
- `ResourceAttributes(...attribute.KeyValue)`
- `OTLPEndpoint(string)`
- `OTLPHeaders(...string)`
- `OTLPProtocol(string)`
- `OTLPLogsEndpoint(string)`
- `OTLPLogsHeaders(...string)`
- `OTLPLogsProtocol(string)`
- `OTLPTracesEndpoint(string)`
- `OTLPTracesHeaders(...string)`
- `OTLPTracesProtocol(string)`
- `OTLPMetricsEndpoint(string)`
- `OTLPMetricsHeaders(...string)`
- `OTLPMetricsProtocol(string)`
- `TraceSampler(sdktrace.Sampler)`
- `Propagators(propagation.TextMapPropagator)`
- `LoggerAddSource(bool)`

## Logging

### Public API
```go
type LogOption interface { applyLog(*logEntry) }
type LogOptions struct{ /* internal */ }

func Opt() *LogOptions
func Log(ctx context.Context, opts ...LogOption)
```

`LogOptions` planned methods:
- `Level(slog.Leveler)`
- `Message(string, ...any)`
- `Fields(...any)`
- `Caller(tcaller.Caller)`
- `Time(time.Time)`

Rules:
- Default level: `slog.LevelInfo`
- Default time: `time.Now()`
- Default caller: callsite of `tlog.Log(...)`
- If message empty and fields empty, skip logging.

### Entry API
```go
type Entry struct{ /* internal */ }

func Entry() *Entry
```

Planned methods:
- `Level(slog.Leveler) *Entry`
- `Message(string, ...any) *Entry`
- `Fields(...any) *Entry`
- `Caller(tcaller.Caller) *Entry`
- `Time(time.Time) *Entry`
- `Log(context.Context)`
- `Print(context.Context)`

### Classic compatibility
- `Print(args ...any)`
- `Printf(format string, args ...any)`
- `Println(args ...any)`

Rules:
- Use `context.Background()`.
- Default level info.

### Context helpers
```go
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context
func AttrsFromContext(ctx context.Context) []slog.Attr

func WithCaller(ctx context.Context, caller tcaller.Caller) context.Context
func WithCurrentCaller(ctx context.Context) context.Context
func CallerFromContext(ctx context.Context) (tcaller.Caller, bool)
```

Rules:
- Use private custom key structs.
- Immutable chain behavior.
- Upstream attrs/caller propagate to downstream.
- Sibling contexts are isolated.
- Caller from context affects logging only, not tracing.

### Stderr behavior
TTY:
```text
[15:04:05.000] [INFO] <file.go:123> message {details}
```

Rules:
- Level colored.
- Caller shown when available.
- Details rendered as colored pretty JSON via `github.com/tidwall/pretty`.
- Header fields excluded from details: `time`, `level`, `msg`, `source`.

Non-TTY:
- Pure JSON output, as close as possible to `slog.JSONHandler`.

### Value normalization
- `time.Duration` -> `duration.String()`.
- `json.RawMessage` -> raw JSON for stderr, JSON string for OTLP attrs.
- `[]byte` valid JSON -> raw JSON for stderr.
- `[]byte` invalid JSON -> string/binary-safe representation.
- `json.Marshaler` -> passthrough for stderr, JSON string for OTLP attrs.
- `encoding.TextMarshaler` -> passthrough text.
- Unknown `any` -> try `json.Marshal`; on failure use `slog.StringValue("ERROR!<err>")`.

### Log correlation
If span context is valid:

For OTLP logs:
- Set top-level `TraceId`, `SpanId`, `TraceFlags`.
- Inject attrs `trace_id`, `span_id`, `trace_flags`.

For stderr / non-OTLP:
- Inject fields `trace_id`, `span_id`, `trace_flags`.

Rules:
- User-provided attrs may override attr copies.
- Top-level OTLP trace context remains canonical from actual span context.

### Logging implementation direction
- Do not use `otelslog` as core implementation.
- Use custom pipeline: normalize record -> stderr sink / OTLP sink.
- Preserve source/caller.
- OTLP source maps to `code.*`.

## Tracing

### Public API
```go
type StartSpanOption interface { applySpan(*startSpanConfig) }
type StartSpanOptions struct{ /* internal */ }

func Span() *StartSpanOptions
func StartSpan(ctx context.Context, opts ...StartSpanOption) (context.Context, trace.Span)
```

`StartSpanOptions` planned methods:
- `Name(string)`
- `Kind(trace.SpanKind)`
- `Attributes(...attribute.KeyValue)`
- `Caller(tcaller.Caller)`
- `StartTime(time.Time)`
- `StartOptions(...trace.SpanStartOption)`
- `TracerName(string)`
- `TracerVersion(string)`
- `Links(...trace.Link)`
- `NewRoot(bool)`

Defaults:
- Span name: short caller name of `telemetry.StartSpan(...)`
- Tracer name: library default
- Start time: `time.Now()`
- Caller: stack location of `telemetry.StartSpan(...)`
- Context caller is not used for tracing

Precedence:
- Name: explicit name -> explicit caller -> auto caller short name
- Tracer name: explicit -> default library tracer name
- Start time: explicit -> now

Tracing metadata:
- Resolve caller for default short span name.
- Fill span attrs `code.function`, `code.filepath`, `code.lineno` when possible.

Trace export behavior:
- Provider remains available for correlation even when exporter disabled.
- Exporter only enabled if OTLP traces endpoint explicitly resolved.

## tcaller
- `Caller` type
- Resolve `uintptr`/PC -> file, line, function
- Cached lookup
- Short function naming helper for tracing
- Pretty formatting helper for logging

## terror
- App-level error classification
- Error enrichment for logs and spans
- Helpers mapping error -> category/status/attrs

## HTTP and SQL
- `thttpclient`: client-side logging/metric/tracing
- `thttpserver`: server-side logging/metric/tracing
- `tsql`: logging/metric/tracing

Rule:
- These packages do not initialize SDK; they only consume global/injected providers.

## Implementation phases
1. Bootstrap project structure and SDK skeleton.
2. Implement `tcaller`.
3. Implement `tlog` core, context scopes, stderr renderers, OTLP log sink.
4. Implement `telemetry.StartSpan` and tracing core.
5. Implement `terror`.
6. Implement `thttpclient` and `thttpserver`.
7. Implement `tsql`.
8. Add tests, examples, and benchmarks.
