# Stores pod and node manifests as snapshots from Kubernetes

Have you ever wished that you could do "kubectl get pod xyz -o yaml" after the pod is long gone?

This tool takes snapshots from both pod and node manifests and stores them for later retrieval into a PostgreSQL database.
It also provides a really simple UI where you can query the stored snapshots and look how they were in a certain point of time.

## Usage

- Setup a PostgreSQL database in a namespace
- Deploy this image as a Deployment with a single replica.
- Pass PSQL_URL environment variable with value like "postgres://username:password@kubehistory-psql.mynamespace.svc.cluster.local:5432/database?sslmode=disable"
