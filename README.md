# Crunchy Users

This is used to set initial permissions for users created in a crunchy-pgo cluster

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
