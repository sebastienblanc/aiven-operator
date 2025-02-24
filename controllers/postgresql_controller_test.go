// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PostgreSQL Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		pg          *v1alpha1.PostgreSQL
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-pg-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)

		By("Creating a new PostgreSQL CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdPostgreSQL := &v1alpha1.PostgreSQL{}
		// We'll need to retry getting this newly created PostgreSQL,
		// given that creation may not immediately happen.
		By("by retrieving PostgreSQL instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, pgLookupKey, createdPostgreSQL)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting PostgreSQL service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, pgLookupKey, createdPostgreSQL)
			if err == nil {
				return meta.IsStatusConditionTrue(createdPostgreSQL.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdPostgreSQL.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating PostgreSQL reconciler behaviour", func() {
		It("should createOrUpdate a new PostgreSQL service", func() {
			createdPostgreSQL := &v1alpha1.PostgreSQL{}
			pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, pgLookupKey, createdPostgreSQL)).Should(Succeed())

			By("by checking that after creation of a PostreSQL service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["PGHOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGPORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGDATABASE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGUSER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGPASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGSSLMODE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["DATABASE_URI"]).NotTo(BeEmpty())

			Expect(createdPostgreSQL.Status.State).Should(Equal("RUNNING"))
		})
	})

	AfterEach(func() {
		By("Ensures that PostgreSQL instance was deleted")
		ensureDelete(ctx, pg)
	})
})

func pgSpec(serviceName, namespace string) *v1alpha1.PostgreSQL {
	return &v1alpha1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "PostgreSQL",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.PostgreSQLSpec{
			DiskSpace: "100Gib",
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: v1alpha1.PostgreSQLUserconfig{
				PublicAccess: v1alpha1.PublicAccessUserConfig{
					Pg:         boolPointer(true),
					Prometheus: boolPointer(true),
				},
				Pg: v1alpha1.PostgreSQLSubUserConfig{
					IdleInTransactionSessionTimeout: int64Pointer(900),
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
