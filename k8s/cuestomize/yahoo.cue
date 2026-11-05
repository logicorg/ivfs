@experiment(try)

package main

if input.providers.yahoo.enabled {
	_components: "yahoo-provider": {
		#name: "yahoo-provider"
		#port: 8080
		#podSpec: {
			try {
				affinity: input.providers.yahoo.affinity?
			}
			try {
				runtimeClassName: input.providers.yahoo.runtimeClassName?
			}
		}
		#container: {
			imagePullPolicy: input.providers.yahoo.imagePullPolicy
			try {
				image: input.providers.yahoo.image?
			}
			env: [_goMemLimit]
			ports: [{name: "http", containerPort: 8080}]
			// It serves no health endpoint, and an answer depends on Yahoo, so
			// a listening socket is all there is to check.
			livenessProbe: tcpSocket: port:  "http"
			readinessProbe: tcpSocket: port: "http"
			resources: input.resources.yahooProvider
		}
	}
}
