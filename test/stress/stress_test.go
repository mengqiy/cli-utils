// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package stress

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/test/e2e/e2eutil"
	"sigs.k8s.io/cli-utils/test/e2e/invconfig"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Parse optional logging flags
// Ex: ginkgo ./test/e2e/... -- -v=5
// Allow init for e2e test (not imported by external code)
// nolint:gochecknoinits
func init() {
	klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)
}

var defaultTestTimeout = 1 * time.Hour
var defaultBeforeTestTimeout = 30 * time.Second
var defaultAfterTestTimeout = 30 * time.Second

var _ = Describe("Applier", func() {

	var c client.Client
	var invConfig invconfig.InventoryConfig

	BeforeSuite(func() {
		// increase from 4000 to handle long event lists
		format.MaxLength = 10000

		cfg, err := ctrl.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		// increase QPS from 5 to 20
		cfg.QPS = 20
		// increase Burst QPS from 10 to 40
		cfg.Burst = 40

		invConfig = invconfig.NewCustomTypeInvConfig(cfg)

		mapper, err := apiutil.NewDynamicRESTMapper(cfg)
		Expect(err).NotTo(HaveOccurred())

		c, err = client.New(cfg, client.Options{
			Scheme: scheme.Scheme,
			Mapper: mapper,
		})
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithTimeout(context.Background(), defaultBeforeTestTimeout)
		defer cancel()
		e2eutil.CreateInventoryCRD(ctx, c)
		Expect(ctx.Err()).To(BeNil(), "BeforeSuite context cancelled or timed out")
	})

	AfterSuite(func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultAfterTestTimeout)
		defer cancel()
		e2eutil.DeleteInventoryCRD(ctx, c)
		Expect(ctx.Err()).To(BeNil(), "AfterSuite context cancelled or timed out")
	})

	Context("StressTest", func() {
		var namespace *v1.Namespace
		var inventoryName string
		var ctx context.Context
		var cancel context.CancelFunc

		BeforeEach(func() {
			ctx, cancel = context.WithTimeout(context.Background(), defaultTestTimeout)
			inventoryName = e2eutil.RandomString("test-inv-")
			namespace = e2eutil.CreateRandomNamespace(ctx, c)
		})

		AfterEach(func() {
			Expect(ctx.Err()).To(BeNil(), "test context cancelled or timed out")
			cancel()
			ctx, cancel = context.WithTimeout(context.Background(), defaultAfterTestTimeout)
			defer cancel()
			// clean up resources created by the tests
			e2eutil.DeleteNamespace(ctx, c, namespace)
		})

		It("ThousandNamespaces", func() {
			thousandNamespacesTest(ctx, c, invConfig, inventoryName, namespace.GetName())
		})
	})
})
