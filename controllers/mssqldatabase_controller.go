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

package controllers

import (
	"bytes"
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devdbv1 "github.com/ashpr/devdb-operator/api/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"fmt"

	_ "github.com/microsoft/go-mssqldb"
)

// MSSQLDatabaseReconciler reconciles a MSSQLDatabase object
type MSSQLDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=devdb.io,resources=mssqldatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devdb.io,resources=mssqldatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devdb.io,resources=mssqldatabases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MSSQLDatabase object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *MSSQLDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	mssqlDatabase := &devdbv1.MSSQLDatabase{}
	mssqlServerPods := &metav1.PartialObjectMetadataList{}
	mssqlServerPods.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("PodList"))

	log.Info("⚡️ Event received! ⚡️")
	log.Info("Request: ", "req", req)

	err := r.Get(ctx, req.NamespacedName, mssqlDatabase)

	if err != nil {
		log.Error(err, "Could not get MSSQLDatabase CR")
		return ctrl.Result{}, err
	}

	log.Info("Checking existing CR State")
	//if meta.FindStatusCondition(mssqlDatabase.Status.Conditions, "Ready").Status == metav1.ConditionTrue {
	//		log.Info("Database is in an OK State. No need to proceed")
	//		return ctrl.Result{}, nil
	//	}

	log.Info("Setting CR State")
	//meta.SetStatusCondition(&mssqlDatabase.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})

	log.Info("Searching for MSSQL Pods")
	err = r.List(ctx, mssqlServerPods, client.MatchingLabels{"app": "mssql-server"})

	if err != nil {
		log.Error(err, "Could not query for SQL Server Pods")
		//	meta.SetStatusCondition(&mssqlDatabase.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Failed", Message: "An error was thrown"})
		return ctrl.Result{}, err
	}

	log.Info("Checking for Operator managed MSSQL Pod")
	if len(mssqlServerPods.Items) == 0 {
		err = errors.New("No pods were found")
		log.Error(nil, "No pods found")
		//	meta.SetStatusCondition(&mssqlDatabase.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Failed", Message: "An error was thrown"})
		return ctrl.Result{}, err
	}

	log.Info("Pod found!")
	serverPod := mssqlServerPods.Items[0]

	log.Info("Running Bash script")
	command := fmt.Sprintf("/opt/mssql-scripts/create-database.sh /opt/overlay/target/%s %s", mssqlDatabase.Spec.CloneDirectory, mssqlDatabase.Name)

	stdout, stderr, err := ExecuteRemoteCommand(&serverPod, command)

	log.Info(stdout)
	log.Info(stderr)

	if err != nil {
		log.Error(err, "Could not create database")
		//	meta.SetStatusCondition(&mssqlDatabase.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Failed", Message: "An error was thrown"})
		return ctrl.Result{}, err
	}

	//meta.SetStatusCondition(&mssqlDatabase.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Database Deployment", Message: "Completed"})

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MSSQLDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devdbv1.MSSQLDatabase{}).
		Complete(r)
}

// Handy script for remote execution on the target pod
func ExecuteRemoteCommand(pod *metav1.PartialObjectMetadata, command string) (string, string, error) {
	kubeCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restCfg, err := kubeCfg.ClientConfig()
	if err != nil {
		return "", "", err
	}
	coreClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return "", "", err
	}

	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := coreClient.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: []string{"/bin/sh", "-c", command},
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(restCfg, "POST", request.URL())
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", "", fmt.Errorf("%w Failed executing command %s on %v/%v", err, command, pod.Namespace, pod.Name)
	}

	return buf.String(), errBuf.String(), nil
}
