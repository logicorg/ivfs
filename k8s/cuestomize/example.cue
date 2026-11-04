@if(example)

package main

includes: {}

// The input `cue export -t example .` renders, as a Cuestomization would set it.
input: {
	hostname: "ivfs.example.com"
	image: tag:         "0.1.0"
	temporal: hostPort: "temporal-frontend.temporal:7233"
	oidc: {
		issuer:   "https://auth.example.com"
		clientID: "ivfs"
		group:    "ivfs"
	}
	secrets: gateway: "ivfs-gateway"
	route: parentRefs: [{name: "public", namespace: "gateway", sectionName: "https"}]
}
