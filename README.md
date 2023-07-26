# loki2tf - Loki YAML Prometheus-compatible Rules to Terraform HCL converter

A tool for converting Loki Prometheus-compatible Rules (in YAML format) into HashiCorp's Terraform configuration language.

The converted `.tf` files are suitable for use with the [Terraform Loki Provider](https://registry.terraform.io/providers/fgouteroux/loki/latest/docs)


## Installation

**Pre-built Binaries**

Download Binary from GitHub [releases](https://github.com/fgouteroux/loki2tf/releases/latest) page.


## YAML to HCL

**Convert a single YAML file and write generated Terraform config to Stdout**

```
$ loki2tf -f test-fixtures/rules.yaml

resource "loki_rule_group_alerting" "should_fire" {
  name = "should_fire"

  rule {
    alert = "HighPercentageError"

    expr = <<EOT
sum(rate({app="foo", env="production"} |= "error" [5m])) by (job)
  /
sum(rate({app="foo", env="production"}[5m])) by (job)
  > 0.05
EOT

    for = "10m"

    labels = {
      severity = "page"
    }

    annotations = {
      summary = "High request latency"
    }
  }
}

resource "loki_rule_group_alerting" "credentials_leak" {
  name = "credentials_leak"

  rule {
    alert = "http-credentials-leaked"
    expr  = "sum by (cluster, job, pod) (count_over_time({namespace=\"prod\"} |~ \"http(s?)://(\\w+):(\\w+)@\" [5m]) > 0)"
    for   = "10m"

    labels = {
      severity = "critical"
    }

    annotations = {
      message = "{{ $labels.job }} is leaking http basic auth credentials."
    }
  }
}

resource "loki_rule_group_recording" "NginxRules" {
  name = "NginxRules"

  rule {
    record = "nginx:requests:rate1m"

    expr = <<EOT
sum(
  rate({container="nginx"}[1m])
)
EOT

    labels = {
      cluster = "us-central1"
    }
  }
}

```

**Convert a single YAML file and write output to file**

```
$ loki2tf -f test-fixtures/rules.yaml -o rules.tf
```

**Convert a directory of Loki YAML files and write output to stdout**

```
$ loki2tf -f test-fixtures/
```

**Convert a directory of Loki YAML files and write output to file**

```
$ loki2tf -f test-fixtures/ -o /tmp/rules.tf
```

## HCL to YAML

**Convert a single HCL file and write yaml output to file**

```
$ loki2tf -r -f test-fixtures/rules.tf -o /tmp/rules.yaml
```

**Convert a directory of Loki HCL files to YAML file**

```
$ loki2tf -r -f test-fixtures -o /tmp/rules.yaml

```

## Building

> **NOTE** Requires a working Golang build environment.

This project uses Golang modules for dependency management, so it can be cloned outside of the `$GOPATH`.

**Clone the repository**

```
$ git clone https://github.com/fgouteroux/loki2tf.git
```

**Build**

```
$ cd loki2tf
$ make build
```
