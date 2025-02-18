/*
Copyright The Kubernetes Authors.

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
// Code generated by informer-gen. DO NOT EDIT.

package v1beta1

import (
	context "context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	apisvisibilityv1beta1 "sigs.k8s.io/kueue/apis/visibility/v1beta1"
	versioned "sigs.k8s.io/kueue/client-go/clientset/versioned"
	internalinterfaces "sigs.k8s.io/kueue/client-go/informers/externalversions/internalinterfaces"
	visibilityv1beta1 "sigs.k8s.io/kueue/client-go/listers/visibility/v1beta1"
)

// ClusterQueueInformer provides access to a shared informer and lister for
// ClusterQueues.
type ClusterQueueInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() visibilityv1beta1.ClusterQueueLister
}

type clusterQueueInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterQueueInformer constructs a new informer for ClusterQueue type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterQueueInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterQueueInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterQueueInformer constructs a new informer for ClusterQueue type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterQueueInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.VisibilityV1beta1().ClusterQueues().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.VisibilityV1beta1().ClusterQueues().Watch(context.TODO(), options)
			},
		},
		&apisvisibilityv1beta1.ClusterQueue{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterQueueInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterQueueInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterQueueInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisvisibilityv1beta1.ClusterQueue{}, f.defaultInformer)
}

func (f *clusterQueueInformer) Lister() visibilityv1beta1.ClusterQueueLister {
	return visibilityv1beta1.NewClusterQueueLister(f.Informer().GetIndexer())
}
