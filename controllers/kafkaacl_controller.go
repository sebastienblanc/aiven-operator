// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

type KafkaACLHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get;list;watch;create;delete

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaACLHandler{}, &v1alpha1.KafkaACL{})
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaACL{}).
		Complete(r)
}

func (h KafkaACLHandler) createOrUpdate(avn *aiven.Client, i client.Object, refs []client.Object) error {
	acl, err := h.convert(i)
	if err != nil {
		return err
	}

	exists, err := h.exists(avn, acl)
	if err != nil {
		return err
	}

	if !exists {
		_, err = avn.KafkaACLs.Create(
			acl.Spec.Project,
			acl.Spec.ServiceName,
			aiven.CreateKafkaACLRequest{
				Permission: acl.Spec.Permission,
				Topic:      acl.Spec.Topic,
				Username:   acl.Spec.Username,
			},
		)
		if err != nil && !aiven.IsAlreadyExists(err) {
			return err
		}
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("CreatedOrUpdate",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "CreatedOrUpdate",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(acl.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaACLHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Server returns 404 if resource doesn't exist OR it's just an invalid URL, so having ID is mandatory
	// This workaround fixes existing ACLs created without ID
	id := acl.Status.ID
	if id == "" {
		aivenACL, err := h.getAivenACL(avn, acl)
		if err != nil {
			return false, err
		}
		id = aivenACL.ID
	}

	err = avn.KafkaACLs.Delete(acl.Spec.Project, acl.Spec.ServiceName, id)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	return true, nil
}

func (h KafkaACLHandler) exists(avn *aiven.Client, acl *v1alpha1.KafkaACL) (bool, error) {
	a, err := h.getAivenACL(avn, acl)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return a != nil, err
}

func (h KafkaACLHandler) getAivenACL(avn *aiven.Client, acl *v1alpha1.KafkaACL) (*aiven.KafkaACL, error) {
	// This object already has ID
	if acl.Status.ID != "" {
		return avn.KafkaACLs.Get(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	}

	// First time lookup
	list, err := avn.KafkaACLs.List(acl.Spec.Project, acl.Spec.ServiceName)
	if err != nil {
		return nil, err
	}

	for _, a := range list {
		if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
			return a, nil
		}
	}

	// Error should mimic client error to play well with aiven.IsNotFound(err)
	return nil, aiven.Error{Status: http.StatusNotFound, Message: "Kafka ACL not found"}
}

func (h KafkaACLHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	acl, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	aivenACL, err := h.getAivenACL(avn, acl)
	if err != nil {
		return nil, err
	}

	acl.Status.ID = aivenACL.ID
	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaACLHandler) checkPreconditions(avn *aiven.Client, i client.Object) (bool, error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(avn, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaACLHandler) convert(i client.Object) (*v1alpha1.KafkaACL, error) {
	acl, ok := i.(*v1alpha1.KafkaACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaACL")
	}

	return acl, nil
}
