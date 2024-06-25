package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	crunchy "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/henrywhitaker3/crunchy-users/internal/logger"
	"github.com/henrywhitaker3/flow"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	WatchLabel          = "crunchy-users.henrywhitaker3.github.com/watch"
	SuperuserAnnotation = "crunchy-users.henrywhitaker3.github.com/superuser"
)

var (
	superusers = flow.NewStore[string]()
)

type ClusterUser struct {
	Name      string
	Databases []string
}

type ClusterResult struct {
	Name      string
	Namespace string
	Superuser string
	Users     []ClusterUser
}

func (c ClusterResult) Key() string {
	return fmt.Sprintf("%s:%s")
}

func WatchClusters(ctx context.Context, client *dynamic.DynamicClient) (<-chan ClusterResult, error) {
	logger := logger.Logger(ctx)

	out := make(chan ClusterResult, 1)

	fac := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		client,
		time.Minute,
		corev1.NamespaceAll,
		nil,
	)
	informer := fac.ForResource(crunchy.GroupVersion.WithResource("postgresclusters")).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			u := obj.(*unstructured.Unstructured)
			cluster := processObject(ctx, logger, u, client)
			if cluster != nil {
				out <- *cluster
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			u := newObj.(*unstructured.Unstructured)
			cluster := processObject(ctx, logger, u, client)
			if cluster != nil {
				out <- *cluster
			}
		},
	})
	logger.Infow("watching clusters")
	go informer.Run(ctx.Done())

	return out, nil
}

func processObject(ctx context.Context, logger *zap.SugaredLogger, u *unstructured.Unstructured, client *dynamic.DynamicClient) *ClusterResult {
	cluster := &crunchy.PostgresCluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), cluster); err != nil {
		logger.Errorw("couldn't cast resource to PostgresCluster", "obj", u)
		return nil
	}
	l := logger.With("cluster", cluster.Name, "namespace", cluster.Namespace)

	watched, ok := cluster.Labels[WatchLabel]
	if !ok || watched != "true" {
		l.Infow("skipping cluster as it is not being watched")
		return nil
	}
	superName, ok := cluster.Annotations[SuperuserAnnotation]
	if !ok {
		l.Errorw("skipping cluster as superuser annotation not set")
		return nil
	}

	users := []ClusterUser{}
	for _, user := range cluster.Spec.Users {
		dbs := []string{}
		for _, db := range user.Databases {
			dbs = append(dbs, string(db))
		}
		users = append(users, ClusterUser{
			Name:      string(user.Name),
			Databases: dbs,
		})
	}

	if len(users) < 1 {
		l.Infow("skipping cluster as there are no users")
		return nil
	}

	super, err := getSuperuser(ctx, client, cluster, superName)
	if err != nil {
		l.Errorw("skipping, could not get super user credentials", "error", err)
		return nil
	}

	return &ClusterResult{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
		Superuser: super,
		Users:     users,
	}
}

func getSuperuser(ctx context.Context, client *dynamic.DynamicClient, cluster *crunchy.PostgresCluster, name string) (string, error) {
	if url, ok := superusers.Get(clusterKey(cluster)); ok {
		return url, nil
	}

	secretName := fmt.Sprintf("%s-pguser-%s", cluster.Name, name)
	usec, err := client.Resource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}).Namespace(cluster.Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	secret := &corev1.Secret{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(usec.UnstructuredContent(), secret); err != nil {
		return "", err
	}

	url, ok := secret.Data["uri"]
	if !ok {
		return "", errors.New("superuser secret missing field uri")
	}

	superusers.Put(clusterKey(cluster), string(url))

	return string(url), nil
}

func clusterKey(cluster *crunchy.PostgresCluster) string {
	return fmt.Sprintf("%s:%s", cluster.Name, cluster.Namespace)
}
