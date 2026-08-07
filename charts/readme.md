# FSAS Rancher Cluster Template (Helm)

This folder contains a Helm chart that renders Rancher provisioning resources for Fujitsu FSAS:

- `Cluster` (`provisioning.cattle.io/v1`)
- `FsasConfig` (`rke-machine-config.cattle.io/v1`)

The rendered manifests can be applied to the Rancher management cluster (for example the local cluster where Rancher runs).

## Chart layout and purpose

- `Chart.yaml`
Describes chart metadata and marks this chart as a Rancher `cluster-template` catalog item.

- `values.yaml`
Default values for all rendered resources.

- `questions.yaml`
Rancher UI form schema used when this chart is published in a Rancher catalog.

- `templates/cluster.yaml`
Renders the Rancher `Cluster` resource, including RKE2 settings and machine pool references.

- `templates/fsasconfig-pool.yaml`
Renders the `FsasConfig` machine config resource consumed by the machine pool.

- `templates/_helpers.tpl`
Naming helper templates used by Helm.

## How values map to rendered resources

The templates currently read these key paths from `values.yaml`:

- `cluster.*` for cluster identity, credential, and Kubernetes version
- `machinePools.pool.*` for the single machine pool definition
- `rkeConfig.*` for RKE2 and upgrade behavior
- `machineSelectorConfig.*` for node-level options
- `fsas.pool.*` for FSAS driver settings (`apiUrl`, credentials, networking, compute/storage JSON, OS image, SSH, etc.)

If you render from CLI, make sure your override file uses the same paths used by templates.

## Prepare your custom values

1. Copy defaults:

```bash
cp values.yaml values.custom.yaml
```

2. Edit `values.custom.yaml` with your environment:

- `cluster.name`
- `cluster.cloudCredentialSecretName` (Rancher cloud credential secret reference)
- `cluster.kubernetesVersion`
- `fsas.pool.apiUrl`
- `fsas.pool.credentialsUsername`
- `fsas.pool.credentialsPassword`
- `fsas.pool.computeConditionsJson`
- `fsas.pool.devicesSpecJson`
- network and SSH fields under `fsas.pool.*`

Example override skeleton:

```yaml
cluster:
	name: fsas-demo
	cloudCredentialSecretName: cattle-global-data:cc-xxxxx
	kubernetesVersion: v1.35.6+rke2r1

machinePools:
	pool:
		name: pool1
		quantity: 1
		controlPlaneRole: true
		etcdRole: true
		workerRole: true

fsas:
	pool:
		apiUrl: <domain-name>
		tenantUuid: <tenant-uuid>
		credentialsUsername: "<username>"
		credentialsPassword: "<password>"
		osImageName: sles16
		computeConditionsJson: |
			[]
		devicesSpecJson: |
			[]
```

## Render Kubernetes manifests

From the `charts/` directory:

```bash
$ helm lint . -f values.custom.yaml
$ helm template fsas-cluster . -f values.custom.yaml > rendered-fsas-cluster.yaml
```

Validate the rendered output contains both resources:

```bash
$ grep -E "^kind: (Cluster|FsasConfig)$" -n rendered-fsas-cluster.yaml
```
The grep response should be something like:
```bash
4:kind: Cluster
89:kind: FsasConfig
```

## Apply rendered file to Rancher

Prerequisites:

- You have a kubeconfig with access to the Rancher management cluster.
- The FSAS node driver CRDs are installed in Rancher (fsas node-driver is installed).
- The cloud credential referenced by `cluster.cloudCredentialSecretName` exists.

1. Apply the rendered manifest:

```bash
kubectl apply -f rendered-fsas-cluster.yaml
```

2. Verify resources:

```bash
kubectl -n fleet-default get cluster.provisioning.cattle.io
kubectl -n fleet-default get fsasconfig.rke-machine-config.cattle.io
```

3. Inspect status/events when troubleshooting:

```bash
kubectl -n fleet-default describe cluster.provisioning.cattle.io <cluster-name>
kubectl -n fleet-default describe fsasconfig.rke-machine-config.cattle.io <cluster-name>-all
```

## Rancher UI workflow (alternative)

If this chart is published as a Rancher catalog `cluster-template`:

1. Open Rancher Manager.
2. Go to Cluster Management and create a cluster from the FSAS template.
3. Fill fields generated from `questions.yaml`.
4. Rancher renders and applies equivalent resources automatically.

## Notes

- Namespace is currently fixed to `fleet-default` in templates.
- Pool machine config name is rendered as `<cluster.name>-all`.
- Treat files containing credentials as sensitive; do not commit secrets.
#