@experiment(try)

package main

import (
	"strings"
	api "cue.dev/x/k8s.io/api/core/v1"
)

_#ImageOptions: {
	image?:          string
	imagePullPolicy: #_ImagePullPolicy
}

// #Input is what a Cuestomization's input may set. It is closed, so a typo
// fails the build instead of silently falling back to a default.
#Input: {
	// namespace is ivfs's own; the module creates it.
	namespace: #Name | *"ivfs"

	// hostname is where browsers reach the gateway, over HTTPS alone: the
	// session cookie is Secure.
	hostname: string & =~"^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$"

	commonLabels: [string]: string | *{}

	image: {
		registry: string | *"ghcr.io/logicorg/ivfs"
		// tag is required: a moving default would upgrade on any re-render.
		tag: =~"^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$" | *""
	}

	// if image.tag is unset, all images must be set explicitly
	if image.tag == "" {
		worker: image:  strings.MinRunes(1)
		job: image:     strings.MinRunes(1)
		gateway: image: strings.MinRunes(1)
		if providers.yahoo.enabled {
			providers: yahoo: image!: strings.MinRunes(1)
		}
	}

	try {
		if len(worker.image?) > 0 && len(job.image?) > 0 && len(gateway.image?) > 0 && (
			!providers.yahoo.enabled || len(providers.yahoo.image?) > 0) {
			image: tag: "" | error("image.tag must be unset when all images are set explicitly")
		}
	}

	worker: {
		_#ImageOptions

		// replicas is how many workers run; the Job queue balances them.
		replicas: int & >=1 | *1

		runtimeClassName?: string

		affinity?: api.#Affinity
	}

	job: {
		_#ImageOptions

		affinity?: api.#Affinity
	}

	gateway: {
		_#ImageOptions

		runtimeClassName?: string

		affinity?: api.#Affinity
	}

	temporal: {
		// hostPort is the frontend, in Temporal's own Kubernetes namespace,
		// such as temporal-frontend.temporal:7233.
		hostPort: string & =~"^[^:]+:[0-9]+$"
		// namespace is Temporal's, not Kubernetes'; the ensure-namespace Job
		// registers it.
		namespace: string | *"ivfs"
		// retention is how long Temporal keeps a closed run's history, if
		// the Job creates the namespace; an existing one keeps its own.
		retention: string & =~"^[0-9]+h$" | *"720h"
	}

	database: {
		// role owns the registry and creates every user's database
		// (ADR 0013).
		role:      #Name | *"ivfs_app"
		name:      #Name | *"ivfs"
		instances: int & >=1 | *1
		// imageName follows compose's major. Minimal suffices: ivfs uses no
		// extension. The tag floats within 18; pin a minor or a digest to
		// control when CNPG rolls the instances.
		imageName: string | *"ghcr.io/cloudnative-pg/postgresql:18-minimal-trixie"
		storage: {
			size:          string | *"10Gi"
			storageClass?: string
		}
	}

	oidc: {
		issuer:   string & =~"^https://"
		clientID: string & !=""
		// group, when set, is the one a user must be in to sign in.
		group?: string & !=""
		// logoutURL, when set, is where signing out sends the browser.
		logoutURL?: string & =~"^https?://"
		// ca, when set, is a ConfigMap of certificate authorities to trust
		// for the issuer, such as a trust-manager bundle.
		ca?: {
			configMap: #Name
			key:       string | *"ca.crt"
		}
	}

	// secrets names the Secrets External Secrets Operator syncs; their keys
	// are fixed, see README.md.
	secrets: {
		gateway: #Name
	}

	// route attaches the gateway to existing Gateway API listeners, whose
	// wildcard certificate already covers hostname. Each listener's
	// allowedRoutes must admit this namespace.
	route: {
		parentRefs: [#ParentRef, ...#ParentRef]
	}

	// networkPolicy says where the peers outside this namespace run, for the
	// CiliumNetworkPolicies that deny everything else.
	networkPolicy: {
		enabled: bool | *true
		dns: #Peer | *{namespace: "kube-system", podLabels: {"k8s-app": "kube-dns"}}
		temporal: #Peer | *{namespace: "temporal", podLabels: {"app.kubernetes.io/name": "temporal", "app.kubernetes.io/component": "frontend"}}
		// cnpgOperator reaches the instances' status endpoint.
		cnpgOperator: #Peer | *{namespace: "cnpg-system", podLabels: {"app.kubernetes.io/name": "cloudnative-pg"}}
	}

	providers: {
		yahoo: {
			enabled: bool | *true

			_#ImageOptions

			affinity?: api.#Affinity

			runtimeClassName?: string
		}
	}

	resources: {
		gateway: #Resources | *{requests: {cpu: "50m", memory: "64Mi"}, limits: memory: "256Mi"}
		worker: #Resources | *{requests: {cpu: "100m", memory: "128Mi"}, limits: memory: "512Mi"}
		database: #Resources | *{requests: {cpu: "100m", memory: "256Mi"}, limits: memory: "1Gi"}
		yahooProvider: #Resources | *{requests: {cpu: "10m", memory: "32Mi"}, limits: memory: "128Mi"}
	}
}

#Peer: {
	namespace: #Name
	podLabels: [string]: string
}

#ParentRef: {
	name:         #Name
	namespace?:   #Name
	sectionName?: #Name
	port?:        int & >0 & <65536
}

#Name: string & =~"^[a-z0-9]([-a-z0-9_]*[a-z0-9])?$" & strings.MaxRunes(63)

#Resources: {
	requests?: [string]: string
	// limits.memory is required: GOMEMLIMIT is derived from it.
	limits: {memory: string, [string]: string}
}
