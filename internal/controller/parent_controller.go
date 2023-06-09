/*
Copyright 2023.

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

package controller

import (
	"context"

	otelv1 "github.com/zoetrope/otel-sample-controller/api/v1"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ParentReconciler reconciles a Parent object
type ParentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Tracer trace.Tracer
}

//+kubebuilder:rbac:groups=otel.zoetrope.github.io,resources=parents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=otel.zoetrope.github.io,resources=parents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=otel.zoetrope.github.io,resources=parents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Parent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ParentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := r.Tracer.Start(ctx, "parent-reconcile")
	defer span.End()

	traceID := span.SpanContext().TraceID()
	spanID := span.SpanContext().SpanID()

	logger := log.FromContext(ctx).WithValues("traceID", traceID, "spanID", spanID)
	logger.Info("Reconcile")

	var parent otelv1.Parent
	err := r.Get(ctx, req.NamespacedName, &parent)
	if err != nil {
		if !errors.IsNotFound(err) {
			span.RecordError(err)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !parent.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	child := &otelv1.Child{
		ObjectMeta: metav1.ObjectMeta{
			Name:      parent.Name + "-child",
			Namespace: parent.Namespace,
		},
	}
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, child, func() error {
		if child.Annotations == nil {
			child.Annotations = make(map[string]string)
		}
		child.Annotations["otel.zoetrope.github.io/trace_id"] = traceID.String()
		child.Annotations["otel.zoetrope.github.io/span_id"] = spanID.String()
		return ctrl.SetControllerReference(&parent, child, r.Scheme)
	})
	if err != nil {
		span.RecordError(err)
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("crate or update child successfully")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ParentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&otelv1.Parent{}).
		Complete(r)
}
