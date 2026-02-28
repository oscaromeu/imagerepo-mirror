# imagerepo-mirror

A [GitOps toolkit](https://fluxcd.io/flux/components/) controller that watches [FluxCD ImageRepository](https://fluxcd.io/flux/components/image/imagerepositories/) objects and mirrors detected container image tags to a destination GCP Artifact Registry.

## 📖 Overview

The controller subscribes to `ImageRepository` resources managed by the [FluxCD image-reflector-controller](https://github.com/fluxcd/image-reflector-controller). When a new set of tags is detected (i.e. the scan revision changes), it mirrors those tags to the configured destination registry using [crane](https://github.com/google/go-containerregistry).

Authentication is handled via [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) on GKE, or Application Default Credentials (ADC) locally.

## 💡 Use cases

- **Environment promotion** — mirror images from a source registry to a destination registry as soon as they are tagged, without any manual intervention.

## ⚙️ How it works

1. Flux's [image-reflector-controller](https://github.com/fluxcd/image-reflector-controller) periodically scans the source registry and updates `ImageRepository` status with the latest tags and a revision hash.
2. `imagerepo-mirror` watches for changes to that revision hash — it only triggers when the set of tags actually changes, not on every scan.
3. For each changed `ImageRepository`, the controller copies the detected tags to the destination registry in parallel using `crane`, authenticated via Workload Identity.

## 🔧 Configuration

| Flag | Default | Description |
|---|---|---|
| `--destination-registry` | `""` | Destination registry prefix (e.g. `europe-west4-docker.pkg.dev/my-project/my-repo`) |
| `--workers` | `4` | Concurrent ImageRepository reconciles |
| `--tag-workers` | `4` | Concurrent tag copies per reconcile |
| `--enable-leader-election` | `false` | Enable leader election (required with `replicas > 1`) |
| `--metrics-addr` | `:8080` | Address for the Prometheus metrics endpoint |

# 📚 Guides

* [Watching for source changes](https://fluxcd.io/flux/gitops-toolkit/source-watcher/)