package main

outputs: "httproute-gateway": {
	apiVersion: "gateway.networking.k8s.io/v1"
	kind:       "HTTPRoute"
	metadata: {
		name:      "gateway"
		namespace: input.namespace
		labels: _labels & {"app.kubernetes.io/name": "gateway", "app.kubernetes.io/component": "gateway"}
	}
	spec: {
		parentRefs: [for p in input.route.parentRefs {
			group: "gateway.networking.k8s.io"
			kind:  "Gateway"
			p
		}]
		// The gateway's Host guard refuses any other name, so the route
		// matches only the one it accepts.
		hostnames: [input.hostname]
		rules: [{
			backendRefs: [{name: "gateway", port: _components.gateway.#port}]
		}]
	}
}
