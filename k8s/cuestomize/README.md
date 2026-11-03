# ivfs cuestomize module

Renders ivfs for Kubernetes through the [cuestomize](https://github.com/logicorg/cuestomize)
KRM function. `schema.cue` is the input contract; `#Input` is closed, so an unknown field fails
the render.

## Secrets

The module reads, never creates, the Secret `input.secrets.gateway` names. External Secrets
Operator is expected to sync it, with these keys:

| Key                  | Becomes              | Value                                                  |
| -------------------- | -------------------- | ------------------------------------------------------ |
| `session-keys`       | `IVFS_SESSION_KEYS`  | comma-separated base64 keys, newest first (`openssl rand -base64 32`) |
| `oidc-client-secret` | `OIDC_CLIENT_SECRET` | the gateway's client secret at the OIDC issuer         |

The database's password comes from the CNPG Cluster's `ivfs-db-app` Secret, and its CA from
`ivfs-db-ca`.

## Rendering locally

```sh
cue export -t example . -e outputs --out yaml
```

`example.cue` holds that input; it is excluded from the published module's default build.
