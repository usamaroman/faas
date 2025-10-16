package knative

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// ServiceConfig describes the desired Knative Service to be created.
type ServiceConfig struct {
	Namespace           string
	ServiceName         string
	Image               string
	ContainerPort       int32
	AdditionalEnv       map[string]string
	Annotations         map[string]string
	TemplateAnnotations map[string]string
	MeterAgentImage     string
	MeterURL            string
	ImagePullPolicy     string
	Tenant              string
}

var (
	knativeServiceGVR = schema.GroupVersionResource{
		Group:    "serving.knative.dev",
		Version:  "v1",
		Resource: "services",
	}
)

// CreateService creates a Knative Service with the provided configuration.
func CreateService(ctx context.Context, restConfig *rest.Config, cfg ServiceConfig) (*unstructured.Unstructured, error) {
	if cfg.ContainerPort == 0 {
		cfg.ContainerPort = 80
	}
	if cfg.ImagePullPolicy == "" {
		cfg.ImagePullPolicy = "IfNotPresent"
	}
	if cfg.MeterAgentImage == "" {
		cfg.MeterAgentImage = "romanchechyotkin/meter_agent:latest"
	}

	// Ensure template annotations exist and set sane defaults if not provided
	if cfg.TemplateAnnotations == nil {
		cfg.TemplateAnnotations = map[string]string{}
	}
	if _, ok := cfg.TemplateAnnotations["autoscaling.knative.dev/minScale"]; !ok {
		cfg.TemplateAnnotations["autoscaling.knative.dev/minScale"] = "1"
	}
	if _, ok := cfg.TemplateAnnotations["autoscaling.knative.dev/maxScale"]; !ok {
		cfg.TemplateAnnotations["autoscaling.knative.dev/maxScale"] = "5"
	}
	if _, ok := cfg.TemplateAnnotations["autoscaling.knative.dev/target"]; !ok {
		cfg.TemplateAnnotations["autoscaling.knative.dev/target"] = "50"
	}
	if _, ok := cfg.TemplateAnnotations["autoscaling.knative.dev/class"]; !ok {
		cfg.TemplateAnnotations["autoscaling.knative.dev/class"] = "kpa.autoscaling.knative.dev"
	}
	if _, ok := cfg.TemplateAnnotations["autoscaling.knative.dev/metric"]; !ok {
		cfg.TemplateAnnotations["autoscaling.knative.dev/metric"] = "rps"
	}
	if _, ok := cfg.TemplateAnnotations["networking.knative.dev/ingress.class"]; !ok {
		cfg.TemplateAnnotations["networking.knative.dev/ingress.class"] = "kourier.ingress.networking.knative.dev"
	}

	dc, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		slog.Error("failed to construct dynamic client", slog.String("error", err.Error()))
		return nil, err
	}

	obj := buildKnativeServiceObject(cfg)
	created, err := dc.Resource(knativeServiceGVR).Namespace(cfg.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating knative service %s/%s: %w", cfg.Namespace, cfg.ServiceName, err)
	}
	return created, nil
}

func buildKnativeServiceObject(cfg ServiceConfig) *unstructured.Unstructured {
	userContainer := map[string]any{
		"name":            "user-container",
		"image":           cfg.Image,
		"imagePullPolicy": cfg.ImagePullPolicy,
		"ports": []any{
			map[string]any{"containerPort": cfg.ContainerPort},
		},
	}

	if len(cfg.AdditionalEnv) > 0 {
		var envList []any
		for k, v := range cfg.AdditionalEnv {
			envList = append(envList, map[string]any{"name": k, "value": v})
		}
		userContainer["env"] = envList
	}

	meterEnv := []any{
		map[string]any{
			"name":  "POD_NAME",
			"value": cfg.ServiceName,
		},
	}
	if cfg.MeterURL != "" {
		meterEnv = append(meterEnv, map[string]any{"name": "METER_URL", "value": cfg.MeterURL})
	}
	if cfg.Tenant != "" {
		meterEnv = append(meterEnv, map[string]any{"name": "TENANT", "value": cfg.Tenant})
	}

	meterAgentContainer := map[string]any{
		"name":  "meter-agent-sidecar",
		"image": cfg.MeterAgentImage,
		"env":   meterEnv,
	}

	containers := []any{userContainer, meterAgentContainer}

	service := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "serving.knative.dev/v1",
		"kind":       "Service",
		"metadata": map[string]any{
			"name":      cfg.ServiceName,
			"namespace": cfg.Namespace,
		},
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": cfg.TemplateAnnotations,
				},
				"spec": map[string]any{
					"containers": containers,
				},
			},
		},
	}}

	return service
}
