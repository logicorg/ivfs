@experiment(try)

package main

_components: gateway: {
	#name: "gateway"
	#port: 8080
	if input.oidc.ca != _|_ {
		#volumes: [{
			name: "oidc-ca"
			configMap: {name: input.oidc.ca.configMap, items: [{key: input.oidc.ca.key, path: "ca.crt"}]}
		}]
	}
	#container: {
		try {
			image: input.gateway.image?
		}
		imagePullPolicy: input.gateway.imagePullPolicy

		// The default listens on loopback alone, for development.
		args: ["--addr=:8080"]
		env: [
			{name: "IVFS_PUBLIC_URL", value: "https://\(input.hostname)"},
			{name: "IVFS_SESSION_KEYS", valueFrom: secretKeyRef: {name: input.secrets.gateway, key: "session-keys"}},
			{name: "OIDC_ISSUER", value: input.oidc.issuer},
			{name: "OIDC_CLIENT_ID", value: input.oidc.clientID},
			{name: "OIDC_CLIENT_SECRET", valueFrom: secretKeyRef: {name: input.secrets.gateway, key: "oidc-client-secret"}},
			if input.oidc.group != _|_ {{name: "OIDC_GROUP", value: input.oidc.group}},
			if input.oidc.logoutURL != _|_ {{name: "OIDC_LOGOUT_URL", value: input.oidc.logoutURL}},
			if input.oidc.ca != _|_ {{name: "OIDC_CA_FILE", value: "/etc/ivfs/oidc-ca/ca.crt"}},
			{name: "IVFS_ASSET_VALUE_PROVIDERS", value: [
				if input.providers.yahoo.enabled {"manual,yahoo=http://yahoo-provider:8080"},
				"manual",
			][0]},
			_goMemLimit,
			for e in _temporalEnv {e},
		]
		ports: [{name: "http", containerPort: 8080}]
		if input.oidc.ca != _|_ {
			volumeMounts: [{name: "oidc-ca", mountPath: "/etc/ivfs/oidc-ca", readOnly: true}]
		}

		// /livez sits in front of the Host guard; /api/health does not, so
		// readiness sends the Host browsers do.
		livenessProbe: httpGet: {path: "/livez", port: "http"}
		readinessProbe: httpGet: {
			path: "/api/health"
			port: "http"
			httpHeaders: [{name: "Host", value: input.hostname}]
		}
		resources: input.resources.gateway
	}
}
