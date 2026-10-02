# opentelemetry-jaeger-tracing

This is a runnable example of req, which uses the built-in tiny github sdk built on req to query and display the information of the specified user.

Best of all, it integrates seamlessly with jaeger tracing and is very easy to extend.

## How to run

Start Jaeger with its OTLP/HTTP receiver exposed on port 4318 and its UI on
port 16686 (see the [Jaeger documentation](https://www.jaegertracing.io/docs/)).
This example uses the supported OpenTelemetry OTLP exporter rather than the
retired Jaeger exporter. The old `JAEGER_ENDPOINT` option is replaced by the
standard `OTEL_EXPORTER_OTLP_ENDPOINT` or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`.

Then, run example:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
go run .
```

PowerShell:

```powershell
$env:OTEL_EXPORTER_OTLP_ENDPOINT = 'http://localhost:4318'
go run .
```

`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` takes a complete URL such as
`http://localhost:4318/v1/traces` and takes precedence over the base endpoint.
Use `go test ./...` to verify export against an in-process collector without
running Jaeger or calling GitHub.
```txt
Please give a github username: 
```

Input a github username, e.g. `imroc`:

```bash
$ go run .
Please give a github username: imroc
The moust popular repo of roc (https://imroc.cc) is req, which have 2500 stars
```

Then enter the Jaeger UI with browser (`http://127.0.0.1:16686/`), checkout the tracing details.

Run example again, try to input some username that doesn't exist, and check the error log in Jaeger UI.
