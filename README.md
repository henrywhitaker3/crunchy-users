# Crunchy Users

This is used to set initial permissions for users and databases created in a crunchy-pgo cluster

It watches all `PostgresCluster` resources in the cluster with the `crunchy-users.henrywhitaker3.github.com/watch` label set to `true` and will update any databases you have defined by running `ALTER DATABASE {db} OWNER TO {user}` using the superuser defined in the `crunchy-users.henrywhitaker3.github.com/superuser` annotation

e.g.

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: crunchy
  labels:
    crunchy-users.henrywhitaker3.github.com/watch: "true"
  annotations:
    crunchy-users.henrywhitaker3.github.com/superuser: "postgres"
spec:
  ...
  users:
    - name: bongo
      databases:
        - bongo1
        - bongo2
```

Both databases `bongo1` and `bongo2` will have their owner set to the user `bongo`.

## Extensions

To create extensions for a database, you can add entries to the `crunchy-users.henrywhitaker3.github.com/extensions` annotation. This expects a json array:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: crunchy
  labels:
    crunchy-users.henrywhitaker3.github.com/watch: "true"
  annotations:
    crunchy-users.henrywhitaker3.github.com/superuser: "postgres"
    crunchy-users.henrywhitaker3.github.com/extensions: |
      [
        {
          "extension": "vector",
          "database": "bongo"
        }
      ]
```

Cascade will default to `false`.

## Installation

The helm chart is hosted at `oci://ghcr.io/henrywhitaker3/crunchy-users-helm`
