# Add a new datasource

The plugin will search for datasources as ConfigMaps in the `openshift-config-managed` namespace with the `console.openshift.io/dashboard-datasource: 'true'` label

The configmap must define a datasource type and an in-cluster service where the data can be fetched:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-custom-prometheus-datasource
  namespace: openshift-config-managed
  labels:
    console.openshift.io/dashboard-datasource: 'true'
data:
  'dashboard-datasource.yaml': |-
    kind: "Datasource"
    metadata:
      name: "my-custom-prometheus-datasource"
      project: "openshift-config-managed"
    spec:
      plugin:
        kind: "PrometheusDatasource"
        spec:
          direct_url: "https://my-custom-prometheus-service.my-service-namespace.svc.cluster.local:9091"
```

After the datasource is added a custom dashboard can be configured to connect to it, this can be in a panel or variable (templating)

```
kind: ConfigMap
apiVersion: v1
metadata:
  name: grafana-dashboard-api-performance-custom
  namespace: openshift-config-managed
  labels:
    console.openshift.io/dashboard: "true"
data:
  api-performance.json: |-
    {
      "panels": [
        {
          ...

          "datasource": {
            "name":"my-custom-prometheus-datasource",
            "type":"prometheus"
          },

          ...
        }
      ],
      "templating": {
        "list": [
          {
            ...

            "datasource": {
              "name":"my-custom-prometheus-datasource",
              "type":"prometheus"
            },

            ...
          }
        ]
      }
    }

```
