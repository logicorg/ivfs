@experiment(try)

package main

import (
	batchv1 "cue.dev/x/k8s.io/api/batch/v1"
)

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
				try {
					runtimeClassName: input.job.runtimeClassName?
				}
				containers: [_container & {
					imagePullPolicy: input.job.imagePullPolicy
					try {
						image: input.job.image?
					}
					image: string | *_componentDefaultImage["ensure-namespace"]
				} & {
					name: "ensure-namespace"
					args: ["ensure-namespace", "--retention=\(input.temporal.retention)"]
					env: _temporalEnv
					resources: {requests: {cpu: "10m", memory: "32Mi"}, limits: memory: "64Mi"}
				}]
				try {
					affinity: input.job.affinity?
				}
			}
		}
	}
} & batchv1.#Job
