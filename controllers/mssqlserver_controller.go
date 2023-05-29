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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devdbv1 "github.com/ashpr/devdb-operator/api/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// MSSQLServerReconciler reconciles a MSSQLServer object
type MSSQLServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=devdb.io,resources=mssqlServers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devdb.io,resources=mssqlServers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devdb.io,resources=mssqlServers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MSSQLServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *MSSQLServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	mssqlServer := &devdbv1.MSSQLServer{}
	existingDeployment := &appsv1.Deployment{}
	existingService := &corev1.Service{}
	existingConfigMap := &corev1.ConfigMap{}

	log.Info("⚡️ Event received! ⚡️")
	log.Info("Request: ", "req", req)

	// CR deleted : check if  the Deployment and the Service must be deleted
	err := r.Get(ctx, req.NamespacedName, mssqlServer)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("MSSQLServer resource not found, check if a deployment must be deleted.")

			// Delete Deployment
			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, existingDeployment)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Nothing to do, no deployment found.")
					return ctrl.Result{}, nil
				} else {
					log.Error(err, "❌ Failed to get Deployment")
					return ctrl.Result{}, err
				}
			} else {
				log.Info("☠️ Deployment exists: delete it. ☠️")
				r.Delete(ctx, existingDeployment)
			}

			// Delete Service
			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, existingService)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("Nothing to do, no service found.")
					return ctrl.Result{}, nil
				} else {
					log.Error(err, "❌ Failed to get Service")
					return ctrl.Result{}, err
				}
			} else {
				log.Info("☠️ Service exists: delete it. ☠️")
				r.Delete(ctx, existingService)
				return ctrl.Result{}, nil
			}
		}
	} else {
		log.Info("ℹ️  CR state ℹ️", "mssqlServer.Name", mssqlServer.Name, " mssqlServer.Namespace", mssqlServer.Namespace)

		// Check if the ConfigMap already exists, if not: create a new one
		err = r.Get(ctx, types.NamespacedName{Name: mssqlServer.Name, Namespace: mssqlServer.Namespace}, existingConfigMap)
		if err != nil && errors.IsNotFound(err) {
			// Define a new deployment
			configMap := r.createConfigMap(mssqlServer)
			log.Info("✨ Creating a new configMap", "configMap.Namespace", configMap.Namespace, "configMap.Name", configMap.Name)

			err = r.Create(ctx, configMap)
			if err != nil {
				log.Error(err, "❌ Failed to create new ConfigMap", "configMap.Namespace", configMap.Namespace, "configMap.Name", configMap.Name)
				return ctrl.Result{}, err
			}

			if err := ctrl.SetControllerReference(mssqlServer, configMap, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// Check for updates here
		} else if err != nil {
			log.Error(err, "Failed to get Deployment")
			return ctrl.Result{}, err
		}

		// Check if the deployment already exists, if not: create a new one.
		err = r.Get(ctx, types.NamespacedName{Name: mssqlServer.Name, Namespace: mssqlServer.Namespace}, existingDeployment)
		if err != nil && errors.IsNotFound(err) {
			// Define a new deployment
			deployment := r.createDeployment(mssqlServer)
			log.Info("✨ Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

			err = r.Create(ctx, deployment)
			if err != nil {
				log.Error(err, "❌ Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
				return ctrl.Result{}, err
			}

			if err := ctrl.SetControllerReference(mssqlServer, deployment, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// Check for updates here
		} else if err != nil {
			log.Error(err, "Failed to get Deployment")
			return ctrl.Result{}, err
		}

		// Check if the service already exists, if not: create a new one
		err = r.Get(ctx, types.NamespacedName{Name: mssqlServer.Name, Namespace: mssqlServer.Namespace}, existingService)
		if err != nil && errors.IsNotFound(err) {
			// Create the Service
			service := r.createService(mssqlServer)
			log.Info("✨ Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			err = r.Create(ctx, service)
			if err != nil {
				log.Error(err, "❌ Failed to create new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
				return ctrl.Result{}, err
			}

			if err := ctrl.SetControllerReference(mssqlServer, service, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// Checks here for if the service has changed
		} else if err != nil {
			log.Error(err, "Failed to get Service")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MSSQLServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devdbv1.MSSQLServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// Create the MSSQL Database Server
func (r *MSSQLServerReconciler) createDeployment(MSSQLServerCR *devdbv1.MSSQLServer) *appsv1.Deployment {
	executeMode := int32(0777)

	volumes := []corev1.Volume{
		{
			Name: fmt.Sprintf("%s-config", MSSQLServerCR.Name),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: MSSQLServerCR.Name,
					},
					DefaultMode: &executeMode,
				},
			},
		},
		{
			Name: "overlay",
			VolumeSource: *&corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	volumes = append(volumes, MSSQLServerCR.Spec.Clones...)

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      fmt.Sprintf("%s-config", MSSQLServerCR.Name),
			MountPath: "/opt/mssql-scripts",
		},
		{
			Name:      "overlay",
			MountPath: "/opt/overlay",
		},
	}

	overlaycmd := ""

	for _, clone := range MSSQLServerCR.Spec.Clones {
		// Map this clone to a volumemount in the Clones directory
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      clone.Name,
			MountPath: "/opt/clones/" + clone.Name,
			ReadOnly:  true,
		})
		overlaycmd = overlaycmd + fmt.Sprintf("mkdir -p /opt/overlay/upper/%s && mkdir -p /opt/overlay/work/%s && mkdir -p /opt/overlay/target/%s && mount -t overlay -o lowerdir=/opt/clones/%s,upperdir=/opt/overlay/upper/%s,workdir=/opt/overlay/work/%s overlay /opt/overlay/target/%s && ",
			clone.Name, clone.Name, clone.Name, clone.Name, clone.Name, clone.Name, clone.Name)
	}

	priviledged := true
	runAsUser := int64(0)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MSSQLServerCR.Name,
			Namespace: MSSQLServerCR.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "mssql-server"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "mssql-server"},
				},
				Spec: corev1.PodSpec{
					Hostname: "mssqlinst",
					Containers: []corev1.Container{{
						Image:   "mcr.microsoft.com/mssql/server:2022-latest",
						Name:    "mssql",
						Command: []string{"/bin/sh", "-c"},
						Args:    []string{fmt.Sprintf("%s /opt/mssql/bin/sqlservr", overlaycmd)},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &priviledged,
							RunAsUser:  &runAsUser,
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 1433,
							Name:          "tcp",
							Protocol:      "TCP",
						}},
						Env: []corev1.EnvVar{{
							Name:  "MSSQL_PID",
							Value: "Developer",
						}, {
							Name:  "ACCEPT_EULA",
							Value: "Y",
						}, {
							Name:  "MSSQL_SA_PASSWORD",
							Value: "DevPassword123",
						}},
						VolumeMounts: volumeMounts,
					}},
					Volumes: volumes,
				},
			},
		},
	}
	return deployment
}

// Create the MSSQL Database Server Service
func (r *MSSQLServerReconciler) createService(MSSQLServerCR *devdbv1.MSSQLServer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MSSQLServerCR.Name,
			Namespace: MSSQLServerCR.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "mssql-server",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "tcp",
					Protocol:   corev1.ProtocolTCP,
					Port:       1433,
					TargetPort: intstr.FromString("tcp"),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

func (r *MSSQLServerReconciler) createConfigMap(MSSQLServerCR *devdbv1.MSSQLServer) *corev1.ConfigMap {
	createDatabaseScript := `#!/bin/bash

	# Check if the folder path and database name are provided
	if [[ $# -lt 2 ]]; then
	  echo "Usage: $0 <folder_path> <database_name>"
	  exit 1
	fi
	
	# Get the folder path and database name from the arguments
	folder_path="$1"
	new_database_name="$2"
	
	# Get the list of MDF and LDF files in the folder
	mdf_files=($(find "$folder_path" -name "*.mdf"))
	ldf_files=($(find "$folder_path" -name "*.ldf"))
	
	# Check if there are MDF and LDF files present
	if [[ ${#mdf_files[@]} -eq 0 || ${#ldf_files[@]} -eq 0 ]]; then
	  echo "No MDF or LDF files found in the specified folder."
	  exit 1
	fi
	
	# Construct the SQL query to attach the MDF and LDF files
	sql_query="CREATE DATABASE [$new_database_name] ON "
	for (( i=0; i<${#mdf_files[@]}; i++ )); do
	  mdf_file=${mdf_files[i]}
	  ldf_file=${ldf_files[i]}
	  sql_query+="(FILENAME = '$mdf_file'), "
	  sql_query+="(FILENAME = '$ldf_file') "
	  if [[ $i -lt $((${#mdf_files[@]} - 1)) ]]; then
		sql_query+=", "
	  fi
	done
	sql_query+="FOR ATTACH"
	
	# Execute the SQL query using sqlcmd
	/opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P $MSSQL_SA_PASSWORD -Q "$sql_query"
	
	# Check if the SQL query execution was successful
	if [[ $? -eq 0 ]]; then
	  echo "Database created and attached successfully."
	else
	  echo "Failed to create and attach the database."
	fi`

	config := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MSSQLServerCR.Name,
			Namespace: MSSQLServerCR.Namespace,
		},
		Data: map[string]string{
			"create-database.sh": createDatabaseScript,
		},
	}

	return config
}
