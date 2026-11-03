package main

_components: worker: {
	#name: "worker"
	#volumes: [{
		name: "postgres-ca"
		secret: {secretName: "\(_dbCluster)-ca", items: [{key: "ca.crt", path: "ca.crt"}]}
	}]
	#container: {
		image: "\(input.image.registry)/worker:\(input.image.tag)"
		env: [
			{name: "IVFS_APP_ROLE", value: input.database.role},
			{name: "IVFS_APP_PASSWORD", valueFrom: secretKeyRef: {name: "\(_dbCluster)-app", key: "password"}},
			{name: "IVFS_REGISTRY_DATABASE", value: input.database.name},
			// The users' databases are on the same server, so this one host
			// is every connection's.
			{name: "POSTGRES_HOST", value: "\(_dbCluster)-rw:5432"},
			// verify-full: CNPG's server certificate names the -rw Service.
			{name: "DSN_QUERY", value: "sslmode=verify-full&sslrootcert=/etc/ivfs/postgres-ca/ca.crt"},
			{name: "IVFS_HEALTH_ADDR", value: ":8081"},
			_goMemLimit,
			for e in _temporalEnv {e},
		]
		ports: [{name: "health", containerPort: 8081}]
		volumeMounts: [{name: "postgres-ca", mountPath: "/etc/ivfs/postgres-ca", readOnly: true}]
		// Liveness checks nothing beyond the process: a database or Temporal
		// outage is waited out, not fixed by a restart.
		livenessProbe: httpGet: {path: "/livez", port: "health"}
		readinessProbe: httpGet: {path: "/readyz", port: "health"}
		resources: input.resources.worker
	}
}
