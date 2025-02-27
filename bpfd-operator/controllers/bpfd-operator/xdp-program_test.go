/*
Copyright 2022.

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

package bpfdoperator

import (
	"context"
	"fmt"
	"testing"

	bpfdiov1alpha1 "github.com/bpfd-dev/bpfd/bpfd-operator/apis/v1alpha1"
	internal "github.com/bpfd-dev/bpfd/bpfd-operator/internal"
	testutils "github.com/bpfd-dev/bpfd/bpfd-operator/internal/test-utils"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestXdpProgramReconcile(t *testing.T) {
	var (
		name         = "fakeXdpProgram"
		bytecodePath = "/tmp/hello.o"
		sectionName  = "test"
		fakeNode     = testutils.NewNode("fake-control-plane")
		fakeInt      = "eth0"
		ctx          = context.TODO()
		bpfProgName  = fmt.Sprintf("%s-%s", name, fakeNode.Name)
	)
	// A XdpProgram object with metadata and spec.
	Xdp := &bpfdiov1alpha1.XdpProgram{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: bpfdiov1alpha1.XdpProgramSpec{
			BpfProgramCommon: bpfdiov1alpha1.BpfProgramCommon{
				SectionName:  sectionName,
				NodeSelector: metav1.LabelSelector{},
				ByteCode: bpfdiov1alpha1.BytecodeSelector{
					Path: &bytecodePath,
				},
			},
			InterfaceSelector: bpfdiov1alpha1.InterfaceSelector{
				Interfaces: &[]string{fakeInt},
			},
			Priority: 0,
			ProceedOn: []bpfdiov1alpha1.XdpProceedOnValue{bpfdiov1alpha1.XdpProceedOnValue("pass"),
				bpfdiov1alpha1.XdpProceedOnValue("dispatcher_return"),
			},
		},
	}

	// The expected accompanying BpfProgram object
	expectedBpfProg := &bpfdiov1alpha1.BpfProgram{
		ObjectMeta: metav1.ObjectMeta{
			Name: bpfProgName,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       Xdp.Name,
					Controller: &[]bool{true}[0],
				},
			},
			Labels:     map[string]string{internal.BpfProgramOwnerLabel: Xdp.Name, internal.K8sHostLabel: fakeNode.Name},
			Finalizers: []string{internal.TcProgramControllerFinalizer},
		},
		Spec: bpfdiov1alpha1.BpfProgramSpec{
			Type: "tc",
		},
		Status: bpfdiov1alpha1.BpfProgramStatus{
			Conditions: []metav1.Condition{bpfdiov1alpha1.BpfProgCondLoaded.Condition()},
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{fakeNode, Xdp, expectedBpfProg}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, Xdp)
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.TcProgramList{})
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.BpfProgram{})
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.BpfProgramList{})

	// Create a fake client to mock API calls.
	cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	rc := ReconcilerCommon{
		Client: cl,
		Scheme: s,
	}

	// Set development Logger so we can see all logs in tests.
	logf.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	// Create a ReconcileMemcached object with the scheme and fake client.
	r := &XdpProgramReconciler{ReconcilerCommon: rc}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
	}

	// First reconcile should add the finalzier to the tcProgram object
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)

	// Check the BpfProgram Object was created successfully
	err = cl.Get(ctx, types.NamespacedName{Name: Xdp.Name, Namespace: metav1.NamespaceAll}, Xdp)
	require.NoError(t, err)

	// Check the bpfd-operator finalizer was successfully added
	require.Contains(t, Xdp.GetFinalizers(), internal.BpfdOperatorFinalizer)

	// Second reconcile should check bpfProgram Status and write Success condition to tcProgram Status
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)

	// Check the BpfProgram Object was created successfully
	err = cl.Get(ctx, types.NamespacedName{Name: Xdp.Name, Namespace: metav1.NamespaceAll}, Xdp)
	require.NoError(t, err)

	require.Equal(t, Xdp.Status.Conditions[0].Type, string(bpfdiov1alpha1.ProgramReconcileSuccess))

}
