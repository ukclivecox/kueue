package util

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	kfmpi "github.com/kubeflow/mpi-operator/pkg/apis/kubeflow/v2beta1"
	kftraining "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/onsi/gomega"
	awv1beta2 "github.com/project-codeflare/appwrapper/api/v1beta2"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	jobset "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
	"sigs.k8s.io/yaml"

	configapi "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	visibility "sigs.k8s.io/kueue/apis/visibility/v1beta1"
	kueueclientset "sigs.k8s.io/kueue/client-go/clientset/versioned"
	visibilityv1beta1 "sigs.k8s.io/kueue/client-go/clientset/versioned/typed/visibility/v1beta1"
)

const (
	// E2eTestSleepImageOld is the image used for testing rolling update.
	E2eTestSleepImageOld = "gcr.io/k8s-staging-perf-tests/sleep:v0.0.3@sha256:00ae8e01dd4439edfb7eb9f1960ac28eba16e952956320cce7f2ac08e3446e6b"
	// E2eTestSleepImage is the image used for testing.
	E2eTestSleepImage = "gcr.io/k8s-staging-perf-tests/sleep:v0.1.0@sha256:8d91ddf9f145b66475efda1a1b52269be542292891b5de2a7fad944052bab6ea"
	// E2eTTestCurlImage is the image used for testing with curl execution.
	E2eTTestCurlImage = "curlimages/curl:8.11.0@sha256:6324a8b41a7f9d80db93c7cf65f025411f55956c6b248037738df3bfca32410c"
)

func CreateClientUsingCluster(kContext string) (client.WithWatch, *rest.Config) {
	cfg, err := config.GetConfigWithContext(kContext)
	if err != nil {
		fmt.Printf("unable to get kubeconfig for context %q: %s", kContext, err)
		os.Exit(1)
	}
	gomega.ExpectWithOffset(1, cfg).NotTo(gomega.BeNil())

	err = kueue.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = kueuealpha.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = visibility.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = jobset.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = kftraining.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = kfmpi.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = leaderworkersetv1.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	cfg.APIPath = "/api"
	cfg.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	cfg.ContentConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	err = awv1beta2.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	err = rayv1.AddToScheme(scheme.Scheme)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())

	client, err := client.NewWithWatch(cfg, client.Options{Scheme: scheme.Scheme})
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred())
	return client, cfg
}

// CreateRestClient creates a *rest.RESTClient using the provided config.
func CreateRestClient(cfg *rest.Config) *rest.RESTClient {
	restClient, err := rest.RESTClientFor(cfg)
	gomega.ExpectWithOffset(1, err).Should(gomega.Succeed())
	gomega.ExpectWithOffset(1, restClient).NotTo(gomega.BeNil())

	return restClient
}

func CreateVisibilityClient(user string) visibilityv1beta1.VisibilityV1beta1Interface {
	cfg, err := config.GetConfigWithContext("")
	if err != nil {
		fmt.Printf("unable to get kubeconfig: %s", err)
		os.Exit(1)
	}
	gomega.ExpectWithOffset(1, cfg).NotTo(gomega.BeNil())

	if user != "" {
		cfg.Impersonate.UserName = user
	}

	kueueClient, err := kueueclientset.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("unable to create kueue clientset: %s", err)
		os.Exit(1)
	}
	visibilityClient := kueueClient.VisibilityV1beta1()
	return visibilityClient
}

func rolloutOperatorDeployment(ctx context.Context, k8sClient client.Client, key types.NamespacedName) {
	deployment := &appsv1.Deployment{}
	var deploymentCondition *appsv1.DeploymentCondition
	expectedDeploymentCondition := &appsv1.DeploymentCondition{
		Type:   appsv1.DeploymentProgressing,
		Status: corev1.ConditionTrue,
		Reason: "NewReplicaSetAvailable",
	}

	gomega.EventuallyWithOffset(2, func(g gomega.Gomega) {
		g.Expect(k8sClient.Get(ctx, key, deployment)).To(gomega.Succeed())
		deploymentCondition = FindDeploymentCondition(deployment, appsv1.DeploymentProgressing)
		g.Expect(deploymentCondition).To(gomega.BeComparableTo(expectedDeploymentCondition, IgnoreDeploymentConditionTimestampsAndMessage))
	}, Timeout, Interval).Should(gomega.Succeed())
	beforeUpdateTime := deploymentCondition.LastUpdateTime

	gomega.EventuallyWithOffset(2, func(g gomega.Gomega) {
		deployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
		g.Expect(k8sClient.Update(ctx, deployment)).To(gomega.Succeed())
	}, Timeout, Interval).Should(gomega.Succeed())

	gomega.EventuallyWithOffset(2, func(g gomega.Gomega) {
		g.Expect(k8sClient.Get(ctx, key, deployment)).To(gomega.Succeed())
		deploymentCondition := FindDeploymentCondition(deployment, appsv1.DeploymentProgressing)
		g.Expect(deploymentCondition).To(gomega.BeComparableTo(expectedDeploymentCondition, IgnoreDeploymentConditionTimestampsAndMessage))
		afterUpdateTime := deploymentCondition.LastUpdateTime
		g.Expect(afterUpdateTime).NotTo(gomega.Equal(beforeUpdateTime))
	}, StartUpTimeout, Interval).Should(gomega.Succeed())
}

func waitForOperatorAvailability(ctx context.Context, k8sClient client.Client, key types.NamespacedName) {
	deployment := &appsv1.Deployment{}
	pods := &corev1.PodList{}
	gomega.EventuallyWithOffset(2, func(g gomega.Gomega) error {
		g.Expect(k8sClient.Get(ctx, key, deployment)).To(gomega.Succeed())
		g.Expect(k8sClient.List(ctx, pods, client.InNamespace(key.Namespace), client.MatchingLabels(deployment.Spec.Selector.MatchLabels))).To(gomega.Succeed())
		for _, pod := range pods.Items {
			for _, cs := range pod.Status.ContainerStatuses {
				// To make sure that we don't have restarts of controller-manager.
				// If we have that's mean that something went wrong, and there is
				// no needs to continue trying check availability.
				if cs.RestartCount > 0 {
					return gomega.StopTrying(fmt.Sprintf("%q in %q has restarted %d times", cs.Name, pod.Name, cs.RestartCount))
				}
			}
		}
		// To verify that webhooks are ready, checking is deployment have condition Available=True.
		g.Expect(deployment.Status.Conditions).To(gomega.ContainElement(gomega.BeComparableTo(
			appsv1.DeploymentCondition{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			cmpopts.IgnoreFields(appsv1.DeploymentCondition{}, "Reason", "Message", "LastUpdateTime", "LastTransitionTime")),
		))
		return nil
	}, StartUpTimeout, Interval).Should(gomega.Succeed())
}

func WaitForKueueAvailability(ctx context.Context, k8sClient client.Client) {
	kcmKey := types.NamespacedName{Namespace: "kueue-system", Name: "kueue-controller-manager"}
	waitForOperatorAvailability(ctx, k8sClient, kcmKey)
}

func WaitForAppWrapperAvailability(ctx context.Context, k8sClient client.Client) {
	jcmKey := types.NamespacedName{Namespace: "appwrapper-system", Name: "appwrapper-controller-manager"}
	waitForOperatorAvailability(ctx, k8sClient, jcmKey)
}

func WaitForJobSetAvailability(ctx context.Context, k8sClient client.Client) {
	jcmKey := types.NamespacedName{Namespace: "jobset-system", Name: "jobset-controller-manager"}
	waitForOperatorAvailability(ctx, k8sClient, jcmKey)
}

func WaitForLeaderWorkerSetAvailability(ctx context.Context, k8sClient client.Client) {
	jcmKey := types.NamespacedName{Namespace: "lws-system", Name: "lws-controller-manager"}
	waitForOperatorAvailability(ctx, k8sClient, jcmKey)
}

func WaitForKubeFlowTrainingOperatorAvailability(ctx context.Context, k8sClient client.Client) {
	kftoKey := types.NamespacedName{Namespace: "kubeflow", Name: "training-operator"}
	waitForOperatorAvailability(ctx, k8sClient, kftoKey)
}

func WaitForKubeFlowMPIOperatorAvailability(ctx context.Context, k8sClient client.Client) {
	kftoKey := types.NamespacedName{Namespace: "mpi-operator", Name: "mpi-operator"}
	waitForOperatorAvailability(ctx, k8sClient, kftoKey)
}

func WaitForKubeRayOperatorAvailability(ctx context.Context, k8sClient client.Client) {
	kroKey := types.NamespacedName{Namespace: "ray-system", Name: "kuberay-operator"}
	waitForOperatorAvailability(ctx, k8sClient, kroKey)
}

func GetKueueConfiguration(ctx context.Context, k8sClient client.Client) *configapi.Configuration {
	var kueueCfg configapi.Configuration
	kcmKey := types.NamespacedName{Namespace: "kueue-system", Name: "kueue-manager-config"}
	configMap := &corev1.ConfigMap{}

	gomega.Expect(k8sClient.Get(ctx, kcmKey, configMap)).To(gomega.Succeed())
	gomega.Expect(yaml.Unmarshal([]byte(configMap.Data["controller_manager_config.yaml"]), &kueueCfg)).To(gomega.Succeed())
	return &kueueCfg
}

func ApplyKueueConfiguration(ctx context.Context, k8sClient client.Client, kueueCfg *configapi.Configuration) {
	configMap := &corev1.ConfigMap{}
	kcmKey := types.NamespacedName{Namespace: "kueue-system", Name: "kueue-manager-config"}
	config, err := yaml.Marshal(kueueCfg)

	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Eventually(func(g gomega.Gomega) {
		g.Expect(k8sClient.Get(ctx, kcmKey, configMap)).To(gomega.Succeed())
		configMap.Data["controller_manager_config.yaml"] = string(config)
		g.Expect(k8sClient.Update(ctx, configMap)).To(gomega.Succeed())
	}, Timeout, Interval).Should(gomega.Succeed())
}

func RestartKueueController(ctx context.Context, k8sClient client.Client) {
	kcmKey := types.NamespacedName{Namespace: "kueue-system", Name: "kueue-controller-manager"}
	rolloutOperatorDeployment(ctx, k8sClient, kcmKey)
}
