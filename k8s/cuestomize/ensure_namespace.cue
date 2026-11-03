package main

// The worker cannot start a Temporal worker in a namespace that does not
// exist, and registering one is an admin call it should not need at runtime.
outputs: "job-ensure-namespace": {
	let labels = _labels & {
		"app.kubernetes.io/name":      "ensure-namespace"
		"app.kubernetes.io/component": "ensure-namespace"
	}
	apiVersion: "batch/v1"
	kind:       "Job"
	metadata: {
		name:      "ensure-namespace"
		namespace: input.namespace
		"labels":  labels
	}
	spec: {
		backoffLimit: 10
		// A Job's template is immutable, so a new image tag could not be
		// applied over a finished one. Deleted after an hour, it is recreated
		// by the next sync, which is harmless: ensuring is idempotent.
		ttlSecondsAfterFinished: 3600
		template: {
			metadata: "labels": labels
			spec: {
				restartPolicy:                "OnFailure"
				serviceAccountName:           "worker"
				automountServiceAccountToken: false
				enableServiceLinks:           false
				securityContext:              _podSecurity
				containers: [_container & {
					name:  "ensure-namespace"
					image: "\(input.image.registry)/worker:\(input.image.tag)"
					args: ["ensure-namespace", "--retention=\(input.temporal.retention)"]
					env: _temporalEnv
					resources: {requests: {cpu: "10m", memory: "32Mi"}, limits: memory: "64Mi"}
				}]
			}
		}
	}
}
