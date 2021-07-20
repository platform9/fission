package mqtrigger

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/executor/util"
	genInformer "github.com/fission/fission/pkg/generated/informers/externalversions"
	"github.com/fission/fission/pkg/utils"
)

var (
	// Group refers to the group name in KEDA CRD
	Group = "keda.sh"
	// Version refers to the version name in KEDA CRD
	Version = "v1alpha1"

	// apiVersion refers to the api version name in KEDA CRD
	apiVersion      = Group + "/" + Version
	scaledObjectGVR = schema.GroupVersionResource{
		Group:    Group,
		Version:  Version,
		Resource: "scaledobjects",
	}
	authTriggerGVR = schema.GroupVersionResource{
		Group:    Group,
		Version:  Version,
		Resource: "triggerauthentications",
	}
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func getScaledObjectClient(namespace string) (dynamic.ResourceInterface, error) {
	dynamicClient, err := crd.GetDynamicClient()
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(scaledObjectGVR).Namespace(namespace), nil
}

func getAuthTriggerClient(namespace string) (dynamic.ResourceInterface, error) {
	dynamicClient, err := crd.GetDynamicClient()
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(authTriggerGVR).Namespace(namespace), nil
}

func mqTriggerEventHandlers(logger *zap.Logger, kubeClient *kubernetes.Clientset, routerURL string) k8sCache.ResourceEventHandlerFuncs {
	return k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			go func() {
				mqt := obj.(*fv1.MessageQueueTrigger)
				if mqt.Spec.MqtKind == "fission" {
					return
				}
				logger.Debug("Create deployment for Scaler Object", zap.Any("mqt", mqt.ObjectMeta), zap.Any("mqt.Spec", mqt.Spec))

				authenticationRef := ""
				if len(mqt.Spec.Secret) > 0 {
					authenticationRef = fmt.Sprintf("%s-auth-trigger", mqt.ObjectMeta.Name)
					err := createAuthTrigger(mqt, authenticationRef, kubeClient)
					if err != nil {
						logger.Error("Failed to create Authentication Trigger", zap.Error(err))
						return
					}
				}

				if err := createDeployment(mqt, routerURL, kubeClient); err != nil {
					logger.Error("Failed to create Deployment", zap.Error(err))
					if len(authenticationRef) > 0 {
						err = deleteAuthTrigger(authenticationRef, mqt.ObjectMeta.Namespace)
						if err != nil {
							logger.Error("Failed to delete Authentication Trigger", zap.Error(err))
						}
					}
					return
				}

				if err := createScaledObject(mqt, authenticationRef); err != nil {
					logger.Error("Failed to create ScaledObject", zap.Error(err))
					if len(authenticationRef) > 0 {
						if err = deleteAuthTrigger(authenticationRef, mqt.ObjectMeta.Namespace); err != nil {
							logger.Error("Failed to delete Authentication Trigger", zap.Error(err))
						}
					}
					if err = deleteDeployment(mqt.ObjectMeta.Name, kubeClient); err != nil {
						logger.Error("Failed to delete Deployment", zap.Error(err))
					}
				}
			}()
		},
		UpdateFunc: func(obj interface{}, newObj interface{}) {
			go func() {
				mqt := obj.(*fv1.MessageQueueTrigger)
				newMqt := newObj.(*fv1.MessageQueueTrigger)
				updated := checkAndUpdateTriggerFields(mqt, newMqt)
				if mqt.Spec.MqtKind == "fission" {
					return
				}
				if !updated {
					logger.Warn(fmt.Sprintf("%s remains unchanged. No changes found in trigger fields", mqt.ObjectMeta.Name))
					return
				}

				authenticationRef := ""
				if len(newMqt.Spec.Secret) > 0 && newMqt.Spec.Secret != mqt.Spec.Secret {
					authenticationRef = fmt.Sprintf("%s-auth-trigger", mqt.ObjectMeta.Name)
					if err := updateAuthTrigger(mqt, authenticationRef, kubeClient); err != nil {
						logger.Error("Failed to update Authentication Trigger", zap.Error(err))
						return
					}
				}

				if err := updateDeployment(mqt, routerURL, kubeClient); err != nil {
					logger.Error("Failed to Update Deployment", zap.Error(err))
					return
				}

				if err := updateScaledObject(mqt, authenticationRef); err != nil {
					logger.Error("Failed to Update ScaledObject", zap.Error(err))
					return
				}
			}()
		},
	}

}

// StartScalerManager watches for changes in MessageQueueTrigger and,
// Based on changes, it Creates, Updates and Deletes Objects of Kind ScaledObjects, AuthenticationTriggers and Deployments
func StartScalerManager(logger *zap.Logger, routerURL string) error {
	fissionClient, kubeClient, _, _, err := crd.MakeFissionClient()
	if err != nil {
		return err
	}
	err = fissionClient.WaitForCRDs()
	if err != nil {
		return errors.Wrap(err, "error waiting for CRDs")
	}
	informerFactory := genInformer.NewSharedInformerFactory(fissionClient, 30*time.Second)
	mqTriggerInformer := informerFactory.Core().V1().MessageQueueTriggers().Informer()
	mqTriggerInformer.AddEventHandler(mqTriggerEventHandlers(logger, kubeClient, routerURL))
	mqTriggerInformer.Run(context.Background().Done())
	return nil
}

func toEnvVar(str string) string {
	envVar := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	envVar = matchAllCap.ReplaceAllString(envVar, "${1}_${2}")
	return strings.ToUpper(envVar)
}

func getEnvVarlist(mqt *fv1.MessageQueueTrigger, routerURL string, kubeClient kubernetes.Interface) ([]apiv1.EnvVar, error) {
	url := routerURL + "/" + strings.TrimPrefix(utils.UrlForFunction(mqt.Spec.FunctionReference.Name, mqt.ObjectMeta.Namespace), "/")
	envVars := []apiv1.EnvVar{
		{
			Name:  "TOPIC",
			Value: mqt.Spec.Topic,
		},
		{
			Name:  "HTTP_ENDPOINT",
			Value: url,
		},
		{
			Name:  "ERROR_TOPIC",
			Value: mqt.Spec.ErrorTopic,
		},
		{
			Name:  "RESPONSE_TOPIC",
			Value: mqt.Spec.ResponseTopic,
		},
		{
			Name:  "SOURCE_NAME",
			Value: mqt.ObjectMeta.Name,
		},
		{
			Name:  "MAX_RETRIES",
			Value: strconv.Itoa(mqt.Spec.MaxRetries),
		},
		{
			Name:  "CONTENT_TYPE",
			Value: mqt.Spec.ContentType,
		},
	}
	// Metadata Fields
	for key, value := range mqt.Spec.Metadata {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  toEnvVar(key),
			Value: value,
		})
	}

	// Add Auth Fields
	secretName := mqt.Spec.Secret
	if len(secretName) > 0 {
		secret, err := kubeClient.CoreV1().Secrets(apiv1.NamespaceDefault).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		for key, value := range secret.Data {
			envVars = append(envVars, apiv1.EnvVar{
				Name:  toEnvVar(key),
				Value: string(value),
			})
		}
	}
	return envVars, nil
}

func checkAndUpdateTriggerFields(mqt, newMqt *fv1.MessageQueueTrigger) bool {
	updated := false
	if len(newMqt.Spec.Topic) > 0 && newMqt.Spec.Topic != mqt.Spec.Topic {
		mqt.Spec.Topic = newMqt.Spec.Topic
		updated = true
	}
	if len(newMqt.Spec.ResponseTopic) > 0 && newMqt.Spec.ResponseTopic != mqt.Spec.ResponseTopic {
		mqt.Spec.ResponseTopic = newMqt.Spec.ResponseTopic
		updated = true
	}
	if len(newMqt.Spec.ErrorTopic) > 0 && newMqt.Spec.ErrorTopic != mqt.Spec.ErrorTopic {
		mqt.Spec.ErrorTopic = newMqt.Spec.ErrorTopic
		updated = true
	}
	if newMqt.Spec.MaxRetries >= 0 && newMqt.Spec.MaxRetries != mqt.Spec.MaxRetries {
		mqt.Spec.MaxRetries = newMqt.Spec.MaxRetries
		updated = true
	}
	if len(newMqt.Spec.FunctionReference.Name) > 0 && newMqt.Spec.FunctionReference.Name != mqt.Spec.FunctionReference.Name {
		mqt.Spec.FunctionReference.Name = newMqt.Spec.FunctionReference.Name
		updated = true
	}
	if len(newMqt.Spec.ContentType) > 0 && newMqt.Spec.ContentType != mqt.Spec.ContentType {
		mqt.Spec.ContentType = newMqt.Spec.ContentType
		updated = true
	}
	if *newMqt.Spec.PollingInterval >= 0 && *newMqt.Spec.PollingInterval != *mqt.Spec.PollingInterval {
		mqt.Spec.PollingInterval = newMqt.Spec.PollingInterval
		updated = true
	}
	if *newMqt.Spec.CooldownPeriod >= 0 && *newMqt.Spec.CooldownPeriod != *mqt.Spec.CooldownPeriod {
		mqt.Spec.CooldownPeriod = newMqt.Spec.CooldownPeriod
		updated = true
	}
	if *newMqt.Spec.MinReplicaCount >= 0 && *newMqt.Spec.MinReplicaCount != *mqt.Spec.MinReplicaCount {
		mqt.Spec.MinReplicaCount = newMqt.Spec.MinReplicaCount
		updated = true
	}
	if *newMqt.Spec.MaxReplicaCount >= 0 && *newMqt.Spec.MaxReplicaCount != *mqt.Spec.MaxReplicaCount {
		mqt.Spec.MaxReplicaCount = newMqt.Spec.MaxReplicaCount
		updated = true
	}
	if len(newMqt.Spec.FunctionReference.Name) > 0 && newMqt.Spec.FunctionReference.Name != mqt.Spec.FunctionReference.Name {
		mqt.Spec.FunctionReference.Name = newMqt.Spec.FunctionReference.Name
		updated = true
	}

	if !reflect.DeepEqual(newMqt.Spec.PodSpec, mqt.Spec.PodSpec) {
		mqt.Spec.PodSpec = newMqt.Spec.PodSpec
		updated = true
	}
	if newMqt.Spec.Sequential != mqt.Spec.Sequential {
		mqt.Spec.Sequential = newMqt.Spec.Sequential
		updated = true
	}

	for key, value := range newMqt.Spec.Metadata {
		if val, ok := mqt.Spec.Metadata[key]; ok && val != value {
			mqt.Spec.Metadata[key] = value
			updated = true
		}
	}

	if len(newMqt.Spec.Secret) > 0 && newMqt.Spec.Secret != mqt.Spec.Secret {
		mqt.Spec.Secret = newMqt.Spec.Secret
		updated = true
	}

	if newMqt.Spec.MqtKind != mqt.Spec.MqtKind {
		mqt.Spec.MqtKind = newMqt.Spec.MqtKind
		updated = true
	}

	return updated
}

func getAuthTriggerSpec(mqt *fv1.MessageQueueTrigger, authenticationRef string, kubeClient kubernetes.Interface) (*unstructured.Unstructured, error) {
	secret, err := kubeClient.CoreV1().Secrets(apiv1.NamespaceDefault).Get(context.TODO(), mqt.Spec.Secret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var secretTargetRefFields []interface{}
	for secretField := range secret.Data {
		secretTargetRefFields = append(secretTargetRefFields, map[string]interface{}{
			"name":      mqt.Spec.Secret,
			"parameter": secretField,
			"key":       secretField,
		})
	}
	authTriggerObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "TriggerAuthentication",
			"apiVersion": apiVersion,
			"metadata": map[string]interface{}{
				"name":      authenticationRef,
				"namespace": mqt.ObjectMeta.Namespace,
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"kind":               "MessageQueueTrigger",
						"apiVersion":         "fission.io/v1",
						"name":               mqt.ObjectMeta.Name,
						"uid":                mqt.ObjectMeta.UID,
						"blockOwnerDeletion": true,
					},
				},
			},
			"spec": map[string]interface{}{
				"secretTargetRef": secretTargetRefFields,
			},
		},
	}
	return authTriggerObj, nil
}

func createAuthTrigger(mqt *fv1.MessageQueueTrigger, authenticationRef string, kubeClient *kubernetes.Clientset) error {
	authTriggerObj, err := getAuthTriggerSpec(mqt, authenticationRef, kubeClient)
	if err != nil {
		return err
	}
	authTriggerClient, err := getAuthTriggerClient(mqt.ObjectMeta.Namespace)
	if err != nil {
		return err
	}
	_, err = authTriggerClient.Create(context.Background(), authTriggerObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func updateAuthTrigger(mqt *fv1.MessageQueueTrigger, authenticationRef string, kubeClient *kubernetes.Clientset) error {
	authTriggerClient, err := getAuthTriggerClient(mqt.ObjectMeta.Namespace)
	if err != nil {
		return err
	}
	oldAuthTriggerObj, err := authTriggerClient.Get(context.Background(), authenticationRef, metav1.GetOptions{})
	if err != nil {
		return err
	}
	resourceVersion := oldAuthTriggerObj.GetResourceVersion()

	authTriggerObj, err := getAuthTriggerSpec(mqt, authenticationRef, kubeClient)
	if err != nil {
		return err
	}
	authTriggerObj.SetResourceVersion(resourceVersion)
	_, err = authTriggerClient.Update(context.Background(), authTriggerObj, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func deleteAuthTrigger(name, namespace string) error {
	authTriggerClient, err := getAuthTriggerClient(namespace)
	if err != nil {
		return err
	}
	err = authTriggerClient.Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func getDeploymentSpec(mqt *fv1.MessageQueueTrigger, routerURL string, kubeClient *kubernetes.Clientset) (*appsv1.Deployment, error) {
	envVars, err := getEnvVarlist(mqt, routerURL, kubeClient)
	if err != nil {
		return nil, err
	}
	imageName := fmt.Sprintf("%s_image", string(mqt.Spec.MessageQueueType))
	image := os.Getenv(strings.ToUpper(imageName))
	imagePullPolicy := utils.GetImagePullPolicy(os.Getenv("CONNECTOR_IMAGE_PULL_POLICY"))

	podSpec := &apiv1.PodSpec{
		Containers: []apiv1.Container{
			{
				Name:            mqt.ObjectMeta.Name,
				Image:           image,
				ImagePullPolicy: imagePullPolicy,
				Env:             envVars,
			},
		},
	}
	podSpec, err = util.MergePodSpec(podSpec, mqt.Spec.PodSpec)
	if err != nil {
		return nil, err
	}

	blockOwnerDeletion := true
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: mqt.ObjectMeta.Name,
			Labels: map[string]string{
				"app": mqt.ObjectMeta.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:               "MessageQueueTrigger",
					APIVersion:         "fission.io/v1",
					Name:               mqt.ObjectMeta.Name,
					UID:                mqt.ObjectMeta.UID,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": mqt.ObjectMeta.Name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": mqt.ObjectMeta.Name,
					},
				},
				Spec: *podSpec,
			},
		},
	}, nil
}

func createDeployment(mqt *fv1.MessageQueueTrigger, routerURL string, kubeClient *kubernetes.Clientset) error {
	deployment, err := getDeploymentSpec(mqt, routerURL, kubeClient)
	if err != nil {
		return err
	}
	_, err = kubeClient.AppsV1().Deployments(apiv1.NamespaceDefault).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func updateDeployment(mqt *fv1.MessageQueueTrigger, routerURL string, kubeClient *kubernetes.Clientset) error {
	deployment, err := getDeploymentSpec(mqt, routerURL, kubeClient)
	if err != nil {
		return err
	}
	_, err = kubeClient.AppsV1().Deployments(apiv1.NamespaceDefault).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func deleteDeployment(name string, kubeClient *kubernetes.Clientset) error {
	deletePolicy := metav1.DeletePropagationForeground
	if err := kubeClient.AppsV1().Deployments(apiv1.NamespaceDefault).Delete(context.TODO(), name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return err
	}
	return nil
}

func getScaledObject(mqt *fv1.MessageQueueTrigger, authenticationRef string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "ScaledObject",
			"apiVersion": apiVersion,
			"metadata": map[string]interface{}{
				"name":      mqt.ObjectMeta.Name,
				"namespace": mqt.ObjectMeta.Namespace,
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"kind":               "MessageQueueTrigger",
						"apiVersion":         "fission.io/v1",
						"name":               mqt.ObjectMeta.Name,
						"uid":                mqt.ObjectMeta.UID,
						"blockOwnerDeletion": true,
					},
				},
			},
			"spec": map[string]interface{}{
				"cooldownPeriod":  &mqt.Spec.CooldownPeriod,
				"maxReplicaCount": &mqt.Spec.MaxReplicaCount,
				"minReplicaCount": &mqt.Spec.MinReplicaCount,
				"pollingInterval": &mqt.Spec.PollingInterval,
				"scaleTargetRef": map[string]interface{}{
					"name": mqt.ObjectMeta.Name,
				},
				"triggers": []interface{}{
					map[string]interface{}{
						"type":     mqt.Spec.MessageQueueType,
						"metadata": mqt.Spec.Metadata,
						"authenticationRef": map[string]interface{}{
							"name": authenticationRef,
						},
					},
				},
			},
		},
	}
}

func createScaledObject(mqt *fv1.MessageQueueTrigger, authenticationRef string) error {
	scaledObject := getScaledObject(mqt, authenticationRef)
	kedaClient, err := getScaledObjectClient(mqt.ObjectMeta.Namespace)
	if err != nil {
		return err
	}
	_, err = kedaClient.Create(context.Background(), scaledObject, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func updateScaledObject(mqt *fv1.MessageQueueTrigger, authenticationRef string) error {
	kedaClient, err := getScaledObjectClient(mqt.ObjectMeta.Namespace)
	if err != nil {
		return err
	}
	oldScaledObject, err := kedaClient.Get(context.Background(), mqt.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	resourceVersion := oldScaledObject.GetResourceVersion()

	scaledObject := getScaledObject(mqt, authenticationRef)
	scaledObject.SetResourceVersion(resourceVersion)

	_, err = kedaClient.Update(context.Background(), scaledObject, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
