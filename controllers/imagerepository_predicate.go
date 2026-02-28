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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	imagev1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

// ImageRepositoryTagsChangePredicate triggers an update event
// when an ImageRepository detects new or removed tags (i.e. the revision changes).
type ImageRepositoryTagsChangePredicate struct {
	predicate.Funcs
}

func (ImageRepositoryTagsChangePredicate) Create(e event.CreateEvent) bool {
	repo, ok := e.Object.(*imagev1beta2.ImageRepository)
	return ok && repo.Status.LastScanResult != nil && repo.Status.LastScanResult.Revision != ""
}

func (ImageRepositoryTagsChangePredicate) Update(e event.UpdateEvent) bool {
	oldRepo, ok1 := e.ObjectOld.(*imagev1beta2.ImageRepository)
	newRepo, ok2 := e.ObjectNew.(*imagev1beta2.ImageRepository)
	if !ok1 || !ok2 {
		return false
	}

	if oldRepo.Status.LastScanResult == nil && newRepo.Status.LastScanResult != nil {
		return true
	}

	if oldRepo.Status.LastScanResult != nil && newRepo.Status.LastScanResult != nil &&
		oldRepo.Status.LastScanResult.Revision != newRepo.Status.LastScanResult.Revision {
		return true
	}

	return false
}
