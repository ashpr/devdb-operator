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
	"database/sql"

	v1 "k8s.io/api/core/v1"
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
	//mssqlServer := &devdbv1.MSSQLServer{}

	log.Info("⚡️ Event received! ⚡️")
	log.Info("Request: ", "req", req)

	err := r.Get(ctx, req.NamespacedName, mssqlDatabase)

	server := mssqlDatabase.Spec.Server //fmt.Sprintf("%s.%s.svc.cluster.local", mssqlDatabase.Spec.Server, mssqlDatabase.Namespace)
	user := "sa"
	password := "DevPassword123"
	port := 1433

	log.Info(fmt.Sprintf("Resolving Database for server %s, %d", server, port))

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d", server, user, password, port)

	conn, err := sql.Open("mssql", connString)
	if err != nil {
		log.Error(err, "Open connection failed:")
		return ctrl.Result{}, err
	}

	// Check if Database already exists
	row := sql.Row{}
	err = conn.QueryRow("select * from sys.databases where name = $p1", mssqlDatabase.Name).Scan(row)

	if err != sql.ErrNoRows {
		log.Info("Database already exists. No action taken")
		return ctrl.Result{}, nil
	}

	log.Info("Database does not exist.")

	// Database does not exist. Time to create it.
	//_, err = conn.Exec(fmt.Sprintf(``, mssqlDatabase.Spec.CloneDirectory, mssqlDatabase.Spec.CloneDirectory, mssqlDatabase.Name))

	if err != nil {
		log.Error(err, "Could not create database")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MSSQLDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devdbv1.MSSQLDatabase{}).
		Complete(r)
}

// Handy script for remote execution on the target pod
func ExecuteRemoteCommand(pod *v1.Pod, command string) (string, string, error) {
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
