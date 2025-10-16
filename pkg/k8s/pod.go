package k8s

import (
	"context"
	"encoding/json"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RunPod creates a Pod with the provided main container image and optional meter-agent sidecar.
// It returns the created Pod name.
func RunPod(
	ctx context.Context,
	restCfg *rest.Config,
	namespace string,
	name string,
	image string,
	envs map[string]string,
	annotations map[string]string,
	meterAgentImage string,
	meterURL string,
) (string, error) {
	cli, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		slog.Error("failed to get k8s client", slog.String("error", err.Error()))
		return "", err
	}

	var envList []v1.EnvVar
	for k, v := range envs {
		envList = append(envList, v1.EnvVar{Name: k, Value: v})
	}

	// base annotations plus a compact JSON of envs and image name for traceability
	if annotations == nil {
		annotations = map[string]string{}
	}
	envJSON, _ := json.Marshal(envs)
	annotations["image"] = image
	annotations["env-json"] = string(envJSON)

	shareProc := true
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: v1.PodSpec{
			ShareProcessNamespace: &shareProc,
			Containers: []v1.Container{
				{
					Name:  "main-app",
					Image: image,
					Env:   envList,
				},
			},
		},
	}

	if meterAgentImage != "" {
		meterEnv := []v1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				}},
			},
		}
		if meterURL != "" {
			meterEnv = append(meterEnv, v1.EnvVar{Name: "METER_URL", Value: meterURL})
		}
		pod.Spec.Containers = append(pod.Spec.Containers, v1.Container{
			Name:  "meter-agent-sidecar",
			Image: meterAgentImage,
			Env:   meterEnv,
		})
	}

	created, err := cli.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return created.Name, nil
}
