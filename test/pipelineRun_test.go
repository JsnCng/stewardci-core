// +build e2e

package test

import (
	"fmt"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
)

type pipelineRunTest struct {
	name        string
	pipelineRun *api.PipelineRun
	check       PipelineRunCheck
}

type pipelineRunTestBuilder = func(string) pipelineRunTest

type testPlan struct {
	testBuilder pipelineRunTestBuilder
	parallel    int
}

func PipelineRunSleep(namespace string) pipelineRunTest {
	return pipelineRunTest{
		name: "sleep",
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "1"),
			)),
		check: PipelineRunHasStateResult(api.ResultSuccess),
	}
}

func PipelineRunFail(namespace string) pipelineRunTest {
	return pipelineRunTest{
		name: "error",
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"error/Jenkinsfile"),
			)),
		check: PipelineRunHasStateResult(api.ResultErrorContent),
	}
}

func PipelineRunOK(namespace string) pipelineRunTest {
	return pipelineRunTest{
		name: "ok",
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"success/Jenkinsfile"),
			)),
		check: PipelineRunHasStateResult(api.ResultSuccess),
	}
}

func Test_PipelineRuns(t *testing.T) {
	executePipelineRunTests(t,
		testPlan{testBuilder: PipelineRunSleep,
			parallel: 1,
		},
		testPlan{testBuilder: PipelineRunFail,
			parallel: 2,
		},
		testPlan{testBuilder: PipelineRunOK,
			parallel: 3,
		},
	)
}

func Test_PipelineRunSuccess(t *testing.T) {
	executeSingleTest(t, PipelineRunOK)
}

func executeSingleTest(t *testing.T, testBuilder pipelineRunTestBuilder) {
	t.Parallel()
	clientFactory, namespace, waiter := setup(t)
	test := TenantSuccessTest(namespace)
	tenant := test.tenant
	tenant, err := CreateTenant(clientFactory, tenant)
	assert.NilError(t, err)

	defer DeleteTenant(clientFactory, tenant)
	check := CreateTenantCondition(tenant, test.check, test.name)
	err = waiter.WaitFor(check)
	assert.NilError(t, err)

	tenant, err = GetTenant(clientFactory, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName
	pipelineTest := testBuilder(tnn)
	pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
	assert.NilError(t, err)
	pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, pipelineTest.name)
	err = waiter.WaitFor(pipelineRunCheck)
	assert.NilError(t, err)
}

func executePipelineRunTests(t *testing.T, testPlans ...testPlan) {
	t.Parallel()
	clientFactory, namespace, waiter := setup(t)
	test := TenantSuccessTest(namespace)
	tenant := test.tenant
	tenant, err := CreateTenant(clientFactory, tenant)
	assert.NilError(t, err)

	defer DeleteTenant(clientFactory, tenant)
	check := CreateTenantCondition(tenant, test.check, test.name)
	err = waiter.WaitFor(check)
	assert.NilError(t, err)
	tenant, err = GetTenant(clientFactory, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName
	for _, testPlan := range testPlans {
		testPlan := testPlan
		for i := 1; i <= testPlan.parallel; i++ {
			testBuilder := testPlan.testBuilder
			pipelineTest := testBuilder(tnn)
			pipelineTest.name =
				fmt.Sprintf("%s_%d", pipelineTest.name, i)
				//    t.Run(pipelineTest.name,func(t *testing.T) {
				//      pipelineTest := pipelineTest
				//      t.Parallel()
			pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
			assert.NilError(t, err)
			pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, pipelineTest.name)
			err = waiter.WaitFor(pipelineRunCheck)
			assert.NilError(t, err)
			//	})
		}
	}
}
