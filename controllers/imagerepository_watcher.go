/*
Copyright 2026 Oscar Romeu

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"path"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/errgroup"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	imagev1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

// ImageRepositoryWatcher watches ImageRepository objects for new tags
// and mirrors them to the configured destination registry.
type ImageRepositoryWatcher struct {
	client.Client
	DestinationRegistry string // e.g. "europe-west4-docker.pkg.dev/my-project/flanks"
	Workers             int    // concurrent ImageRepository reconciles
	TagWorkers          int    // concurrent tag copies per reconcile
	tokenSource         oauth2.TokenSource
}

func (r *ImageRepositoryWatcher) SetupWithManager(mgr ctrl.Manager) error {
	ts, err := google.DefaultTokenSource(context.Background(), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("failed to initialize GCP credentials: %w", err)
	}
	r.tokenSource = ts

	return ctrl.NewControllerManagedBy(mgr).
		For(&imagev1beta2.ImageRepository{}, builder.WithPredicates(ImageRepositoryTagsChangePredicate{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.Workers}).
		Complete(r)
}

// +kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagerepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagerepositories/status,verbs=get

func (r *ImageRepositoryWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var imageRepo imagev1beta2.ImageRepository
	if err := r.Get(ctx, req.NamespacedName, &imageRepo); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	scan := imageRepo.Status.LastScanResult
	imageName := path.Base(imageRepo.Status.CanonicalImageName)

	log.Info("New tags detected", "image", imageRepo.Status.CanonicalImageName, "tags", scan.LatestTags)

	token, err := r.tokenSource.Token()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get GCP token: %w", err)
	}
	auth := authn.FromConfig(authn.AuthConfig{
		Username: "oauth2accesstoken",
		Password: token.AccessToken,
	})

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(r.TagWorkers)
	for _, tag := range scan.LatestTags {
		tag := tag
		g.Go(func() error {
			src := fmt.Sprintf("%s:%s", imageRepo.Status.CanonicalImageName, tag)
			dst := fmt.Sprintf("%s/%s:%s", r.DestinationRegistry, imageName, tag)
			log.Info("Mirroring image", "src", src, "dst", dst)
			if err := crane.Copy(src, dst, crane.WithAuth(auth)); err != nil {
				log.Error(err, "failed to mirror image", "src", src, "dst", dst)
				return nil // log and continue, don't abort the group
			}
			log.Info("Mirrored successfully", "dst", dst)
			return nil
		})
	}
	g.Wait()

	return ctrl.Result{}, nil
}
