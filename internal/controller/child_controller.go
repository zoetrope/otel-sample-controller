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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ChildReconciler reconciles a Child object
type ChildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Tracer trace.Tracer
}

//+kubebuilder:rbac:groups=otel.zoetrope.github.io,resources=children,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=otel.zoetrope.github.io,resources=children/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=otel.zoetrope.github.io,resources=children/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Child object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ChildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var child otelv1.Child
	err := r.Get(ctx, req.NamespacedName, &child)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !child.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{},
		SpanID:     trace.SpanID{},
		TraceState: trace.TraceState{},
		TraceFlags: 01,
		Remote:     false,
	})

	if parentTraceID, ok := child.Annotations["otel.zoetrope.github.io/trace_id"]; ok {
		traceID, err := trace.TraceIDFromHex(parentTraceID)
		if err != nil {
			return ctrl.Result{}, err
		}
		spanCtx = spanCtx.WithTraceID(traceID)
	}
	if parentSpanID, ok := child.Annotations["otel.zoetrope.github.io/span_id"]; ok {
		spanID, err := trace.SpanIDFromHex(parentSpanID)
		if err != nil {
			return ctrl.Result{}, err
		}
		spanCtx = spanCtx.WithSpanID(spanID)
	}
	log.FromContext(ctx).Info("SpanContext is valid", "isValid", spanCtx.IsValid())

	ctx, span := r.Tracer.Start(trace.ContextWithSpanContext(ctx, spanCtx), "child-reconcile")
	defer span.End()

	traceID := span.SpanContext().TraceID()
	spanID := span.SpanContext().SpanID()

	logger := log.FromContext(ctx).WithValues("traceID", traceID, "spanID", spanID)
	logger.Info("Reconcile")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&otelv1.Child{}).
		Complete(r)
}
