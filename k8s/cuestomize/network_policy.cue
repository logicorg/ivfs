package main

import "strings"

// Every pod here is denied traffic both ways but for what these policies
// allow. The kubelet's probes still pass: Cilium lets the host reach its
// local pods.

let np = input.networkPolicy

// Policies select peers after Service translation, by the backend pods'
// labels and ports, so the Temporal port is the frontend pods' own.
let temporalPort = strings.Split(input.temporal.hostPort, ":")[1]

let issuerHostPort = strings.Split(strings.TrimPrefix(input.oidc.issuer, "https://"), "/")[0]
let issuerHost = strings.Split(issuerHostPort, ":")[0]
let issuerPort = [
	if strings.Contains(issuerHostPort, ":") {strings.Split(issuerHostPort, ":")[1]},
	"443",
][0]

let tcp = {
	_p: string
	{ports: [{port: _p, protocol: "TCP"}]}
}

_peer: {
	_in: #Peer
	{"k8s:io.kubernetes.pod.namespace": _in.namespace} & _in.podLabels
}
_local: {
	_name: string
	{"k8s:io.kubernetes.pod.namespace": input.namespace, "app.kubernetes.io/name": _name}
}

let toTemporal = {
	toEndpoints: [{matchLabels: (_peer & {_in: np.temporal})}]
	toPorts: [tcp & {_p: temporalPort}]
}

_policies: [Name=string]: {
	apiVersion: "cilium.io/v2"
	kind:       "CiliumNetworkPolicy"
	metadata: {
		name:      Name
		namespace: input.namespace
		labels:    _labels
	}
}

_policies: "default-deny": spec: {
	endpointSelector: {}
	// Explicit, so ingress is denied too although this policy allows none.
	enableDefaultDeny: {ingress: true, egress: true}
	egress: [{
		toEndpoints: [{matchLabels: (_peer & {_in: np.dns})}]
		toPorts: [{
			ports: [{port: "53", protocol: "UDP"}, {port: "53", protocol: "TCP"}]
			// Through Cilium's DNS proxy, which the FQDN rules below need to
			// learn the names' addresses.
			rules: dns: [{matchPattern: "*"}]
		}]
	}]
}

_policies: gateway: spec: {
	endpointSelector: matchLabels: "app.kubernetes.io/name": "gateway"
	// Cilium's Gateway API Envoy carries the identity ingress.
	ingress: [{
		fromEntities: ["ingress"]
		toPorts: [tcp & {_p: "8080"}]
	}]
	egress: [
		toTemporal,
		{
			toFQDNs: [{matchName: issuerHost}]
			toPorts: [tcp & {_p: issuerPort}]
		},
		if input.providers.yahoo.enabled {
			toEndpoints: [{matchLabels: (_local & {_name: "yahoo-provider"})}]
			toPorts: [tcp & {_p: "8080"}]
		},
	]
}

if input.providers.yahoo.enabled {
	_policies: "yahoo-provider": spec: {
		endpointSelector: matchLabels: "app.kubernetes.io/name": "yahoo-provider"
		ingress: [{
			fromEndpoints: [{matchLabels: (_local & {_name: "gateway"})}]
			toPorts: [tcp & {_p: "8080"}]
		}]
		egress: [{
			toFQDNs: [{matchName: "query1.finance.yahoo.com"}]
			toPorts: [tcp & {_p: "443"}]
		}]
	}
}

_policies: worker: spec: {
	endpointSelector: matchLabels: "app.kubernetes.io/name": "worker"
	egress: [
		toTemporal,
		{
			toEndpoints: [{matchLabels: {"cnpg.io/cluster": _dbCluster}}]
			toPorts: [tcp & {_p: "5432"}]
		},
	]
}

_policies: "ensure-namespace": spec: {
	endpointSelector: matchLabels: "app.kubernetes.io/name": "ensure-namespace"
	egress: [toTemporal]
}

_policies: database: spec: {
	let instances = {matchLabels: {"cnpg.io/cluster": _dbCluster}}
	endpointSelector: instances
	ingress: [
		{
			fromEndpoints: [{matchLabels: (_local & {_name: "worker"})}]
			toPorts: [tcp & {_p: "5432"}]
		},
		{
			fromEndpoints: [{matchLabels: (_peer & {_in: np.cnpgOperator})}]
			toPorts: [tcp & {_p: "8000"}]
		},
		// Replicas stream from the primary and ask each other's status.
		{
			fromEndpoints: [instances]
			toPorts: [{ports: [{port: "5432", protocol: "TCP"}, {port: "8000", protocol: "TCP"}]}]
		},
	]
	egress: [
		// The instance manager watches its Cluster and writes its status.
		{toEntities: ["kube-apiserver"]},
		{
			toEndpoints: [instances]
			toPorts: [{ports: [{port: "5432", protocol: "TCP"}, {port: "8000", protocol: "TCP"}]}]
		},
	]
}

for name, p in _policies {
	outputs: "ciliumnetworkpolicy-\(name)": p
}
