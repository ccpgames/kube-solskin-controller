# solskin
A simple service to log or suppress Kubernetes resources in a cluster that do not meet basic best practices.

## Documentation
TODO

## Best Practices Checks
There are currently four primary checks that the _solskin_ service will perform on every pod, deployment, and daemon set:
  - **Observability**: does the resource export Prometheus metrics of some sort?
  - **Liveness**: does the resource possess a liveness check?
  - **Readiness**: does the resource possess a readiness check?
  - **Resource Requests**: does the resource possess resource requests?
  - **Resource Limits**: does the resource possess resource limits?

## Contributing
Please feel free to create issues or PRs, or just join the discussion! This repository is a prototype at best and could probably use a good rework, as well as good discussions for more / better checks.

## FAQs
TODO