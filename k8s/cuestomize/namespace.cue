package main

outputs: namespace: {
	apiVersion: "v1"
	kind:       "Namespace"
	metadata: {
		name: input.namespace
		labels: _labels & {
			// Every pod here already meets restricted; enforcing it keeps a
			// later change from quietly loosening one.
			"pod-security.kubernetes.io/enforce": "restricted"
			"pod-security.kubernetes.io/audit":   "restricted"
			"pod-security.kubernetes.io/warn":    "restricted"
		}
	}
}
