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

package bpfdagent

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	bpfdiov1alpha1 "github.com/bpfd-dev/bpfd/bpfd-operator/apis/v1alpha1"
	agenttestutils "github.com/bpfd-dev/bpfd/bpfd-operator/controllers/bpfd-agent/internal/test-utils"
	internal "github.com/bpfd-dev/bpfd/bpfd-operator/internal"
	testutils "github.com/bpfd-dev/bpfd/bpfd-operator/internal/test-utils"

	gobpfd "github.com/bpfd-dev/bpfd/clients/gobpfd/v1"
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

func TestXdpProgramControllerCreate(t *testing.T) {
	var (
		name         = "fakeXdpProgram"
		namespace    = "bpfd"
		bytecodePath = "/tmp/hello.o"
		sectionName  = "test"
		fakeNode     = testutils.NewNode("fake-control-plane")
		fakeInt      = "eth0"
		ctx          = context.TODO()
		bpfProgName  = fmt.Sprintf("%s-%s-%s", name, fakeNode.Name, fakeInt)
		bpfProg      = &bpfdiov1alpha1.BpfProgram{}
		fakeUID      = "ef71d42c-aa21-48e8-a697-82391d801a81"
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

	// Objects to track in the fake client.
	objs := []runtime.Object{fakeNode, Xdp}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, Xdp)
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.XdpProgramList{})
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.BpfProgram{})

	// Create a fake client to mock API calls.
	cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	cli := agenttestutils.NewBpfdClientFake()

	rc := ReconcilerCommon{
		Client:       cl,
		Scheme:       s,
		BpfdClient:   cli,
		NodeName:     fakeNode.Name,
		expectedMaps: map[string]string{},
	}

	// Set development Logger so we can see all logs in tests.
	logf.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	// Create a ReconcileMemcached object with the scheme and fake client.
	r := &XdpProgramReconciler{ReconcilerCommon: rc, ourNode: fakeNode}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	// First reconcile should create the bpf program object
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Check the BpfProgram Object was created successfully
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName, Namespace: metav1.NamespaceAll}, bpfProg)
	require.NoError(t, err)

	require.NotEmpty(t, bpfProg)
	// Finalizer is written
	require.Equal(t, r.getFinalizer(), bpfProg.Finalizers[0])
	// owningConfig Label was correctly set
	require.Equal(t, bpfProg.Labels[internal.BpfProgramOwnerLabel], name)
	// node Label was correctly set
	require.Equal(t, bpfProg.Labels[internal.K8sHostLabel], fakeNode.Name)
	// Type is set
	require.Equal(t, r.getRecType(), bpfProg.Spec.Type)
	// Require no requeue
	require.False(t, res.Requeue)

	// Update UID of bpfProgram with Fake UID since the fake API server won't
	bpfProg.UID = types.UID(fakeUID)
	err = cl.Update(ctx, bpfProg)
	require.NoError(t, err)

	// Second reconcile should create the bpfd Load Request and update the
	// BpfProgram object's 'Programs' field.
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)
	id := string(bpfProg.UID)
	mapOwnerUuid := ""

	expectedLoadReq := &gobpfd.LoadRequest{
		Common: &gobpfd.LoadRequestCommon{
			Location: &gobpfd.LoadRequestCommon_File{
				File: bytecodePath,
			},
			SectionName:  sectionName,
			ProgramType:  *internal.Xdp.Uint32(),
			Id:           &id,
			MapOwnerUuid: &mapOwnerUuid,
		},
		AttachInfo: &gobpfd.LoadRequest_XdpAttachInfo{
			XdpAttachInfo: &gobpfd.XDPAttachInfo{
				Iface:     fakeInt,
				Priority:  0,
				ProceedOn: []int32{2, 31},
			},
		},
	}
	// Check the bpfLoadRequest was correctly Built
	if !cmp.Equal(expectedLoadReq, cli.LoadRequests[id], protocmp.Transform()) {
		t.Logf("Diff %v", cmp.Diff(expectedLoadReq, cli.LoadRequests[id], protocmp.Transform()))
		t.Fatal("Built bpfd LoadRequest does not match expected")
	}

	// Check that the bpfProgram's programs was correctly updated
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName, Namespace: metav1.NamespaceAll}, bpfProg)
	require.NoError(t, err)

	require.Nil(t, bpfProg.Spec.Maps)

	// Third reconcile should update the bpfPrograms status to loaded
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)

	// Check that the bpfProgram's status was correctly updated
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName, Namespace: metav1.NamespaceAll}, bpfProg)
	require.NoError(t, err)

	require.Equal(t, string(bpfdiov1alpha1.BpfProgCondLoaded), bpfProg.Status.Conditions[0].Type)
}

func TestXdpProgramControllerCreateMultiIntf(t *testing.T) {
	var (
		name         = "fakeXdpProgram"
		namespace    = "bpfd"
		bytecodePath = "/tmp/hello.o"
		sectionName  = "test"
		fakeNode     = testutils.NewNode("fake-control-plane")
		fakeInts     = []string{"eth0", "eth1"}
		ctx          = context.TODO()
		bpfProgName0 = fmt.Sprintf("%s-%s-%s", name, fakeNode.Name, fakeInts[0])
		bpfProgName1 = fmt.Sprintf("%s-%s-%s", name, fakeNode.Name, fakeInts[1])
		bpfProgEth0  = &bpfdiov1alpha1.BpfProgram{}
		bpfProgEth1  = &bpfdiov1alpha1.BpfProgram{}
		fakeUID0     = "ef71d42c-aa21-48e8-a697-82391d801a80"
		fakeUID1     = "ef71d42c-aa21-48e8-a697-82391d801a81"
	)
	// A XdpProgram object with metadata and spec.
	xdp := &bpfdiov1alpha1.XdpProgram{
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
				Interfaces: &fakeInts,
			},
			Priority: 0,
			ProceedOn: []bpfdiov1alpha1.XdpProceedOnValue{bpfdiov1alpha1.XdpProceedOnValue("pass"),
				bpfdiov1alpha1.XdpProceedOnValue("dispatcher_return"),
			},
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{fakeNode, xdp}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, xdp)
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.XdpProgramList{})
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.BpfProgram{})
	s.AddKnownTypes(bpfdiov1alpha1.SchemeGroupVersion, &bpfdiov1alpha1.BpfProgramList{})

	// Create a fake client to mock API calls.
	cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()

	cli := agenttestutils.NewBpfdClientFake()

	rc := ReconcilerCommon{
		Client:       cl,
		Scheme:       s,
		BpfdClient:   cli,
		NodeName:     fakeNode.Name,
		expectedMaps: map[string]string{},
	}

	// Set development Logger so we can see all logs in tests.
	logf.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	// Create a ReconcileMemcached object with the scheme and fake client.
	r := &XdpProgramReconciler{ReconcilerCommon: rc, ourNode: fakeNode}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	// First reconcile should create the first bpf program object
	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Check the first BpfProgram Object was created successfully
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName0, Namespace: metav1.NamespaceAll}, bpfProgEth0)
	require.NoError(t, err)

	require.NotEmpty(t, bpfProgEth0)
	// owningConfig Label was correctly set
	require.Equal(t, bpfProgEth0.Labels[internal.BpfProgramOwnerLabel], name)
	// node Label was correctly set
	require.Equal(t, bpfProgEth0.Labels[internal.K8sHostLabel], fakeNode.Name)
	// Finalizer is written
	require.Equal(t, r.getFinalizer(), bpfProgEth0.Finalizers[0])
	// Type is set
	require.Equal(t, r.getRecType(), bpfProgEth0.Spec.Type)
	// Require no requeue
	require.False(t, res.Requeue)

	// Update UID of bpfProgram with Fake UID since the fake API server won't
	bpfProgEth0.UID = types.UID(fakeUID0)
	err = cl.Update(ctx, bpfProgEth0)
	require.NoError(t, err)

	// Second reconcile should create the bpfd Load Requests for the first bpfProgram.
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)

	// Third reconcile should create the second bpf program object
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Check the Second BpfProgram Object was created successfully
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName1, Namespace: metav1.NamespaceAll}, bpfProgEth1)
	require.NoError(t, err)

	require.NotEmpty(t, bpfProgEth1)
	// owningConfig Label was correctly set
	require.Equal(t, bpfProgEth1.Labels[internal.BpfProgramOwnerLabel], name)
	// node Label was correctly set
	require.Equal(t, bpfProgEth1.Labels[internal.K8sHostLabel], fakeNode.Name)
	// Finalizer is written
	require.Equal(t, r.getFinalizer(), bpfProgEth1.Finalizers[0])
	// Type is set
	require.Equal(t, r.getRecType(), bpfProgEth1.Spec.Type)
	// Require no requeue
	require.False(t, res.Requeue)

	// Update UID of bpfProgram with Fake UID since the fake API server won't
	bpfProgEth1.UID = types.UID(fakeUID1)
	err = cl.Update(ctx, bpfProgEth1)
	require.NoError(t, err)

	// Fourth reconcile should create the bpfd Load Requests for the second bpfProgram.
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)

	id0 := string(bpfProgEth0.UID)
	mapOwnerUuid := ""

	expectedLoadReq0 := &gobpfd.LoadRequest{
		Common: &gobpfd.LoadRequestCommon{
			Location: &gobpfd.LoadRequestCommon_File{
				File: bytecodePath,
			},
			SectionName:  sectionName,
			ProgramType:  *internal.Xdp.Uint32(),
			Id:           &id0,
			MapOwnerUuid: &mapOwnerUuid,
		},
		AttachInfo: &gobpfd.LoadRequest_XdpAttachInfo{
			XdpAttachInfo: &gobpfd.XDPAttachInfo{
				Iface:     fakeInts[0],
				Priority:  0,
				ProceedOn: []int32{2, 31},
			},
		},
	}

	id1 := string(bpfProgEth1.UID)

	expectedLoadReq1 := &gobpfd.LoadRequest{
		Common: &gobpfd.LoadRequestCommon{
			Location: &gobpfd.LoadRequestCommon_File{
				File: bytecodePath,
			},
			SectionName:  sectionName,
			ProgramType:  *internal.Xdp.Uint32(),
			Id:           &id1,
			MapOwnerUuid: &mapOwnerUuid,
		},
		AttachInfo: &gobpfd.LoadRequest_XdpAttachInfo{
			XdpAttachInfo: &gobpfd.XDPAttachInfo{
				Iface:     fakeInts[1],
				Priority:  0,
				ProceedOn: []int32{2, 31},
			},
		},
	}

	// Check the bpfLoadRequest was correctly built
	if !cmp.Equal(expectedLoadReq0, cli.LoadRequests[id0], protocmp.Transform()) {
		t.Logf("Diff %v", cmp.Diff(expectedLoadReq0, cli.LoadRequests[id0], protocmp.Transform()))
		t.Fatal("Built bpfd LoadRequest does not match expected")
	}

	// Check the bpfLoadRequest was correctly built
	if !cmp.Equal(expectedLoadReq1, cli.LoadRequests[id1], protocmp.Transform()) {
		t.Logf("Diff %v", cmp.Diff(expectedLoadReq1, cli.LoadRequests[id1], protocmp.Transform()))
		t.Fatal("Built bpfd LoadRequest does not match expected")
	}

	// Check that the bpfProgram's maps was correctly updated
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName0, Namespace: metav1.NamespaceAll}, bpfProgEth0)
	require.NoError(t, err)

	require.Nil(t, bpfProgEth0.Spec.Maps)

	// Check that the bpfProgram's maps was correctly updated
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName1, Namespace: metav1.NamespaceAll}, bpfProgEth1)
	require.NoError(t, err)

	require.Nil(t, bpfProgEth1.Spec.Maps)

	// Third reconcile should update the bpfPrograms status to loaded
	res, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Require no requeue
	require.False(t, res.Requeue)

	// Check that the bpfProgram's status was correctly updated
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName0, Namespace: metav1.NamespaceAll}, bpfProgEth0)
	require.NoError(t, err)

	require.Equal(t, string(bpfdiov1alpha1.BpfProgCondLoaded), bpfProgEth0.Status.Conditions[0].Type)

	// Check that the bpfProgram's status was correctly updated
	err = cl.Get(ctx, types.NamespacedName{Name: bpfProgName1, Namespace: metav1.NamespaceAll}, bpfProgEth1)
	require.NoError(t, err)

	require.Equal(t, string(bpfdiov1alpha1.BpfProgCondLoaded), bpfProgEth1.Status.Conditions[0].Type)
}
