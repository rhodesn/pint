---
layout: default
parent: Checks
grand_parent: Documentation
---

# promql/series

This check will also query Prometheus servers, it is used to warn about queries
that are using metrics not currently present in Prometheus.
It parses `expr` query from every rule, finds individual metric selectors and
checks if they return any values.

Let's say we have a rule this query: `sum(my_metric{foo="bar"}) > 10`.
This checks would query all configured server for the existence of
`my_metric{foo="bar"}` series and report a warning if it's missing.

## Configuration

This check doesn't have any configuration options.

## How to enable it

This check is enabled by default for all configured Prometheus servers.

Example:

```js
prometheus "prod" {
  uri     = "https://prometheus-prod.example.com"
  timeout = "60s"
  paths = [
    "rules/prod/.*",
    "rules/common/.*",
  ]
}

prometheus "dev" {
  uri     = "https://prometheus-dev.example.com"
  timeout = "30s"
  paths = [
    "rules/dev/.*",
    "rules/common/.*",
  ]
}
```

## How to disable it

You can disable this check globally by adding this config block:

```js
checks {
  disabled = ["promql/series"]
}
```

Or you can disable it per rule by adding a comment to it:

`# pint disable promql/series`

If you want to disable only individual instances of this check
you can add a more specific comment.

`# pint disable promql/series($prometheus)`

Where `$prometheus` is the name of Prometheus server to disable.

Example:

`# pint disable promql/series(prod)`

You can also disable `promql/series` for specific metric using
`# pint disable promql/series($selector)` comment.

Just like with PromQL if a selector doesn't have any labels then it will match all instances,
if you pass any labels it will only pass time series with those labels.

Disable warnings about missing `my_metric_name`:

```YAML
# pint disable promql/series(my_metric_name)
```

Disable it only for `my_metric_name{cluster="dev"}` but still warn about
`my_metric_name{cluster="prod"}`:

```YAML
# pint disable promql/series(my_metric_name{cluster="dev"})
```