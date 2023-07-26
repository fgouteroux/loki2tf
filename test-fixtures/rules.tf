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

