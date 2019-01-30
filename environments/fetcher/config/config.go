package container

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	crd "github.com/fission/fission/crd"
)

type Config struct {
	fetcherImage           string
	fetcherImagePullPolicy apiv1.PullPolicy

	resourceRequirements apiv1.ResourceRequirements

	// used by generic pool when creating env deployment to specify the share volume path for fetcher & env
	// change this may break v1 compatibility, since most of the v1 environments have hard-coded "/userfunc" in loading path
	sharedMountPath  string
	sharedSecretPath string
	sharedCfgMapPath string

	dockerRegistryAuthDomain string
	dockerRegistryUsername   string
	dockerRegistryPassword   string

	serviceAccount string

	jaegerCollectorEndpoint string
}

func getFetcherResources() (apiv1.ResourceRequirements, error) {
	mincpu, err := resource.ParseQuantity(os.Getenv("FETCHER_MINCPU"))
	if err != nil {
		return apiv1.ResourceRequirements{}, err
	}

	minmem, err := resource.ParseQuantity(os.Getenv("FETCHER_MINMEM"))
	if err != nil {
		return apiv1.ResourceRequirements{}, err
	}

	maxcpu, err := resource.ParseQuantity(os.Getenv("FETCHER_MAXCPU"))
	if err != nil {
		return apiv1.ResourceRequirements{}, err
	}

	maxmem, err := resource.ParseQuantity(os.Getenv("FETCHER_MAXMEM"))
	if err != nil {
		return apiv1.ResourceRequirements{}, err
	}

	return apiv1.ResourceRequirements{
		Requests: map[apiv1.ResourceName]resource.Quantity{
			apiv1.ResourceCPU:    mincpu,
			apiv1.ResourceMemory: minmem,
		},
		Limits: map[apiv1.ResourceName]resource.Quantity{
			apiv1.ResourceCPU:    maxcpu,
			apiv1.ResourceMemory: maxmem,
		},
	}, nil
}

func MakeFetcherConfig(sharedMountPath string) (*Config, error) {
	resources, err := getFetcherResources()
	if err != nil {
		return nil, err
	}

	fetcherImage := os.Getenv("FETCHER_IMAGE")
	if len(fetcherImage) == 0 {
		fetcherImage = "fission/fetcher"
	}

	fetcherImagePullPolicy := os.Getenv("FETCHER_IMAGE_PULL_POLICY")
	if len(fetcherImagePullPolicy) == 0 {
		fetcherImagePullPolicy = "IfNotPresent"
	}

	return &Config{
		resourceRequirements:    resources,
		fetcherImage:            fetcherImage,
		fetcherImagePullPolicy:  fission.GetImagePullPolicy(fetcherImagePullPolicy),
		sharedMountPath:         sharedMountPath,
		sharedSecretPath:        "/secrets",
		sharedCfgMapPath:        "/configmaps",
		jaegerCollectorEndpoint: os.Getenv("OPENCENSUS_TRACE_JAEGER_COLLECTOR_ENDPOINT"),
		serviceAccount:          fission.FissionFetcherSA,
	}, nil
}

func (cfg *Config) SetupServiceAccount(kubernetesClient *kubernetes.Clientset, namespace string, context interface{}) error {
	_, err := fission.SetupSA(kubernetesClient, fission.FissionFetcherSA, namespace)
	if err != nil {
		log.Printf("Error : %v creating %s in ns : %s for: %#v", err, fission.FissionFetcherSA, namespace, context)
		return err
	}

	return nil
}

func (cfg *Config) SharedMountPath() string {
	return cfg.sharedMountPath
}

func (cfg *Config) NewSpecializeRequest(fn *crd.Function, env *crd.Environment) fission.FunctionSpecializeRequest {
	// for backward compatibility, since most v1 env
	// still try to load user function from hard coded
	// path /userfunc/user
	targetFilename := "user"
	if env.Spec.Version >= 2 {
		targetFilename = string(fn.Metadata.UID)
	}

	return fission.FunctionSpecializeRequest{
		FetchReq: fission.FunctionFetchRequest{
			FetchType: fission.FETCH_DEPLOYMENT,
			Package: metav1.ObjectMeta{
				Namespace: fn.Spec.Package.PackageRef.Namespace,
				Name:      fn.Spec.Package.PackageRef.Name,
			},
			Filename:    targetFilename,
			Secrets:     fn.Spec.Secrets,
			ConfigMaps:  fn.Spec.ConfigMaps,
			KeepArchive: env.Spec.KeepArchive,
		},
		LoadReq: fission.FunctionLoadRequest{
			FilePath:         filepath.Join(cfg.sharedMountPath, targetFilename),
			FunctionName:     fn.Spec.Package.FunctionName,
			FunctionMetadata: &fn.Metadata,
			EnvVersion:       env.Spec.Version,
		},
	}
}

func (cfg *Config) AddFetcherToPodSpec(podSpec *apiv1.PodSpec, mainContainerName string) error {
	return cfg.addFetcherToPodSpecWithCommand(podSpec, mainContainerName, cfg.fetcherCommand())
}

func (cfg *Config) AddSpecializingFetcherToPodSpec(podSpec *apiv1.PodSpec, mainContainerName string, fn *crd.Function, env *crd.Environment) error {
	specializeReq := cfg.NewSpecializeRequest(fn, env)
	specializePayload, err := json.Marshal(specializeReq)
	if err != nil {
		return err
	}

	return cfg.addFetcherToPodSpecWithCommand(
		podSpec,
		mainContainerName,
		cfg.fetcherCommand(
			"-specialize-on-startup",
			"-specialize-request", string(specializePayload),
		),
	)
}

func (cfg *Config) fetcherCommand(extraArgs ...string) []string {
	command := []string{"/fetcher",
		"-secret-dir", cfg.sharedSecretPath,
		"-cfgmap-dir", cfg.sharedCfgMapPath,
		"-jaeger-collector-endpoint", cfg.jaegerCollectorEndpoint,
	}

	command = append(command, extraArgs...)
	command = append(command, cfg.sharedMountPath)
	return command
}

func (cfg *Config) volumesWithMounts() ([]apiv1.Volume, []apiv1.VolumeMount) {
	volumes := []apiv1.Volume{
		{
			Name: fission.SharedVolumeUserfunc,
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: fission.SharedVolumeSecrets,
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: fission.SharedVolumeConfigmaps,
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
	mounts := []apiv1.VolumeMount{
		{
			Name:      fission.SharedVolumeUserfunc,
			MountPath: cfg.sharedMountPath,
		},
		{
			Name:      fission.SharedVolumeSecrets,
			MountPath: cfg.sharedSecretPath,
		},
		{
			Name:      fission.SharedVolumeConfigmaps,
			MountPath: cfg.sharedCfgMapPath,
		},
	}

	return volumes, mounts
}

func (cfg *Config) addFetcherToPodSpecWithCommand(podSpec *apiv1.PodSpec, mainContainerName string, command []string) error {
	volumes, mounts := cfg.volumesWithMounts()
	c := apiv1.Container{
		Name:                   "fetcher",
		Command:                command,
		Image:                  cfg.fetcherImage,
		ImagePullPolicy:        cfg.fetcherImagePullPolicy,
		TerminationMessagePath: "/dev/termination-log",
		VolumeMounts:           mounts,
		Resources:              cfg.resourceRequirements,
		ReadinessProbe: &apiv1.Probe{
			InitialDelaySeconds: 1,
			PeriodSeconds:       1,
			FailureThreshold:    30,
			Handler: apiv1.Handler{
				HTTPGet: &apiv1.HTTPGetAction{
					Path: "/readniess-healthz",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8000,
					},
				},
			},
		},
		LivenessProbe: &apiv1.Probe{
			InitialDelaySeconds: 1,
			PeriodSeconds:       5,
			Handler: apiv1.Handler{
				HTTPGet: &apiv1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8000,
					},
				},
			},
		},
	}

	// Pod is removed from endpoints list for service when it's
	// state became "Termination". We used preStop hook as the
	// workaround for connection draining since pod maybe shutdown
	// before grace period expires.
	// https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods
	// https://github.com/kubernetes/kubernetes/issues/47576#issuecomment-308900172
	if podSpec.TerminationGracePeriodSeconds != nil {
		c.Lifecycle = &apiv1.Lifecycle{
			PreStop: &apiv1.Handler{
				Exec: &apiv1.ExecAction{
					Command: []string{
						"/bin/sleep",
						fmt.Sprintf("%v", *podSpec.TerminationGracePeriodSeconds),
					},
				},
			},
		}
	}

	found := false
	for ix, container := range podSpec.Containers {
		if container.Name != mainContainerName {
			continue
		}

		found = true
		container.VolumeMounts = append(container.VolumeMounts, mounts...)
		podSpec.Containers[ix] = container
	}
	if !found {
		existingContainerNames := make([]string, len(podSpec.Containers))
		for _, existingContainer := range podSpec.Containers {
			existingContainerNames = append(existingContainerNames, existingContainer.Name)
		}
		return fmt.Errorf("Could not find main container '%s' in given PodSpec. Found: %v",
			mainContainerName,
			existingContainerNames)
	}

	podSpec.Volumes = append(podSpec.Volumes, volumes...)
	podSpec.Containers = append(podSpec.Containers, c)
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = fission.FissionFetcherSA
	}

	return nil
}
