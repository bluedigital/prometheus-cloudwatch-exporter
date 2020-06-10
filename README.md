# Prometheus CloudWatch Exporter

Why another CloudWatch exporter? Mainly 'cause other exporters don't really do auto discovery (requiring you to tag each resource, or to wait for them to support the namespace in the exporter) of available metrics or use the [GetMetricData](https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetMetricData.html) API optimizing costs for API calls.

## Features

* Support hot reload of configuration through GET request to `/reload` or `SIGHUP`
* Walk through past metrics advancing a period on each scrape
* Auto discovery done correctly
* Support to use timestamp annotations (using CloudWatch data point timestamp)
* Custom labels and metrics names
* All configurations formats supported by [viper](https://github.com/spf13/viper)

## Project Status

This project is currently on alpha stage and although we intend to keep the API compatibility, it needs to be tested, documented. Fill an issue if you want to contribute! We're actively working on Prometheus and its ecosystem.

## Installing

`cd cmd/prometheus-cloudwatch-exporter && go install`

## Setup

You can run `prometheus-cloudwatch-exporter --help` to see the list of all available command args. The configuration to collect metrics is described below:

```yml
exporter:
  # If not specified the exporter will use the "--aws-region" configuration
  regions:
    - us-east-1
  metrics:
    # Specify the AWS CloudWatch metric namespace, example: "AWS/EC2"
    - namespace: CWAgent
      # Period to apply to GetMetricData query (seconds, optional, default is 60)
      period: 60
      # Delay to move back start and end time for GetMetricData query (seconds, optional, default is 600)
      delay: 72000
      # Which dimensions should the exporter use to get the metrics?
      dimensions: [InstanceId, InstanceType, ImageId, cpu]
      # Metrics specifications
      specs:
          # CloudWatch metric name
        - name: cpu_usage_user
          # Which statistic to get metric
          statistic: Maximum
          # Which unit to get metric
          unit: Percent
          # What is the name of the metric to be exposed at the metrics endpoint?
          promName: cpu_usage_user_maximum
        - name: cpu_usage_idle
          statistic: Minimum
          # Override the default region and the exporter's default regions
          regions: [sa-east-1]
          unit: Percent
          promName: cpu_usage_idle_minimum
          # Use the CW timestamp for metric data point? (default is false)
          timestamp: true
          # Custom labels to add to Prometheus metric (key: value)
          customLabels:
            account: XYZ
```

Save the file as **config.yml** and the automagic happens running:

```bash
prometheus-cloudwatch-exporter \
    --aws-access-key-id="YOUR_ACCESS_KEY_ID" \
    --aws-secret-access-key="YOUR_SECRET_ACCESS_KEY" \
    --config "config.yml"
```

Now open your browser at [localhost:9016/metrics](http://localhost:9016/metrics).

## Roadmap

- [ ] Test
- [ ] Documentation
- [ ] Contributing Guide
- [ ] Support multiple AWS accounts (each metric should support specific AWS credentials?)
