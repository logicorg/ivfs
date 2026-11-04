package main

import (
	api "cue.dev/x/k8s.io/api/core/v1"
	appsv1 "cue.dev/x/k8s.io/api/apps/v1"
)

apiVersion: "cuestomize.dev/v1alpha1"
kind:       "Cuestomization"

input: #Input

includes: _

// outputs is a struct so each file adds its own resources by key.
outputs: [string]: {
	apiVersion: string
	kind:       string
	metadata: name: string
}

#_ImagePullPolicy: "Never" | "Always" | *"IfNotPresent"

// _dbCluster is the CNPG Cluster's name; its -rw Service, -app and -ca
// Secrets follow from it.
_dbCluster: "ivfs-db"

_labels: input.commonLabels & {
	"app.kubernetes.io/part-of": string | *"ivfs"
}

_temporalEnv: [
	{name: "TEMPORAL_HOST_PORT", value: input.temporal.hostPort},
	{name: "TEMPORAL_NAMESPACE", value: input.temporal.namespace},
]

// _goMemLimit keeps the Go heap under the container's limit, so the GC works
// harder before the kernel kills the pod.
_goMemLimit: {name: "GOMEMLIMIT", valueFrom: resourceFieldRef: resource: "limits.memory"}

// _container is what every container shares: the images are distroless,
// nonroot and need no writable filesystem.
_container: {
	imagePullPolicy: #_ImagePullPolicy
	securityContext: {
		allowPrivilegeEscalation: false
		readOnlyRootFilesystem:   true
		capabilities: drop: ["ALL"]
	}
}

_podSecurity: {
	runAsNonRoot: true
	runAsUser:    65532
	runAsGroup:   65532
	seccompProfile: type: "RuntimeDefault"
}

_componentDefaultImage: {
	"ensure-namespace": "\(input.image.registry)/worker:\(input.image.tag)"
	"worker":           "\(input.image.registry)/worker:\(input.image.tag)"
	"gateway":          "\(input.image.registry)/gateway:\(input.image.tag)"
	"yahoo-provider":   "\(input.image.registry)/yahoo-provider:\(input.image.tag)"
}

// _component is a Deployment, its ServiceAccount and, with a port, its
// Service, for one ivfs program.
_component: {
	X=#name: string
	#container: api.#Container & {
		image: string | *_componentDefaultImage[X]
	}
	#port?: int
	#volumes: [...{...}] | *[]

	let podLabels = _labels & {
		"app.kubernetes.io/name":      X
		"app.kubernetes.io/component": X
	}
	let podSelector = {"app.kubernetes.io/name": X}
	let meta = {name: X, namespace: input.namespace, labels: podLabels}

	serviceAccount: {
		apiVersion: "v1"
		kind:       "ServiceAccount"
		metadata:   meta
		// None of them calls the Kubernetes API.
		automountServiceAccountToken: false
	} & api.#ServiceAccount

	deployment: {
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata:   meta
		spec: {
			replicas: 1
			selector: matchLabels: podSelector
			template: {
				metadata: labels: podLabels
				spec: {
					serviceAccountName:           X
					automountServiceAccountToken: false
					enableServiceLinks:           false
					securityContext:              _podSecurity
					containers: [_container & {name: X} & #container]
					if len(#volumes) > 0 {
						volumes: #volumes
					}
				}
			}
		}
	} & appsv1.#Deployment

	if #port != _|_ {
		service: {
			apiVersion: "v1"
			kind:       "Service"
			metadata:   meta
			spec: {
				selector: podSelector
				ports: [{name: "http", port: #port, targetPort: "http"}]
			}
		} & api.#Service
	}
}

// _components turn into outputs, keyed by kind and name.
_components: [string]: _component
for c in _components {
	outputs: "serviceaccount-\(c.#name)": c.serviceAccount
	outputs: "deployment-\(c.#name)":     c.deployment
	if c.service != _|_ {
		outputs: "service-\(c.#name)": c.service
	}
}
