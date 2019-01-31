# kube-solskin-controller
A simple service to log or suppress Kubernetes resources in a cluster that do not meet basic best practices.

## Best Practices Checks
There are currently four primary checks that the _kube-solskin-controller_ service will perform on every pod, deployment, and daemon set:
  - **Observability**: does the resource export Prometheus metrics of some sort?
  - **Liveness**: does the resource possess a liveness check?
  - **Readiness**: does the resource possess a readiness check?
  - **Resource Requests**: does the resource possess resource requests?
  - **Resource Limits**: does the resource possess resource limits?

These checks are extremely simple. At present they only check to see if the resource has any kind of configuration set for these properties. This forces the owner of the resource to at least give some thought to these practices, but doesn't limit them in any way.

## Configuration
At the time of this writing, the service is only configurable via environment variables, but uses `micro/go-config` thus adding more sources of configuration will be relatively simple. Below is a table of configurable values for the service.

| Key | Description | Default |
|-----|-------------|---------|
| SOLSKIN_ELIGIBLITY_AGE_LIMIT | Kubernetes resources that are younger than the supplied duration here are ignored. Format is dictated by `time.ParseDuration`. A value of `off` disables this check. | off |
| SOLSKIN_ELIGIBILITY_EXCLUDE_NAMESPACE | Namespaces matching this regular expression will be exempt from suppression by this service. | ^kube- |
| SOLSKIN_METRICS_ENDPOINT | The endpoint that serves the metrics. | metrics |
| SOLSKIN_METRICS_PORT | The port that the webserver listen on. | 8080 |
| SOLSKIN_SUPPRESSOR_ACTION | The action the suppressor service will take when it detects a subpar resource. Available values are `none`, `log`, and `suppress`. | log |

## Gotchas
Due to the fact that suppression of Kubernetes resources is a **destructive** action, the default value for the action the suppressor should take is set to `log`. This value must be set to `suppress` before the suppressor will actively manage resources.

## Contributing
Please feel free to create issues or PRs, or just join the discussion! This repository is a prototype at best and could probably use a good rework, as well as good discussions for more / better checks.