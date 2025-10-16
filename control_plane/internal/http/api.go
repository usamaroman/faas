package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/usamaroman/faas_demo/control_plane/internal/config"
	"github.com/usamaroman/faas_demo/pkg/knative"
	"k8s.io/client-go/rest"
)

type API struct {
	cfg      config.Config
	producer *kafka.Writer // optional
	restCfg  *rest.Config
}

func New(cfg config.Config, producer *kafka.Writer, restCfg *rest.Config) *API {
	return &API{cfg: cfg, producer: producer, restCfg: restCfg}
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/functions/run", a.handleRun)
}

type RunRequest struct {
	ImageName string            `json:"image_name" example:"ealen/echo-server:latest"`
	Envs      map[string]string `json:"envs"`
	Email     string            `json:"email" example:"user@example.com"`
}

type RunResponse struct {
	DeploymentID int64  `json:"deployment_id"`
	ContainerID  string `json:"container_id"`
	Status       string `json:"status"`
}

// handleRun godoc
//
//	@Summary		Run a function
//	@Description	Create a Knative service for the provided function image and envs
//	@Tags			functions
//	@Accept			json
//	@Produce		json
//	@Param			input	body		RunRequest	true	"Request body"
//	@Success		200		{object}	RunResponse
//	@Failure		400		{string}	string	"invalid json"
//	@Failure		405		{string}	string	"method not allowed"
//	@Failure		500		{string}	string
//	@Router			/v1/functions/run [post]
func (a *API) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RunRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Составим имя (простое, по времени) и аннотации с tenant=email
	name := fmt.Sprintf("func-%s-%s", uuid.NewString(), string(req.Email[0]))
	envs := req.Envs
	annotations := map[string]string{}
	if req.Email != "" {
		annotations["tenant"] = req.Email
	}
	// включаем имя образа и envs (в json) прямо в метаданные сервиса
	if req.ImageName != "" {
		annotations["image"] = req.ImageName
	}
	if len(envs) > 0 {
		if b, _ := json.Marshal(envs); len(b) > 0 {
			annotations["env-json"] = string(b)
		}
	}

	// Создаём Knative Service с нужными аннотациями и сайдкаром meter-agent
	image := req.ImageName
	if image == "" {
		image = "ealen/echo-server:latest"
	}

	templateAnnotations := map[string]string{
		"autoscaling.knative.dev/minScale":     "1",
		"autoscaling.knative.dev/maxScale":     "5",
		"autoscaling.knative.dev/target":       "50",
		"autoscaling.knative.dev/class":        "kpa.autoscaling.knative.dev",
		"autoscaling.knative.dev/metric":       "rps",
		"networking.knative.dev/ingress.class": "kourier.ingress.networking.knative.dev",
	}

	_, err := knative.CreateService(ctx, a.restCfg, knative.ServiceConfig{
		Namespace:           a.cfg.K8S.Namespace,
		ServiceName:         name,
		Image:               image,
		ContainerPort:       80,
		AdditionalEnv:       envs,
		Annotations:         annotations,
		TemplateAnnotations: templateAnnotations,
		MeterAgentImage:     "romanchechyotkin/meter_agent:latest",
		MeterURL:            a.cfg.Meter.URL,
		Tenant:              req.Email,
	})
	if err != nil {
		httpError(w, err)
		return
	}

	if a.producer != nil && a.cfg.Kafka.Topic != "" {
		evt := map[string]any{
			"tenant": req.Email,
			"pod":    name,
			"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		}
		_ = publishJSON(a.producer, evt)
	}

	writeJSON(w, http.StatusOK, RunResponse{
		// Можно вернуть имя сервиса как идентификатор
		ContainerID: name,
		Status:      "creating",
	})
}

func httpError(w http.ResponseWriter, err error) {
	var code = http.StatusInternalServerError
	if errors.Is(err, context.DeadlineExceeded) {
		code = http.StatusGatewayTimeout
	}
	http.Error(w, err.Error(), code)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// lightweight kafka wrapper to avoid extra dependencies in control_plane module
func publishJSON(w *kafka.Writer, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := kafka.Message{Value: b}
	return w.WriteMessages(context.Background(), msg)
}

// toStringMap converts map[string]any to map[string]string using fmt.Sprint for values.
func toStringMap(in map[string]any) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		switch t := v.(type) {
		case nil:
			out[k] = ""
		case string:
			out[k] = t
		default:
			out[k] = fmt.Sprint(t)
		}
	}
	return out
}
