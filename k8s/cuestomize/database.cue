package main

outputs: "cluster-\(_dbCluster)": {
	apiVersion: "postgresql.cnpg.io/v1"
	kind:       "Cluster"
	metadata: {
		name:      _dbCluster
		namespace: input.namespace
		labels: _labels & {"app.kubernetes.io/name": _dbCluster, "app.kubernetes.io/component": "database"}
	}
	spec: {
		instances: input.database.instances
		imageName: input.database.imageName
		storage: {
			size: input.database.storage.size
			if input.database.storage.storageClass != _|_ {
				storageClass: input.database.storage.storageClass
			}
		}
		resources: input.resources.database
		// The worker needs no superuser: the role below owns everything.
		enableSuperuserAccess: false
		bootstrap: initdb: {
			database: input.database.name
			owner:    input.database.role
		}
		// initdb creates the owner without CREATEDB, which provisioning needs
		// for every user's database (ADR 0013). Managing the role reconciles
		// the attribute; with no passwordSecret, the -app Secret's password
		// stays as initdb set it.
		managed: roles: [{
			name:     input.database.role
			ensure:   "present"
			login:    true
			createdb: true
			inherit:  true
		}]
	}
}
