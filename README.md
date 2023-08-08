# JZ NOTES: 
This branch is for local-testing of both the `console-dashboars-plugin` and `monitoring-plugin`. 
`start-console.sh` has been modified `BRIDGE_PLUGIN`` and `BRIDGE_PROXY` to point at the `monitoring-plugin`. 

```
cd console-dashboards-plugin
git branch local-dev-monitoring 
make install 
make start-backend 
# in a different terminal
make start-frontent 
# in a different terminal
make start-console

# start monitoring-plugin for local development
 cd monitoring-plugin
 git chekcout branch local-dev-dashboards
 yarn install
 yarn start 

 # point your browser to localhost:9000, both dynamic plugins (`console-dashboards-plugin` and `monitoring-plugin`) should be loaded
```




# Dashboards Dynamic Plugin for OpenShift Console

This plugin adds custom datasources for OpenShift dashboards. It requires OpenShift 4.10+

##

- [Development](#development)
- [Deployment on cluster](#deployment-on-cluster)
- [Add a new Datasource](#add-a-new-datasource)

## Development

[Node.js](https://nodejs.org/en/), [npm](https://www.npmjs.com/) and [go](https://go.dev/) are required
to build and run the plugin. To run OpenShift console in a container, either
[Docker](https://www.docker.com) or [podman 3.2.0+](https://podman.io) and
[oc](https://console.redhat.com/openshift/downloads) are required.

### Running locally

1. Install the dependencies with `make install`
2. Start the backend with `make start-backend`
3. In a different terminal, start the frontend with `make start-frontend`
4. In a different terminal, start the console
   a. `oc login` (requires [oc](https://console.redhat.com/openshift/downloads) and an [OpenShift cluster](https://console.redhat.com/openshift/create))
   b. `make start-console` (requires [Docker](https://www.docker.com) or [podman 3.2.0+](https://podman.io))

This will run the OpenShift console in a container connected to the cluster you've logged into. The plugin backend server
runs on port 9002 with CORS enabled.

Navigate to <http://localhost:9000> to see the running plugin.

### Building the image

```sh
make build-image
```

## Deployment on cluster

You can deploy the plugin into a cluster by running the helm chart at `charts/console-dashboards-plugin`.
It will use the image from `quay.io/gbernal/console-dashboards-plugin:0.0.1` and run a go HTTP server
to serve the plugin's assets and proxy to the configured datasources.

```sh
helm upgrade -i console-dashboards-plugin charts/console-dashboards-plugin -n console-dashboards --create-namespace
```

JZ image: 
```sh
helm upgrade -i console-dashboards-plugin charts/console-dashboards-plugin -n console-dashboards --create-namespace --set plugin.image=quay.io/jezhu/console-dashboards-plugin:0.0.1
```

## Add a new Datasource

See [add datasource docs](docs/add-datasource.md)
